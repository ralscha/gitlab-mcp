package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

type ListMergeRequestsOptions struct {
	Project      string
	State        string
	Scope        string
	Search       string
	SourceBranch string
	TargetBranch string
	Labels       []string
	AuthorID     int
	AssigneeID   int
	ReviewerID   int
	Draft        *bool
	OrderBy      string
	Sort         string
	Page         int
	PerPage      int
}

func (c *Client) ListMergeRequests(ctx context.Context, opts ListMergeRequestsOptions) ([]MergeRequest, PageInfo, error) {
	query := pageQuery(opts.Page, opts.PerPage)
	setIf(query, "state", opts.State)
	setIf(query, "scope", opts.Scope)
	setIf(query, "search", opts.Search)
	setIf(query, "source_branch", opts.SourceBranch)
	setIf(query, "target_branch", opts.TargetBranch)
	setIf(query, "labels", joinComma(opts.Labels))
	setIf(query, "order_by", opts.OrderBy)
	setIf(query, "sort", opts.Sort)
	setBool(query, "draft", opts.Draft)
	if opts.AuthorID > 0 {
		query.Set("author_id", strconv.Itoa(opts.AuthorID))
	}
	if opts.AssigneeID > 0 {
		query.Set("assignee_id", strconv.Itoa(opts.AssigneeID))
	}
	if opts.ReviewerID > 0 {
		query.Set("reviewer_id", strconv.Itoa(opts.ReviewerID))
	}
	path := "merge_requests"
	if opts.Project != "" {
		path = projectPath(opts.Project) + "/merge_requests"
	}
	var mergeRequests []MergeRequest
	headers, err := c.doJSON(ctx, http.MethodGet, path, query, nil, &mergeRequests)
	return mergeRequests, pagination(headers, opts.Page, opts.PerPage, len(mergeRequests)), err
}

func (c *Client) GetMergeRequest(ctx context.Context, project string, iid int) (*MergeRequest, error) {
	var mergeRequest MergeRequest
	path := fmt.Sprintf("%s/merge_requests/%d", projectPath(project), iid)
	_, err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &mergeRequest)
	return &mergeRequest, err
}

type CreateMergeRequestInput struct {
	SourceBranch       string
	TargetBranch       string
	Title              string
	Description        string
	Draft              bool
	AssigneeIDs        []int
	ReviewerIDs        []int
	Labels             []string
	MilestoneID        int
	RemoveSourceBranch *bool
	Squash             *bool
	AllowCollaboration *bool
}

func (c *Client) CreateMergeRequest(ctx context.Context, project string, in CreateMergeRequestInput) (*MergeRequest, error) {
	body := map[string]any{"source_branch": in.SourceBranch, "target_branch": in.TargetBranch, "title": in.Title}
	if in.Description != "" {
		body["description"] = in.Description
	}
	if in.Draft {
		body["draft"] = true
	}
	if len(in.AssigneeIDs) > 0 {
		body["assignee_ids"] = in.AssigneeIDs
	}
	if len(in.ReviewerIDs) > 0 {
		body["reviewer_ids"] = in.ReviewerIDs
	}
	if len(in.Labels) > 0 {
		body["labels"] = joinComma(in.Labels)
	}
	if in.MilestoneID > 0 {
		body["milestone_id"] = in.MilestoneID
	}
	if in.RemoveSourceBranch != nil {
		body["remove_source_branch"] = *in.RemoveSourceBranch
	}
	if in.Squash != nil {
		body["squash"] = *in.Squash
	}
	if in.AllowCollaboration != nil {
		body["allow_collaboration"] = *in.AllowCollaboration
	}
	var mergeRequest MergeRequest
	_, err := c.doJSON(ctx, http.MethodPost, projectPath(project)+"/merge_requests", nil, body, &mergeRequest)
	return &mergeRequest, err
}

type UpdateMergeRequestInput struct {
	Title              *string
	Description        *string
	TargetBranch       *string
	StateEvent         *string
	Draft              *bool
	AssigneeIDs        *[]int
	ReviewerIDs        *[]int
	Labels             *[]string
	MilestoneID        *int
	RemoveSourceBranch *bool
	Squash             *bool
}

func (c *Client) UpdateMergeRequest(ctx context.Context, project string, iid int, in UpdateMergeRequestInput) (*MergeRequest, error) {
	body := map[string]any{}
	if in.Title != nil {
		body["title"] = *in.Title
	}
	if in.Description != nil {
		body["description"] = *in.Description
	}
	if in.TargetBranch != nil {
		body["target_branch"] = *in.TargetBranch
	}
	if in.StateEvent != nil {
		body["state_event"] = *in.StateEvent
	}
	if in.Draft != nil {
		body["draft"] = *in.Draft
	}
	if in.AssigneeIDs != nil {
		body["assignee_ids"] = *in.AssigneeIDs
	}
	if in.ReviewerIDs != nil {
		body["reviewer_ids"] = *in.ReviewerIDs
	}
	if in.Labels != nil {
		body["labels"] = joinComma(*in.Labels)
	}
	if in.MilestoneID != nil {
		body["milestone_id"] = *in.MilestoneID
	}
	if in.RemoveSourceBranch != nil {
		body["remove_source_branch"] = *in.RemoveSourceBranch
	}
	if in.Squash != nil {
		body["squash"] = *in.Squash
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("gitlab: UpdateMergeRequest requires at least one field")
	}
	var mergeRequest MergeRequest
	path := fmt.Sprintf("%s/merge_requests/%d", projectPath(project), iid)
	_, err := c.doJSON(ctx, http.MethodPut, path, nil, body, &mergeRequest)
	return &mergeRequest, err
}

type AcceptMergeRequestInput struct {
	SHA                       string
	MergeCommitMessage        string
	SquashCommitMessage       string
	Squash                    *bool
	ShouldRemoveSourceBranch  *bool
	MergeWhenPipelineSucceeds bool
}

func (c *Client) AcceptMergeRequest(ctx context.Context, project string, iid int, in AcceptMergeRequestInput) (*MergeRequest, error) {
	body := map[string]any{}
	if in.SHA != "" {
		body["sha"] = in.SHA
	}
	if in.MergeCommitMessage != "" {
		body["merge_commit_message"] = in.MergeCommitMessage
	}
	if in.SquashCommitMessage != "" {
		body["squash_commit_message"] = in.SquashCommitMessage
	}
	if in.Squash != nil {
		body["squash"] = *in.Squash
	}
	if in.ShouldRemoveSourceBranch != nil {
		body["should_remove_source_branch"] = *in.ShouldRemoveSourceBranch
	}
	if in.MergeWhenPipelineSucceeds {
		body["merge_when_pipeline_succeeds"] = true
	}
	var mergeRequest MergeRequest
	path := fmt.Sprintf("%s/merge_requests/%d/merge", projectPath(project), iid)
	_, err := c.doJSON(ctx, http.MethodPut, path, nil, body, &mergeRequest)
	return &mergeRequest, err
}

func (c *Client) ListMergeRequestNotes(ctx context.Context, project string, iid, page, perPage int, sort, orderBy string) ([]Note, PageInfo, error) {
	query := pageQuery(page, perPage)
	setIf(query, "sort", sort)
	setIf(query, "order_by", orderBy)
	path := fmt.Sprintf("%s/merge_requests/%d/notes", projectPath(project), iid)
	var notes []Note
	headers, err := c.doJSON(ctx, http.MethodGet, path, query, nil, &notes)
	return notes, pagination(headers, page, perPage, len(notes)), err
}

func (c *Client) AddMergeRequestNote(ctx context.Context, project string, iid int, body string, internal bool) (*Note, error) {
	payload := map[string]any{"body": body}
	if internal {
		payload["internal"] = true
	}
	path := fmt.Sprintf("%s/merge_requests/%d/notes", projectPath(project), iid)
	var note Note
	_, err := c.doJSON(ctx, http.MethodPost, path, nil, payload, &note)
	return &note, err
}
