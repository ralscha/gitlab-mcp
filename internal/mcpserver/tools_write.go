package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gitlab-mcp/internal/gitlab"
)

var nonDestructiveHint = &mcp.ToolAnnotations{DestructiveHint: new(false)}
var destructiveHint = &mcp.ToolAnnotations{DestructiveHint: new(true)}

type CreateIssueInput struct {
	Project      string   `json:"project" jsonschema:"numeric project id or path such as group/project"`
	Title        string   `json:"title" jsonschema:"issue title"`
	Description  string   `json:"description,omitempty" jsonschema:"issue description in GitLab Flavored Markdown"`
	Confidential bool     `json:"confidential,omitempty" jsonschema:"make the issue confidential"`
	Labels       []string `json:"labels,omitempty" jsonschema:"labels to assign"`
	AssigneeIDs  []int    `json:"assignee_ids,omitempty" jsonschema:"numeric GitLab user ids to assign"`
	MilestoneID  int      `json:"milestone_id,omitempty" jsonschema:"numeric project milestone id"`
	DueDate      string   `json:"due_date,omitempty" jsonschema:"due date as YYYY-MM-DD"`
	IssueType    string   `json:"issue_type,omitempty" jsonschema:"issue type such as issue, incident, test_case, or task"`
	Weight       int      `json:"weight,omitempty" jsonschema:"issue weight where supported by the GitLab tier"`
}

func createIssue(client *gitlab.Client) mcp.ToolHandlerFor[CreateIssueInput, gitlab.Issue] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in CreateIssueInput) (*mcp.CallToolResult, gitlab.Issue, error) {
		result, err := client.CreateIssue(ctx, in.Project, gitlab.CreateIssueInput{
			Title: in.Title, Description: in.Description, Confidential: in.Confidential, Labels: in.Labels,
			AssigneeIDs: in.AssigneeIDs, MilestoneID: in.MilestoneID, DueDate: in.DueDate, IssueType: in.IssueType, Weight: in.Weight,
		})
		if err != nil {
			return nil, gitlab.Issue{}, fmt.Errorf("create issue in %s: %w", in.Project, err)
		}
		return nil, *result, nil
	}
}

type UpdateIssueInput struct {
	Project      string    `json:"project" jsonschema:"numeric project id or path such as group/project"`
	IID          int       `json:"iid" jsonschema:"project-scoped issue IID"`
	Title        *string   `json:"title,omitempty" jsonschema:"new issue title"`
	Description  *string   `json:"description,omitempty" jsonschema:"new description in GitLab Flavored Markdown; empty clears it"`
	Confidential *bool     `json:"confidential,omitempty" jsonschema:"new confidential status"`
	Labels       *[]string `json:"labels,omitempty" jsonschema:"replacement labels; an empty list removes all labels"`
	AssigneeIDs  *[]int    `json:"assignee_ids,omitempty" jsonschema:"replacement numeric assignee ids; an empty list unassigns everyone"`
	MilestoneID  *int      `json:"milestone_id,omitempty" jsonschema:"new milestone id; use 0 to remove the milestone"`
	DueDate      *string   `json:"due_date,omitempty" jsonschema:"new due date as YYYY-MM-DD; empty clears it"`
	IssueType    *string   `json:"issue_type,omitempty" jsonschema:"new issue type"`
	Weight       *int      `json:"weight,omitempty" jsonschema:"new issue weight; use 0 to remove it"`
	StateEvent   *string   `json:"state_event,omitempty" jsonschema:"state transition: close or reopen"`
}

func updateIssue(client *gitlab.Client) mcp.ToolHandlerFor[UpdateIssueInput, gitlab.Issue] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in UpdateIssueInput) (*mcp.CallToolResult, gitlab.Issue, error) {
		result, err := client.UpdateIssue(ctx, in.Project, in.IID, gitlab.UpdateIssueInput{
			Title: in.Title, Description: in.Description, Confidential: in.Confidential, Labels: in.Labels,
			AssigneeIDs: in.AssigneeIDs, MilestoneID: in.MilestoneID, DueDate: in.DueDate,
			IssueType: in.IssueType, Weight: in.Weight, StateEvent: in.StateEvent,
		})
		if err != nil {
			return nil, gitlab.Issue{}, fmt.Errorf("update issue %s#%d: %w", in.Project, in.IID, err)
		}
		return nil, *result, nil
	}
}

type AddNoteInput struct {
	Project  string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	IID      int    `json:"iid" jsonschema:"project-scoped issue or merge request IID"`
	Body     string `json:"body" jsonschema:"note body in GitLab Flavored Markdown"`
	Internal bool   `json:"internal,omitempty" jsonschema:"make the note visible only to project members with sufficient access"`
}

func addIssueNote(client *gitlab.Client) mcp.ToolHandlerFor[AddNoteInput, gitlab.Note] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in AddNoteInput) (*mcp.CallToolResult, gitlab.Note, error) {
		result, err := client.AddIssueNote(ctx, in.Project, in.IID, in.Body, in.Internal)
		if err != nil {
			return nil, gitlab.Note{}, fmt.Errorf("add note to issue %s#%d: %w", in.Project, in.IID, err)
		}
		return nil, *result, nil
	}
}

type CreateMergeRequestInput struct {
	Project            string   `json:"project" jsonschema:"numeric project id or path such as group/project"`
	SourceBranch       string   `json:"source_branch" jsonschema:"branch containing the proposed changes"`
	TargetBranch       string   `json:"target_branch" jsonschema:"branch to merge into"`
	Title              string   `json:"title" jsonschema:"merge request title"`
	Description        string   `json:"description,omitempty" jsonschema:"merge request description in GitLab Flavored Markdown"`
	Draft              bool     `json:"draft,omitempty" jsonschema:"create as a draft merge request"`
	AssigneeIDs        []int    `json:"assignee_ids,omitempty" jsonschema:"numeric GitLab user ids to assign"`
	ReviewerIDs        []int    `json:"reviewer_ids,omitempty" jsonschema:"numeric GitLab user ids to request as reviewers"`
	Labels             []string `json:"labels,omitempty" jsonschema:"labels to assign"`
	MilestoneID        int      `json:"milestone_id,omitempty" jsonschema:"numeric project milestone id"`
	RemoveSourceBranch *bool    `json:"remove_source_branch,omitempty" jsonschema:"whether GitLab should remove the source branch after merge"`
	Squash             *bool    `json:"squash,omitempty" jsonschema:"whether commits should be squashed on merge"`
	AllowCollaboration *bool    `json:"allow_collaboration,omitempty" jsonschema:"allow commits from members who can merge to the target branch"`
}

func createMergeRequest(client *gitlab.Client) mcp.ToolHandlerFor[CreateMergeRequestInput, gitlab.MergeRequest] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in CreateMergeRequestInput) (*mcp.CallToolResult, gitlab.MergeRequest, error) {
		result, err := client.CreateMergeRequest(ctx, in.Project, gitlab.CreateMergeRequestInput{
			SourceBranch: in.SourceBranch, TargetBranch: in.TargetBranch, Title: in.Title, Description: in.Description,
			Draft: in.Draft, AssigneeIDs: in.AssigneeIDs, ReviewerIDs: in.ReviewerIDs, Labels: in.Labels,
			MilestoneID: in.MilestoneID, RemoveSourceBranch: in.RemoveSourceBranch, Squash: in.Squash, AllowCollaboration: in.AllowCollaboration,
		})
		if err != nil {
			return nil, gitlab.MergeRequest{}, fmt.Errorf("create merge request in %s: %w", in.Project, err)
		}
		return nil, *result, nil
	}
}

type UpdateMergeRequestInput struct {
	Project            string    `json:"project" jsonschema:"numeric project id or path such as group/project"`
	IID                int       `json:"iid" jsonschema:"project-scoped merge request IID"`
	Title              *string   `json:"title,omitempty" jsonschema:"new title"`
	Description        *string   `json:"description,omitempty" jsonschema:"new description; empty clears it"`
	TargetBranch       *string   `json:"target_branch,omitempty" jsonschema:"new target branch"`
	StateEvent         *string   `json:"state_event,omitempty" jsonschema:"state transition: close or reopen"`
	Draft              *bool     `json:"draft,omitempty" jsonschema:"new draft status"`
	AssigneeIDs        *[]int    `json:"assignee_ids,omitempty" jsonschema:"replacement assignee ids"`
	ReviewerIDs        *[]int    `json:"reviewer_ids,omitempty" jsonschema:"replacement reviewer ids"`
	Labels             *[]string `json:"labels,omitempty" jsonschema:"replacement labels"`
	MilestoneID        *int      `json:"milestone_id,omitempty" jsonschema:"new milestone id; use 0 to remove"`
	RemoveSourceBranch *bool     `json:"remove_source_branch,omitempty" jsonschema:"whether to remove source branch after merge"`
	Squash             *bool     `json:"squash,omitempty" jsonschema:"whether commits should be squashed on merge"`
}

func updateMergeRequest(client *gitlab.Client) mcp.ToolHandlerFor[UpdateMergeRequestInput, gitlab.MergeRequest] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in UpdateMergeRequestInput) (*mcp.CallToolResult, gitlab.MergeRequest, error) {
		result, err := client.UpdateMergeRequest(ctx, in.Project, in.IID, gitlab.UpdateMergeRequestInput{
			Title: in.Title, Description: in.Description, TargetBranch: in.TargetBranch, StateEvent: in.StateEvent,
			Draft: in.Draft, AssigneeIDs: in.AssigneeIDs, ReviewerIDs: in.ReviewerIDs, Labels: in.Labels,
			MilestoneID: in.MilestoneID, RemoveSourceBranch: in.RemoveSourceBranch, Squash: in.Squash,
		})
		if err != nil {
			return nil, gitlab.MergeRequest{}, fmt.Errorf("update merge request %s!%d: %w", in.Project, in.IID, err)
		}
		return nil, *result, nil
	}
}

func addMergeRequestNote(client *gitlab.Client) mcp.ToolHandlerFor[AddNoteInput, gitlab.Note] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in AddNoteInput) (*mcp.CallToolResult, gitlab.Note, error) {
		result, err := client.AddMergeRequestNote(ctx, in.Project, in.IID, in.Body, in.Internal)
		if err != nil {
			return nil, gitlab.Note{}, fmt.Errorf("add note to merge request %s!%d: %w", in.Project, in.IID, err)
		}
		return nil, *result, nil
	}
}

type AcceptMergeRequestInput struct {
	Project                   string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	IID                       int    `json:"iid" jsonschema:"project-scoped merge request IID"`
	SHA                       string `json:"sha,omitempty" jsonschema:"expected current HEAD SHA; the merge fails with 409 if it changed"`
	MergeCommitMessage        string `json:"merge_commit_message,omitempty" jsonschema:"custom merge commit message"`
	SquashCommitMessage       string `json:"squash_commit_message,omitempty" jsonschema:"custom squash commit message"`
	Squash                    *bool  `json:"squash,omitempty" jsonschema:"whether to squash commits"`
	RemoveSourceBranch        *bool  `json:"remove_source_branch,omitempty" jsonschema:"whether to remove the source branch after merge"`
	AutoMerge                 bool   `json:"auto_merge,omitempty" jsonschema:"merge automatically when all checks pass"`
	MergeWhenPipelineSucceeds bool   `json:"merge_when_pipeline_succeeds,omitempty" jsonschema:"deprecated alias for auto_merge"`
}

func acceptMergeRequest(client *gitlab.Client) mcp.ToolHandlerFor[AcceptMergeRequestInput, gitlab.MergeRequest] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in AcceptMergeRequestInput) (*mcp.CallToolResult, gitlab.MergeRequest, error) {
		result, err := client.AcceptMergeRequest(ctx, in.Project, in.IID, gitlab.AcceptMergeRequestInput{
			SHA: in.SHA, MergeCommitMessage: in.MergeCommitMessage, SquashCommitMessage: in.SquashCommitMessage,
			Squash: in.Squash, ShouldRemoveSourceBranch: in.RemoveSourceBranch,
			AutoMerge: in.AutoMerge || in.MergeWhenPipelineSucceeds,
		})
		if err != nil {
			return nil, gitlab.MergeRequest{}, fmt.Errorf("accept merge request %s!%d: %w", in.Project, in.IID, err)
		}
		return nil, *result, nil
	}
}

type CreateBranchInput struct {
	Project string `json:"project" jsonschema:"numeric project id or path such as group/project"`
	Branch  string `json:"branch" jsonschema:"name of the branch to create"`
	Ref     string `json:"ref" jsonschema:"existing branch, tag, or commit SHA to branch from"`
}

func createBranch(client *gitlab.Client) mcp.ToolHandlerFor[CreateBranchInput, gitlab.Branch] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in CreateBranchInput) (*mcp.CallToolResult, gitlab.Branch, error) {
		result, err := client.CreateBranch(ctx, in.Project, in.Branch, in.Ref)
		if err != nil {
			return nil, gitlab.Branch{}, fmt.Errorf("create branch %s in %s: %w", in.Branch, in.Project, err)
		}
		return nil, *result, nil
	}
}

type CreateCommitInput struct {
	Project       string                `json:"project" jsonschema:"numeric project id or path such as group/project"`
	Branch        string                `json:"branch" jsonschema:"target branch; can be new when start_branch or start_sha is supplied"`
	CommitMessage string                `json:"commit_message" jsonschema:"commit message"`
	StartBranch   string                `json:"start_branch,omitempty" jsonschema:"branch to create the target branch from"`
	StartSHA      string                `json:"start_sha,omitempty" jsonschema:"commit SHA to create the target branch from"`
	AuthorEmail   string                `json:"author_email,omitempty" jsonschema:"commit author email"`
	AuthorName    string                `json:"author_name,omitempty" jsonschema:"commit author name"`
	Actions       []gitlab.CommitAction `json:"actions" jsonschema:"file actions; action is create, update, move, delete, or chmod; content may use text or base64 encoding"`
}

func createCommit(client *gitlab.Client) mcp.ToolHandlerFor[CreateCommitInput, gitlab.Commit] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in CreateCommitInput) (*mcp.CallToolResult, gitlab.Commit, error) {
		result, err := client.CreateCommit(ctx, in.Project, gitlab.CreateCommitInput{
			Branch: in.Branch, CommitMessage: in.CommitMessage, StartBranch: in.StartBranch, StartSHA: in.StartSHA,
			AuthorEmail: in.AuthorEmail, AuthorName: in.AuthorName, Actions: in.Actions,
		})
		if err != nil {
			return nil, gitlab.Commit{}, fmt.Errorf("create commit in %s: %w", in.Project, err)
		}
		return nil, *result, nil
	}
}

type CreatePipelineInput struct {
	Project   string                    `json:"project" jsonschema:"numeric project id or path such as group/project"`
	Ref       string                    `json:"ref" jsonschema:"branch or tag to run the pipeline for"`
	Variables []gitlab.PipelineVariable `json:"variables,omitempty" jsonschema:"pipeline variables; variable_type may be env_var or file"`
	Inputs    map[string]any            `json:"inputs,omitempty" jsonschema:"typed CI/CD inputs defined by the pipeline configuration, as key-value pairs"`
}

func createPipeline(client *gitlab.Client) mcp.ToolHandlerFor[CreatePipelineInput, gitlab.Pipeline] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in CreatePipelineInput) (*mcp.CallToolResult, gitlab.Pipeline, error) {
		result, err := client.CreatePipeline(ctx, in.Project, in.Ref, in.Variables, in.Inputs)
		if err != nil {
			return nil, gitlab.Pipeline{}, fmt.Errorf("create pipeline in %s: %w", in.Project, err)
		}
		return nil, *result, nil
	}
}

func retryPipeline(client *gitlab.Client) mcp.ToolHandlerFor[PipelineInput, gitlab.Pipeline] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in PipelineInput) (*mcp.CallToolResult, gitlab.Pipeline, error) {
		result, err := client.RetryPipeline(ctx, in.Project, in.PipelineID)
		if err != nil {
			return nil, gitlab.Pipeline{}, fmt.Errorf("retry pipeline %d in %s: %w", in.PipelineID, in.Project, err)
		}
		return nil, *result, nil
	}
}

func registerWriteTools(s *mcp.Server, client *gitlab.Client) {
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_create_issue", Description: "Create a GitLab issue.", Annotations: nonDestructiveHint}, createIssue(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_update_issue", Description: "Update, close, or reopen a GitLab issue.", Annotations: &mcp.ToolAnnotations{DestructiveHint: new(false), IdempotentHint: true}}, updateIssue(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_add_issue_note", Description: "Add a Markdown note to a GitLab issue.", Annotations: nonDestructiveHint}, addIssueNote(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_create_merge_request", Description: "Create a GitLab merge request.", Annotations: nonDestructiveHint}, createMergeRequest(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_update_merge_request", Description: "Update, close, or reopen a GitLab merge request.", Annotations: &mcp.ToolAnnotations{DestructiveHint: new(false), IdempotentHint: true}}, updateMergeRequest(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_add_merge_request_note", Description: "Add a Markdown note to a GitLab merge request.", Annotations: nonDestructiveHint}, addMergeRequestNote(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_accept_merge_request", Description: "Merge a GitLab merge request now or automatically when all checks pass.", Annotations: destructiveHint}, acceptMergeRequest(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_create_branch", Description: "Create a GitLab repository branch.", Annotations: nonDestructiveHint}, createBranch(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_create_commit", Description: "Create a GitLab commit with one or more create, update, move, delete, or chmod file actions.", Annotations: destructiveHint}, createCommit(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_create_pipeline", Description: "Run a new GitLab CI/CD pipeline. Pipelines can have deployment side effects.", Annotations: destructiveHint}, createPipeline(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_retry_pipeline", Description: "Retry failed or canceled jobs in a GitLab pipeline. Jobs can have deployment side effects.", Annotations: destructiveHint}, retryPipeline(client))
}
