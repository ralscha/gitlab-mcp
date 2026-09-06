package gitlab

type PageInfo struct {
	Page         int  `json:"page"`
	PerPage      int  `json:"per_page"`
	Total        int  `json:"total,omitempty"`
	TotalPages   int  `json:"total_pages,omitempty"`
	NextPage     int  `json:"next_page,omitempty"`
	PreviousPage int  `json:"previous_page,omitempty"`
	LastPage     bool `json:"last_page"`
}

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	State     string `json:"state,omitempty"`
	WebURL    string `json:"web_url,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Email     string `json:"email,omitempty"`
}

type Namespace struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	FullPath string `json:"full_path"`
	Kind     string `json:"kind"`
	WebURL   string `json:"web_url,omitempty"`
}

type Project struct {
	ID                int        `json:"id"`
	Name              string     `json:"name"`
	NameWithNamespace string     `json:"name_with_namespace"`
	Path              string     `json:"path"`
	PathWithNamespace string     `json:"path_with_namespace"`
	Description       string     `json:"description,omitempty"`
	WebURL            string     `json:"web_url"`
	DefaultBranch     string     `json:"default_branch,omitempty"`
	Visibility        string     `json:"visibility"`
	Archived          bool       `json:"archived"`
	Topics            []string   `json:"topics,omitempty"`
	LastActivityAt    string     `json:"last_activity_at,omitempty"`
	Namespace         *Namespace `json:"namespace,omitempty"`
}

type Milestone struct {
	ID    int    `json:"id"`
	IID   int    `json:"iid"`
	Title string `json:"title"`
	State string `json:"state,omitempty"`
}

type References struct {
	Short    string `json:"short,omitempty"`
	Relative string `json:"relative,omitempty"`
	Full     string `json:"full,omitempty"`
}

type Issue struct {
	ID           int         `json:"id"`
	IID          int         `json:"iid"`
	ProjectID    int         `json:"project_id"`
	Title        string      `json:"title"`
	Description  string      `json:"description,omitempty"`
	State        string      `json:"state"`
	IssueType    string      `json:"issue_type,omitempty"`
	Confidential bool        `json:"confidential"`
	Labels       []string    `json:"labels,omitempty"`
	Author       *User       `json:"author,omitempty"`
	Assignees    []User      `json:"assignees,omitempty"`
	Milestone    *Milestone  `json:"milestone,omitempty"`
	References   *References `json:"references,omitempty"`
	WebURL       string      `json:"web_url"`
	DueDate      string      `json:"due_date,omitempty"`
	CreatedAt    string      `json:"created_at,omitempty"`
	UpdatedAt    string      `json:"updated_at,omitempty"`
	ClosedAt     string      `json:"closed_at,omitempty"`
}

type PipelineRef struct {
	ID     int    `json:"id"`
	Status string `json:"status,omitempty"`
	Ref    string `json:"ref,omitempty"`
	SHA    string `json:"sha,omitempty"`
	WebURL string `json:"web_url,omitempty"`
}

type MergeRequest struct {
	ID                        int          `json:"id"`
	IID                       int          `json:"iid"`
	ProjectID                 int          `json:"project_id"`
	Title                     string       `json:"title"`
	Description               string       `json:"description,omitempty"`
	State                     string       `json:"state"`
	Draft                     bool         `json:"draft"`
	SourceBranch              string       `json:"source_branch"`
	TargetBranch              string       `json:"target_branch"`
	Author                    *User        `json:"author,omitempty"`
	Assignees                 []User       `json:"assignees,omitempty"`
	Reviewers                 []User       `json:"reviewers,omitempty"`
	Labels                    []string     `json:"labels,omitempty"`
	Milestone                 *Milestone   `json:"milestone,omitempty"`
	DetailedMergeStatus       string       `json:"detailed_merge_status,omitempty"`
	MergeWhenPipelineSucceeds bool         `json:"merge_when_pipeline_succeeds"`
	HasConflicts              bool         `json:"has_conflicts"`
	SHA                       string       `json:"sha,omitempty"`
	MergeCommitSHA            string       `json:"merge_commit_sha,omitempty"`
	Pipeline                  *PipelineRef `json:"pipeline,omitempty"`
	WebURL                    string       `json:"web_url"`
	CreatedAt                 string       `json:"created_at,omitempty"`
	UpdatedAt                 string       `json:"updated_at,omitempty"`
	MergedAt                  string       `json:"merged_at,omitempty"`
	ClosedAt                  string       `json:"closed_at,omitempty"`
}

type MergeRequestDiff struct {
	OldPath       string `json:"old_path"`
	NewPath       string `json:"new_path"`
	OldMode       string `json:"a_mode"`
	NewMode       string `json:"b_mode"`
	Diff          string `json:"diff"`
	NewFile       bool   `json:"new_file"`
	RenamedFile   bool   `json:"renamed_file"`
	DeletedFile   bool   `json:"deleted_file"`
	GeneratedFile bool   `json:"generated_file"`
	Collapsed     bool   `json:"collapsed"`
	TooLarge      bool   `json:"too_large"`
}

type Note struct {
	ID         int    `json:"id"`
	Body       string `json:"body"`
	Author     *User  `json:"author,omitempty"`
	System     bool   `json:"system"`
	Internal   bool   `json:"internal"`
	Resolvable bool   `json:"resolvable"`
	Resolved   bool   `json:"resolved"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

type Commit struct {
	ID            string         `json:"id"`
	ShortID       string         `json:"short_id"`
	Title         string         `json:"title"`
	Message       string         `json:"message,omitempty"`
	AuthorName    string         `json:"author_name,omitempty"`
	AuthorEmail   string         `json:"author_email,omitempty"`
	AuthoredDate  string         `json:"authored_date,omitempty"`
	CommittedDate string         `json:"committed_date,omitempty"`
	CreatedAt     string         `json:"created_at,omitempty"`
	ParentIDs     []string       `json:"parent_ids,omitempty"`
	WebURL        string         `json:"web_url,omitempty"`
	Stats         map[string]int `json:"stats,omitempty"`
}

type Branch struct {
	Name      string  `json:"name"`
	Merged    bool    `json:"merged"`
	Protected bool    `json:"protected"`
	Default   bool    `json:"default"`
	CanPush   bool    `json:"can_push"`
	WebURL    string  `json:"web_url,omitempty"`
	Commit    *Commit `json:"commit,omitempty"`
}

type TreeItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type RepositoryFile struct {
	FileName      string `json:"file_name"`
	FilePath      string `json:"file_path"`
	Size          int    `json:"size"`
	Encoding      string `json:"encoding"`
	Content       string `json:"content"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
	Ref           string `json:"ref"`
	BlobID        string `json:"blob_id"`
	CommitID      string `json:"commit_id"`
	LastCommitID  string `json:"last_commit_id"`
}

type Pipeline struct {
	ID             int     `json:"id"`
	IID            int     `json:"iid,omitempty"`
	ProjectID      int     `json:"project_id,omitempty"`
	Name           string  `json:"name,omitempty"`
	Status         string  `json:"status"`
	Source         string  `json:"source,omitempty"`
	Ref            string  `json:"ref,omitempty"`
	SHA            string  `json:"sha,omitempty"`
	WebURL         string  `json:"web_url,omitempty"`
	CreatedAt      string  `json:"created_at,omitempty"`
	UpdatedAt      string  `json:"updated_at,omitempty"`
	StartedAt      string  `json:"started_at,omitempty"`
	FinishedAt     string  `json:"finished_at,omitempty"`
	Duration       float64 `json:"duration,omitempty"`
	QueuedDuration float64 `json:"queued_duration,omitempty"`
	User           *User   `json:"user,omitempty"`
}

type Job struct {
	ID             int          `json:"id"`
	Name           string       `json:"name"`
	Stage          string       `json:"stage"`
	Status         string       `json:"status"`
	Ref            string       `json:"ref,omitempty"`
	Tag            bool         `json:"tag"`
	AllowFailure   bool         `json:"allow_failure"`
	WebURL         string       `json:"web_url,omitempty"`
	CreatedAt      string       `json:"created_at,omitempty"`
	StartedAt      string       `json:"started_at,omitempty"`
	FinishedAt     string       `json:"finished_at,omitempty"`
	Duration       float64      `json:"duration,omitempty"`
	QueuedDuration float64      `json:"queued_duration,omitempty"`
	User           *User        `json:"user,omitempty"`
	Pipeline       *PipelineRef `json:"pipeline,omitempty"`
}

type Metadata struct {
	Version    string         `json:"version"`
	Revision   string         `json:"revision"`
	Enterprise bool           `json:"enterprise"`
	KAS        map[string]any `json:"kas,omitempty"`
}
