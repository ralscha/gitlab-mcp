package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

type ListPipelinesOptions struct {
	Status       string
	Ref          string
	SHA          string
	Source       string
	Username     string
	OrderBy      string
	Sort         string
	UpdatedAfter string
	Page         int
	PerPage      int
}

func (c *Client) ListPipelines(ctx context.Context, project string, opts ListPipelinesOptions) ([]Pipeline, PageInfo, error) {
	query := pageQuery(opts.Page, opts.PerPage)
	setIf(query, "status", opts.Status)
	setIf(query, "ref", opts.Ref)
	setIf(query, "sha", opts.SHA)
	setIf(query, "source", opts.Source)
	setIf(query, "username", opts.Username)
	setIf(query, "order_by", opts.OrderBy)
	setIf(query, "sort", opts.Sort)
	setIf(query, "updated_after", opts.UpdatedAfter)
	var pipelines []Pipeline
	headers, err := c.doJSON(ctx, http.MethodGet, projectPath(project)+"/pipelines", query, nil, &pipelines)
	return pipelines, pagination(headers, opts.Page, opts.PerPage, len(pipelines)), err
}

func (c *Client) GetPipeline(ctx context.Context, project string, pipelineID int) (*Pipeline, error) {
	var pipeline Pipeline
	path := fmt.Sprintf("%s/pipelines/%d", projectPath(project), pipelineID)
	_, err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &pipeline)
	return &pipeline, err
}

func (c *Client) ListPipelineJobs(ctx context.Context, project string, pipelineID int, scopes []string, includeRetried bool, page, perPage int) ([]Job, PageInfo, error) {
	query := pageQuery(page, perPage)
	for _, scope := range scopes {
		query.Add("scope[]", scope)
	}
	if includeRetried {
		query.Set("include_retried", "true")
	}
	path := fmt.Sprintf("%s/pipelines/%d/jobs", projectPath(project), pipelineID)
	var jobs []Job
	headers, err := c.doJSON(ctx, http.MethodGet, path, query, nil, &jobs)
	return jobs, pagination(headers, page, perPage, len(jobs)), err
}

type PipelineVariable struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	VariableType string `json:"variable_type,omitempty"`
}

func (c *Client) CreatePipeline(ctx context.Context, project, ref string, variables []PipelineVariable, inputs map[string]any) (*Pipeline, error) {
	body := map[string]any{"ref": ref}
	if len(variables) > 0 {
		body["variables"] = variables
	}
	if len(inputs) > 0 {
		body["inputs"] = inputs
	}
	var pipeline Pipeline
	_, err := c.doJSON(ctx, http.MethodPost, projectPath(project)+"/pipeline", nil, body, &pipeline)
	return &pipeline, err
}

func (c *Client) RetryPipeline(ctx context.Context, project string, pipelineID int) (*Pipeline, error) {
	var pipeline Pipeline
	path := fmt.Sprintf("%s/pipelines/%s/retry", projectPath(project), strconv.Itoa(pipelineID))
	_, err := c.doJSON(ctx, http.MethodPost, path, nil, nil, &pipeline)
	return &pipeline, err
}
