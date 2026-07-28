# gitlab-mcp

A [Model Context Protocol](https://modelcontextprotocol.io/) server for the
current [GitLab REST API v4](https://docs.gitlab.com/api/rest/), written in Go
with the official
[`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

It supports GitLab.com and GitLab Self-Managed, stdio and streamable HTTP
transports, paginated results, and separate `readonly` and `readwrite` modes.
The default is `readonly`.

## Quick start

### Using binary releases

Download a pre-built binary from the [releases page](https://github.com/ralscha/gitlab-mcp/releases):

```bash
export GITLAB_TOKEN=glpat-your-token
./gitlab-mcp --mode=readonly --transport=http --http-addr=127.0.0.1:8080 \
  --auth-token=choose-a-separate-mcp-token
```

### Building from source

Create a GitLab personal, project, or group access token, then run:

```bash
export GITLAB_TOKEN=glpat-your-token
go run ./cmd/gitlab-mcp --mode=readonly --transport=http --http-addr=127.0.0.1:8080 \
  --auth-token=choose-a-separate-mcp-token
```

For a self-managed instance:

```bash
export GITLAB_BASE_URL=https://gitlab.example.com
export GITLAB_TOKEN=glpat-your-token
go run ./cmd/gitlab-mcp
```

`GITLAB_BASE_URL` is the instance URL, without `/api/v4`; the server appends
the stable GitLab REST API path itself.

## Configuration

Flags override environment variables.

| Environment variable | CLI flag | Default | Description |
| --- | --- | --- | --- |
| `GITLAB_BASE_URL` | `--gitlab-base-url` | `https://gitlab.com` | GitLab instance URL |
| `GITLAB_TOKEN` | `--gitlab-token` | required | Personal, project, or group access token |
| `GITLAB_MODE` | `--mode` | `readonly` | `readonly` or `readwrite` |
| `MCP_TRANSPORT` | `--transport` | `stdio` | `stdio` or `http` |
| `MCP_HTTP_ADDR` | `--http-addr` | `:8080` | HTTP listen address |
| `MCP_AUTH_TOKEN` | `--auth-token` | empty | Bearer token required from HTTP MCP clients |

Run `gitlab-mcp --version` to print the build version.

## Tools

Project arguments accept either a numeric project ID or an unescaped full path
such as `group/subgroup/project`.

### Read-only tools

| Tool | Purpose |
| --- | --- |
| `gitlab_get_metadata` | Get the connected GitLab version and instance metadata |
| `gitlab_get_current_user` | Get the user represented by the configured token |
| `gitlab_list_projects` | List and filter visible projects |
| `gitlab_get_project` | Get a project by ID or path |
| `gitlab_list_issues` | List issues in a project or across projects |
| `gitlab_get_issue` | Get an issue by project and IID |
| `gitlab_list_issue_notes` | List issue comments and system notes |
| `gitlab_list_merge_requests` | List merge requests in a project or across projects |
| `gitlab_get_merge_request` | Get a merge request by project and IID |
| `gitlab_list_merge_request_notes` | List merge request notes |
| `gitlab_list_repository_tree` | Browse a repository tree |
| `gitlab_get_file` | Read and decode a repository file at a ref |
| `gitlab_list_branches` | List repository branches |
| `gitlab_list_commits` | List repository commits |
| `gitlab_get_commit` | Get one commit |
| `gitlab_list_pipelines` | List project pipelines |
| `gitlab_get_pipeline` | Get one pipeline |
| `gitlab_list_pipeline_jobs` | List jobs in a pipeline |
| `gitlab_search` | Search globally or within a group or project |

### Write tools

These tools are registered only when `GITLAB_MODE=readwrite` or
`--mode=readwrite` is set.

| Tool | Purpose |
| --- | --- |
| `gitlab_create_issue` | Create an issue |
| `gitlab_update_issue` | Update, close, or reopen an issue |
| `gitlab_add_issue_note` | Add an issue note |
| `gitlab_create_merge_request` | Create a merge request |
| `gitlab_update_merge_request` | Update, close, or reopen a merge request |
| `gitlab_add_merge_request_note` | Add a merge request note |
| `gitlab_accept_merge_request` | Merge now or after the pipeline succeeds |
| `gitlab_create_branch` | Create a branch from a ref |
| `gitlab_create_commit` | Commit a batch of file create/update/move/delete/chmod actions |
| `gitlab_create_pipeline` | Run a pipeline, optionally with variables and typed inputs |
| `gitlab_retry_pipeline` | Retry failed or canceled pipeline jobs |

Issue and merge-request descriptions and notes use GitLab Flavored Markdown
directly. Repository file responses are capped at 25 MiB. Transient `429`,
`502`, `503`, and `504` responses are retried up to three times, honoring
`Retry-After`.

## Authentication and token scopes

The server sends the configured token in GitLab's recommended `PRIVATE-TOKEN`
header. Use the narrowest token and project membership that supports the tools
you enable:

- `readonly`: use `read_api`; add `read_repository` when repository content is
  needed and the token type supports that scope.
- `readwrite`: use `api`, because issue, merge request, repository, and pipeline
  writes require API write access.

GitLab authorization still applies to every request. A token cannot access or
modify resources that its user, project, or group membership cannot access.

## Transports

### stdio

```bash
gitlab-mcp --transport=stdio
```

### Streamable HTTP

```bash
gitlab-mcp --transport=http --http-addr=127.0.0.1:8080 \
  --auth-token=choose-a-separate-mcp-token
```

HTTP clients must then send `Authorization: Bearer <mcp-token>`. This token
protects the MCP endpoint and is separate from `GITLAB_TOKEN`. If no MCP auth
token is configured, the HTTP endpoint has no application-level
authentication. The server logs a warning when that is combined with
`readwrite` mode.

## MCP client configuration

Claude Desktop:

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "gitlab-mcp",
      "args": [],
      "env": {
        "GITLAB_BASE_URL": "https://gitlab.com",
        "GITLAB_TOKEN": "glpat-your-token",
        "GITLAB_MODE": "readonly"
      }
    }
  }
}
```

VS Code `.vscode/mcp.json`:

```json
{
  "servers": {
    "gitlab": {
      "command": "gitlab-mcp",
      "args": [],
      "env": {
        "GITLAB_BASE_URL": "https://gitlab.com",
        "GITLAB_TOKEN": "glpat-your-token",
        "GITLAB_MODE": "readonly"
      }
    }
  }
}
```

## Development

Requires Go 1.26 or newer.

```bash
go build ./...
go test ./...
go vet ./...
```

Tests cover configuration, authentication, project-path and file-path
encoding, pagination, API errors, retry handling, file decoding, commit
payloads, tool mode gating, and end-to-end stdio and HTTP MCP connections.

## License

MIT. See [LICENSE](LICENSE).
