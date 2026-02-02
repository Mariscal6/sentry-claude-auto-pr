package gitprovider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"
)

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// GitHubProvider implements Provider for GitHub repositories.
type GitHubProvider struct {
	client *github.Client
	owner  string
	repo   string
}

// NewGitHubProvider creates a new GitHub provider.
func NewGitHubProvider(token, owner, repo string) *GitHubProvider {
	client := github.NewClient(nil).WithAuthToken(token)
	return &GitHubProvider{
		client: client,
		owner:  owner,
		repo:   repo,
	}
}

// Owner returns the repository owner.
func (g *GitHubProvider) Owner() string {
	return g.owner
}

// Repo returns the repository name.
func (g *GitHubProvider) Repo() string {
	return g.repo
}

// FetchFile retrieves file content at a specific ref.
func (g *GitHubProvider) FetchFile(ctx context.Context, path, ref string) (*FileContent, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	content, _, _, err := g.client.Repositories.GetContents(ctx, g.owner, g.repo, path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file %s: %w", path, err)
	}

	if content == nil {
		return nil, fmt.Errorf("path %s is a directory, not a file", path)
	}

	decoded, err := content.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to decode content: %w", err)
	}

	return &FileContent{
		Path:     path,
		Content:  decoded,
		SHA:      content.GetSHA(),
		Size:     int64(content.GetSize()),
		Encoding: content.GetEncoding(),
	}, nil
}

// SearchCode searches for code matching the query.
func (g *GitHubProvider) SearchCode(ctx context.Context, query string) ([]SearchResult, error) {
	// Scope search to this repository
	scopedQuery := fmt.Sprintf("%s repo:%s/%s", query, g.owner, g.repo)

	results, _, err := g.client.Search.Code(ctx, scopedQuery, &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 30},
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var searchResults []SearchResult
	for _, result := range results.CodeResults {
		sr := SearchResult{
			Path:       result.GetPath(),
			Repository: result.GetRepository().GetFullName(),
		}

		// Extract text matches if available
		for _, match := range result.TextMatches {
			for _, fragment := range match.Matches {
				sr.Matches = append(sr.Matches, SearchMatch{
					Content: fragment.GetText(),
				})
			}
		}

		searchResults = append(searchResults, sr)
	}

	return searchResults, nil
}

// ListDirectory lists contents of a directory at a specific ref.
func (g *GitHubProvider) ListDirectory(ctx context.Context, path, ref string) ([]DirEntry, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	_, dirContents, _, err := g.client.Repositories.GetContents(ctx, g.owner, g.repo, path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory %s: %w", path, err)
	}

	var entries []DirEntry
	for _, item := range dirContents {
		entries = append(entries, DirEntry{
			Name: item.GetName(),
			Path: item.GetPath(),
			Type: item.GetType(),
			Size: int64(item.GetSize()),
			SHA:  item.GetSHA(),
		})
	}

	return entries, nil
}

// GetDefaultBranch returns the repository's default branch name.
func (g *GitHubProvider) GetDefaultBranch(ctx context.Context) (string, error) {
	repo, _, err := g.client.Repositories.Get(ctx, g.owner, g.repo)
	if err != nil {
		return "", fmt.Errorf("failed to get repository: %w", err)
	}
	return repo.GetDefaultBranch(), nil
}

// GetLatestCommitSHA returns the SHA of the latest commit on a branch.
func (g *GitHubProvider) GetLatestCommitSHA(ctx context.Context, branch string) (string, error) {
	ref, _, err := g.client.Git.GetRef(ctx, g.owner, g.repo, "refs/heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("failed to get ref for branch %s: %w", branch, err)
	}
	return ref.GetObject().GetSHA(), nil
}

// CreateBranch creates a new branch from a base commit SHA.
func (g *GitHubProvider) CreateBranch(ctx context.Context, name, baseSHA string) error {
	ref := &github.Reference{
		Ref:    ptr("refs/heads/" + name),
		Object: &github.GitObject{SHA: ptr(baseSHA)},
	}

	_, _, err := g.client.Git.CreateRef(ctx, g.owner, g.repo, ref)
	if err != nil {
		// Check if branch already exists
		if strings.Contains(err.Error(), "Reference already exists") {
			return nil // Branch already exists, which is fine
		}
		return fmt.Errorf("failed to create branch %s: %w", name, err)
	}

	return nil
}

// CommitFiles commits file changes to a branch.
func (g *GitHubProvider) CommitFiles(ctx context.Context, branch string, files []FileChange, message string) (string, error) {
	// Get the current commit SHA for the branch
	ref, _, err := g.client.Git.GetRef(ctx, g.owner, g.repo, "refs/heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch ref: %w", err)
	}
	parentSHA := ref.GetObject().GetSHA()

	// Get the tree from the parent commit
	parentCommit, _, err := g.client.Git.GetCommit(ctx, g.owner, g.repo, parentSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get parent commit: %w", err)
	}
	baseTreeSHA := parentCommit.GetTree().GetSHA()

	// Create tree entries for changed files
	var treeEntries []*github.TreeEntry
	for _, file := range files {
		mode := file.Mode
		if mode == "" {
			mode = "100644"
		}
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path:    ptr(file.Path),
			Mode:    ptr(mode),
			Type:    ptr("blob"),
			Content: ptr(file.Content),
		})
	}

	// Create new tree
	newTree, _, err := g.client.Git.CreateTree(ctx, g.owner, g.repo, baseTreeSHA, treeEntries)
	if err != nil {
		return "", fmt.Errorf("failed to create tree: %w", err)
	}

	// Create commit
	commit := &github.Commit{
		Message: ptr(message),
		Tree:    newTree,
		Parents: []*github.Commit{{SHA: ptr(parentSHA)}},
	}

	newCommit, _, err := g.client.Git.CreateCommit(ctx, g.owner, g.repo, commit, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	// Update branch ref
	ref.Object.SHA = newCommit.SHA
	_, _, err = g.client.Git.UpdateRef(ctx, g.owner, g.repo, ref, false)
	if err != nil {
		return "", fmt.Errorf("failed to update branch ref: %w", err)
	}

	return newCommit.GetSHA(), nil
}

// CreatePullRequest creates a pull request.
func (g *GitHubProvider) CreatePullRequest(ctx context.Context, req PRRequest) (*PRResponse, error) {
	pr := &github.NewPullRequest{
		Title: ptr(req.Title),
		Body:  ptr(req.Body),
		Head:  ptr(req.Head),
		Base:  ptr(req.Base),
		Draft: ptr(req.Draft),
	}

	created, _, err := g.client.PullRequests.Create(ctx, g.owner, g.repo, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	// Add labels if specified
	if len(req.Labels) > 0 {
		_, _, err = g.client.Issues.AddLabelsToIssue(ctx, g.owner, g.repo, created.GetNumber(), req.Labels)
		if err != nil {
			// Non-fatal, just log
			fmt.Printf("warning: failed to add labels: %v\n", err)
		}
	}

	// Add assignees if specified
	if len(req.Assignees) > 0 {
		_, _, err = g.client.Issues.AddAssignees(ctx, g.owner, g.repo, created.GetNumber(), req.Assignees)
		if err != nil {
			// Non-fatal, just log
			fmt.Printf("warning: failed to add assignees: %v\n", err)
		}
	}

	return &PRResponse{
		Number:  created.GetNumber(),
		URL:     created.GetURL(),
		HTMLURL: created.GetHTMLURL(),
	}, nil
}

// decodeBase64Content decodes base64-encoded content.
func decodeBase64Content(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
