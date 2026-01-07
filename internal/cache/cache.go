// ABOUTME: Caches repository data to avoid slow API calls on startup.
// ABOUTME: Uses time-based invalidation with 5-minute TTL.

package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/user/gh-repo-review/internal/repo"
)

const (
	cacheTTL = 5 * time.Minute
)

// CachedData holds the cached repository data with metadata.
type CachedData struct {
	Username string      `json:"username"`
	CachedAt time.Time   `json:"cached_at"`
	Repos    []repo.Repo `json:"repos"`
}

// cacheDir returns the cache directory path.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "gh-repo-review"), nil
}

// cacheFilePath returns the cache file path for a given username.
func cacheFilePath(username string) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, username+"-repos.json"), nil
}

// Load reads cached repos for a user. Returns repos, whether cache is fresh, and any error.
// If cache doesn't exist or is corrupted, returns nil repos with no error.
func Load(username string) ([]repo.Repo, bool, error) {
	path, err := cacheFilePath(username)
	if err != nil {
		return nil, false, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var cached CachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		// Corrupted cache, treat as miss
		return nil, false, nil
	}

	fresh := time.Since(cached.CachedAt) < cacheTTL
	return cached.Repos, fresh, nil
}

// Save writes repos to cache for a user.
func Save(username string, repos []repo.Repo) error {
	dir, err := cacheDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := cacheFilePath(username)
	if err != nil {
		return err
	}

	cached := CachedData{
		Username: username,
		CachedAt: time.Now(),
		Repos:    repos,
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
