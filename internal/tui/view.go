package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	gh "github-tui/internal/github"
)

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading…"
	}
	body := m.viewBody()
	screen := lipgloss.JoinVertical(lipgloss.Left, m.viewTitle(), m.viewTabs(), body, m.viewFooter(), m.viewStatus())
	return lipgloss.NewStyle().Width(m.width).Render(screen)
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
	left := fmt.Sprintf(" github-tui   %s   %s   %s", srv, user, repo)
	if m.loading {
		left += "   loading…"
	}
	return titleBarStyle.Width(m.width).Render(fit(left, m.width))
}

func (m Model) viewTabs() string {
	var b strings.Builder
	for i, label := range tabLabels {
		b.WriteString(tabStyle(label, m.tab == tabID(i) && m.state != stateJobLog))
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
	case stateJobLog:
		return joinHints([][2]string{{"j/k", "scroll"}, {"g/G", "top/bottom"}, {"esc", "back"}})
	case stateDetail:
		if m.detailRun != nil {
			return joinHints([][2]string{
				{"j/k", "job"}, {"enter", "log"}, {"R", "rerun"}, {"c", "cancel"},
				{"o", "open"}, {"y", "yank"}, {"esc", "back"},
			})
		}
		return joinHints([][2]string{
			{"j/k", "scroll"}, {"C", "comment"}, {"m", "merge"}, {"x", "close"},
			{"O", "reopen"}, {"o", "open"}, {"y", "yank"}, {"esc", "back"},
		})
	default:
		h := [][2]string{
			{"1-4", "tabs"}, {"j/k", "move"}, {"enter", "open"}, {"r", "refresh"},
			{"n/p", "page"}, {"o", "browser"}, {"y", "yank"}, {"S", "server"}, {"q", "quit"},
		}
		switch m.tab {
		case tabPRs:
			h = append([][2]string{{"s", "state"}, {"c", "create"}, {"m", "merge"}, {"x", "close"}, {"O", "reopen"}}, h...)
		case tabIssues:
			h = append([][2]string{{"s", "state"}, {"c", "create"}, {"x", "close"}, {"O", "reopen"}}, h...)
		case tabActions:
			h = append([][2]string{{"R", "rerun"}, {"c", "cancel"}}, h...)
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
		return m.placeDialog(subtitleStyle.Render("Confirm"), "", m.confirmMsg)
	case stateServerSelect:
		return m.viewServers()
	case stateLinkSelect:
		return m.viewLinks()
	case stateComment:
		return m.placeDialog(subtitleStyle.Render("Comment"), "ctrl+s posts the comment", m.bodyInput.View())
	case stateCreate:
		return m.viewCreate()
	case stateJobLog:
		return m.viewLog()
	case stateDetail:
		if m.detailRun != nil {
			return m.viewRun()
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
	case stateConfirm, stateComment, stateCreate, stateServerSelect, stateLinkSelect:
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
	repoVisColW  = 9
	repoNameColW = 42
)

func (m Model) viewListCore() string {
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
	if m.tab == tabRepos {
		b.WriteString("  ")
		b.WriteString(m.repoInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.repoListHeader(max(20, m.width-2)))
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

func (m Model) listTitle() string {
	switch m.tab {
	case tabPRs:
		return fmt.Sprintf("Pull requests · %s · page %d", m.prState, m.prPage)
	case tabIssues:
		return fmt.Sprintf("Issues · %s · page %d", m.issueState, m.issuePage)
	case tabActions:
		return fmt.Sprintf("Actions · page %d", m.runPage)
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
			rows[i] = m.renderItem(pullBadge(pr), pullRest(pr), i == m.prCursor, width)
		}
		return rows
	case tabIssues:
		rows := make([]string, len(m.issues))
		for i, issue := range m.issues {
			rows[i] = m.renderItem(statusBadge(issue.State), issueRest(issue), i == m.issueCursor, width)
		}
		return rows
	case tabActions:
		rows := make([]string, len(m.runs))
		for i, run := range m.runs {
			rows[i] = m.renderItem(statusBadge(run.Badge()), runRest(run), i == m.runCursor, width)
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
	return st.Width(width - 2).Render(mark + line)
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

func pullRest(pr *gh.PullInfo) string {
	return fmt.Sprintf("#%d  %s  %s → %s  %s  %s", pr.Number, pr.Title, pr.Head, pr.Base, pr.Author, shortTime(pr.UpdatedAt))
}

func issueRest(issue *gh.IssueInfo) string {
	return fmt.Sprintf("#%d  %s  %s  %s", issue.Number, issue.Title, issue.Author, shortTime(issue.UpdatedAt))
}

func runRest(run *gh.RunInfo) string {
	return fmt.Sprintf("%s  %s  %s  %s", run.Name, run.Branch, run.Event, shortTime(run.UpdatedAt))
}

func shortTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("Jan 02 15:04")
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
	var lines []string
	add := func(s string) { lines = append(lines, s) }
	switch {
	case m.detailPR != nil:
		pr := m.detailPR
		add(pullBadge(pr) + "  " + boldStyle.Render(fmt.Sprintf("#%d  %s", pr.Number, pr.Title)))
		add(dimStyle.Render(fmt.Sprintf("%s → %s   %s   %s", pr.Head, pr.Base, pr.Author, shortTime(pr.UpdatedAt))))
		add("")
		lines = append(lines, wrapStyled(pr.Body, width)...)
	case m.detailIssue != nil:
		issue := m.detailIssue
		add(statusBadge(issue.State) + "  " + boldStyle.Render(fmt.Sprintf("#%d  %s", issue.Number, issue.Title)))
		add(dimStyle.Render(fmt.Sprintf("%s   %s", issue.Author, shortTime(issue.UpdatedAt))))
		add("")
		lines = append(lines, wrapStyled(issue.Body, width)...)
	default:
		m.detailLines = nil
		return
	}
	if m.commentNote != "" {
		lines = append(lines, "", warningStyle.Render(m.commentNote))
	}
	lines = append(lines, "", subtitleStyle.Render("Comments"))
	if len(m.comments) == 0 {
		lines = append(lines, dimStyle.Render("No comments."))
	}
	for _, c := range m.comments {
		lines = append(lines, "")
		lines = append(lines, accentStyle.Render(c.Author)+"  "+dimStyle.Render(shortTime(c.UpdatedAt)))
		lines = append(lines, wrapStyled(c.Body, width)...)
	}
	m.detailLines = lines
	m.clampDetail()
}

func wrapStyled(text string, width int) []string {
	if strings.TrimSpace(text) == "" {
		return []string{dimStyle.Render("(no description)")}
	}
	raw := wrapBlock(text, width)
	out := make([]string, len(raw))
	for i, line := range raw {
		out[i] = line
	}
	return out
}

func (m Model) viewRun() string {
	run := m.detailRun
	var b strings.Builder
	b.WriteString(statusBadge(run.Badge()))
	b.WriteString("  ")
	b.WriteString(boldStyle.Render(run.Name))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("%s   %s   %s   run %d", run.Branch, run.Event, shortTime(run.UpdatedAt), run.ID)))
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
	b.WriteString(subtitleStyle.Render("Log  " + m.logName))
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
	h := m.height - 4
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) listHeight() int {
	h := m.bodyHeight() - 2
	if m.tab == tabRepos {
		h -= 4 // search, blank line, header, rule
	}
	if h < 3 {
		return 3
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
