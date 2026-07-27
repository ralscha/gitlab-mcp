// Package mcpserver registers GitLab MCP tools using the official Go SDK.
package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gitlab-mcp/internal/gitlab"
)

var readOnlyHint = &mcp.ToolAnnotations{ReadOnlyHint: true}

type EmptyInput struct{}

func getMetadata(client *gitlab.Client) mcp.ToolHandlerFor[EmptyInput, gitlab.Metadata] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, gitlab.Metadata, error) {
		result, err := client.Metadata(ctx)
		if err != nil {
			return nil, gitlab.Metadata{}, fmt.Errorf("get GitLab metadata: %w", err)
		}
		return nil, *result, nil
	}
}

func getCurrentUser(client *gitlab.Client) mcp.ToolHandlerFor[EmptyInput, gitlab.User] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, gitlab.User, error) {
		result, err := client.CurrentUser(ctx)
		if err != nil {
			return nil, gitlab.User{}, fmt.Errorf("get current GitLab user: %w", err)
		}
		return nil, *result, nil
	}
}

type ListProjectsInput struct {
	Search     string `json:"search,omitempty" jsonschema:"text to match against project paths and names"`
	Membership *bool  `json:"membership,omitempty" jsonschema:"when true, only projects where the current user is a member"`
	Owned      *bool  `json:"owned,omitempty" jsonschema:"when true, only projects explicitly owned by the current user"`
	Archived   *bool  `json:"archived,omitempty" jsonschema:"filter by archived status"`
	Visibility string `json:"visibility,omitempty" jsonschema:"visibility filter: public, internal, or private"`
	OrderBy    string `json:"order_by,omitempty" jsonschema:"sort field, such as id, name, path, created_at, updated_at, or last_activity_at"`
	Sort       string `json:"sort,omitempty" jsonschema:"sort direction: asc or desc"`
	Page       int    `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage    int    `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type ProjectsPage struct {
	Pagination gitlab.PageInfo  `json:"pagination"`
	Projects   []gitlab.Project `json:"projects"`
}

func listProjects(client *gitlab.Client) mcp.ToolHandlerFor[ListProjectsInput, ProjectsPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListProjectsInput) (*mcp.CallToolResult, ProjectsPage, error) {
		items, page, err := client.ListProjects(ctx, gitlab.ListProjectsOptions{
			Search: in.Search, Membership: in.Membership, Owned: in.Owned, Archived: in.Archived,
			Visibility: in.Visibility, OrderBy: in.OrderBy, Sort: in.Sort, Page: in.Page, PerPage: in.PerPage,
		})
		if err != nil {
			return nil, ProjectsPage{}, fmt.Errorf("list GitLab projects: %w", err)
		}
		return nil, ProjectsPage{Pagination: page, Projects: items}, nil
	}
}

type ProjectInput struct {
	Project string `json:"project" jsonschema:"numeric project id or full path such as group/project; do not URL-encode it"`
}

func getProject(client *gitlab.Client) mcp.ToolHandlerFor[ProjectInput, gitlab.Project] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ProjectInput) (*mcp.CallToolResult, gitlab.Project, error) {
		result, err := client.GetProject(ctx, in.Project)
		if err != nil {
			return nil, gitlab.Project{}, fmt.Errorf("get GitLab project %s: %w", in.Project, err)
		}
		return nil, *result, nil
	}
}

type ListIssuesInput struct {
	Project      string   `json:"project,omitempty" jsonschema:"optional project id or path; omit to list issues across projects"`
	State        string   `json:"state,omitempty" jsonschema:"state filter: opened or closed"`
	Scope        string   `json:"scope,omitempty" jsonschema:"scope filter: created_by_me, assigned_to_me, or all"`
	Search       string   `json:"search,omitempty" jsonschema:"search issue title and description"`
	Labels       []string `json:"labels,omitempty" jsonschema:"issues must have all of these labels"`
	AssigneeID   int      `json:"assignee_id,omitempty" jsonschema:"filter by numeric assignee user id"`
	AuthorID     int      `json:"author_id,omitempty" jsonschema:"filter by numeric author user id"`
	Milestone    string   `json:"milestone,omitempty" jsonschema:"filter by milestone title; use None or Any for special filters"`
	OrderBy      string   `json:"order_by,omitempty" jsonschema:"sort field such as created_at, updated_at, priority, due_date, or label_priority"`
	Sort         string   `json:"sort,omitempty" jsonschema:"sort direction: asc or desc"`
	UpdatedAfter string   `json:"updated_after,omitempty" jsonschema:"ISO 8601 timestamp lower bound for updated_at"`
	Page         int      `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage      int      `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type IssuesPage struct {
	Pagination gitlab.PageInfo `json:"pagination"`
	Issues     []gitlab.Issue  `json:"issues"`
}

func listIssues(client *gitlab.Client) mcp.ToolHandlerFor[ListIssuesInput, IssuesPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListIssuesInput) (*mcp.CallToolResult, IssuesPage, error) {
		items, page, err := client.ListIssues(ctx, gitlab.ListIssuesOptions{
			Project: in.Project, State: in.State, Scope: in.Scope, Search: in.Search, Labels: in.Labels,
			AssigneeID: in.AssigneeID, AuthorID: in.AuthorID, Milestone: in.Milestone, OrderBy: in.OrderBy,
			Sort: in.Sort, UpdatedAfter: in.UpdatedAfter, Page: in.Page, PerPage: in.PerPage,
		})
		if err != nil {
			return nil, IssuesPage{}, fmt.Errorf("list GitLab issues: %w", err)
		}
		return nil, IssuesPage{Pagination: page, Issues: items}, nil
	}
}

type IssueInput struct {
	Project string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	IID     int    `json:"iid" jsonschema:"project-scoped issue IID shown as #123 in GitLab"`
}

func getIssue(client *gitlab.Client) mcp.ToolHandlerFor[IssueInput, gitlab.Issue] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in IssueInput) (*mcp.CallToolResult, gitlab.Issue, error) {
		result, err := client.GetIssue(ctx, in.Project, in.IID)
		if err != nil {
			return nil, gitlab.Issue{}, fmt.Errorf("get GitLab issue %s#%d: %w", in.Project, in.IID, err)
		}
		return nil, *result, nil
	}
}

type ListNotesInput struct {
	Project string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	IID     int    `json:"iid" jsonschema:"project-scoped issue or merge request IID"`
	OrderBy string `json:"order_by,omitempty" jsonschema:"sort field: created_at or updated_at"`
	Sort    string `json:"sort,omitempty" jsonschema:"sort direction: asc or desc"`
	Page    int    `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type NotesPage struct {
	Pagination gitlab.PageInfo `json:"pagination"`
	Notes      []gitlab.Note   `json:"notes"`
}

func listIssueNotes(client *gitlab.Client) mcp.ToolHandlerFor[ListNotesInput, NotesPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListNotesInput) (*mcp.CallToolResult, NotesPage, error) {
		items, page, err := client.ListIssueNotes(ctx, in.Project, in.IID, in.Page, in.PerPage, in.Sort, in.OrderBy)
		if err != nil {
			return nil, NotesPage{}, fmt.Errorf("list notes for issue %s#%d: %w", in.Project, in.IID, err)
		}
		return nil, NotesPage{Pagination: page, Notes: items}, nil
	}
}

type ListMergeRequestsInput struct {
	Project      string   `json:"project,omitempty" jsonschema:"optional project id or path; omit to list merge requests across projects"`
	State        string   `json:"state,omitempty" jsonschema:"state filter: opened, closed, locked, merged, or all"`
	Scope        string   `json:"scope,omitempty" jsonschema:"scope filter: created_by_me, assigned_to_me, or all"`
	Search       string   `json:"search,omitempty" jsonschema:"search merge request title and description"`
	SourceBranch string   `json:"source_branch,omitempty" jsonschema:"filter by source branch"`
	TargetBranch string   `json:"target_branch,omitempty" jsonschema:"filter by target branch"`
	Labels       []string `json:"labels,omitempty" jsonschema:"merge requests must have all of these labels"`
	AuthorID     int      `json:"author_id,omitempty" jsonschema:"filter by numeric author id"`
	AssigneeID   int      `json:"assignee_id,omitempty" jsonschema:"filter by numeric assignee id"`
	ReviewerID   int      `json:"reviewer_id,omitempty" jsonschema:"filter by numeric reviewer id"`
	Draft        *bool    `json:"draft,omitempty" jsonschema:"filter draft or non-draft merge requests"`
	OrderBy      string   `json:"order_by,omitempty" jsonschema:"sort field such as created_at or updated_at"`
	Sort         string   `json:"sort,omitempty" jsonschema:"sort direction: asc or desc"`
	Page         int      `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage      int      `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type MergeRequestsPage struct {
	Pagination    gitlab.PageInfo       `json:"pagination"`
	MergeRequests []gitlab.MergeRequest `json:"merge_requests"`
}

func listMergeRequests(client *gitlab.Client) mcp.ToolHandlerFor[ListMergeRequestsInput, MergeRequestsPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListMergeRequestsInput) (*mcp.CallToolResult, MergeRequestsPage, error) {
		items, page, err := client.ListMergeRequests(ctx, gitlab.ListMergeRequestsOptions{
			Project: in.Project, State: in.State, Scope: in.Scope, Search: in.Search, SourceBranch: in.SourceBranch,
			TargetBranch: in.TargetBranch, Labels: in.Labels, AuthorID: in.AuthorID, AssigneeID: in.AssigneeID,
			ReviewerID: in.ReviewerID, Draft: in.Draft, OrderBy: in.OrderBy, Sort: in.Sort, Page: in.Page, PerPage: in.PerPage,
		})
		if err != nil {
			return nil, MergeRequestsPage{}, fmt.Errorf("list GitLab merge requests: %w", err)
		}
		return nil, MergeRequestsPage{Pagination: page, MergeRequests: items}, nil
	}
}

type MergeRequestInput struct {
	Project string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	IID     int    `json:"iid" jsonschema:"project-scoped merge request IID shown as !123 in GitLab"`
}

func getMergeRequest(client *gitlab.Client) mcp.ToolHandlerFor[MergeRequestInput, gitlab.MergeRequest] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in MergeRequestInput) (*mcp.CallToolResult, gitlab.MergeRequest, error) {
		result, err := client.GetMergeRequest(ctx, in.Project, in.IID)
		if err != nil {
			return nil, gitlab.MergeRequest{}, fmt.Errorf("get GitLab merge request %s!%d: %w", in.Project, in.IID, err)
		}
		return nil, *result, nil
	}
}

func listMergeRequestNotes(client *gitlab.Client) mcp.ToolHandlerFor[ListNotesInput, NotesPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListNotesInput) (*mcp.CallToolResult, NotesPage, error) {
		items, page, err := client.ListMergeRequestNotes(ctx, in.Project, in.IID, in.Page, in.PerPage, in.Sort, in.OrderBy)
		if err != nil {
			return nil, NotesPage{}, fmt.Errorf("list notes for merge request %s!%d: %w", in.Project, in.IID, err)
		}
		return nil, NotesPage{Pagination: page, Notes: items}, nil
	}
}

type ListTreeInput struct {
	Project   string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	Path      string `json:"path,omitempty" jsonschema:"repository directory path; omit for the root"`
	Ref       string `json:"ref,omitempty" jsonschema:"branch, tag, or commit SHA; defaults to the default branch"`
	Recursive bool   `json:"recursive,omitempty" jsonschema:"return the tree recursively"`
	Page      int    `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage   int    `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type TreePage struct {
	Pagination gitlab.PageInfo   `json:"pagination"`
	Items      []gitlab.TreeItem `json:"items"`
}

func listRepositoryTree(client *gitlab.Client) mcp.ToolHandlerFor[ListTreeInput, TreePage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListTreeInput) (*mcp.CallToolResult, TreePage, error) {
		items, page, err := client.ListRepositoryTree(ctx, in.Project, gitlab.ListTreeOptions{Path: in.Path, Ref: in.Ref, Recursive: in.Recursive, Page: in.Page, PerPage: in.PerPage})
		if err != nil {
			return nil, TreePage{}, fmt.Errorf("list repository tree for %s: %w", in.Project, err)
		}
		return nil, TreePage{Pagination: page, Items: items}, nil
	}
}

type GetFileInput struct {
	Project  string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	FilePath string `json:"file_path" jsonschema:"full path to the file in the repository"`
	Ref      string `json:"ref" jsonschema:"branch, tag, or commit SHA"`
}

func getFile(client *gitlab.Client) mcp.ToolHandlerFor[GetFileInput, gitlab.FileContent] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in GetFileInput) (*mcp.CallToolResult, gitlab.FileContent, error) {
		result, err := client.GetFile(ctx, in.Project, in.FilePath, in.Ref)
		if err != nil {
			return nil, gitlab.FileContent{}, fmt.Errorf("get %s at %s in %s: %w", in.FilePath, in.Ref, in.Project, err)
		}
		return nil, *result, nil
	}
}

type ListBranchesInput struct {
	Project string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	Search  string `json:"search,omitempty" jsonschema:"branch name search expression"`
	Regex   string `json:"regex,omitempty" jsonschema:"RE2 branch name regular expression"`
	Page    int    `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type BranchesPage struct {
	Pagination gitlab.PageInfo `json:"pagination"`
	Branches   []gitlab.Branch `json:"branches"`
}

func listBranches(client *gitlab.Client) mcp.ToolHandlerFor[ListBranchesInput, BranchesPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListBranchesInput) (*mcp.CallToolResult, BranchesPage, error) {
		items, page, err := client.ListBranches(ctx, in.Project, gitlab.ListBranchesOptions{Search: in.Search, Regex: in.Regex, Page: in.Page, PerPage: in.PerPage})
		if err != nil {
			return nil, BranchesPage{}, fmt.Errorf("list branches for %s: %w", in.Project, err)
		}
		return nil, BranchesPage{Pagination: page, Branches: items}, nil
	}
}

type ListCommitsInput struct {
	Project     string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	RefName     string `json:"ref_name,omitempty" jsonschema:"branch, tag, or revision range"`
	Path        string `json:"path,omitempty" jsonschema:"repository path to filter commits"`
	Since       string `json:"since,omitempty" jsonschema:"ISO 8601 lower timestamp bound"`
	Until       string `json:"until,omitempty" jsonschema:"ISO 8601 upper timestamp bound"`
	WithStats   bool   `json:"with_stats,omitempty" jsonschema:"include commit statistics"`
	FirstParent bool   `json:"first_parent,omitempty" jsonschema:"follow only first-parent history"`
	Page        int    `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage     int    `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type CommitsPage struct {
	Pagination gitlab.PageInfo `json:"pagination"`
	Commits    []gitlab.Commit `json:"commits"`
}

func listCommits(client *gitlab.Client) mcp.ToolHandlerFor[ListCommitsInput, CommitsPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListCommitsInput) (*mcp.CallToolResult, CommitsPage, error) {
		items, page, err := client.ListCommits(ctx, in.Project, gitlab.ListCommitsOptions{
			RefName: in.RefName, Path: in.Path, Since: in.Since, Until: in.Until, WithStats: in.WithStats,
			FirstParent: in.FirstParent, Page: in.Page, PerPage: in.PerPage,
		})
		if err != nil {
			return nil, CommitsPage{}, fmt.Errorf("list commits for %s: %w", in.Project, err)
		}
		return nil, CommitsPage{Pagination: page, Commits: items}, nil
	}
}

type GetCommitInput struct {
	Project string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	SHA     string `json:"sha" jsonschema:"commit SHA, branch, or tag"`
	Stats   bool   `json:"stats,omitempty" jsonschema:"include commit statistics"`
}

func getCommit(client *gitlab.Client) mcp.ToolHandlerFor[GetCommitInput, gitlab.Commit] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in GetCommitInput) (*mcp.CallToolResult, gitlab.Commit, error) {
		result, err := client.GetCommit(ctx, in.Project, in.SHA, in.Stats)
		if err != nil {
			return nil, gitlab.Commit{}, fmt.Errorf("get commit %s in %s: %w", in.SHA, in.Project, err)
		}
		return nil, *result, nil
	}
}

type ListPipelinesInput struct {
	Project      string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	Status       string `json:"status,omitempty" jsonschema:"pipeline status such as running, pending, success, failed, or canceled"`
	Ref          string `json:"ref,omitempty" jsonschema:"filter by ref"`
	SHA          string `json:"sha,omitempty" jsonschema:"filter by commit SHA"`
	Source       string `json:"source,omitempty" jsonschema:"pipeline source such as push, web, schedule, api, or merge_request_event"`
	Username     string `json:"username,omitempty" jsonschema:"filter by triggering username"`
	OrderBy      string `json:"order_by,omitempty" jsonschema:"sort field: id, status, ref, updated_at, or user_id"`
	Sort         string `json:"sort,omitempty" jsonschema:"sort direction: asc or desc"`
	UpdatedAfter string `json:"updated_after,omitempty" jsonschema:"ISO 8601 timestamp lower bound"`
	Page         int    `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage      int    `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type PipelinesPage struct {
	Pagination gitlab.PageInfo   `json:"pagination"`
	Pipelines  []gitlab.Pipeline `json:"pipelines"`
}

func listPipelines(client *gitlab.Client) mcp.ToolHandlerFor[ListPipelinesInput, PipelinesPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListPipelinesInput) (*mcp.CallToolResult, PipelinesPage, error) {
		items, page, err := client.ListPipelines(ctx, in.Project, gitlab.ListPipelinesOptions{
			Status: in.Status, Ref: in.Ref, SHA: in.SHA, Source: in.Source, Username: in.Username,
			OrderBy: in.OrderBy, Sort: in.Sort, UpdatedAfter: in.UpdatedAfter, Page: in.Page, PerPage: in.PerPage,
		})
		if err != nil {
			return nil, PipelinesPage{}, fmt.Errorf("list pipelines for %s: %w", in.Project, err)
		}
		return nil, PipelinesPage{Pagination: page, Pipelines: items}, nil
	}
}

type PipelineInput struct {
	Project    string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	PipelineID int    `json:"pipeline_id" jsonschema:"numeric pipeline id"`
}

func getPipeline(client *gitlab.Client) mcp.ToolHandlerFor[PipelineInput, gitlab.Pipeline] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in PipelineInput) (*mcp.CallToolResult, gitlab.Pipeline, error) {
		result, err := client.GetPipeline(ctx, in.Project, in.PipelineID)
		if err != nil {
			return nil, gitlab.Pipeline{}, fmt.Errorf("get pipeline %d in %s: %w", in.PipelineID, in.Project, err)
		}
		return nil, *result, nil
	}
}

type ListPipelineJobsInput struct {
	Project        string   `json:"project" jsonschema:"numeric project id or path such as group/project"`
	PipelineID     int      `json:"pipeline_id" jsonschema:"numeric pipeline id"`
	Scopes         []string `json:"scopes,omitempty" jsonschema:"job status filters such as pending, running, failed, success, canceled, manual, or skipped"`
	IncludeRetried bool     `json:"include_retried,omitempty" jsonschema:"include retried jobs"`
	Page           int      `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage        int      `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type JobsPage struct {
	Pagination gitlab.PageInfo `json:"pagination"`
	Jobs       []gitlab.Job    `json:"jobs"`
}

func listPipelineJobs(client *gitlab.Client) mcp.ToolHandlerFor[ListPipelineJobsInput, JobsPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListPipelineJobsInput) (*mcp.CallToolResult, JobsPage, error) {
		items, page, err := client.ListPipelineJobs(ctx, in.Project, in.PipelineID, in.Scopes, in.IncludeRetried, in.Page, in.PerPage)
		if err != nil {
			return nil, JobsPage{}, fmt.Errorf("list jobs for pipeline %d in %s: %w", in.PipelineID, in.Project, err)
		}
		return nil, JobsPage{Pagination: page, Jobs: items}, nil
	}
}

type SearchInput struct {
	Scope   string `json:"scope" jsonschema:"GitLab search scope, such as projects, issues, merge_requests, blobs, commits, notes, or users"`
	Search  string `json:"search" jsonschema:"search expression"`
	Project string `json:"project,omitempty" jsonschema:"optional project id or path for project-scoped search"`
	GroupID int    `json:"group_id,omitempty" jsonschema:"optional numeric group id for group-scoped search; ignored when project is set"`
	OrderBy string `json:"order_by,omitempty" jsonschema:"scope-dependent sort field"`
	Sort    string `json:"sort,omitempty" jsonschema:"sort direction: asc or desc"`
	Page    int    `json:"page,omitempty" jsonschema:"page number, starting at 1"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"results per page, defaults to 20 and is capped at 100"`
}

type SearchPage struct {
	Pagination gitlab.PageInfo  `json:"pagination"`
	Results    []map[string]any `json:"results"`
}

func search(client *gitlab.Client) mcp.ToolHandlerFor[SearchInput, SearchPage] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, SearchPage, error) {
		items, page, err := client.Search(ctx, gitlab.SearchOptions{Scope: in.Scope, Search: in.Search, Project: in.Project, GroupID: in.GroupID, OrderBy: in.OrderBy, Sort: in.Sort, Page: in.Page, PerPage: in.PerPage})
		if err != nil {
			return nil, SearchPage{}, fmt.Errorf("search GitLab %s: %w", in.Scope, err)
		}
		return nil, SearchPage{Pagination: page, Results: items}, nil
	}
}

func registerReadTools(s *mcp.Server, client *gitlab.Client) {
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_metadata", Description: "Get the connected GitLab instance version and metadata.", Annotations: readOnlyHint}, getMetadata(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_current_user", Description: "Get the GitLab user represented by the configured token.", Annotations: readOnlyHint}, getCurrentUser(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_projects", Description: "List GitLab projects visible to the current user.", Annotations: readOnlyHint}, listProjects(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_project", Description: "Get one GitLab project by numeric id or path.", Annotations: readOnlyHint}, getProject(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_issues", Description: "List and filter GitLab issues in one project or across projects.", Annotations: readOnlyHint}, listIssues(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_issue", Description: "Get one GitLab issue by project and IID.", Annotations: readOnlyHint}, getIssue(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_issue_notes", Description: "List notes (comments and system events) on a GitLab issue.", Annotations: readOnlyHint}, listIssueNotes(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_merge_requests", Description: "List and filter GitLab merge requests in one project or across projects.", Annotations: readOnlyHint}, listMergeRequests(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_merge_request", Description: "Get one GitLab merge request by project and IID.", Annotations: readOnlyHint}, getMergeRequest(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_merge_request_notes", Description: "List notes on a GitLab merge request.", Annotations: readOnlyHint}, listMergeRequestNotes(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_repository_tree", Description: "List files and directories in a GitLab repository tree.", Annotations: readOnlyHint}, listRepositoryTree(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_file", Description: "Read a repository file at a branch, tag, or commit.", Annotations: readOnlyHint}, getFile(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_branches", Description: "List branches in a GitLab repository.", Annotations: readOnlyHint}, listBranches(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_commits", Description: "List commits in a GitLab repository.", Annotations: readOnlyHint}, listCommits(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_commit", Description: "Get one commit from a GitLab repository.", Annotations: readOnlyHint}, getCommit(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_pipelines", Description: "List CI/CD pipelines for a GitLab project.", Annotations: readOnlyHint}, listPipelines(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_pipeline", Description: "Get one GitLab CI/CD pipeline.", Annotations: readOnlyHint}, getPipeline(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_list_pipeline_jobs", Description: "List jobs in a GitLab CI/CD pipeline.", Annotations: readOnlyHint}, listPipelineJobs(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_search", Description: "Search GitLab globally or within a group or project.", Annotations: readOnlyHint}, search(client))
}
