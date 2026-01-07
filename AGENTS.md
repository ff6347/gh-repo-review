# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gh-repo-review is a TUI tool for managing GitHub repositories, built as a `gh` CLI extension. It provides an interactive terminal interface for listing, filtering, archiving, and deleting repositories.

## Commands

```bash
make build       # Build binary
make test        # Run tests: go test -v ./...
make lint        # Run linter: golangci-lint run
make run         # Build and run locally
make install     # Build and install as gh extension
make dev         # Hot reload development (requires air)
```

Run a single test:
```bash
go test -v -run TestFunctionName ./internal/repo/
```

## Architecture

The project has three internal packages with clear responsibilities:

**internal/gh/client.go** - GitHub API wrapper
- All GitHub operations go through the `gh` CLI (not direct API calls)
- Uses GraphQL with pagination for repository fetching
- Exposes: `CheckAuth()`, `GetUsername()`, `ListRepositories()`, `ArchiveRepository()`, `UnarchiveRepository()`, `DeleteRepository()`, `OpenInBrowser()`, `GetRepoStats()`

**internal/repo/repo.go** - Data models and business logic
- `Repo` struct holds all repository metadata
- `FilterOptions` struct defines filtering/sorting criteria
- Implements `ApplyFilters()` for filtering and sorting repository lists

**internal/tui/model.go** - Bubble Tea application
- Main TUI model managing application state across 6 views: List, Filter, Detail, ConfirmArchive, ConfirmDelete, Help
- Implements `Init()`, `Update()`, `View()` for Bubble Tea
- Handles keyboard input with context-aware bindings per view
- Async operations (load, archive, delete) use Bubble Tea's command pattern

**internal/tui/styles.go** - Lip Gloss styling
- Color palette: purple (primary), green (secondary), amber (warning), red (danger)
- Language-specific color mappings for display

## Key Patterns

- All GitHub operations are commands that return `tea.Cmd` for async execution
- Repository selection state is tracked via `selected` map for bulk operations
- Filter state persists during navigation between views
- Views are switched via `currentView` enum, not separate components
