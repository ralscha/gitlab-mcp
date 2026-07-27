package gitlab

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ListTreeOptions struct {
	Path      string
	Ref       string
	Recursive bool
	Page      int
	PerPage   int
}

func (c *Client) ListRepositoryTree(ctx context.Context, project string, opts ListTreeOptions) ([]TreeItem, PageInfo, error) {
	query := pageQuery(opts.Page, opts.PerPage)
	setIf(query, "path", opts.Path)
	setIf(query, "ref", opts.Ref)
	if opts.Recursive {
		query.Set("recursive", "true")
	}
	var items []TreeItem
	headers, err := c.doJSON(ctx, http.MethodGet, projectPath(project)+"/repository/tree", query, nil, &items)
	return items, pagination(headers, opts.Page, opts.PerPage, len(items)), err
}

type FileContent struct {
	FileName      string `json:"file_name"`
	FilePath      string `json:"file_path"`
	Size          int    `json:"size"`
	Content       string `json:"content"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
	Ref           string `json:"ref"`
	BlobID        string `json:"blob_id"`
	CommitID      string `json:"commit_id"`
	LastCommitID  string `json:"last_commit_id"`
}

func (c *Client) GetFile(ctx context.Context, project, filePath, ref string) (*FileContent, error) {
	query := url.Values{"ref": {ref}}
	var file RepositoryFile
	path := projectPath(project) + "/repository/files/" + url.PathEscape(filePath)
	if _, err := c.doJSON(ctx, http.MethodGet, path, query, nil, &file); err != nil {
		return nil, err
	}
	if file.Encoding != "base64" {
		return nil, fmt.Errorf("gitlab: unsupported repository file encoding %q", file.Encoding)
	}
	compact := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, file.Content)
	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return nil, fmt.Errorf("gitlab: decoding repository file: %w", err)
	}
	if len(decoded) > MaxFileBytes {
		return nil, fmt.Errorf("%w: decoded file is %d bytes, limit is %d", ErrTooLarge, len(decoded), MaxFileBytes)
	}
	return &FileContent{
		FileName: file.FileName, FilePath: file.FilePath, Size: file.Size,
		Content: string(decoded), ContentSHA256: file.ContentSHA256, Ref: file.Ref,
		BlobID: file.BlobID, CommitID: file.CommitID, LastCommitID: file.LastCommitID,
	}, nil
}

type ListBranchesOptions struct {
	Search  string
	Regex   string
	Page    int
	PerPage int
}

func (c *Client) ListBranches(ctx context.Context, project string, opts ListBranchesOptions) ([]Branch, PageInfo, error) {
	query := pageQuery(opts.Page, opts.PerPage)
	setIf(query, "search", opts.Search)
	setIf(query, "regex", opts.Regex)
	var branches []Branch
	headers, err := c.doJSON(ctx, http.MethodGet, projectPath(project)+"/repository/branches", query, nil, &branches)
	return branches, pagination(headers, opts.Page, opts.PerPage, len(branches)), err
}

func (c *Client) CreateBranch(ctx context.Context, project, branch, ref string) (*Branch, error) {
	body := map[string]any{"branch": branch, "ref": ref}
	var result Branch
	_, err := c.doJSON(ctx, http.MethodPost, projectPath(project)+"/repository/branches", nil, body, &result)
	return &result, err
}

type ListCommitsOptions struct {
	RefName     string
	Path        string
	Since       string
	Until       string
	WithStats   bool
	FirstParent bool
	Page        int
	PerPage     int
}

func (c *Client) ListCommits(ctx context.Context, project string, opts ListCommitsOptions) ([]Commit, PageInfo, error) {
	query := pageQuery(opts.Page, opts.PerPage)
	setIf(query, "ref_name", opts.RefName)
	setIf(query, "path", opts.Path)
	setIf(query, "since", opts.Since)
	setIf(query, "until", opts.Until)
	if opts.WithStats {
		query.Set("with_stats", "true")
	}
	if opts.FirstParent {
		query.Set("first_parent", "true")
	}
	var commits []Commit
	headers, err := c.doJSON(ctx, http.MethodGet, projectPath(project)+"/repository/commits", query, nil, &commits)
	return commits, pagination(headers, opts.Page, opts.PerPage, len(commits)), err
}

func (c *Client) GetCommit(ctx context.Context, project, sha string, stats bool) (*Commit, error) {
	query := url.Values{"stats": {strconv.FormatBool(stats)}}
	var commit Commit
	path := projectPath(project) + "/repository/commits/" + url.PathEscape(sha)
	_, err := c.doJSON(ctx, http.MethodGet, path, query, nil, &commit)
	return &commit, err
}

type CommitAction struct {
	Action          string `json:"action"`
	FilePath        string `json:"file_path"`
	PreviousPath    string `json:"previous_path,omitempty"`
	Content         string `json:"content,omitempty"`
	Encoding        string `json:"encoding,omitempty"`
	LastCommitID    string `json:"last_commit_id,omitempty"`
	ExecuteFilemode *bool  `json:"execute_filemode,omitempty"`
}

type CreateCommitInput struct {
	Branch        string
	CommitMessage string
	StartBranch   string
	StartSHA      string
	AuthorEmail   string
	AuthorName    string
	Actions       []CommitAction
}

func (c *Client) CreateCommit(ctx context.Context, project string, in CreateCommitInput) (*Commit, error) {
	if len(in.Actions) == 0 {
		return nil, fmt.Errorf("gitlab: CreateCommit requires at least one action")
	}
	for i, action := range in.Actions {
		switch action.Action {
		case "create", "delete", "move", "update", "chmod":
		default:
			return nil, fmt.Errorf("gitlab: action %d has invalid action %q", i, action.Action)
		}
		if action.FilePath == "" {
			return nil, fmt.Errorf("gitlab: action %d requires file_path", i)
		}
		if action.Encoding != "" && action.Encoding != "text" && action.Encoding != "base64" {
			return nil, fmt.Errorf("gitlab: action %d has invalid encoding %q", i, action.Encoding)
		}
		if action.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(action.Content)
			if err != nil {
				return nil, fmt.Errorf("gitlab: action %d has invalid base64 content: %w", i, err)
			}
			if len(decoded) > MaxFileBytes {
				return nil, fmt.Errorf("%w: action %d content is %d bytes", ErrTooLarge, i, len(decoded))
			}
		} else if len(action.Content) > MaxFileBytes {
			return nil, fmt.Errorf("%w: action %d content is %d bytes", ErrTooLarge, i, len(action.Content))
		}
	}
	body := map[string]any{"branch": in.Branch, "commit_message": in.CommitMessage, "actions": in.Actions}
	if in.StartBranch != "" {
		body["start_branch"] = in.StartBranch
	}
	if in.StartSHA != "" {
		body["start_sha"] = in.StartSHA
	}
	if in.AuthorEmail != "" {
		body["author_email"] = in.AuthorEmail
	}
	if in.AuthorName != "" {
		body["author_name"] = in.AuthorName
	}
	var commit Commit
	_, err := c.doJSON(ctx, http.MethodPost, projectPath(project)+"/repository/commits", nil, body, &commit)
	return &commit, err
}
