package repo

import (
	"fmt"
	"strings"
	"time"
)

// Repo represents a GitHub repository
type Repo struct {
	Name            string    `json:"name"`
	FullName        string    `json:"nameWithOwner"`
	Description     string    `json:"description"`
	URL             string    `json:"url"`
	SSHURL          string    `json:"sshUrl"`
	IsPrivate       bool      `json:"isPrivate"`
	IsArchived      bool      `json:"isArchived"`
	IsFork          bool      `json:"isFork"`
	IsTemplate      bool      `json:"isTemplate"`
	StargazerCount  int       `json:"stargazerCount"`
	ForkCount       int       `json:"forkCount"`
	OpenIssuesCount int       `json:"openIssues"`
	PrimaryLanguage string    `json:"primaryLanguage"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	PushedAt        time.Time `json:"pushedAt"`
	DiskUsage       int       `json:"diskUsage"` // in KB
	Selected        bool      // for multi-select in TUI
}

// FilterOptions holds the filter criteria
type FilterOptions struct {
	ShowArchived     bool
	ShowPrivate      bool
	ShowPublic       bool
	ShowForks        bool
	Language         string
	MinStars         int
	MaxStars         int
	InactiveForDays  int // repos not updated in X days
	SearchQuery      string
	SortBy           SortField
	SortDesc         bool
}

// SortField represents sortable fields
type SortField int

const (
	SortByName SortField = iota
	SortByUpdated
	SortByCreated
	SortByStars
	SortByForks
	SortBySize
)

func (s SortField) String() string {
	switch s {
	case SortByName:
		return "Name"
	case SortByUpdated:
		return "Last Updated"
	case SortByCreated:
		return "Created"
	case SortByStars:
		return "Stars"
	case SortByForks:
		return "Forks"
	case SortBySize:
		return "Size"
	default:
		return "Unknown"
	}
}

// DefaultFilterOptions returns sensible default filters
func DefaultFilterOptions() FilterOptions {
	return FilterOptions{
		ShowArchived:    false,
		ShowPrivate:     true,
		ShowPublic:      true,
		ShowForks:       true,
		Language:        "",
		MinStars:        -1,
		MaxStars:        -1,
		InactiveForDays: 0,
		SearchQuery:     "",
		SortBy:          SortByUpdated,
		SortDesc:        true,
	}
}

// Filter applies the filter options to a list of repos
func Filter(repos []Repo, opts FilterOptions) []Repo {
	var result []Repo

	cutoff := time.Time{}
	if opts.InactiveForDays > 0 {
		cutoff = time.Now().AddDate(0, 0, -opts.InactiveForDays)
	}

	for _, r := range repos {
		// Skip archived repos if not showing them
		if r.IsArchived && !opts.ShowArchived {
			continue
		}

		// Privacy filter
		if r.IsPrivate && !opts.ShowPrivate {
			continue
		}
		if !r.IsPrivate && !opts.ShowPublic {
			continue
		}

		// Fork filter
		if r.IsFork && !opts.ShowForks {
			continue
		}

		// Language filter
		if opts.Language != "" && !strings.EqualFold(r.PrimaryLanguage, opts.Language) {
			continue
		}

		// Stars filter
		if opts.MinStars >= 0 && r.StargazerCount < opts.MinStars {
			continue
		}
		if opts.MaxStars >= 0 && r.StargazerCount > opts.MaxStars {
			continue
		}

		// Inactivity filter
		if opts.InactiveForDays > 0 && r.PushedAt.After(cutoff) {
			continue
		}

		// Search query
		if opts.SearchQuery != "" {
			query := strings.ToLower(opts.SearchQuery)
			name := strings.ToLower(r.Name)
			desc := strings.ToLower(r.Description)
			if !strings.Contains(name, query) && !strings.Contains(desc, query) {
				continue
			}
		}

		result = append(result, r)
	}

	return result
}

// Sort sorts repos by the specified field
func Sort(repos []Repo, sortBy SortField, desc bool) {
	n := len(repos)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			swap := false
			switch sortBy {
			case SortByName:
				swap = repos[j].Name > repos[j+1].Name
			case SortByUpdated:
				swap = repos[j].UpdatedAt.Before(repos[j+1].UpdatedAt)
			case SortByCreated:
				swap = repos[j].CreatedAt.Before(repos[j+1].CreatedAt)
			case SortByStars:
				swap = repos[j].StargazerCount < repos[j+1].StargazerCount
			case SortByForks:
				swap = repos[j].ForkCount < repos[j+1].ForkCount
			case SortBySize:
				swap = repos[j].DiskUsage < repos[j+1].DiskUsage
			}
			if desc {
				swap = !swap
			}
			if swap {
				repos[j], repos[j+1] = repos[j+1], repos[j]
			}
		}
	}
}

// DaysSinceUpdate returns the number of days since last push
func (r Repo) DaysSinceUpdate() int {
	return int(time.Since(r.PushedAt).Hours() / 24)
}

// SizeString returns a human-readable size string
func (r Repo) SizeString() string {
	kb := r.DiskUsage
	if kb < 1024 {
		return fmt.Sprintf("%d KB", kb)
	}
	mb := float64(kb) / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.2f GB", gb)
}

// VisibilityString returns "Private" or "Public"
func (r Repo) VisibilityString() string {
	if r.IsPrivate {
		return "Private"
	}
	return "Public"
}

// StatusString returns archive/fork/template status
func (r Repo) StatusString() string {
	var parts []string
	if r.IsArchived {
		parts = append(parts, "Archived")
	}
	if r.IsFork {
		parts = append(parts, "Fork")
	}
	if r.IsTemplate {
		parts = append(parts, "Template")
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}
