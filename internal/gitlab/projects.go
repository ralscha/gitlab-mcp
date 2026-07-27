package gitlab

import (
	"context"
	"net/http"
	"strconv"
)

type ListProjectsOptions struct {
	Search     string
	Membership *bool
	Owned      *bool
	Archived   *bool
	Visibility string
	OrderBy    string
	Sort       string
	Page       int
	PerPage    int
}

func (c *Client) ListProjects(ctx context.Context, opts ListProjectsOptions) ([]Project, PageInfo, error) {
	query := pageQuery(opts.Page, opts.PerPage)
	setIf(query, "search", opts.Search)
	setIf(query, "visibility", opts.Visibility)
	setIf(query, "order_by", opts.OrderBy)
	setIf(query, "sort", opts.Sort)
	setBool(query, "membership", opts.Membership)
	setBool(query, "owned", opts.Owned)
	setBool(query, "archived", opts.Archived)
	var projects []Project
	headers, err := c.doJSON(ctx, http.MethodGet, "projects", query, nil, &projects)
	return projects, pagination(headers, opts.Page, opts.PerPage, len(projects)), err
}

func (c *Client) GetProject(ctx context.Context, project string) (*Project, error) {
	var result Project
	_, err := c.doJSON(ctx, http.MethodGet, projectPath(project), nil, nil, &result)
	return &result, err
}

func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	var result User
	_, err := c.doJSON(ctx, http.MethodGet, "user", nil, nil, &result)
	return &result, err
}

func (c *Client) Metadata(ctx context.Context) (*Metadata, error) {
	var result Metadata
	_, err := c.doJSON(ctx, http.MethodGet, "metadata", nil, nil, &result)
	return &result, err
}

type SearchOptions struct {
	Scope   string
	Search  string
	Project string
	GroupID int
	OrderBy string
	Sort    string
	Page    int
	PerPage int
}

// Search runs the GitLab search endpoint. Results remain untyped because the
// response shape depends on scope (projects, issues, merge_requests, blobs,
// commits, notes, users, and other GitLab-supported scopes).
func (c *Client) Search(ctx context.Context, opts SearchOptions) ([]map[string]any, PageInfo, error) {
	query := pageQuery(opts.Page, opts.PerPage)
	query.Set("scope", opts.Scope)
	query.Set("search", opts.Search)
	setIf(query, "order_by", opts.OrderBy)
	setIf(query, "sort", opts.Sort)
	path := "search"
	if opts.Project != "" {
		path = projectPath(opts.Project) + "/search"
	} else if opts.GroupID > 0 {
		path = "groups/" + strconv.Itoa(opts.GroupID) + "/search"
	}
	var results []map[string]any
	headers, err := c.doJSON(ctx, http.MethodGet, path, query, nil, &results)
	return results, pagination(headers, opts.Page, opts.PerPage, len(results)), err
}

func joinComma(values []string) string {
	result := ""
	for i, value := range values {
		if i > 0 {
			result += ","
		}
		result += value
	}
	return result
}
