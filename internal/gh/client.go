package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/user/gh-repo-review/internal/repo"
)

// Client wraps the gh CLI for GitHub API operations
type Client struct{}

// NewClient creates a new GitHub client
func NewClient() *Client {
	return &Client{}
}

// ghResponse represents the raw JSON response from gh api
type ghRepoResponse struct {
	Name        string `json:"name"`
	FullName    string `json:"nameWithOwner"`
	Description string `json:"description"`
	URL         string `json:"url"`
	SSHURL      string `json:"sshUrl"`
	IsPrivate   bool   `json:"isPrivate"`
	IsArchived  bool   `json:"isArchived"`
	IsFork      bool   `json:"isFork"`
	IsTemplate  bool   `json:"isTemplate"`
	Stargazers  struct {
		TotalCount int `json:"totalCount"`
	} `json:"stargazerCount"`
	Forks struct {
		TotalCount int `json:"totalCount"`
	} `json:"forkCount"`
	Issues struct {
		TotalCount int `json:"totalCount"`
	} `json:"issues"`
	PrimaryLanguage *struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	PushedAt  string `json:"pushedAt"`
	DiskUsage int    `json:"diskUsage"`
}

// ListReposResponse is the response from listing repos
type listReposResponse struct {
	Repos    []ghRepoResponse `json:"repos"`
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
}

// CheckAuth verifies that gh is authenticated
func (c *Client) CheckAuth() error {
	cmd := exec.Command("gh", "auth", "status")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh auth check failed: %s", stderr.String())
	}
	return nil
}

// GetCurrentUser returns the authenticated user's login
func (c *Client) GetCurrentUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ListRepos fetches all repositories for the authenticated user
func (c *Client) ListRepos() ([]repo.Repo, error) {
	// Use GraphQL for efficient fetching with pagination
	query := `
query($cursor: String) {
  viewer {
    repositories(first: 100, after: $cursor, ownerAffiliations: [OWNER]) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        name
        nameWithOwner
        description
        url
        sshUrl
        isPrivate
        isArchived
        isFork
        isTemplate
        stargazerCount
        forkCount
        issues(states: OPEN) {
          totalCount
        }
        primaryLanguage {
          name
        }
        createdAt
        updatedAt
        pushedAt
        diskUsage
      }
    }
  }
}
`

	var allRepos []repo.Repo
	cursor := ""

	for {
		args := []string{"api", "graphql", "-f", fmt.Sprintf("query=%s", query)}
		if cursor != "" {
			args = append(args, "-f", fmt.Sprintf("cursor=%s", cursor))
		}

		cmd := exec.Command("gh", args...)
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("gh api failed: %s", string(exitErr.Stderr))
			}
			return nil, fmt.Errorf("failed to execute gh: %w", err)
		}

		var result struct {
			Data struct {
				Viewer struct {
					Repositories struct {
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
						Nodes []struct {
							Name        string `json:"name"`
							FullName    string `json:"nameWithOwner"`
							Description string `json:"description"`
							URL         string `json:"url"`
							SSHURL      string `json:"sshUrl"`
							IsPrivate   bool   `json:"isPrivate"`
							IsArchived  bool   `json:"isArchived"`
							IsFork      bool   `json:"isFork"`
							IsTemplate  bool   `json:"isTemplate"`
							Stargazers  int    `json:"stargazerCount"`
							ForkCount   int    `json:"forkCount"`
							Issues      struct {
								TotalCount int `json:"totalCount"`
							} `json:"issues"`
							PrimaryLanguage *struct {
								Name string `json:"name"`
							} `json:"primaryLanguage"`
							CreatedAt string `json:"createdAt"`
							UpdatedAt string `json:"updatedAt"`
							PushedAt  string `json:"pushedAt"`
							DiskUsage int    `json:"diskUsage"`
						} `json:"nodes"`
					} `json:"repositories"`
				} `json:"viewer"`
			} `json:"data"`
		}

		if err := json.Unmarshal(output, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		for _, r := range result.Data.Viewer.Repositories.Nodes {
			createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
			updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)
			pushedAt, _ := time.Parse(time.RFC3339, r.PushedAt)

			lang := ""
			if r.PrimaryLanguage != nil {
				lang = r.PrimaryLanguage.Name
			}

			allRepos = append(allRepos, repo.Repo{
				Name:            r.Name,
				FullName:        r.FullName,
				Description:     r.Description,
				URL:             r.URL,
				SSHURL:          r.SSHURL,
				IsPrivate:       r.IsPrivate,
				IsArchived:      r.IsArchived,
				IsFork:          r.IsFork,
				IsTemplate:      r.IsTemplate,
				StargazerCount:  r.Stargazers,
				ForkCount:       r.ForkCount,
				OpenIssuesCount: r.Issues.TotalCount,
				PrimaryLanguage: lang,
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
				PushedAt:        pushedAt,
				DiskUsage:       r.DiskUsage,
			})
		}

		if !result.Data.Viewer.Repositories.PageInfo.HasNextPage {
			break
		}
		cursor = result.Data.Viewer.Repositories.PageInfo.EndCursor
	}

	return allRepos, nil
}

// ArchiveRepo archives a repository
func (c *Client) ArchiveRepo(fullName string) error {
	cmd := exec.Command("gh", "repo", "archive", fullName, "--yes")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to archive %s: %s", fullName, stderr.String())
	}
	return nil
}

// UnarchiveRepo unarchives a repository
func (c *Client) UnarchiveRepo(fullName string) error {
	cmd := exec.Command("gh", "repo", "unarchive", fullName, "--yes")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unarchive %s: %s", fullName, stderr.String())
	}
	return nil
}

// DeleteRepo deletes a repository (dangerous!)
func (c *Client) DeleteRepo(fullName string) error {
	cmd := exec.Command("gh", "repo", "delete", fullName, "--yes")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete %s: %s", fullName, stderr.String())
	}
	return nil
}

// OpenInBrowser opens the repository in the default browser
func (c *Client) OpenInBrowser(fullName string) error {
	cmd := exec.Command("gh", "repo", "view", fullName, "--web")
	return cmd.Start()
}

// GetRepoStats returns detailed stats for a repo
func (c *Client) GetRepoStats(fullName string) (map[string]interface{}, error) {
	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s", fullName))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo stats: %w", err)
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return stats, nil
}
