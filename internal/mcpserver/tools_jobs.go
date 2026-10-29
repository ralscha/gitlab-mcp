package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gitlab-mcp/internal/gitlab"
)

type JobInput struct {
	Project string `json:"project" jsonschema:"numeric project id or unescaped path such as group/project"`
	JobID   int    `json:"job_id" jsonschema:"numeric GitLab job id"`
}

type JobTrace struct {
	JobID int    `json:"job_id"`
	Trace string `json:"trace"`
}

func getJob(client *gitlab.Client) mcp.ToolHandlerFor[JobInput, gitlab.Job] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in JobInput) (*mcp.CallToolResult, gitlab.Job, error) {
		result, err := client.GetJob(ctx, in.Project, in.JobID)
		if err != nil {
			return nil, gitlab.Job{}, fmt.Errorf("get job %d in %s: %w", in.JobID, in.Project, err)
		}
		return nil, *result, nil
	}
}

func getJobTrace(client *gitlab.Client) mcp.ToolHandlerFor[JobInput, JobTrace] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in JobInput) (*mcp.CallToolResult, JobTrace, error) {
		trace, err := client.GetJobTrace(ctx, in.Project, in.JobID)
		if err != nil {
			return nil, JobTrace{}, fmt.Errorf("get trace for job %d in %s: %w", in.JobID, in.Project, err)
		}
		return nil, JobTrace{JobID: in.JobID, Trace: trace}, nil
	}
}

func registerJobTools(s *mcp.Server, client *gitlab.Client) {
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_job", Description: "Get one GitLab CI/CD job by its job ID.", Annotations: readOnlyHint}, getJob(client))
	mcp.AddTool(s, &mcp.Tool{Name: "gitlab_get_job_trace", Description: "Read the plain-text log of a GitLab CI/CD job. Logs over 1 MiB return a size-limit error; missing or erased logs return GitLab's error.", Annotations: readOnlyHint}, getJobTrace(client))
}
