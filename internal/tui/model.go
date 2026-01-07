package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/gh-repo-review/internal/cache"
	"github.com/user/gh-repo-review/internal/gh"
	"github.com/user/gh-repo-review/internal/repo"
)

// View represents different screens in the app
type View int

const (
	ViewList View = iota
	ViewFilter
	ViewDetail
	ViewConfirmArchive
	ViewConfirmDelete
	ViewHelp
)

// Model is the main application model
type Model struct {
	// Data
	repos         []repo.Repo
	filteredRepos []repo.Repo
	client        *gh.Client
	username      string

	// State
	view           View
	cursor         int
	offset         int
	loading        bool
	err            error
	message        string
	messageIsError bool

	// Filtering
	filterOpts  repo.FilterOptions
	searchInput textinput.Model

	// UI state
	width      int
	height     int
	spinner    spinner.Model
	showDetail bool

	// Selection for bulk operations
	selectedCount int
}

// Messages
type reposLoadedMsg struct {
	repos    []repo.Repo
	username string
}

type cacheLoadedMsg struct {
	repos    []repo.Repo
	username string
	fresh    bool
}

type backgroundRefreshMsg struct {
	repos []repo.Repo
}

type errorMsg struct{ err error }
type archiveCompleteMsg struct{ name string }
type deleteCompleteMsg struct{ name string }
type actionMsg string

// NewModel creates a new Model with default settings
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.CharLimit = 50
	ti.Width = 30

	return Model{
		view:       ViewList,
		loading:    true,
		spinner:    s,
		filterOpts: repo.DefaultFilterOptions(),
		searchInput: ti,
		width:      80,
		height:     24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadReposWithCache,
	)
}

// loadReposWithCache tries cache first, falls back to API
func loadReposWithCache() tea.Msg {
	client := gh.NewClient()

	if err := client.CheckAuth(); err != nil {
		return errorMsg{err: fmt.Errorf("not authenticated with gh CLI: %w", err)}
	}

	username, err := client.GetCurrentUser()
	if err != nil {
		return errorMsg{err: err}
	}

	// Try loading from cache
	repos, fresh, err := cache.Load(username)
	if err == nil && repos != nil {
		return cacheLoadedMsg{repos: repos, username: username, fresh: fresh}
	}

	// No cache, fetch from API
	repos, err = client.ListRepos()
	if err != nil {
		return errorMsg{err: err}
	}

	// Save to cache
	_ = cache.Save(username, repos)

	return reposLoadedMsg{repos: repos, username: username}
}

// refreshRepos fetches fresh data from API (for background refresh)
func refreshRepos(username string) tea.Cmd {
	return func() tea.Msg {
		client := gh.NewClient()
		repos, err := client.ListRepos()
		if err != nil {
			// Silent failure for background refresh
			return nil
		}
		_ = cache.Save(username, repos)
		return backgroundRefreshMsg{repos: repos}
	}
}

// forceRefreshRepos always fetches from API (for manual refresh)
func forceRefreshRepos() tea.Msg {
	client := gh.NewClient()

	username, err := client.GetCurrentUser()
	if err != nil {
		return errorMsg{err: err}
	}

	repos, err := client.ListRepos()
	if err != nil {
		return errorMsg{err: err}
	}

	_ = cache.Save(username, repos)
	return reposLoadedMsg{repos: repos, username: username}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case reposLoadedMsg:
		m.loading = false
		m.repos = msg.repos
		m.username = msg.username
		m.client = gh.NewClient()
		m.applyFilters()
		m.message = fmt.Sprintf("Loaded %d repositories", len(m.repos))

	case cacheLoadedMsg:
		m.loading = false
		m.repos = msg.repos
		m.username = msg.username
		m.client = gh.NewClient()
		m.applyFilters()
		if msg.fresh {
			m.message = fmt.Sprintf("Loaded %d repositories (cached)", len(m.repos))
		} else {
			m.message = fmt.Sprintf("Loaded %d repositories (refreshing...)", len(m.repos))
			cmds = append(cmds, refreshRepos(msg.username))
		}

	case backgroundRefreshMsg:
		if msg.repos != nil {
			// Preserve selection state
			selectedNames := make(map[string]bool)
			for _, r := range m.repos {
				if r.Selected {
					selectedNames[r.FullName] = true
				}
			}
			m.repos = msg.repos
			for i := range m.repos {
				if selectedNames[m.repos[i].FullName] {
					m.repos[i].Selected = true
				}
			}
			m.applyFilters()
			m.message = fmt.Sprintf("Refreshed %d repositories", len(m.repos))
		}

	case errorMsg:
		m.loading = false
		m.err = msg.err
		m.message = msg.err.Error()
		m.messageIsError = true

	case archiveCompleteMsg:
		m.message = fmt.Sprintf("Archived: %s", msg.name)
		m.messageIsError = false
		// Mark as archived in our list
		for i := range m.repos {
			if m.repos[i].FullName == msg.name {
				m.repos[i].IsArchived = true
				m.repos[i].Selected = false
				break
			}
		}
		m.applyFilters()

	case deleteCompleteMsg:
		m.message = fmt.Sprintf("Deleted: %s", msg.name)
		m.messageIsError = false
		// Remove from our list
		for i := range m.repos {
			if m.repos[i].FullName == msg.name {
				m.repos = append(m.repos[:i], m.repos[i+1:]...)
				break
			}
		}
		m.applyFilters()

	case actionMsg:
		m.message = string(msg)
		m.messageIsError = false
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c", "q":
		if m.view == ViewList && !m.searchInput.Focused() {
			return m, tea.Quit
		}
	}

	// Handle based on current view
	switch m.view {
	case ViewList:
		return m.handleListKeys(msg)
	case ViewFilter:
		return m.handleFilterKeys(msg)
	case ViewDetail:
		return m.handleDetailKeys(msg)
	case ViewConfirmArchive:
		return m.handleConfirmArchiveKeys(msg)
	case ViewConfirmDelete:
		return m.handleConfirmDeleteKeys(msg)
	case ViewHelp:
		return m.handleHelpKeys(msg)
	}

	return m, nil
}

// handleListKeys handles keys in the list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchInput.Focused() {
		switch msg.String() {
		case "enter", "esc":
			m.searchInput.Blur()
			m.filterOpts.SearchQuery = m.searchInput.Value()
			m.applyFilters()
			return m, nil
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.filterOpts.SearchQuery = m.searchInput.Value()
			m.applyFilters()
			return m, cmd
		}
	}

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.adjustOffset()
		}
	case "down", "j":
		if m.cursor < len(m.filteredRepos)-1 {
			m.cursor++
			m.adjustOffset()
		}
	case "pgup":
		m.cursor -= m.visibleRows()
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.adjustOffset()
	case "pgdown":
		m.cursor += m.visibleRows()
		if m.cursor >= len(m.filteredRepos) {
			m.cursor = len(m.filteredRepos) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.adjustOffset()
	case "home", "g":
		m.cursor = 0
		m.offset = 0
	case "end", "G":
		m.cursor = len(m.filteredRepos) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.adjustOffset()

	case "/":
		m.searchInput.Focus()
		return m, textinput.Blink

	case "f":
		m.view = ViewFilter
		return m, nil

	case "enter", "l":
		if len(m.filteredRepos) > 0 {
			m.view = ViewDetail
		}
		return m, nil

	case " ", "x":
		if len(m.filteredRepos) > 0 {
			idx := m.getActualIndex(m.cursor)
			if idx >= 0 {
				m.repos[idx].Selected = !m.repos[idx].Selected
				m.filteredRepos[m.cursor].Selected = m.repos[idx].Selected
				m.updateSelectedCount()
			}
		}

	case "a":
		if len(m.filteredRepos) > 0 {
			if m.selectedCount > 0 {
				m.view = ViewConfirmArchive
			} else {
				// Archive single repo
				idx := m.getActualIndex(m.cursor)
				if idx >= 0 {
					m.repos[idx].Selected = true
					m.updateSelectedCount()
					m.view = ViewConfirmArchive
				}
			}
		}

	case "d":
		if len(m.filteredRepos) > 0 {
			if m.selectedCount > 0 {
				m.view = ViewConfirmDelete
			} else {
				// Delete single repo under cursor
				idx := m.getActualIndex(m.cursor)
				if idx >= 0 {
					m.repos[idx].Selected = true
					m.updateSelectedCount()
					m.view = ViewConfirmDelete
				}
			}
		}

	case "A":
		// Select all visible
		for i := range m.filteredRepos {
			if !m.filteredRepos[i].IsArchived {
				m.filteredRepos[i].Selected = true
				idx := m.getActualIndex(i)
				if idx >= 0 {
					m.repos[idx].Selected = true
				}
			}
		}
		m.updateSelectedCount()

	case "D":
		// Deselect all
		for i := range m.repos {
			m.repos[i].Selected = false
		}
		for i := range m.filteredRepos {
			m.filteredRepos[i].Selected = false
		}
		m.selectedCount = 0

	case "o":
		if len(m.filteredRepos) > 0 && m.client != nil {
			r := m.filteredRepos[m.cursor]
			m.client.OpenInBrowser(r.FullName)
			return m, func() tea.Msg { return actionMsg("Opening in browser...") }
		}

	case "s":
		m.cycleSortField()
		m.applyFilters()

	case "S":
		m.filterOpts.SortDesc = !m.filterOpts.SortDesc
		m.applyFilters()

	case "r":
		m.loading = true
		return m, forceRefreshRepos

	case "?":
		m.view = ViewHelp

	case "1":
		m.filterOpts.ShowArchived = !m.filterOpts.ShowArchived
		m.applyFilters()
	case "2":
		m.filterOpts.ShowPrivate = !m.filterOpts.ShowPrivate
		m.applyFilters()
	case "3":
		m.filterOpts.ShowPublic = !m.filterOpts.ShowPublic
		m.applyFilters()
	case "4":
		m.filterOpts.ShowForks = !m.filterOpts.ShowForks
		m.applyFilters()
	}

	return m, nil
}

// handleFilterKeys handles keys in filter view
func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "f":
		m.view = ViewList
	case "1":
		m.filterOpts.ShowArchived = !m.filterOpts.ShowArchived
		m.applyFilters()
	case "2":
		m.filterOpts.ShowPrivate = !m.filterOpts.ShowPrivate
		m.applyFilters()
	case "3":
		m.filterOpts.ShowPublic = !m.filterOpts.ShowPublic
		m.applyFilters()
	case "4":
		m.filterOpts.ShowForks = !m.filterOpts.ShowForks
		m.applyFilters()
	case "5":
		m.filterOpts.InactiveForDays = cycleInactiveDays(m.filterOpts.InactiveForDays)
		m.applyFilters()
	case "s":
		m.cycleSortField()
		m.applyFilters()
	case "S":
		m.filterOpts.SortDesc = !m.filterOpts.SortDesc
		m.applyFilters()
	case "r":
		m.filterOpts = repo.DefaultFilterOptions()
		m.searchInput.SetValue("")
		m.applyFilters()
	}
	return m, nil
}

// handleDetailKeys handles keys in detail view
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "h":
		m.view = ViewList
	case "o":
		if m.client != nil && len(m.filteredRepos) > 0 {
			r := m.filteredRepos[m.cursor]
			m.client.OpenInBrowser(r.FullName)
		}
	case "a":
		if len(m.filteredRepos) > 0 {
			idx := m.getActualIndex(m.cursor)
			if idx >= 0 {
				m.repos[idx].Selected = true
				m.updateSelectedCount()
				m.view = ViewConfirmArchive
			}
		}
	}
	return m, nil
}

// handleConfirmArchiveKeys handles the archive confirmation dialog
func (m Model) handleConfirmArchiveKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		var cmds []tea.Cmd
		for i := range m.repos {
			if m.repos[i].Selected && !m.repos[i].IsArchived {
				name := m.repos[i].FullName
				cmds = append(cmds, func() tea.Msg {
					if m.client != nil {
						if err := m.client.ArchiveRepo(name); err != nil {
							return errorMsg{err: err}
						}
					}
					return archiveCompleteMsg{name: name}
				})
			}
		}
		m.view = ViewList
		return m, tea.Batch(cmds...)

	case "n", "N", "esc", "q":
		// Clear selections and go back
		for i := range m.repos {
			m.repos[i].Selected = false
		}
		m.selectedCount = 0
		m.view = ViewList
	}
	return m, nil
}

// handleConfirmDeleteKeys handles the delete confirmation dialog
func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		var cmds []tea.Cmd
		for i := range m.repos {
			if m.repos[i].Selected {
				name := m.repos[i].FullName
				cmds = append(cmds, func() tea.Msg {
					if m.client != nil {
						if err := m.client.DeleteRepo(name); err != nil {
							return errorMsg{err: err}
						}
					}
					return deleteCompleteMsg{name: name}
				})
			}
		}
		m.view = ViewList
		return m, tea.Batch(cmds...)

	case "n", "N", "esc", "q":
		for i := range m.repos {
			m.repos[i].Selected = false
		}
		m.selectedCount = 0
		m.view = ViewList
	}
	return m, nil
}

// handleHelpKeys handles keys in help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?":
		m.view = ViewList
	}
	return m, nil
}

// Helper methods

func (m *Model) applyFilters() {
	m.filteredRepos = repo.Filter(m.repos, m.filterOpts)
	repo.Sort(m.filteredRepos, m.filterOpts.SortBy, m.filterOpts.SortDesc)

	// Ensure cursor is valid
	if m.cursor >= len(m.filteredRepos) {
		m.cursor = len(m.filteredRepos) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustOffset()
}

func (m *Model) adjustOffset() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func (m Model) visibleRows() int {
	// Account for header, footer, etc.
	return m.height - 10
}

func (m *Model) updateSelectedCount() {
	count := 0
	for _, r := range m.repos {
		if r.Selected {
			count++
		}
	}
	m.selectedCount = count
}

func (m Model) getActualIndex(filteredIdx int) int {
	if filteredIdx < 0 || filteredIdx >= len(m.filteredRepos) {
		return -1
	}
	name := m.filteredRepos[filteredIdx].FullName
	for i, r := range m.repos {
		if r.FullName == name {
			return i
		}
	}
	return -1
}

func (m Model) findFilteredIndex(fullName string) int {
	for i, r := range m.filteredRepos {
		if r.FullName == fullName {
			return i
		}
	}
	return -1
}

func (m *Model) cycleSortField() {
	m.filterOpts.SortBy = (m.filterOpts.SortBy + 1) % 6
}

func cycleInactiveDays(current int) int {
	options := []int{0, 30, 90, 180, 365, 730}
	for i, opt := range options {
		if opt == current {
			return options[(i+1)%len(options)]
		}
	}
	return 0
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// View renders the UI
func (m Model) View() string {
	if m.loading {
		return appStyle.Render(
			fmt.Sprintf("\n%s Loading repositories...\n\n", m.spinner.View()),
		)
	}

	if m.err != nil && len(m.repos) == 0 {
		return appStyle.Render(
			fmt.Sprintf("\n❌ Error: %s\n\nPress q to quit.\n", m.err.Error()),
		)
	}

	switch m.view {
	case ViewList:
		return m.viewList()
	case ViewFilter:
		return m.viewFilter()
	case ViewDetail:
		return m.viewDetail()
	case ViewConfirmArchive:
		return m.viewConfirmArchive()
	case ViewConfirmDelete:
		return m.viewConfirmDelete()
	case ViewHelp:
		return m.viewHelp()
	}

	return ""
}

func (m Model) viewList() string {
	var b strings.Builder

	// Title
	title := fmt.Sprintf(" gh-repo-review | %s | %d repos ", m.username, len(m.filteredRepos))
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Quick filter status
	var filters []string
	if m.filterOpts.ShowArchived {
		filters = append(filters, "archived")
	}
	if !m.filterOpts.ShowForks {
		filters = append(filters, "no-forks")
	}
	if m.filterOpts.InactiveForDays > 0 {
		filters = append(filters, fmt.Sprintf(">%dd inactive", m.filterOpts.InactiveForDays))
	}
	if m.filterOpts.SearchQuery != "" {
		filters = append(filters, fmt.Sprintf("search:%s", m.filterOpts.SearchQuery))
	}

	filterLine := fmt.Sprintf("Sort: %s %s", m.filterOpts.SortBy.String(), sortDirArrow(m.filterOpts.SortDesc))
	if len(filters) > 0 {
		filterLine += " | Filters: " + strings.Join(filters, ", ")
	}
	b.WriteString(statsStyle.Render(filterLine))
	b.WriteString("\n\n")

	// Search input
	if m.searchInput.Focused() {
		b.WriteString(filterInputStyle.Render(m.searchInput.View()))
		b.WriteString("\n\n")
	}

	// Repository list
	visible := m.visibleRows()
	if visible < 1 {
		visible = 10
	}

	end := m.offset + visible
	if end > len(m.filteredRepos) {
		end = len(m.filteredRepos)
	}

	if len(m.filteredRepos) == 0 {
		b.WriteString(mutedStyle.Render("  No repositories match the current filters.\n"))
		b.WriteString(mutedStyle.Render("  Press 'f' to adjust filters or 'r' to reload.\n"))
	}

	for i := m.offset; i < end; i++ {
		r := m.filteredRepos[i]

		// Cursor
		var cursor string
		if i == m.cursor {
			cursor = cursorStyle.Render(">")
		} else {
			cursor = " "
		}

		// Checkbox
		var checkbox string
		if r.Selected {
			checkbox = checkboxStyle.Render("[✓]")
		} else {
			checkbox = uncheckedStyle.Render("[ ]")
		}

		// Name (bold for selected row)
		var name string
		if i == m.cursor {
			name = selectedItemStyle.Render(r.Name)
		} else {
			name = repoNameStyle.Render(r.Name)
		}

		// Tags
		var tagParts []string
		if r.IsPrivate {
			tagParts = append(tagParts, privateTagStyle.Render("private"))
		}
		if r.IsArchived {
			tagParts = append(tagParts, archivedTagStyle.Render("archived"))
		}
		if r.IsFork {
			tagParts = append(tagParts, forkTagStyle.Render("fork"))
		}
		tags := ""
		if len(tagParts) > 0 {
			tags = " " + strings.Join(tagParts, " ")
		}

		// Stats
		var statParts []string
		statParts = append(statParts, fmt.Sprintf("★ %d", r.StargazerCount))
		statParts = append(statParts, fmt.Sprintf("⑂ %d", r.ForkCount))
		if r.PrimaryLanguage != "" {
			statParts = append(statParts, GetLangStyle(r.PrimaryLanguage).Render(r.PrimaryLanguage))
		}
		statParts = append(statParts, fmt.Sprintf("%dd", r.DaysSinceUpdate()))
		stats := statsStyle.Render(strings.Join(statParts, " "))

		// Build line without lipgloss padding (causes issues with ANSI codes)
		b.WriteString(fmt.Sprintf("  %s %s %s%s  %s\n", cursor, checkbox, name, tags, stats))
	}

	// Selection count
	if m.selectedCount > 0 {
		b.WriteString("\n")
		b.WriteString(successStyle.Render(fmt.Sprintf("  %d %s selected", m.selectedCount, pluralize(m.selectedCount, "repo", "repos"))))
	}

	// Status message
	if m.message != "" {
		b.WriteString("\n")
		if m.messageIsError {
			b.WriteString(dangerStyle.Render("  " + m.message))
		} else {
			b.WriteString(successStyle.Render("  " + m.message))
		}
	}

	// Help line
	b.WriteString("\n\n")
	helpItems := []string{
		helpKeyStyle.Render("/") + " search",
		helpKeyStyle.Render("f") + " filter",
		helpKeyStyle.Render("space") + " select",
		helpKeyStyle.Render("a") + " archive",
		helpKeyStyle.Render("d") + " delete",
		helpKeyStyle.Render("o") + " open",
		helpKeyStyle.Render("?") + " help",
		helpKeyStyle.Render("q") + " quit",
	}
	b.WriteString(helpStyle.Render(strings.Join(helpItems, "  ")))

	return appStyle.Render(b.String())
}

func (m Model) viewFilter() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Filter Options "))
	b.WriteString("\n\n")

	// Toggle options
	options := []struct {
		key     string
		name    string
		enabled bool
	}{
		{"1", "Show Archived", m.filterOpts.ShowArchived},
		{"2", "Show Private", m.filterOpts.ShowPrivate},
		{"3", "Show Public", m.filterOpts.ShowPublic},
		{"4", "Show Forks", m.filterOpts.ShowForks},
	}

	for _, opt := range options {
		check := uncheckedStyle.Render("[ ]")
		if opt.enabled {
			check = checkboxStyle.Render("[✓]")
		}
		b.WriteString(fmt.Sprintf("  %s %s %s\n", helpKeyStyle.Render(opt.key), check, opt.name))
	}

	b.WriteString("\n")

	// Inactive days
	inactiveStr := "All repos"
	if m.filterOpts.InactiveForDays > 0 {
		inactiveStr = fmt.Sprintf("> %d days inactive", m.filterOpts.InactiveForDays)
	}
	b.WriteString(fmt.Sprintf("  %s Inactive: %s\n", helpKeyStyle.Render("5"), inactiveStr))

	b.WriteString("\n")

	// Sort
	sortDir := "↑ Asc"
	if m.filterOpts.SortDesc {
		sortDir = "↓ Desc"
	}
	b.WriteString(fmt.Sprintf("  %s Sort by: %s\n", helpKeyStyle.Render("s"), m.filterOpts.SortBy.String()))
	b.WriteString(fmt.Sprintf("  %s Direction: %s\n", helpKeyStyle.Render("S"), sortDir))

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s Reset to defaults\n", helpKeyStyle.Render("r")))

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press esc or f to return"))

	return appStyle.Render(b.String())
}

func (m Model) viewDetail() string {
	if len(m.filteredRepos) == 0 {
		return appStyle.Render("No repository selected")
	}

	r := m.filteredRepos[m.cursor]
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf(" %s ", r.FullName)))
	b.WriteString("\n\n")

	if r.Description != "" {
		b.WriteString(repoDescStyle.Render(r.Description))
		b.WriteString("\n\n")
	}

	// Info grid
	info := []struct {
		label string
		value string
	}{
		{"Visibility", r.VisibilityString()},
		{"Status", r.StatusString()},
		{"Language", r.PrimaryLanguage},
		{"Stars", fmt.Sprintf("%d", r.StargazerCount)},
		{"Forks", fmt.Sprintf("%d", r.ForkCount)},
		{"Open Issues", fmt.Sprintf("%d", r.OpenIssuesCount)},
		{"Size", r.SizeString()},
		{"Created", r.CreatedAt.Format("Jan 02, 2006")},
		{"Last Updated", r.UpdatedAt.Format("Jan 02, 2006")},
		{"Last Push", fmt.Sprintf("%s (%d days ago)", r.PushedAt.Format("Jan 02, 2006"), r.DaysSinceUpdate())},
	}

	for _, item := range info {
		if item.value != "" && item.value != "-" {
			label := statsStyle.Render(fmt.Sprintf("%-14s", item.label+":"))
			b.WriteString(fmt.Sprintf("  %s %s\n", label, item.value))
		}
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  URL: %s\n", mutedStyle.Render(r.URL)))
	b.WriteString(fmt.Sprintf("  SSH: %s\n", mutedStyle.Render(r.SSHURL)))

	b.WriteString("\n\n")

	// Actions
	b.WriteString(helpKeyStyle.Render("o") + " Open in browser  ")
	if !r.IsArchived {
		b.WriteString(helpKeyStyle.Render("a") + " Archive  ")
	}
	b.WriteString(helpKeyStyle.Render("esc") + " Back")

	return appStyle.Render(b.String())
}

func (m Model) viewConfirmArchive() string {
	var b strings.Builder

	title := dialogTitleStyle.Render("⚠ Confirm Archive")
	b.WriteString(title)
	b.WriteString("\n\n")

	// List repos to be archived
	count := 0
	for _, r := range m.repos {
		if r.Selected && !r.IsArchived {
			count++
			if count <= 5 {
				b.WriteString(fmt.Sprintf("  • %s\n", r.FullName))
			}
		}
	}
	if count > 5 {
		b.WriteString(fmt.Sprintf("  ... and %d more\n", count-5))
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Archive %d %s?\n", count, pluralize(count, "repository", "repositories")))
	b.WriteString("Archived repos are read-only but can be unarchived later.\n\n")

	b.WriteString(helpKeyStyle.Render("y") + " Yes, archive  ")
	b.WriteString(helpKeyStyle.Render("n") + " No, cancel")

	return appStyle.Render(dialogStyle.Render(b.String()))
}

func (m Model) viewConfirmDelete() string {
	var b strings.Builder

	title := dialogTitleStyle.Render("🗑 DANGER: Confirm Delete")
	b.WriteString(title)
	b.WriteString("\n\n")

	// List repos to be deleted
	count := 0
	for _, r := range m.repos {
		if r.Selected {
			count++
			if count <= 5 {
				b.WriteString(fmt.Sprintf("  • %s\n", r.FullName))
			}
		}
	}
	if count > 5 {
		b.WriteString(fmt.Sprintf("  ... and %d more\n", count-5))
	}

	b.WriteString("\n")
	b.WriteString(dangerStyle.Render(fmt.Sprintf("PERMANENTLY DELETE %d %s?\n", count, pluralize(count, "repository", "repositories"))))
	b.WriteString(dangerStyle.Render("This action CANNOT be undone!\n\n"))

	b.WriteString(helpKeyStyle.Render("y") + " Yes, DELETE  ")
	b.WriteString(helpKeyStyle.Render("n") + " No, cancel")

	return appStyle.Render(dialogStyle.Render(b.String()))
}

func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Keyboard Shortcuts "))
	b.WriteString("\n\n")

	sections := []struct {
		name  string
		binds []struct{ key, desc string }
	}{
		{
			"Navigation",
			[]struct{ key, desc string }{
				{"↑/k", "Move up"},
				{"↓/j", "Move down"},
				{"PgUp/PgDn", "Page up/down"},
				{"g/G", "Go to top/bottom"},
				{"Enter/l", "View details"},
				{"esc/h", "Go back"},
			},
		},
		{
			"Search & Filter",
			[]struct{ key, desc string }{
				{"/", "Search repositories"},
				{"f", "Open filter panel"},
				{"s", "Cycle sort field"},
				{"S", "Toggle sort direction"},
				{"1-4", "Toggle filter options"},
			},
		},
		{
			"Selection",
			[]struct{ key, desc string }{
				{"Space/x", "Toggle selection"},
				{"A", "Select all visible"},
				{"D", "Deselect all"},
			},
		},
		{
			"Actions",
			[]struct{ key, desc string }{
				{"a", "Archive selected"},
				{"d", "Delete selected (dangerous!)"},
				{"o", "Open in browser"},
				{"r", "Reload repositories"},
			},
		},
		{
			"General",
			[]struct{ key, desc string }{
				{"?", "Show/hide help"},
				{"q", "Quit"},
			},
		},
	}

	for _, section := range sections {
		b.WriteString(repoNameStyle.Render(section.name))
		b.WriteString("\n")
		for _, bind := range section.binds {
			key := helpKeyStyle.Render(fmt.Sprintf("%-12s", bind.key))
			b.WriteString(fmt.Sprintf("  %s %s\n", key, bind.desc))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("Press ? or esc to return"))

	return appStyle.Render(b.String())
}

func sortDirArrow(desc bool) string {
	if desc {
		return "↓"
	}
	return "↑"
}
