// Package gitlab implements the subset of GitLab REST API v4 used by the
// MCP server. It intentionally uses the stable REST contract directly so it
// works with GitLab.com and current GitLab Self-Managed releases.
package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	MaxFileBytes         = 25 << 20 // 25 MiB
	maxJSONResponseBytes = 40 << 20
	maxRetries           = 3
	maxRetryDelay        = 30 * time.Second
)

var ErrTooLarge = errors.New("gitlab: payload exceeds size limit")

type Client struct {
	httpClient *http.Client
	apiBaseURL *url.URL
	token      string
}

// NewClient creates a GitLab REST API v4 client. baseURL is the instance URL,
// such as https://gitlab.com or https://gitlab.example.com/gitlab.
func NewClient(baseURL, token string, httpClient *http.Client) (*Client, error) {
	instance, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("gitlab: invalid base URL: %w", err)
	}
	if !instance.IsAbs() || instance.Host == "" {
		return nil, errors.New("gitlab: base URL must be absolute")
	}
	if instance.User != nil || instance.RawQuery != "" || instance.Fragment != "" {
		return nil, errors.New("gitlab: base URL must not contain user information, a query string, or a fragment")
	}
	instance.Path = strings.TrimRight(instance.Path, "/") + "/api/v4/"
	instance.RawPath = ""
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	clientCopy := *httpClient
	originalCheckRedirect := clientCopy.CheckRedirect
	clientCopy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != instance.Scheme || !strings.EqualFold(req.URL.Host, instance.Host) {
			return fmt.Errorf("gitlab: refusing cross-origin redirect to %s", req.URL.Redacted())
		}
		if originalCheckRedirect != nil {
			return originalCheckRedirect(req, via)
		}
		if len(via) >= 10 {
			return errors.New("gitlab: stopped after 10 redirects")
		}
		return nil
	}
	return &Client{httpClient: &clientCopy, apiBaseURL: instance, token: token}, nil
}

type APIError struct {
	StatusCode int
	Message    string
	Details    map[string]any
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("gitlab: request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("gitlab: request failed with status %d: %s", e.StatusCode, e.Message)
}

func (c *Client) do(req *http.Request, maxBytes int64) (*http.Response, []byte, error) {
	for attempt := 0; ; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("gitlab: request failed: %w", err)
		}
		body, readErr := readLimited(resp.Body, maxBytes)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, nil, readErr
		}
		if attempt >= maxRetries || !isRetryable(resp.StatusCode) {
			return resp, body, nil
		}
		select {
		case <-req.Context().Done():
			return nil, nil, fmt.Errorf("gitlab: request failed: %w", req.Context().Err())
		case <-time.After(retryDelay(resp, attempt)):
		}
		if req.GetBody != nil {
			rewound, err := req.GetBody()
			if err != nil {
				return nil, nil, fmt.Errorf("gitlab: rewinding request body for retry: %w", err)
			}
			req.Body = rewound
		}
	}
}

func isRetryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if value := strings.TrimSpace(resp.Header.Get("Retry-After")); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
			return min(time.Duration(seconds)*time.Second, maxRetryDelay)
		}
		if at, err := http.ParseTime(value); err == nil {
			return min(max(time.Until(at), 0), maxRetryDelay)
		}
	}
	return min(500*time.Millisecond<<attempt, maxRetryDelay)
}

func readLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("gitlab: reading response body: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%w: response larger than %d bytes", ErrTooLarge, maxBytes)
	}
	return body, nil
}

// doJSON sends a JSON request and returns response headers, which GitLab uses
// for pagination metadata.
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body, out any) (http.Header, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("gitlab: encoding request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := c.newRequest(ctx, method, path, query, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	resp, respBody, err := c.do(req, maxJSONResponseBytes)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, parseAPIError(resp.StatusCode, respBody)
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return resp.Header, fmt.Errorf("gitlab: decoding response body: %w", err)
		}
	}
	return resp.Header, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Request, error) {
	relative, err := url.Parse(strings.TrimLeft(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("gitlab: invalid API path: %w", err)
	}
	endpoint := c.apiBaseURL.ResolveReference(relative)
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("gitlab: building request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	return req, nil
}

func parseAPIError(status int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status}
	var payload map[string]any
	if json.Unmarshal(body, &payload) == nil {
		apiErr.Details = payload
		if message, ok := payload["message"].(string); ok {
			apiErr.Message = message
		} else if message := payload["message"]; message != nil {
			if encoded, err := json.Marshal(message); err == nil {
				apiErr.Message = string(encoded)
			}
		} else if description, ok := payload["error_description"].(string); ok {
			apiErr.Message = description
		} else if message, ok := payload["error"].(string); ok {
			apiErr.Message = message
		}
	}
	if apiErr.Message == "" {
		apiErr.Message = strings.TrimSpace(string(body))
	}
	return apiErr
}

func projectPath(project string) string { return "projects/" + url.PathEscape(project) }

func pageQuery(page, perPage int) url.Values {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	perPage = min(perPage, 100)
	return url.Values{"page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
}

func pagination(headers http.Header, page, perPage, resultCount int) PageInfo {
	info := PageInfo{Page: page, PerPage: perPage}
	info.Total, _ = strconv.Atoi(headers.Get("X-Total"))
	info.TotalPages, _ = strconv.Atoi(headers.Get("X-Total-Pages"))
	info.NextPage, _ = strconv.Atoi(headers.Get("X-Next-Page"))
	info.PreviousPage, _ = strconv.Atoi(headers.Get("X-Prev-Page"))
	if info.Page <= 0 {
		info.Page = 1
	}
	if info.PerPage <= 0 {
		info.PerPage = 20
	}
	if info.TotalPages == 0 && info.NextPage == 0 && resultCount < info.PerPage {
		info.LastPage = true
	} else {
		info.LastPage = info.NextPage == 0
	}
	return info
}

func setIf(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func setBool(query url.Values, key string, value *bool) {
	if value != nil {
		query.Set(key, strconv.FormatBool(*value))
	}
}
