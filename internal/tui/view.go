package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	gh "github-tui/internal/github"
)

func (m Model) showTabs() bool {
	return m.state == stateMain
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading…"
	}
	body := padHeight(m.viewBody(), m.bodyHeight())
	parts := []string{m.viewTitle()}
	if m.showTabs() {
		parts = append(parts, m.viewTabs())
	}
	parts = append(parts, body, m.viewFooter(), m.viewStatus())
	view := clipHeight(lipgloss.JoinVertical(lipgloss.Left, parts...), m.height)
	return m.applyYankOverlays(view)
}

func clipHeight(s string, height int) string {
	if height <= 0 {
		return s
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= height {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:height], "\n")
}

func padHeight(s string, height int) string {
	if height <= 0 {
		return s
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > height {
		return strings.Join(lines[:height], "\n")
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewTitle() string {
	srv := "github"
	if m.serverIdx >= 0 && m.serverIdx < len(m.cfg.Servers) {
		srv = m.cfg.Servers[m.serverIdx].Name
	}
	repo := "no repository"
	if m.repo != nil {
		repo = m.repo.FullName
	}
	user := m.username
	if user == "" {
		user = "…"
	}
	left := fmt.Sprintf(" github-tui   %s   %s", srv, user)
	if m.tab != tabRepos {
		left += "   " + repo
	}
	if m.loading {
		left += "   loading…"
	}
	avail := m.width - 4 // titleBarStyle horizontal padding
	if avail < 1 {
		avail = m.width
	}
	return titleBarStyle.Render(fit(left, avail))
}

func (m Model) viewTabs() string {
	var b strings.Builder
	for i, label := range tabLabels {
		b.WriteString(tabStyle(label, m.tab == tabID(i)))
	}
	return b.String()
}

func (m Model) viewStatus() string {
	if m.status == "" {
		return ""
	}
	return warningStyle.Width(m.width).Render(fit(m.status, m.width))
}

func (m Model) viewFooter() string {
	hints := m.hints()
	return mutedStyle.Width(m.width).Render(fit(hints, m.width))
}

func (m Model) hints() string {
	switch m.state {
	case stateConfirm:
		return joinHints([][2]string{{"y", "confirm"}, {"n", "cancel"}})
	case stateServerSelect:
		return joinHints([][2]string{{"j/k", "move"}, {"enter", "switch"}, {"esc", "back"}})
	case stateLinkSelect:
		return joinHints([][2]string{{"j/k", "move"}, {"enter", "open"}, {"esc", "back"}})
	case stateComment:
		return joinHints([][2]string{{"ctrl+s", "post"}, {"esc", "cancel"}})
	case stateCreate:
		return joinHints([][2]string{{"tab", "next field"}, {"ctrl+s", "save"}, {"esc", "cancel"}})
	case stateDispatchSelect:
		return joinHints([][2]string{{"j/k", "move"}, {"enter", "select"}, {"esc", "back"}})
	case stateDispatch:
		return joinHints([][2]string{{"tab", "next field"}, {"j/k", "choice"}, {"ctrl+s", "run"}, {"esc", "cancel"}})
	case stateJobLog:
		h := [][2]string{{"j/k", "scroll"}, {"g/G", "top/bottom"}, {"r", "refresh"}}
		if m.logTruncated {
			h = append(h, [2]string{"L", "load rest"})
		}
		h = append(h, [2]string{"esc", "back"})
		return joinHints(h)
	case stateDetail:
		if m.inRefDetail() {
			if m.branchCommitDiffPanelOpen || m.tagCommitDiffPanelOpen {
				return joinHints([][2]string{
					{"j/k", "scroll diff"}, {"n/p", "file"}, {"J/K", "hunk"},
					{"tab", "close diff"}, {"esc", "back"}, {"q", "quit"},
				})
			}
			h := [][2]string{{"j/k", "commits"}, {"tab", "diff"}, {"esc", "back"}, {"q", "quit"}}
			if m.tab == tabTags {
				h = append([][2]string{{"e", "edit tag"}}, h...)
			}
			return joinHints(h)
		}
		if m.prDiffPanelOpen && m.detailPR != nil {
			return joinHints([][2]string{
				{"j/k", "scroll diff"}, {"n/p", "file"}, {"J/K", "hunk"},
				{"tab", "close diff"}, {"esc", "back"}, {"q", "quit"},
			})
		}
		if m.detailRun != nil {
			return joinHints([][2]string{
				{"j/k", "job"}, {"enter", "log"}, {"w", "run"}, {"R", "rerun"}, {"c", "cancel"},
				{"r", "refresh"}, {"o", "open"}, {"y", "yank"}, {"esc", "back"},
			})
		}
		h := [][2]string{
			{"j/k", "scroll"}, {"C", "comment"}, {"b", "branch"}, {"m", "merge"}, {"x", "close"},
			{"O", "reopen"}, {"o", "open"}, {"y", "yank"}, {"esc", "back"},
		}
		if m.detailPR != nil {
			h = append([][2]string{{"tab", "diff"}}, h...)
		}
		if m.detailPR != nil || m.detailIssue != nil {
			h = append([][2]string{{"+", "vote up"}, {"-", "vote down"}}, h...)
		}
		return joinHints(h)
	default:
		h := [][2]string{
			{"1-6", "tabs"}, {"j/k", "move"}, {"enter", "open"}, {"r", "refresh"},
			{"n/p", "page"}, {"o", "browser"}, {"y", "yank"}, {"S", "server"}, {"q", "quit"},
		}
		switch m.tab {
		case tabPRs:
			h = append([][2]string{{"s", "state"}, {"c", "create"}, {"m", "merge"}, {"x", "close"}, {"O", "reopen"}}, h...)
		case tabBranches:
			h = append([][2]string{{"c", "create PR"}, {"C", "compare"}, {"d", "delete"}}, h...)
		case tabTags:
			h = append([][2]string{{"c", "create tag"}, {"d", "delete"}}, h...)
		case tabIssues:
			h = append([][2]string{{"s", "state"}, {"c", "create"}, {"b", "branch"}, {"x", "close"}, {"O", "reopen"}}, h...)
		case tabActions:
			h = append([][2]string{{"s", "workflow"}, {"w", "run"}, {"R", "rerun"}, {"c", "cancel"}}, h...)
		case tabRepos:
			return joinHints([][2]string{
				{"type", "search"}, {"up/down", "move"}, {"enter", "use repo"},
				{"esc", "clear"}, {"pgup/pgdn", "page"}, {"tab", "tabs"}, {"ctrl+c", "quit"},
			})
		}
		return joinHints(h)
	}
}

func joinHints(pairs [][2]string) string {
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, keyHint(p[0], p[1]))
	}
	return " " + strings.Join(parts, "  ")
}

func (m Model) viewBody() string {
	switch m.state {
	case stateConfirm:
		body := m.confirmMsg
		if body != "" {
			body += "\n\n"
		}
		body += dimStyle.Render("y confirm · n cancel")
		return m.placeDialog(subtitleStyle.Render("Confirm"), "", body)
	case stateServerSelect:
		return m.viewServers()
	case stateLinkSelect:
		return m.viewLinks()
	case stateComment:
		return m.placeDialog(subtitleStyle.Render("Comment"), "ctrl+s posts the comment", m.bodyInput.View())
	case stateCreate:
		return m.viewCreate()
	case stateCreateTag:
		return m.viewCreateTag()
	case stateEditTag:
		return m.viewEditTag()
	case stateCreateIssueBranch:
		return m.viewCreateIssueBranch()
	case stateDispatchSelect:
		return m.viewDispatchSelect()
	case stateDispatch:
		return m.viewDispatch()
	case stateCompareBranchSelect:
		return m.viewCompareBranchSelect()
	case stateJobLog:
		return m.viewLog()
	case stateDetail:
		if m.inRefDetail() {
			return m.viewRefDetail()
		}
		if m.detailRun != nil {
			return m.viewRun()
		}
		if m.prDiffPanelOpen && m.detailPR != nil {
			return m.viewPRDetailSplit()
		}
		return m.viewTextDetail()
	default:
		return m.viewList()
	}
}

func (m Model) viewBodyForState(st appState) string {
	switch st {
	case stateDetail:
		if m.detailRun != nil {
			return m.viewRun()
		}
		if m.prDiffPanelOpen && m.detailPR != nil {
			return m.viewPRDetailSplit()
		}
		return m.viewTextDetail()
	case stateJobLog:
		return m.viewLog()
	default:
		return m.viewListCore()
	}
}

func (m Model) placeDialog(title, note, body string) string {
	w := m.width - 8
	if w > 88 {
		w = 88
	}
	if w < 24 {
		w = 24
	}
	var b strings.Builder
	if title != "" {
		b.WriteString(title)
		b.WriteString("\n\n")
	}
	if note != "" {
		b.WriteString(dimStyle.Render(note))
		b.WriteString("\n\n")
	}
	b.WriteString(body)
	box := dialogStyle.Width(w).Render(b.String())

	bg := m.dialogBackground()
	height := m.bodyHeight()
	dlgWidth := lipgloss.Width(box)
	dlgHeight := lipgloss.Height(box)
	startX := (m.width - dlgWidth) / 2
	startY := (height - dlgHeight) / 2
	if startY < 0 {
		startY = 0
	}
	return overlay(bg, box, m.width, height, startX, startY)
}

func (m Model) dialogBackground() string {
	bg := ""
	switch m.state {
	case stateConfirm, stateComment, stateCreate, stateServerSelect, stateLinkSelect, stateDispatchSelect, stateDispatch:
		bg = m.viewBodyForState(m.returnState)
	}
	return m.padBodyHeight(bg)
}

func (m Model) padBodyHeight(content string) string {
	height := m.bodyHeight()
	if height < 1 {
		height = 1
	}
	lines := strings.Split(content, "\n")
	if content == "" {
		lines = nil
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewCreate() string {
	var b strings.Builder
	if m.formKind == "pr" {
		b.WriteString(subtitleStyle.Render("New pull request"))
		b.WriteString("\n\n")
		b.WriteString(m.fieldLabel("Head", m.formFocus == 0))
		b.WriteString("\n")
		b.WriteString(m.headInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.fieldLabel("Base", m.formFocus == 1))
		b.WriteString("\n")
		b.WriteString(m.baseInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.fieldLabel("Title", m.formFocus == 2))
		b.WriteString("\n")
		b.WriteString(m.titleInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.fieldLabel("Description", m.formFocus == 3))
		b.WriteString("\n")
		b.WriteString(m.bodyInput.View())
	} else {
		b.WriteString(subtitleStyle.Render("New issue"))
		b.WriteString("\n\n")
		b.WriteString(m.fieldLabel("Title", m.formFocus == 0))
		b.WriteString("\n")
		b.WriteString(m.titleInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.fieldLabel("Description", m.formFocus == 1))
		b.WriteString("\n")
		b.WriteString(m.bodyInput.View())
	}
	return m.placeDialog("", "", strings.TrimPrefix(b.String(), "\n"))
}

func (m Model) fieldLabel(label string, active bool) string {
	if active {
		return accentStyle.Render(label)
	}
	return dimStyle.Render(label)
}

func (m Model) viewServers() string {
	var lines []string
	for i, srv := range m.cfg.Servers {
		mark := "  "
		if i == m.serverIdx {
			mark = "* "
		}
		line := mark + srv.Name + "  " + srv.URL
		lines = append(lines, m.renderChoice(line, i == m.serverCursor, 70))
	}
	return m.placeDialog(subtitleStyle.Render("Servers"), "", strings.Join(lines, "\n"))
}

func (m Model) viewLinks() string {
	var lines []string
	width := min(m.width-12, 100)
	for i, link := range m.links {
		label := link.Label
		if label != link.URL {
			label = link.Label + "  " + link.URL
		}
		lines = append(lines, m.renderChoice(label, i == m.linkCursor, width))
	}
	return m.placeDialog(subtitleStyle.Render("Open link"), "", strings.Join(lines, "\n"))
}

func (m Model) viewList() string {
	if m.tab != tabRepos && m.repo == nil {
		return m.placeDialog(subtitleStyle.Render("No repository"), "Open the Repos tab and press enter, or run github-tui inside a GitHub clone.", "")
	}
	return m.viewListCore()
}

const (
	repoVisColW     = 9
	repoNameColW    = 42
	prIDColW        = 6
	prTitleColMax   = 55
	prStateColW     = 16
	prAuthorColW    = 14
	prUpdatedColW   = 16
	runIDColW       = 12
	runWorkflowColW = 14
	runRefColMax    = 22
	runStatusColW   = 16
	runActorColW    = 14
	runSourceColW   = 12
	runUpdatedColW  = 16
)

func (m Model) viewListCore() string {
	if m.tab == tabBranches {
		return m.viewBranchList(m.listHeight() + 2)
	}
	if m.tab == tabTags {
		return m.viewTagList(m.listHeight() + 2)
	}
	title := m.listTitle()
	rows := m.listRows()
	height := m.listHeight()
	_, offset := m.cursorPair()
	start := *offset
	if start < 0 {
		start = 0
	}
	var b strings.Builder
	b.WriteString(subtitleStyle.Render(title))
	b.WriteString("\n")
	width := max(20, m.width-2)
	if m.tab == tabRepos {
		b.WriteString("  ")
		b.WriteString(m.repoInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.repoListHeader(width))
		b.WriteString("\n")
	}
	if (m.tab == tabPRs || m.tab == tabIssues) && len(rows) > 0 {
		b.WriteString(m.prListHeader(width))
		b.WriteString("\n")
	}
	if m.tab == tabActions && len(rows) > 0 {
		b.WriteString(m.actionsListHeader(width))
		b.WriteString("\n")
	}
	if len(rows) == 0 {
		b.WriteString(dimStyle.Render(m.emptyList()))
		b.WriteString("\n")
	}
	for i := 0; i < height; i++ {
		idx := start + i
		if idx >= len(rows) {
			b.WriteString("\n")
			continue
		}
		b.WriteString(rows[idx])
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) repoListHeader(width int) string {
	header := lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(2).Render(
		fmt.Sprintf("%-*s  %-*s  %s", repoVisColW, "Vis", repoNameColW, "Repository", "Description"),
	)
	rule := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", width))
	return header + "\n" + rule
}

func (m Model) prListHeader(width int) string {
	titleW := prTitleWidth(width)
	header := lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(2).Render(
		fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s",
			prIDColW, "#",
			titleW, "Title",
			prStateColW, "State",
			prAuthorColW, "Author",
			prUpdatedColW, "Updated"),
	)
	rule := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", width))
	return header + "\n" + rule
}

func prTitleWidth(rowWidth int) int {
	const mark = 2
	const gaps = 8 // 4 column gaps of "  "
	fixed := mark + prIDColW + gaps + prStateColW + prAuthorColW + prUpdatedColW
	w := rowWidth - fixed
	if w > prTitleColMax {
		return prTitleColMax
	}
	if w < 8 {
		return 8
	}
	return w
}

func (m Model) actionsListHeader(width int) string {
	refW := runRefWidth(width)
	header := lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(2).Render(
		fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
			runIDColW, "ID",
			runWorkflowColW, "Workflow",
			refW, "Ref",
			runStatusColW, "Status",
			runActorColW, "Triggered by",
			runSourceColW, "Source",
			runUpdatedColW, "Updated"),
	)
	rule := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", width))
	return header + "\n" + rule
}

func runRefWidth(rowWidth int) int {
	const mark = 2
	const gaps = 12 // 6 column gaps of "  "
	fixed := mark + runIDColW + runWorkflowColW + gaps + runStatusColW + runActorColW + runSourceColW + runUpdatedColW
	w := rowWidth - fixed
	if w > runRefColMax {
		return runRefColMax
	}
	if w < 8 {
		return 8
	}
	return w
}

func (m Model) listTitle() string {
	switch m.tab {
	case tabPRs:
		return fmt.Sprintf("Pull requests · %s · page %d", m.prState, m.prPage)
	case tabBranches:
		return "Branches"
	case tabTags:
		return "Tags"
	case tabIssues:
		return fmt.Sprintf("Issues · %s · page %d", m.issueState, m.issuePage)
	case tabActions:
		name := "all"
		if wf := m.filteredWorkflow(); wf != nil {
			name = wf.Name
		}
		return fmt.Sprintf("Actions · %s · page %d", name, m.runPage)
	default:
		extra := ""
		if m.repoHasNext || m.repoPage > 1 {
			extra = fmt.Sprintf(" · page %d", m.repoPage)
		}
		return "Repositories" + extra
	}
}

func (m Model) emptyList() string {
	switch m.tab {
	case tabPRs:
		return "No pull requests."
	case tabBranches:
		return "No branches."
	case tabTags:
		return "No tags."
	case tabIssues:
		return "No issues."
	case tabActions:
		return "No workflow runs."
	default:
		if strings.TrimSpace(m.repoInput.Value()) != "" {
			return "No repositories match."
		}
		return "No repositories."
	}
}

func (m Model) listRows() []string {
	width := max(20, m.width-2)
	switch m.tab {
	case tabPRs:
		rows := make([]string, len(m.prs))
		for i, pr := range m.prs {
			rows[i] = m.renderPRRow(pr, i == m.prCursor, width)
		}
		return rows
	case tabIssues:
		rows := make([]string, len(m.issues))
		for i, issue := range m.issues {
			rows[i] = m.renderIssueRow(issue, i == m.issueCursor, width)
		}
		return rows
	case tabActions:
		rows := make([]string, len(m.runs))
		for i, run := range m.runs {
			rows[i] = m.renderRunRow(run, i == m.runCursor, width)
		}
		return rows
	default:
		repos := m.visibleRepos()
		rows := make([]string, len(repos))
		for i, repo := range repos {
			rows[i] = m.renderRepoRow(repo, i == m.repoCursor, width)
		}
		return rows
	}
}

func (m Model) renderPRRow(pr *gh.PullInfo, selected bool, width int) string {
	title := pr.Title
	if pr.Draft {
		title = "[DRAFT] " + title
	}
	titleW := prTitleWidth(width)
	id := padColumn(fmt.Sprintf("#%-4d", pr.Number), prIDColW)
	titleCol := padColumn(fit(title, titleW), titleW)
	state := padStatusBadge(statusBadge(pr.State), prStateColW)
	author := padColumn(fit(pr.Author, prAuthorColW), prAuthorColW)
	updated := dimStyle.Render(tableTime(pr.UpdatedAt))
	draft := ""
	if pr.Draft {
		draft = warningStyle.Render(" DRAFT")
	}
	line := id + "  " + titleCol + "  " + state + "  " + author + "  " + updated + draft
	st := normalItemStyle
	mark := "  "
	if selected {
		st = selectedStyle
		mark = "▶ "
	}
	return st.Width(width).Render(mark + line)
}

func (m Model) renderIssueRow(issue *gh.IssueInfo, selected bool, width int) string {
	titleW := prTitleWidth(width)
	id := padColumn(fmt.Sprintf("#%-4d", issue.Number), prIDColW)
	titleCol := padColumn(fit(issue.Title, titleW), titleW)
	state := padStatusBadge(statusBadge(issue.State), prStateColW)
	author := padColumn(fit(issue.Author, prAuthorColW), prAuthorColW)
	updated := dimStyle.Render(tableTime(issue.UpdatedAt))
	line := id + "  " + titleCol + "  " + state + "  " + author + "  " + updated
	st := normalItemStyle
	mark := "  "
	if selected {
		st = selectedStyle
		mark = "▶ "
	}
	return st.Width(width).Render(mark + line)
}

func (m Model) renderRunRow(run *gh.RunInfo, selected bool, width int) string {
	refW := runRefWidth(width)
	id := padColumn(fit(fmt.Sprintf("#%d", run.ID), runIDColW), runIDColW)
	workflow := padColumn(fit(run.Name, runWorkflowColW), runWorkflowColW)
	ref := padColumn(fit(run.Branch, refW), refW)
	status := padStatusBadge(statusBadge(run.Badge()), runStatusColW)
	actor := padColumn(fit(run.Actor, runActorColW), runActorColW)
	source := padColumn(fit(run.Event, runSourceColW), runSourceColW)
	updated := dimStyle.Render(tableTime(run.UpdatedAt))
	line := id + "  " + workflow + "  " + ref + "  " + status + "  " + actor + "  " + source + "  " + updated
	st := normalItemStyle
	mark := "  "
	if selected {
		st = selectedStyle
		mark = "▶ "
	}
	return st.Width(width).Render(mark + line)
}

func (m Model) renderRepoRow(repo *gh.RepoInfo, selected bool, width int) string {
	vis := padColumn(repoVisibilityLabel(repo.Private), repoVisColW)
	name := padColumn(fit(repo.FullName, repoNameColW), repoNameColW)
	descW := width - 2 - repoVisColW - 2 - repoNameColW - 2
	if descW < 8 {
		descW = 8
	}
	desc := dimStyle.Render(fit(repo.Description, descW))
	line := vis + "  " + name + "  " + desc
	st := normalItemStyle
	mark := "  "
	if selected {
		st = selectedStyle
		mark = "▶ "
	}
	return st.Width(width).Render(mark + line)
}

func (m Model) renderItem(badge, rest string, selected bool, width int) string {
	b := padStatusBadge(badge, 14)
	avail := width - lipgloss.Width(b) - 1
	if avail < 8 {
		avail = 8
	}
	st := normalItemStyle
	if selected {
		st = selectedStyle
	}
	return b + st.Width(avail).Render(fit(rest, avail))
}

func (m Model) renderChoice(line string, selected bool, width int) string {
	st := normalItemStyle
	if selected {
		st = selectedStyle
	}
	return st.Width(width).Render(fit(line, width))
}

func pullBadge(pr *gh.PullInfo) string {
	if pr.Draft && pr.State == "open" {
		return statusBadge("draft")
	}
	return statusBadge(pr.State)
}

func shortTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("Jan 02 15:04")
}

func tableTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}

func (m Model) viewTextDetail() string {
	m.clampDetail()
	height := m.detailHeight()
	start := m.detailScroll
	var b strings.Builder
	for i := 0; i < height; i++ {
		idx := start + i
		if idx >= len(m.detailLines) {
			b.WriteString("\n")
			continue
		}
		b.WriteString(m.detailLines[idx])
		b.WriteString("\n")
	}
	return b.String()
}

func (m *Model) rebuildDetail() {
	width := max(20, m.width-2)
	if m.prDiffPanelOpen && m.detailPR != nil {
		leftW, _ := m.splitPaneWidths()
		width = max(20, leftW-2)
	}
	switch {
	case m.detailPR != nil:
		m.detailLines = m.prDetailLines(width)
	case m.detailIssue != nil:
		m.detailLines = m.issueDetailLines(width)
	default:
		m.detailLines = nil
		return
	}
	m.clampDetail()
}

func (m Model) prDetailLines(width int) []string {
	pr := m.detailPR
	var lines []string
	add := func(s string) { lines = append(lines, s) }
	add(pullBadge(pr) + "  " + boldStyle.Render(fmt.Sprintf("#%d  %s", pr.Number, pr.Title)))
	add(dimStyle.Render(fmt.Sprintf("%s → %s   %s   %s", pr.Head, pr.Base, pr.Author, shortTime(pr.UpdatedAt))))
	add("")
	lines = append(lines, wrapStyled(pr.Body, width)...)
	add("")
	add(dimStyle.Render(fmt.Sprintf("👍 %d  👎 %d", pr.PlusOne, pr.MinusOne)))
	return m.appendComments(lines, width)
}

func (m Model) issueDetailLines(width int) []string {
	issue := m.detailIssue
	inner := width - 2
	if inner < 8 {
		inner = 8
	}
	pad := func(s string) string { return "  " + s }
	divider := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", inner))

	title := boldStyle.Render(fmt.Sprintf("#%d  %s", issue.Number, issue.Title))
	meta := lipgloss.JoinHorizontal(lipgloss.Center,
		statusBadge(issue.State),
		"  ",
		dimStyle.Render("Author: "),
		accentStyle.Render(issue.Author),
	)
	dates := dimStyle.Render("Updated: " + tableTime(issue.UpdatedAt) + "  Created: " + tableTime(issue.CreatedAt))

	var lines []string
	lines = append(lines, "", pad(title), pad(meta), pad(dates), "", pad(divider), "")
	if strings.TrimSpace(issue.Body) == "" {
		lines = append(lines, pad(dimStyle.Italic(true).Render("No description provided.")))
	} else {
		for _, line := range wrapStyled(issue.Body, inner) {
			lines = append(lines, pad(line))
		}
	}
	comments := issue.Comments
	if comments == 0 {
		for _, c := range m.comments {
			if !c.System {
				comments++
			}
		}
	}
	lines = append(lines,
		"",
		pad(dimStyle.Render(fmt.Sprintf("👍 %d  👎 %d  💬 %d", issue.PlusOne, issue.MinusOne, comments))),
		"",
		pad(infoStyle.Render("🔗 "+issue.HTMLURL)),
		"",
		pad(subtitleStyle.Render("💬 Discussions & Comments")),
		pad(divider),
	)
	if m.commentNote != "" {
		lines = append(lines, pad(warningStyle.Render(m.commentNote)))
	}
	if len(m.comments) == 0 {
		lines = append(lines, pad(dimStyle.Italic(true).Render("No comments yet.")))
		return lines
	}

	sawUser := false
	for _, c := range m.comments {
		if c.System {
			note := "• " + styleSystemNote(c.Body) + " " + dimStyle.Render("("+tableTime(c.UpdatedAt)+")")
			lines = append(lines, pad(note), pad(divider))
			continue
		}
		if !sawUser {
			lines = append(lines, pad(accentStyle.Render("General Thread")))
			sawUser = true
		}
		author := boldStyle.Render("@" + c.Author)
		when := dimStyle.Render(" on " + tableTime(c.UpdatedAt))
		lines = append(lines, pad(author+when))
		bodyWidth := inner - 2
		if bodyWidth < 8 {
			bodyWidth = 8
		}
		if strings.TrimSpace(c.Body) == "" {
			lines = append(lines, pad("  "+dimStyle.Italic(true).Render("(empty comment)")))
		} else {
			for _, line := range wrapStyled(c.Body, bodyWidth) {
				lines = append(lines, pad("  "+line))
			}
		}
		lines = append(lines, pad(divider))
	}
	return lines
}

func (m Model) appendComments(lines []string, width int) []string {
	if m.commentNote != "" {
		lines = append(lines, "", warningStyle.Render(m.commentNote))
	}
	lines = append(lines, "", subtitleStyle.Render("Comments"))
	if len(m.comments) == 0 {
		return append(lines, dimStyle.Render("No comments."))
	}
	for _, c := range m.comments {
		lines = append(lines, "", accentStyle.Render(c.Author)+"  "+dimStyle.Render(shortTime(c.UpdatedAt)))
		lines = append(lines, wrapStyled(c.Body, width)...)
	}
	return lines
}

func wrapStyled(text string, width int) []string {
	if strings.TrimSpace(text) == "" {
		return []string{dimStyle.Italic(true).Render("No description provided.")}
	}
	styled := markdownToStyled(text)
	wrapped := lipgloss.NewStyle().Width(width).Render(styled)
	return strings.Split(wrapped, "\n")
}

func (m Model) viewRun() string {
	run := m.detailRun
	var b strings.Builder
	b.WriteString(statusBadge(run.Badge()))
	b.WriteString("  ")
	b.WriteString(boldStyle.Render(run.Name))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("%s   %s   %s   %s   run %d", run.Branch, run.Event, run.Actor, shortTime(run.UpdatedAt), run.ID)))
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render("Jobs"))
	b.WriteString("\n")
	height := m.jobListHeight()
	if len(m.jobs) == 0 {
		b.WriteString(dimStyle.Render("No jobs."))
		b.WriteString("\n")
	}
	width := max(20, m.width-2)
	for i := 0; i < height; i++ {
		idx := m.jobOffset + i
		if idx >= len(m.jobs) {
			b.WriteString("\n")
			continue
		}
		job := m.jobs[idx]
		b.WriteString(m.renderItem(statusBadge(job.Badge()), job.Name, idx == m.jobCursor, width))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) viewLog() string {
	m.clampLog()
	var b strings.Builder
	title := "Log  " + m.logName
	if m.logTruncated {
		title += "  (truncated)"
	}
	b.WriteString(subtitleStyle.Render(title))
	b.WriteString("\n")
	height := m.logHeight()
	width := max(20, m.width-1)
	for i := 0; i < height; i++ {
		idx := m.logScroll + i
		if idx >= len(m.logLines) {
			b.WriteString("\n")
			continue
		}
		b.WriteString(fit(plain(m.logLines[idx]), width))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) bodyHeight() int {
	h := m.height - 2 // title, footer
	if m.showTabs() {
		h--
	}
	if m.status != "" {
		h--
	}
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) listHeight() int {
	h := m.bodyHeight() - 2 // list subtitle + one list row slot
	if m.listLen() == 0 {
		h -= 1 // empty-list message line
	}
	if m.tab == tabRepos {
		h -= 5 // search, blank line, column header, rule, spacer
	}
	if (m.tab == tabPRs || m.tab == tabIssues || m.tab == tabActions) && m.listLen() > 0 {
		h -= 2 // column header, rule
	}
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) detailHeight() int {
	h := m.bodyHeight()
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) jobListHeight() int {
	h := m.bodyHeight() - 4
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) logHeight() int {
	h := m.bodyHeight() - 1
	if h < 3 {
		return 3
	}
	return h
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
