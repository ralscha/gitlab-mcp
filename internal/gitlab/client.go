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
		if attempt >= maxRetries || !isRetryable(req, resp.StatusCode) {
			return resp, body, nil
		}
		timer := time.NewTimer(retryDelay(resp, attempt))
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, nil, fmt.Errorf("gitlab: request failed: %w", req.Context().Err())
		case <-timer.C:
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

func isRetryable(req *http.Request, status int) bool {
	// A rate-limited request has not been accepted and is safe to retry. For
	// ambiguous gateway failures, retry only methods defined as idempotent so a
	// create operation cannot be replayed after GitLab already processed it.
	if status == http.StatusTooManyRequests {
		return req.Body == nil || req.GetBody != nil
	}
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		switch req.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
			return req.Body == nil || req.GetBody != nil
		}
	default:
	}
	return false
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
	page, perPage = normalizePage(page, perPage)
	return url.Values{"page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
}

func normalizePage(page, perPage int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	perPage = min(perPage, 100)
	return page, perPage
}

func pagination(headers http.Header, page, perPage, resultCount int) PageInfo {
	page, perPage = normalizePage(page, perPage)
	info := PageInfo{Page: page, PerPage: perPage}
	setFromHeader(&info.Page, headers, "X-Page")
	setFromHeader(&info.PerPage, headers, "X-Per-Page")
	info.Total, _ = strconv.Atoi(headers.Get("X-Total"))
	info.TotalPages, _ = strconv.Atoi(headers.Get("X-Total-Pages"))
	info.NextPage, _ = strconv.Atoi(headers.Get("X-Next-Page"))
	info.PreviousPage, _ = strconv.Atoi(headers.Get("X-Prev-Page"))
	nextLinkPage, hasNextLink := linkedPage(headers.Get("Link"), "next")
	if info.NextPage == 0 {
		info.NextPage = nextLinkPage
	}
	if info.PreviousPage == 0 {
		info.PreviousPage, _ = linkedPage(headers.Get("Link"), "prev")
	}
	if info.TotalPages == 0 {
		info.TotalPages, _ = linkedPage(headers.Get("Link"), "last")
	}
	hasNextHeader := headerPresent(headers, "X-Next-Page")
	switch {
	case info.NextPage > 0 || hasNextLink:
		info.LastPage = false
	case info.TotalPages > 0:
		info.LastPage = info.Page >= info.TotalPages
	case hasNextHeader && strings.TrimSpace(headers.Get("X-Next-Page")) == "":
		info.LastPage = true
	case headers.Get("Link") != "":
		info.LastPage = true
	default:
		// Without pagination headers, a full page is ambiguous. Do not claim it
		// is the last page and risk making callers stop early.
		info.LastPage = resultCount < info.PerPage
	}
	return info
}

func setFromHeader(target *int, headers http.Header, name string) {
	if value, err := strconv.Atoi(headers.Get(name)); err == nil && value > 0 {
		*target = value
	}
}

func headerPresent(headers http.Header, name string) bool {
	_, ok := headers[http.CanonicalHeaderKey(name)]
	return ok
}

func linkedPage(linkHeader, relation string) (int, bool) {
	for _, link := range strings.Split(linkHeader, ",") {
		parts := strings.Split(link, ";")
		matched := false
		for _, parameter := range parts[1:] {
			name, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if ok && strings.EqualFold(name, "rel") && strings.EqualFold(strings.Trim(value, `"`), relation) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		target := strings.Trim(strings.TrimSpace(parts[0]), "<>")
		parsed, err := url.Parse(target)
		if err != nil {
			return 0, true
		}
		page, _ := strconv.Atoi(parsed.Query().Get("page"))
		return page, true
	}
	return 0, false
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
