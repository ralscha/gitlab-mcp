package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

type ListIssuesOptions struct {
	Project      string
	State        string
	Scope        string
	Search       string
	Labels       []string
	AssigneeID   int
	AuthorID     int
	Milestone    string
	OrderBy      string
	Sort         string
	UpdatedAfter string
	Page         int
	PerPage      int
}

func (c *Client) ListIssues(ctx context.Context, opts ListIssuesOptions) ([]Issue, PageInfo, error) {
	query := pageQuery(opts.Page, opts.PerPage)
	setIf(query, "state", opts.State)
	setIf(query, "scope", opts.Scope)
	setIf(query, "search", opts.Search)
	setIf(query, "labels", joinComma(opts.Labels))
	setIf(query, "milestone", opts.Milestone)
	setIf(query, "order_by", opts.OrderBy)
	setIf(query, "sort", opts.Sort)
	setIf(query, "updated_after", opts.UpdatedAfter)
	if opts.AssigneeID > 0 {
		query.Set("assignee_id", strconv.Itoa(opts.AssigneeID))
	}
	if opts.AuthorID > 0 {
		query.Set("author_id", strconv.Itoa(opts.AuthorID))
	}
	path := "issues"
	if opts.Project != "" {
		path = projectPath(opts.Project) + "/issues"
	}
	var issues []Issue
	headers, err := c.doJSON(ctx, http.MethodGet, path, query, nil, &issues)
	return issues, pagination(headers, opts.Page, opts.PerPage, len(issues)), err
}

func (c *Client) GetIssue(ctx context.Context, project string, iid int) (*Issue, error) {
	var issue Issue
	path := fmt.Sprintf("%s/issues/%d", projectPath(project), iid)
	_, err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &issue)
	return &issue, err
}

type CreateIssueInput struct {
	Title        string
	Description  string
	Confidential bool
	Labels       []string
	AssigneeIDs  []int
	MilestoneID  int
	DueDate      string
	IssueType    string
	Weight       int
}

func (c *Client) CreateIssue(ctx context.Context, project string, in CreateIssueInput) (*Issue, error) {
	body := map[string]any{"title": in.Title}
	if in.Description != "" {
		body["description"] = in.Description
	}
	if in.Confidential {
		body["confidential"] = true
	}
	if len(in.Labels) > 0 {
		body["labels"] = joinComma(in.Labels)
	}
	if len(in.AssigneeIDs) > 0 {
		body["assignee_ids"] = in.AssigneeIDs
	}
	if in.MilestoneID > 0 {
		body["milestone_id"] = in.MilestoneID
	}
	if in.DueDate != "" {
		body["due_date"] = in.DueDate
	}
	if in.IssueType != "" {
		body["issue_type"] = in.IssueType
	}
	if in.Weight > 0 {
		body["weight"] = in.Weight
	}
	var issue Issue
	_, err := c.doJSON(ctx, http.MethodPost, projectPath(project)+"/issues", nil, body, &issue)
	return &issue, err
}

type UpdateIssueInput struct {
	Title        *string
	Description  *string
	Confidential *bool
	Labels       *[]string
	AssigneeIDs  *[]int
	MilestoneID  *int
	DueDate      *string
	IssueType    *string
	Weight       *int
	StateEvent   *string
}

func (c *Client) UpdateIssue(ctx context.Context, project string, iid int, in UpdateIssueInput) (*Issue, error) {
	body := map[string]any{}
	if in.Title != nil {
		body["title"] = *in.Title
	}
	if in.Description != nil {
		body["description"] = *in.Description
	}
	if in.Confidential != nil {
		body["confidential"] = *in.Confidential
	}
	if in.Labels != nil {
		body["labels"] = joinComma(*in.Labels)
	}
	if in.AssigneeIDs != nil {
		body["assignee_ids"] = *in.AssigneeIDs
	}
	if in.MilestoneID != nil {
		body["milestone_id"] = *in.MilestoneID
	}
	if in.DueDate != nil {
		body["due_date"] = *in.DueDate
	}
	if in.IssueType != nil {
		body["issue_type"] = *in.IssueType
	}
	if in.Weight != nil {
		body["weight"] = *in.Weight
	}
	if in.StateEvent != nil {
		body["state_event"] = *in.StateEvent
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("gitlab: UpdateIssue requires at least one field")
	}
	var issue Issue
	path := fmt.Sprintf("%s/issues/%d", projectPath(project), iid)
	_, err := c.doJSON(ctx, http.MethodPut, path, nil, body, &issue)
	return &issue, err
}

func (c *Client) ListIssueNotes(ctx context.Context, project string, iid, page, perPage int, sort, orderBy string) ([]Note, PageInfo, error) {
	query := pageQuery(page, perPage)
	setIf(query, "sort", sort)
	setIf(query, "order_by", orderBy)
	path := fmt.Sprintf("%s/issues/%d/notes", projectPath(project), iid)
	var notes []Note
	headers, err := c.doJSON(ctx, http.MethodGet, path, query, nil, &notes)
	return notes, pagination(headers, page, perPage, len(notes)), err
}

func (c *Client) AddIssueNote(ctx context.Context, project string, iid int, body string, internal bool) (*Note, error) {
	payload := map[string]any{"body": body}
	if internal {
		payload["internal"] = true
	}
	path := fmt.Sprintf("%s/issues/%d/notes", projectPath(project), iid)
	var note Note
	_, err := c.doJSON(ctx, http.MethodPost, path, nil, payload, &note)
	return &note, err
}
