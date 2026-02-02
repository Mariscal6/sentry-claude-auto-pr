package gitprovider

import "context"

// FileContent represents the content of a file from a git provider.
type FileContent struct {
	Path     string
	Content  string
	SHA      string
	Size     int64
	Encoding string
}

// SearchResult represents a code search result.
type SearchResult struct {
	Path       string
	Repository string
	Matches    []SearchMatch
}

// SearchMatch represents a single match within a file.
type SearchMatch struct {
	LineNumber int
	Content    string
}

// DirEntry represents a directory entry.
type DirEntry struct {
	Name string
	Path string
	Type string // "file", "dir", "symlink", "submodule"
	Size int64
	SHA  string
}

// FileChange represents a file modification for a commit.
type FileChange struct {
	Path    string
	Content string
	Mode    string // "100644" for regular files, "100755" for executables
}

// PRRequest represents a pull request creation request.
type PRRequest struct {
	Title     string
	Body      string
	Head      string // source branch
	Base      string // target branch (e.g., "main")
	Draft     bool
	Labels    []string
	Assignees []string
}

// PRResponse represents a created pull request.
type PRResponse struct {
	Number  int
	URL     string
	HTMLURL string
}

// Provider defines the interface for git operations.
// This abstraction allows supporting multiple git providers (GitHub, GitLab, Bitbucket).
type Provider interface {
	// FetchFile retrieves file content at a specific ref (branch, tag, commit SHA).
	FetchFile(ctx context.Context, path, ref string) (*FileContent, error)

	// SearchCode searches for code matching the query.
	SearchCode(ctx context.Context, query string) ([]SearchResult, error)

	// ListDirectory lists contents of a directory at a specific ref.
	ListDirectory(ctx context.Context, path, ref string) ([]DirEntry, error)

	// GetDefaultBranch returns the repository's default branch name.
	GetDefaultBranch(ctx context.Context) (string, error)

	// GetLatestCommitSHA returns the SHA of the latest commit on a branch.
	GetLatestCommitSHA(ctx context.Context, branch string) (string, error)

	// CreateBranch creates a new branch from a base commit SHA.
	CreateBranch(ctx context.Context, name, baseSHA string) error

	// CommitFiles commits file changes to a branch.
	CommitFiles(ctx context.Context, branch string, files []FileChange, message string) (string, error)

	// CreatePullRequest creates a pull request.
	CreatePullRequest(ctx context.Context, req PRRequest) (*PRResponse, error)

	// Owner returns the repository owner.
	Owner() string

	// Repo returns the repository name.
	Repo() string
}

// ProviderType represents the type of git provider.
type ProviderType string

const (
	ProviderTypeGitHub ProviderType = "github"
)
