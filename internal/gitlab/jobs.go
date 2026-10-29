package gitlab

import (
	"context"
	"fmt"
	"net/http"
)

const MaxJobTraceBytes = 1 << 20 // 1 MiB

func (c *Client) GetJob(ctx context.Context, project string, jobID int) (*Job, error) {
	var job Job
	path := fmt.Sprintf("%s/jobs/%d", projectPath(project), jobID)
	_, err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &job)
	return &job, err
}

// GetJobTrace reads a job's plain-text log. Oversized logs fail explicitly
// instead of returning an incomplete log that could hide the cause of a failure.
func (c *Client) GetJobTrace(ctx context.Context, project string, jobID int) (string, error) {
	path := fmt.Sprintf("%s/jobs/%d/trace", projectPath(project), jobID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/plain")
	resp, body, err := c.do(req, MaxJobTraceBytes)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", parseAPIError(resp.StatusCode, body)
	}
	return string(body), nil
}
