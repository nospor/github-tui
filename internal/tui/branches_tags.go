package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	gh "github-tui/internal/github"
)

func (m Model) diffPanelHeight() int {
	return m.bodyHeight()
}

const (
	createIssueBranchFieldName  = 0
	createIssueBranchFieldRef   = 1
	createIssueBranchFieldCount = 2

	createTagNameField    = 0
	createTagRefField     = 1
	createTagMessageField = 2
	createTagFieldCount   = 3
)

// ─── Branches support ─────────────────────────────────────────────────────────

func (m Model) cmdLoadBranches() tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		branches, err := m.client.ListBranches(full)
		if err != nil {
			return errMsg{err}
		}
		return branchesLoadedMsg{branches}
	}
}

func (m Model) cmdDeleteBranch(branch string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		err := m.client.DeleteBranch(full, branch)
		if err != nil {
			return errMsg{err}
		}
		return doneMsg{text: fmt.Sprintf("Branch %s deleted successfully!", branch), reloadList: true}
	}
}

func (m Model) cmdLoadBranchCommits(branch string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		commits, err := m.client.ListCommits(full, branch)
		if err != nil {
			return errMsg{err}
		}
		return branchCommitsLoadedMsg{branch: branch, commits: commits}
	}
}

func (m Model) cmdCompareBranches(target, source string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		comp, err := m.client.Compare(full, target, source)
		if err != nil {
			return errMsg{err}
		}
		return branchCompareLoadedMsg{
			targetBranch: target,
			sourceBranch: source,
			compare:      comp,
		}
	}
}

func (m Model) startCreatePRFromBranch(srcBranch string) (Model, tea.Cmd) {
	nm, cmd := m.startCreatePR()
	if mm, ok := nm.(Model); ok {
		mm.headInput.SetValue(srcBranch)
		return mm, cmd
	}
	m.headInput.SetValue(srcBranch)
	return m, cmd
}

func (m Model) compareBranchList() []string {
	var list []string
	if m.branchCursor >= len(m.branches) {
		return nil
	}
	current := m.branches[m.branchCursor]
	for _, b := range m.branches {
		if b != current {
			list = append(list, b)
		}
	}
	return list
}

func (m Model) handleCompareBranchSelectKey(key string) (tea.Model, tea.Cmd) {
	list := m.compareBranchList()
	switch key {
	case "esc":
		m.state = m.returnState
		return m, nil
	case "j", "down":
		if m.compareSelectCursor < len(list)-1 {
			m.compareSelectCursor++
		}
	case "k", "up":
		if m.compareSelectCursor > 0 {
			m.compareSelectCursor--
		}
	case "enter":
		if len(list) > 0 && m.compareSelectCursor < len(list) {
			targetBranch := list[m.compareSelectCursor]
			sourceBranch := m.branches[m.branchCursor]
			m.returnState = stateMain
			m.branchDetailView = branchViewCompare
			return m.track(m.cmdCompareBranches(targetBranch, sourceBranch))
		}
	}
	return m, nil
}

func (m Model) viewCompareBranchSelect() string {
	var rows []string
	rows = append(rows, subtitleStyle.Render("Select branch to compare with"), "")
	
	list := m.compareBranchList()
	if len(list) == 0 {
		rows = append(rows, dimStyle.Render("  No other branches found."))
	} else {
		maxVisible := 12
		start := m.compareSelectCursor - maxVisible/2
		if start < 0 {
			start = 0
		}
		end := start + maxVisible
		if end > len(list) {
			end = len(list)
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}
		if start > 0 {
			rows = append(rows, dimStyle.Render(fmt.Sprintf("  ↑ %d more", start)))
		}
		for i := start; i < end; i++ {
			b := list[i]
			label := fmt.Sprintf("%-50s", fit(b, 50))
			if i == m.compareSelectCursor {
				rows = append(rows, selectedStyle.Render("▶ "+label))
			} else {
				rows = append(rows, normalItemStyle.Render("  "+label))
			}
		}
		if end < len(list) {
			rows = append(rows, dimStyle.Render(fmt.Sprintf("  ↓ %d more", len(list)-end)))
		}
	}
	rows = append(rows, "", dimStyle.Render("↑↓ navigate  Enter compare  Esc cancel"))
	return m.placeDialog(subtitleStyle.Render("Compare branches"), "", strings.Join(rows, "\n"))
}

func (m Model) viewBranchList(bodyH int) string {
	if len(m.branches) == 0 {
		return dimStyle.Padding(2).Render("No branches found.")
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		PaddingLeft(2).
		Render(fmt.Sprintf("%-60s", "Branch Name"))
	header += "\n" + lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", m.width-2))

	listH := bodyH - 3
	if listH < 1 {
		listH = 1
	}

	maxVisible := listH
	start := m.branchCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.branches) {
		end = len(m.branches)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		b := m.branches[i]
		selected := i == m.branchCursor

		line := fmt.Sprintf("%-60s", fit(b, 60))

		if selected {
			rows = append(rows, selectedStyle.Width(m.width-2).Render("▶ "+line))
		} else {
			rows = append(rows, normalItemStyle.Width(m.width-2).Render("  "+line))
		}
	}

	return header + "\n" + strings.Join(rows, "\n")
}

func (m Model) viewBranchCommits(bodyH int) string {
	if m.branchCommitDiffPanelOpen {
		return m.viewBranchCommitsSplit(bodyH)
	}

	if len(m.branchCommits) == 0 {
		return dimStyle.Padding(2).Render("Loading commits...")
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		PaddingLeft(2).
		Render(fmt.Sprintf("Commits for branch: %s", m.branchDetailName))
	header += "\n" + lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", m.width-2))

	listH := bodyH - 3
	if listH < 1 {
		listH = 1
	}

	maxVisible := listH
	start := m.branchCommitCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.branchCommits) {
		end = len(m.branchCommits)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		c := m.branchCommits[i]
		selected := i == m.branchCommitCursor

		line := fmt.Sprintf("%-10s  %-50s  %-15s  %s",
			c.ShortID,
			fit(c.Title, 50),
			fit(c.AuthorName, 15),
			dimStyle.Render(c.Date),
		)

		if selected {
			rows = append(rows, selectedStyle.Width(m.width-2).Render("▶ "+line))
		} else {
			rows = append(rows, normalItemStyle.Width(m.width-2).Render("  "+line))
		}
	}

	return header + "\n" + strings.Join(rows, "\n")
}

func (m Model) viewBranchCompare(bodyH int) string {
	if m.branchCompare == nil {
		return dimStyle.Padding(2).Render("Comparing branch...")
	}

	if m.branchCommitDiffPanelOpen {
		return m.viewBranchCompareSplit(bodyH)
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		PaddingLeft(2).
		Render(fmt.Sprintf("Compare: %s ... %s (target ... source)", m.branchCompareTarget, m.branchDetailName))
	header += "\n" + lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", m.width-2))

	listH := bodyH - 3
	if listH < 1 {
		listH = 1
	}

	maxVisible := listH
	start := m.branchCompareCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.branchCompare.Commits) {
		end = len(m.branchCompare.Commits)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		c := m.branchCompare.Commits[i]
		selected := i == m.branchCompareCursor

		line := fmt.Sprintf("%-10s  %-50s  %-15s  %s",
			c.ShortID,
			fit(c.Title, 50),
			fit(c.AuthorName, 15),
			dimStyle.Render(c.Date),
		)

		if selected {
			rows = append(rows, selectedStyle.Width(m.width-2).Render("▶ "+line))
		} else {
			rows = append(rows, normalItemStyle.Width(m.width-2).Render("  "+line))
		}
	}

	return header + "\n" + strings.Join(rows, "\n")
}

func (m Model) viewBranchCompareSplit(bodyH int) string {
	leftW := m.width * 2 / 5
	rightW := m.width - leftW - 1 // -1 for separator

	if leftW < 20 {
		leftW = 20
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		PaddingLeft(2).
		Render(fmt.Sprintf("Compare: %s ... %s", m.branchCompareTarget, m.branchDetailName))
	header += "\n" + lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", leftW-2))

	listH := bodyH - 3
	if listH < 1 {
		listH = 1
	}

	maxVisible := listH
	start := m.branchCompareCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.branchCompare.Commits) {
		end = len(m.branchCompare.Commits)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		c := m.branchCompare.Commits[i]
		selected := i == m.branchCompareCursor

		// In split view, format compactly
		line := fmt.Sprintf("%-10s  %s",
			c.ShortID,
			fit(c.Title, leftW-15),
		)

		if selected {
			rows = append(rows, selectedStyle.Width(leftW-2).Render("▶ "+line))
		} else {
			rows = append(rows, normalItemStyle.Width(leftW-2).Render("  "+line))
		}
	}

	leftContent := header + "\n" + strings.Join(rows, "\n")
	left := lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(leftContent)

	// Separator
	sepContent := strings.Repeat("│\n", bodyH)
	if bodyH > 0 {
		sepContent = sepContent[:len(sepContent)-1]
	}
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(sepContent)

	// Right: diff panel
	rightContent := m.viewBranchCommitDiffPanel(rightW, bodyH)
	right := lipgloss.NewStyle().Width(rightW).Height(bodyH).MaxHeight(bodyH).Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

func (m Model) cmdLoadCommitDiff(sha string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		files, err := m.client.GetCommitDiffs(full, sha)
		if err != nil {
			return errMsg{err}
		}
		return commitDiffsLoadedMsg{sha: sha, files: files}
	}
}

func (m *Model) branchCommitDiffLineCursorDown() {
	if len(m.branchCommitDiffFiles) == 0 {
		return
	}
	f := m.branchCommitDiffFiles[m.branchCommitDiffFileIdx]
	if m.branchCommitDiffLineCursor < len(f.Lines)-1 {
		m.branchCommitDiffLineCursor++
	} else if m.branchCommitDiffFileIdx < len(m.branchCommitDiffFiles)-1 {
		// Move to next file
		m.branchCommitDiffFileIdx++
		m.branchCommitDiffLineCursor = 0
		m.branchCommitDiffScrollOffset = 0
	}
}

func (m *Model) branchCommitDiffLineCursorUp() {
	if m.branchCommitDiffLineCursor > 0 {
		m.branchCommitDiffLineCursor--
	} else if m.branchCommitDiffFileIdx > 0 {
		m.branchCommitDiffFileIdx--
		m.branchCommitDiffScrollOffset = 0
		if len(m.branchCommitDiffFiles[m.branchCommitDiffFileIdx].Lines) > 0 {
			m.branchCommitDiffLineCursor = len(m.branchCommitDiffFiles[m.branchCommitDiffFileIdx].Lines) - 1
		}
	}
}

func (m *Model) branchCommitDiffNextHunk() {
	if len(m.branchCommitDiffFiles) == 0 {
		return
	}
	f := m.branchCommitDiffFiles[m.branchCommitDiffFileIdx]
	for i := m.branchCommitDiffLineCursor + 1; i < len(f.Lines); i++ {
		if f.Lines[i].Type == "hunk" {
			m.branchCommitDiffLineCursor = i
			return
		}
	}
}

func (m *Model) branchCommitDiffPrevHunk() {
	if len(m.branchCommitDiffFiles) == 0 {
		return
	}
	f := m.branchCommitDiffFiles[m.branchCommitDiffFileIdx]
	for i := m.branchCommitDiffLineCursor - 1; i >= 0; i-- {
		if f.Lines[i].Type == "hunk" {
			m.branchCommitDiffLineCursor = i
			return
		}
	}
}

func (m Model) branchCommitDiffHeight() int {
	bodyH := m.diffPanelHeight()
	if len(m.branchCommitDiffFiles) == 0 {
		return bodyH - 4
	}
	startIdx := m.branchCommitDiffFileIdx - 1
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + 3
	if endIdx > len(m.branchCommitDiffFiles) {
		endIdx = len(m.branchCommitDiffFiles)
		startIdx = endIdx - 3
		if startIdx < 0 {
			startIdx = 0
		}
	}
	tabsLen := endIdx - startIdx
	dh := bodyH - (4 + tabsLen)
	if dh < 1 {
		dh = 1
	}
	return dh
}

func (m *Model) updateBranchCommitDiffScroll() {
	if len(m.branchCommitDiffFiles) == 0 {
		return
	}
	f := m.branchCommitDiffFiles[m.branchCommitDiffFileIdx]
	totalLines := len(f.Lines)
	diffHeight := m.branchCommitDiffHeight()

	if m.branchCommitDiffLineCursor < m.branchCommitDiffScrollOffset {
		m.branchCommitDiffScrollOffset = m.branchCommitDiffLineCursor
	}

	if m.branchCommitDiffLineCursor >= m.branchCommitDiffScrollOffset + diffHeight {
		m.branchCommitDiffScrollOffset = m.branchCommitDiffLineCursor - diffHeight + 1
	}

	if m.branchCommitDiffScrollOffset >= totalLines {
		m.branchCommitDiffScrollOffset = totalLines - 1
	}
	if m.branchCommitDiffScrollOffset < 0 {
		m.branchCommitDiffScrollOffset = 0
	}
}

// viewBranchCommitDiffPanel renders the changes panel for the selected commit.
func (m Model) viewBranchCommitDiffPanel(w, h int) string {
	var lines []string

	if m.branchCommitDiffLoading {
		lines = append(lines,
			subtitleStyle.Render("  Changes"),
			"",
			dimStyle.Render("  Loading diffs..."),
		)
		return strings.Join(lines, "\n")
	}

	if len(m.branchCommitDiffFiles) == 0 {
		lines = append(lines,
			subtitleStyle.Render("  Changes"),
			"",
			dimStyle.Render("  No files changed or no diff found."),
		)
		return strings.Join(lines, "\n")
	}

	// File list header
	fileCount := len(m.branchCommitDiffFiles)
	headerLine := subtitleStyle.Render("  Changes ") +
		dimStyle.Render(fmt.Sprintf("(%d file(s))  n/p=file, J/K=hunk", fileCount))
	lines = append(lines, headerLine)

	// File tabs (show nearby files)
	var fileTabs []string
	startIdx := m.branchCommitDiffFileIdx - 1
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + 3
	if endIdx > len(m.branchCommitDiffFiles) {
		endIdx = len(m.branchCommitDiffFiles)
		startIdx = endIdx - 3
		if startIdx < 0 {
			startIdx = 0
		}
	}

	for i := startIdx; i < endIdx; i++ {
		f := m.branchCommitDiffFiles[i]
		name := f.NewPath
		counts := diffCounts(f)
		limit := w - 7 - len(counts)
		if limit < 35 {
			limit = 35
		}
		name = truncatePath(name, limit)
		label := fmt.Sprintf("%s %s", counts, name)
		if i == m.branchCommitDiffFileIdx {
			fileTabs = append(fileTabs, accentStyle.Render(" ▶ "+label))
		} else {
			fileTabs = append(fileTabs, dimStyle.Render("   "+label))
		}
	}
	lines = append(lines, strings.Join(fileTabs, "\n"))
	lines = append(lines, lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", w-2)))

	tabsLen := endIdx - startIdx
	diffHeight := h - (4 + tabsLen)
	if diffHeight < 1 {
		diffHeight = 1
	}

	// Current file diff lines
	f := m.branchCommitDiffFiles[m.branchCommitDiffFileIdx]
	renderedCount := 0

	if len(f.Lines) == 0 {
		lines = append(lines, dimStyle.Render("  (diff unavailable — file is too large or collapsed)"))
		renderedCount++
	}

	for i := m.branchCommitDiffScrollOffset; i < len(f.Lines) && renderedCount < diffHeight; i++ {
		dl := f.Lines[i]
		selected := i == m.branchCommitDiffLineCursor
		content := dl.Content
		// Clip to panel width (use display-width, not byte length)
		avail := w - 5
		if avail < 1 {
			avail = 1
		}
		if lipgloss.Width(content) > avail {
			content = ansi.Truncate(content, avail-1, "…")
		}

		var rendered string
		switch dl.Type {
		case "added":
			st := lipgloss.NewStyle().Foreground(colorSuccess)
			if selected {
				st = st.Background(colorBgHover).Bold(true)
			}
			rendered = st.Render("▶ " + content)
		case "removed":
			st := lipgloss.NewStyle().Foreground(colorError)
			if selected {
				st = st.Background(colorBgHover).Bold(true)
			}
			rendered = st.Render("▶ " + content)
		case "hunk":
			st := lipgloss.NewStyle().Foreground(colorInfo).Italic(true)
			if selected {
				st = st.Background(colorBgHover).Bold(true)
			}
			rendered = st.Render("  " + content)
		default:
			st := lipgloss.NewStyle().Foreground(colorTextDim)
			if selected {
				st = st.Background(colorBgHover)
			}
			rendered = st.Render("  " + content)
		}
		lines = append(lines, rendered)
		renderedCount++
	}

	return strings.Join(lines, "\n")
}

func (m Model) viewBranchCommitsSplit(bodyH int) string {
	leftW := m.width * 2 / 5
	rightW := m.width - leftW - 1 // -1 for separator

	if leftW < 20 {
		leftW = 20
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		PaddingLeft(2).
		Render(fmt.Sprintf("Commits: %s", m.branchDetailName))
	header += "\n" + lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", leftW-2))

	listH := bodyH - 3
	if listH < 1 {
		listH = 1
	}

	maxVisible := listH
	start := m.branchCommitCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.branchCommits) {
		end = len(m.branchCommits)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		c := m.branchCommits[i]
		selected := i == m.branchCommitCursor

		// In split view, format compactly
		line := fmt.Sprintf("%-10s  %s",
			c.ShortID,
			fit(c.Title, leftW-15),
		)

		if selected {
			rows = append(rows, selectedStyle.Width(leftW-2).Render("▶ "+line))
		} else {
			rows = append(rows, normalItemStyle.Width(leftW-2).Render("  "+line))
		}
	}

	leftContent := header + "\n" + strings.Join(rows, "\n")
	left := lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(leftContent)

	// Separator
	sepContent := strings.Repeat("│\n", bodyH)
	if bodyH > 0 {
		sepContent = sepContent[:len(sepContent)-1]
	}
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(sepContent)

	// Right: diff panel
	rightContent := m.viewBranchCommitDiffPanel(rightW, bodyH)
	right := lipgloss.NewStyle().Width(rightW).Height(bodyH).MaxHeight(bodyH).Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

// ─── Tags support ─────────────────────────────────────────────────────────────

func (m Model) cmdDeleteTag(tag string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		err := m.client.DeleteTag(full, tag)
		if err != nil {
			return errMsg{err}
		}
		return doneMsg{text: fmt.Sprintf("Tag '%s' deleted successfully!", tag), reloadList: true}
	}
}

func (m Model) cmdCreateTag(name, ref, message string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		err := m.client.CreateTag(full, name, ref, message)
		if err != nil {
			return errMsg{err}
		}
		return doneMsg{text: fmt.Sprintf("🏷️ Tag '%s' created successfully!", name), reloadList: true}
	}
}

func (m Model) cmdLoadTags() tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		tags, err := m.client.ListTags(full)
		if err != nil {
			return errMsg{err}
		}
		return tagsLoadedMsg{tags}
	}
}

func (m Model) cmdLoadTagCommits(tag string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		commits, err := m.client.ListCommits(full, tag)
		if err != nil {
			return errMsg{err}
		}
		return tagCommitsLoadedMsg{tag: tag, commits: commits}
	}
}

func (m Model) startCreateTag() (Model, tea.Cmd) {
	if m.repo == nil {
		return m, nil
	}
	m.createTagName.SetValue("")
	m.createTagMessage.SetValue("")
	m.createTagBranchCursor = 0
	if m.repo != nil && m.repo.DefaultBranch != "" {
		for i, b := range m.branches {
			if b == m.repo.DefaultBranch {
				m.createTagBranchCursor = i
				break
			}
		}
	}
	m.createTagField = createTagNameField

	m.returnState = m.state
	m.state = stateCreateTag

	var cmds []tea.Cmd
	cmds = append(cmds, m.createTagName.Focus())
	if len(m.branches) == 0 {
		cmds = append(cmds, m.cmdLoadBranches())
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleCreateTagKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.createTagName.Blur()
		m.createTagMessage.Blur()
		m.state = m.returnState
		return m, nil
	case "tab":
		m.createTagField = (m.createTagField + 1) % createTagFieldCount
		return m.focusCreateTagField()
	case "shift+tab":
		m.createTagField = (m.createTagField - 1 + createTagFieldCount) % createTagFieldCount
		return m.focusCreateTagField()
	case "ctrl+s":
		return m.submitCreateTag()
	}

	if m.createTagField == createTagRefField {
		switch key {
		case "j", "down":
			if m.createTagBranchCursor < len(m.branches)-1 {
				m.createTagBranchCursor++
			}
			return m, nil
		case "k", "up":
			if m.createTagBranchCursor > 0 {
				m.createTagBranchCursor--
			}
			return m, nil
		case "enter":
			m.createTagField = createTagMessageField
			return m.focusCreateTagField()
		}
	} else if m.createTagField == createTagNameField {
		if key == "enter" {
			m.createTagField = createTagRefField
			return m.focusCreateTagField()
		}
	}

	var cmd tea.Cmd
	switch m.createTagField {
	case createTagNameField:
		m.createTagName, cmd = m.createTagName.Update(msg)
	case createTagMessageField:
		m.createTagMessage, cmd = m.createTagMessage.Update(msg)
	}
	return m, cmd
}

func (m Model) focusCreateTagField() (Model, tea.Cmd) {
	m.createTagName.Blur()
	m.createTagMessage.Blur()
	switch m.createTagField {
	case createTagNameField:
		cmd := m.createTagName.Focus()
		return m, cmd
	case createTagRefField:
		return m, nil
	case createTagMessageField:
		cmd := m.createTagMessage.Focus()
		return m, cmd
	}
	return m, nil
}

func (m Model) submitCreateTag() (Model, tea.Cmd) {
	tagName := strings.TrimSpace(m.createTagName.Value())
	message := strings.TrimSpace(m.createTagMessage.Value())
	if tagName == "" {
		return m, nil
	}

	ref := ""
	if len(m.branches) > 0 && m.createTagBranchCursor < len(m.branches) {
		ref = m.branches[m.createTagBranchCursor]
	}
	if ref == "" && m.repo != nil {
		ref = m.repo.DefaultBranch
	}
	if ref == "" {
		ref = "main"
	}

	m.createTagName.Blur()
	m.createTagMessage.Blur()
	m.state = m.returnState

	return m, m.cmdCreateTag(tagName, ref, message)
}

func (m *Model) tagCommitDiffLineCursorDown() {
	if len(m.tagCommitDiffFiles) == 0 {
		return
	}
	f := m.tagCommitDiffFiles[m.tagCommitDiffFileIdx]
	if m.tagCommitDiffLineCursor < len(f.Lines)-1 {
		m.tagCommitDiffLineCursor++
	} else if m.tagCommitDiffFileIdx < len(m.tagCommitDiffFiles)-1 {
		m.tagCommitDiffFileIdx++
		m.tagCommitDiffLineCursor = 0
		m.tagCommitDiffScrollOffset = 0
	}
}

func (m *Model) tagCommitDiffLineCursorUp() {
	if len(m.tagCommitDiffFiles) == 0 {
		return
	}
	if m.tagCommitDiffLineCursor > 0 {
		m.tagCommitDiffLineCursor--
	} else if m.tagCommitDiffFileIdx > 0 {
		m.tagCommitDiffFileIdx--
		f := m.tagCommitDiffFiles[m.tagCommitDiffFileIdx]
		m.tagCommitDiffLineCursor = len(f.Lines) - 1
		m.tagCommitDiffScrollOffset = 0
	}
}

func (m *Model) tagCommitDiffNextHunk() {
	if len(m.tagCommitDiffFiles) == 0 {
		return
	}
	f := m.tagCommitDiffFiles[m.tagCommitDiffFileIdx]
	for i := m.tagCommitDiffLineCursor + 1; i < len(f.Lines); i++ {
		if f.Lines[i].Type == "hunk" {
			m.tagCommitDiffLineCursor = i
			return
		}
	}
	if m.tagCommitDiffFileIdx < len(m.tagCommitDiffFiles)-1 {
		m.tagCommitDiffFileIdx++
		m.tagCommitDiffLineCursor = 0
		m.tagCommitDiffScrollOffset = 0
	}
}

func (m *Model) tagCommitDiffPrevHunk() {
	if len(m.tagCommitDiffFiles) == 0 {
		return
	}
	f := m.tagCommitDiffFiles[m.tagCommitDiffFileIdx]
	for i := m.tagCommitDiffLineCursor - 1; i >= 0; i-- {
		if f.Lines[i].Type == "hunk" {
			m.tagCommitDiffLineCursor = i
			return
		}
	}
	if m.tagCommitDiffFileIdx > 0 {
		m.tagCommitDiffFileIdx--
		m.tagCommitDiffLineCursor = 0
		m.tagCommitDiffScrollOffset = 0
	}
}

func (m *Model) updateTagCommitDiffScroll() {
	h := m.height - 8
	if h < 5 {
		h = 5
	}
	if m.tagCommitDiffLineCursor < m.tagCommitDiffScrollOffset {
		m.tagCommitDiffScrollOffset = m.tagCommitDiffLineCursor
	} else if m.tagCommitDiffLineCursor >= m.tagCommitDiffScrollOffset+h {
		m.tagCommitDiffScrollOffset = m.tagCommitDiffLineCursor - h + 1
	}
}

func (m Model) viewTagList(bodyH int) string {
	if len(m.tags) == 0 {
		return dimStyle.Padding(2).Render("No tags found.")
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		PaddingLeft(2).
		Render(fmt.Sprintf("%-25s  %-12s  %-40s", "Tag Name", "Commit", "Message"))
	header += "\n" + lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", m.width-2))

	listH := bodyH - 3
	if listH < 1 {
		listH = 1
	}

	maxVisible := listH
	start := m.tagCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.tags) {
		end = len(m.tags)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		t := m.tags[i]
		selected := i == m.tagCursor

		commitStr := t.ShortID
		if commitStr == "" {
			commitStr = t.Target
		}

		cleanMsg := strings.ReplaceAll(strings.ReplaceAll(t.Message, "\r\n", " "), "\n", " ")
		line := fmt.Sprintf("%-25s  %-12s  %-40s",
			fit(t.Name, 25),
			fit(commitStr, 12),
			fit(cleanMsg, 40),
		)

		if selected {
			rows = append(rows, selectedStyle.Width(m.width-2).Render("▶ "+line))
		} else {
			rows = append(rows, normalItemStyle.Width(m.width-2).Render("  "+line))
		}
	}

	return header + "\n" + strings.Join(rows, "\n")
}

func (m Model) viewTagCommits(bodyH int) string {
	if m.tagCommitDiffPanelOpen {
		return m.viewTagCommitsSplit(bodyH)
	}

	if len(m.tagCommits) == 0 {
		return dimStyle.Padding(2).Render("Loading commits...")
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		PaddingLeft(2).
		Render(fmt.Sprintf("Commits for tag: %s", m.tagDetailName))
	header += "\n" + lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", m.width-2))

	listH := bodyH - 3
	if listH < 1 {
		listH = 1
	}

	maxVisible := listH
	start := m.tagCommitCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.tagCommits) {
		end = len(m.tagCommits)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		c := m.tagCommits[i]
		selected := i == m.tagCommitCursor

		line := fmt.Sprintf("%-10s  %-50s  %-15s  %s",
			c.ShortID,
			fit(c.Title, 50),
			fit(c.AuthorName, 15),
			c.Date,
		)

		if selected {
			rows = append(rows, selectedStyle.Width(m.width-2).Render("▶ "+line))
		} else {
			rows = append(rows, normalItemStyle.Width(m.width-2).Render("  "+line))
		}
	}

	return header + "\n" + strings.Join(rows, "\n")
}

func (m Model) viewTagCommitsSplit(bodyH int) string {
	leftW := m.width * 2 / 5
	rightW := m.width - leftW - 1 // -1 for separator

	if leftW < 20 {
		leftW = 20
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		PaddingLeft(2).
		Render(fmt.Sprintf("Commits: %s", m.tagDetailName))
	header += "\n" + lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", leftW-2))

	listH := bodyH - 3
	if listH < 1 {
		listH = 1
	}

	maxVisible := listH
	start := m.tagCommitCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.tagCommits) {
		end = len(m.tagCommits)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		c := m.tagCommits[i]
		selected := i == m.tagCommitCursor

		line := fmt.Sprintf("%-10s  %s",
			c.ShortID,
			fit(c.Title, leftW-15),
		)

		if selected {
			rows = append(rows, selectedStyle.Width(leftW-2).Render("▶ "+line))
		} else {
			rows = append(rows, normalItemStyle.Width(leftW-2).Render("  "+line))
		}
	}

	leftContent := header + "\n" + strings.Join(rows, "\n")
	left := lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(leftContent)

	// Separator
	sepContent := strings.Repeat("│\n", bodyH)
	if bodyH > 0 {
		sepContent = sepContent[:len(sepContent)-1]
	}
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(sepContent)

	// Right: diff panel
	rightContent := m.viewTagCommitDiffPanel(rightW, bodyH)
	right := lipgloss.NewStyle().Width(rightW).Height(bodyH).MaxHeight(bodyH).Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

func (m Model) viewTagCommitDiffPanel(w, h int) string {
	var lines []string

	if m.tagCommitDiffLoading {
		lines = append(lines,
			subtitleStyle.Render("  Changes"),
			"",
			dimStyle.Render("  Loading diffs..."),
		)
		return strings.Join(lines, "\n")
	}

	if len(m.tagCommitDiffFiles) == 0 {
		lines = append(lines,
			subtitleStyle.Render("  Changes"),
			"",
			dimStyle.Render("  No files changed or no diff found."),
		)
		return strings.Join(lines, "\n")
	}

	// File list header
	fileCount := len(m.tagCommitDiffFiles)
	headerLine := subtitleStyle.Render("  Changes ") +
		dimStyle.Render(fmt.Sprintf("(%d file(s))  n/p=file, J/K=hunk", fileCount))
	lines = append(lines, headerLine)

	// File tabs
	var fileTabs []string
	startIdx := m.tagCommitDiffFileIdx - 1
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + 3
	if endIdx > len(m.tagCommitDiffFiles) {
		endIdx = len(m.tagCommitDiffFiles)
		startIdx = endIdx - 3
		if startIdx < 0 {
			startIdx = 0
		}
	}

	for i := startIdx; i < endIdx; i++ {
		f := m.tagCommitDiffFiles[i]
		name := f.NewPath
		counts := diffCounts(f)
		limit := w - 7 - len(counts)
		if limit < 35 {
			limit = 35
		}
		name = truncatePath(name, limit)
		label := fmt.Sprintf("%s %s", counts, name)
		if i == m.tagCommitDiffFileIdx {
			fileTabs = append(fileTabs, accentStyle.Render(" ▶ "+label))
		} else {
			fileTabs = append(fileTabs, dimStyle.Render("   "+label))
		}
	}
	lines = append(lines, strings.Join(fileTabs, "\n"))
	lines = append(lines, lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", w-2)))

	tabsLen := endIdx - startIdx
	diffHeight := h - (4 + tabsLen)
	if diffHeight < 1 {
		diffHeight = 1
	}

	// File diff content
	if m.tagCommitDiffFileIdx < len(m.tagCommitDiffFiles) {
		f := m.tagCommitDiffFiles[m.tagCommitDiffFileIdx]
		renderedCount := 0

		if len(f.Lines) == 0 {
			lines = append(lines, dimStyle.Render("  (diff unavailable — file is too large or collapsed)"))
			renderedCount++
		}

		for i := m.tagCommitDiffScrollOffset; i < len(f.Lines) && renderedCount < diffHeight; i++ {
			dl := f.Lines[i]
			selected := i == m.tagCommitDiffLineCursor
			content := dl.Content
			avail := w - 5
			if avail < 1 {
				avail = 1
			}
			if lipgloss.Width(content) > avail {
				content = ansi.Truncate(content, avail-1, "…")
			}

			var rendered string
			switch dl.Type {
			case "added":
				st := lipgloss.NewStyle().Foreground(colorSuccess)
				if selected {
					st = st.Background(colorBgHover).Bold(true)
				}
				rendered = st.Render("▶ " + content)
			case "removed":
				st := lipgloss.NewStyle().Foreground(colorError)
				if selected {
					st = st.Background(colorBgHover).Bold(true)
				}
				rendered = st.Render("▶ " + content)
			case "hunk":
				st := lipgloss.NewStyle().Foreground(colorInfo).Italic(true)
				if selected {
					st = st.Background(colorBgHover).Bold(true)
				}
				rendered = st.Render("  " + content)
			default:
				st := lipgloss.NewStyle().Foreground(colorTextDim)
				if selected {
					st = st.Background(colorBgHover)
				}
				rendered = st.Render("  " + content)
			}
			lines = append(lines, rendered)
			renderedCount++
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) viewCreateTag() string {
	var rows []string

	rows = append(rows, subtitleStyle.Render("🏷️ Create Tag"), "")

	fieldLabel := func(idx int, label string) string {
		if m.createTagField == idx {
			return accentStyle.Render("▶ " + label)
		}
		return dimStyle.Render("  " + label)
	}

	// Field 0: Tag Name (Title)
	nameBorderColor := colorBorder
	if m.createTagField == createTagNameField {
		nameBorderColor = colorAccent
	}
	nameBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(nameBorderColor).
		Padding(0, 1).
		Width(58).
		MarginLeft(2).
		Render(m.createTagName.View())
	rows = append(rows, fieldLabel(createTagNameField, "Tag Name (Title):"))
	rows = append(rows, nameBox)
	rows = append(rows, "")

	// Field 1: Create From Branch (Selector List)
	refBorderColor := colorBorder
	if m.createTagField == createTagRefField {
		refBorderColor = colorAccent
	}

	var branchLines []string
	if len(m.branches) == 0 {
		branchLines = append(branchLines, dimStyle.Render("  Loading branches..."))
	} else {
		maxVisible := 4
		start := m.createTagBranchCursor - maxVisible/2
		if start < 0 {
			start = 0
		}
		end := start + maxVisible
		if end > len(m.branches) {
			end = len(m.branches)
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}

		for i := start; i < end; i++ {
			b := m.branches[i]
			selected := i == m.createTagBranchCursor
			isDefault := m.repo != nil && b == m.repo.DefaultBranch

			label := b
			if isDefault {
				label += " (default)"
			}

			if selected {
				branchLines = append(branchLines, selectedStyle.Width(54).Render("▶ "+fit(label, 50)))
			} else {
				branchLines = append(branchLines, normalItemStyle.Width(54).Render("  "+fit(label, 50)))
			}
		}
	}

	refBoxContent := strings.Join(branchLines, "\n")
	refBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(refBorderColor).
		Padding(0, 1).
		Width(58).
		MarginLeft(2).
		Render(refBoxContent)
	rows = append(rows, fieldLabel(createTagRefField, "Create From Branch:"))
	rows = append(rows, refBox)
	rows = append(rows, "")

	// Field 2: Description / Message (Multiline Textarea)
	msgBorderColor := colorBorder
	if m.createTagField == createTagMessageField {
		msgBorderColor = colorAccent
	}
	msgBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(msgBorderColor).
		Padding(0, 1).
		Width(58).
		MarginLeft(2).
		Render(m.createTagMessage.View())
	rows = append(rows, fieldLabel(createTagMessageField, "Description / Message:"))
	rows = append(rows, msgBox)
	rows = append(rows, "")

	hints := []string{
		keyHint("Tab/Shift+Tab", "switch fields"),
		keyHint("↑/↓ or j/k", "select branch"),
		keyHint("Ctrl+S", "create tag"),
		keyHint("Esc", "cancel"),
	}
	rows = append(rows, strings.Join(hints, "  "))

	content := strings.Join(rows, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// ─── Edit Tag support ─────────────────────────────────────────────────────────

func (m Model) cmdUpdateTagRelease(tagName, description string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	full := m.repoFull()
	return func() tea.Msg {
		err := m.client.UpdateTagRelease(full, tagName, description)
		if err != nil {
			return errMsg{err}
		}
		return doneMsg{text: fmt.Sprintf("🏷️ Tag '%s' release notes updated!", tagName), reloadList: true}
	}
}

func (m Model) startEditTag(tag *gh.TagInfo) (Model, tea.Cmd) {
	if tag == nil || m.repo == nil {
		return m, nil
	}
	m.editTagName = tag.Name
	// Pre-fill with existing release description if present, otherwise tag message
	existing := tag.ReleaseDesc
	if existing == "" {
		existing = tag.Message
	}
	m.editTagDescription.SetValue(existing)
	m.editTagDescription.Blur()

	m.returnState = m.state
	m.state = stateEditTag

	cmd := m.editTagDescription.Focus()
	m.editTagDescription.CursorEnd()
	return m, cmd
}

func (m Model) handleEditTagKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.editTagDescription.Blur()
		m.state = m.returnState
		return m, nil
	case "ctrl+s":
		return m.submitEditTag()
	default:
		var cmd tea.Cmd
		m.editTagDescription, cmd = m.editTagDescription.Update(msg)
		return m, cmd
	}
}

func (m Model) submitEditTag() (Model, tea.Cmd) {
	description := strings.TrimSpace(m.editTagDescription.Value())
	m.editTagDescription.Blur()
	m.state = m.returnState
	return m, m.cmdUpdateTagRelease(m.editTagName, description)
}

func (m Model) viewEditTag() string {
	var rows []string

	rows = append(rows, subtitleStyle.Render("✏️  Edit Tag"), "")

	// Tag name (read-only)
	rows = append(rows, dimStyle.Render("  Tag Name:"))
	nameBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Width(58).
		MarginLeft(2).
		Render(lipgloss.NewStyle().Foreground(colorAccent).Render(m.editTagName))
	rows = append(rows, nameBox)
	rows = append(rows, "")

	// Release description (editable textarea)
	rows = append(rows, accentStyle.Render("▶ Release Description / Notes:"))
	descBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(0, 1).
		Width(58).
		MarginLeft(2).
		Render(m.editTagDescription.View())
	rows = append(rows, descBox)
	rows = append(rows, "")

	hints := []string{
		keyHint("Ctrl+S", "save"),
		keyHint("Esc", "cancel"),
	}
	rows = append(rows, strings.Join(hints, "  "))

	content := strings.Join(rows, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func diffCounts(f *gh.DiffFile) string {
	if len(f.Lines) == 0 && (f.NewFile || f.DeletedFile || f.RenamedFile || f.TooLarge || f.Collapsed || f.Overflow || f.AMode != f.BMode) {
		return "+? -?"
	}
	return fmt.Sprintf("+%d -%d", f.Added, f.Deleted)
}

func truncatePath(path string, limit int) string {
	if len(path) <= limit {
		return path
	}
	if limit <= 1 {
		return "…"
	}
	return "…" + path[len(path)-(limit-1):]
}
