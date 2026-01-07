# gh-repo-review

A TUI tool to review all your GitHub repositories and automate the process of archiving old repos. Built as a `gh` CLI extension.

![Demo](https://via.placeholder.com/800x400?text=gh-repo-review+TUI)

## Features

- **List all repositories** - View all your GitHub repositories in a beautiful TUI
- **Filter repositories** - Filter by visibility (public/private), archived status, forks, language, and inactivity period
- **Search** - Quick search through repository names and descriptions
- **Sort** - Sort by name, last updated, created date, stars, forks, or size
- **Bulk selection** - Select multiple repositories for batch operations
- **Archive repos** - Archive old/unused repositories with confirmation
- **Delete repos** - Permanently delete repositories (with extra confirmation)
- **Open in browser** - Quickly open any repository in your default browser
- **Keyboard-driven** - Full keyboard navigation for efficient workflow

## Prerequisites

- [Go 1.21+](https://golang.org/dl/) (for building from source)
- [GitHub CLI (gh)](https://cli.github.com/) - must be installed and authenticated

## Permissions

The default `gh` authentication works for listing, archiving, and unarchiving repositories.

**To delete repositories**, you need the `delete_repo` scope:

```bash
gh auth refresh -s delete_repo
```

This will prompt you to re-authenticate and grant the additional permission.

## Installation

### As a gh extension (recommended)

```bash
gh extension install user/gh-repo-review
```

Then run:
```bash
gh repo-review
```

### From source

```bash
git clone https://github.com/user/gh-repo-review.git
cd gh-repo-review
go build -o gh-repo-review .
./gh-repo-review
```

### Install locally as gh extension

```bash
git clone https://github.com/user/gh-repo-review.git
cd gh-repo-review
go build -o gh-repo-review .
gh extension install .
```

## Usage

Simply run the command to start the TUI:

```bash
gh repo-review
# or if installed from source:
./gh-repo-review
```

## Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `PgUp` / `PgDn` | Page up/down |
| `g` / `G` | Go to top/bottom |
| `Enter` / `l` | View repository details |
| `Esc` / `h` | Go back |

### Search & Filter
| Key | Action |
|-----|--------|
| `/` | Search repositories |
| `f` | Open filter panel |
| `s` | Cycle sort field |
| `S` | Toggle sort direction |
| `1` | Toggle archived repos |
| `2` | Toggle private repos |
| `3` | Toggle public repos |
| `4` | Toggle forks |

### Selection
| Key | Action |
|-----|--------|
| `Space` / `x` | Toggle selection |
| `A` | Select all visible |
| `D` | Deselect all |

### Actions
| Key | Action |
|-----|--------|
| `a` | Archive selected repos |
| `d` | Delete selected repos (dangerous!) |
| `o` | Open in browser |
| `r` | Reload repositories |

### General
| Key | Action |
|-----|--------|
| `?` | Show/hide help |
| `q` | Quit |

## Filtering

The filter panel (`f`) allows you to:

- **Show Archived** - Include archived repositories in the list
- **Show Private** - Include private repositories
- **Show Public** - Include public repositories
- **Show Forks** - Include forked repositories
- **Inactive Period** - Only show repos not updated in X days (30, 90, 180, 365, 730)

## Common Workflows

### Find and archive old repos

1. Press `f` to open filters
2. Press `5` to cycle to "inactive > 365 days"
3. Press `f` to close filters
4. Press `A` to select all visible repos
5. Review the selection
6. Press `a` to archive
7. Press `y` to confirm

### Clean up forks

1. Press `f` to open filters
2. Press `2` to hide private repos
3. Press `3` to hide public repos (wait, keep this on)
4. Actually: Press `1`, `2`, `3` to set desired visibility
5. Use `/` to search for specific patterns
6. Select repos with `Space`
7. Archive with `a` or delete with `d`

## Development

### Building

```bash
go build -o gh-repo-review .
```

### Running tests

```bash
go test ./...
```

### Project Structure

```
.
├── main.go                 # Entry point
├── internal/
│   ├── cache/
│   │   └── cache.go       # Repository list caching
│   ├── gh/
│   │   └── client.go      # GitHub API client (via gh CLI)
│   ├── repo/
│   │   └── repo.go        # Repository model and filtering
│   └── tui/
│       ├── model.go       # Bubble Tea model and views
│       └── styles.go      # Lipgloss styles
├── go.mod
├── go.sum
└── README.md
```

Cache is stored at `~/.cache/gh-repo-review/` with a 5-minute TTL. Press `r` to force refresh.

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
