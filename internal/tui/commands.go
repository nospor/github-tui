package tui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github-tui/internal/config"
	gh "github-tui/internal/github"
)

func (m Model) cmdWhoAmI() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		name, err := client.WhoAmI()
		if err != nil {
			return errMsg{err}
		}
		return whoAmIMsg{name: name}
	}
}

func (m Model) cmdRepos() tea.Cmd {
	client := m.client
	page := m.repoPage
	return func() tea.Msg {
		items, hasNext, err := client.ListRepos(page)
		if err != nil {
			return errMsg{err}
		}
		return reposMsg{items: items, hasNext: hasNext, page: page}
	}
}

func (m Model) cmdPulls() tea.Cmd {
	client := m.client
	full := m.repoFull()
	state := m.prState
	page := m.prPage
	return func() tea.Msg {
		items, hasNext, err := client.ListPulls(full, state, page)
		if err != nil {
			return errMsg{err}
		}
		return pullsMsg{items: items, hasNext: hasNext, page: page}
	}
}

func (m Model) cmdIssues() tea.Cmd {
	client := m.client
	full := m.repoFull()
	state := m.issueState
	page := m.issuePage
	return func() tea.Msg {
		items, hasNext, err := client.ListIssues(full, state, page)
		if err != nil {
			return errMsg{err}
		}
		return issuesMsg{items: items, hasNext: hasNext, page: page}
	}
}

func (m Model) cmdRuns() tea.Cmd {
	client := m.client
	full := m.repoFull()
	page := m.runPage
	return func() tea.Msg {
		items, hasNext, err := client.ListRuns(full, page)
		if err != nil {
			return errMsg{err}
		}
		return runsMsg{items: items, hasNext: hasNext, page: page}
	}
}

func (m Model) cmdPull(number int) tea.Cmd {
	client := m.client
	full := m.repoFull()
	return func() tea.Msg {
		item, comments, truncated, err := client.GetPull(full, number)
		if item == nil && err != nil {
			return errMsg{err}
		}
		msg := pullDetailMsg{item: item, comments: comments, truncated: truncated}
		if err != nil {
			msg.commentErr = err.Error()
		}
		files, derr := client.GetPullDiffs(full, number)
		msg.files = files
		if derr != nil {
			msg.diffErr = derr.Error()
		}
		return msg
	}
}

func (m Model) cmdIssue(number int) tea.Cmd {
	client := m.client
	full := m.repoFull()
	return func() tea.Msg {
		item, comments, truncated, err := client.GetIssue(full, number)
		if item == nil && err != nil {
			return errMsg{err}
		}
		msg := issueDetailMsg{item: item, comments: comments, truncated: truncated}
		if err != nil {
			msg.commentErr = err.Error()
		}
		return msg
	}
}

func (m Model) cmdRun(id int64) tea.Cmd {
	client := m.client
	full := m.repoFull()
	return func() tea.Msg {
		item, jobs, err := client.GetRun(full, id)
		if item == nil && err != nil {
			return errMsg{err}
		}
		msg := runDetailMsg{item: item, jobs: jobs}
		if err != nil {
			msg.jobErr = err.Error()
		}
		return msg
	}
}

func (m Model) cmdOpenInitial() tea.Cmd {
	switch m.openKind {
	case "pr":
		return m.cmdPull(int(m.openNumber))
	case "issue":
		return m.cmdIssue(int(m.openNumber))
	case "run":
		return m.cmdRun(m.openNumber)
	default:
		return nil
	}
}

func (m Model) cmdJobLog(job *gh.JobInfo) tea.Cmd {
	client := m.client
	full := m.repoFull()
	name := job.Name
	id := job.ID
	return func() tea.Msg {
		text, err := client.JobLog(full, id)
		if err != nil {
			return errMsg{err}
		}
		if strings.TrimSpace(text) == "" {
			text = "(no log output)"
		}
		return logMsg{name: name, text: text}
	}
}

func (m Model) cmdComment(body string) tea.Cmd {
	client := m.client
	full := m.repoFull()
	var number int
	switch {
	case m.detailPR != nil:
		number = m.detailPR.Number
	case m.detailIssue != nil:
		number = m.detailIssue.Number
	default:
		return func() tea.Msg { return errMsg{errString("nothing to comment on")} }
	}
	return func() tea.Msg {
		if err := client.Comment(full, number, body); err != nil {
			return errMsg{err}
		}
		return doneMsg{text: "Comment posted", reloadDetail: true}
	}
}

func (m Model) cmdCreateIssue(title, body string) tea.Cmd {
	client := m.client
	full := m.repoFull()
	return func() tea.Msg {
		issue, err := client.CreateIssue(full, title, body)
		if err != nil {
			return errMsg{err}
		}
		return doneMsg{text: "Created issue #" + itoa(issue.Number), reloadList: true}
	}
}

func (m Model) cmdCreatePull(head, base, title, body string) tea.Cmd {
	client := m.client
	full := m.repoFull()
	return func() tea.Msg {
		pr, err := client.CreatePull(full, head, base, title, body)
		if err != nil {
			return errMsg{err}
		}
		return doneMsg{text: "Created pull request #" + itoa(pr.Number), reloadList: true}
	}
}

func (m Model) cmdSetPull(number int, state string) tea.Cmd {
	return m.cmdMutate(func(client *gh.Client, full string) error {
		return client.SetPullState(full, number, state)
	}, "Pull request #"+itoa(number)+" "+state)
}

func (m Model) cmdSetIssue(number int, state string) tea.Cmd {
	return m.cmdMutate(func(client *gh.Client, full string) error {
		return client.SetIssueState(full, number, state)
	}, "Issue #"+itoa(number)+" "+state)
}

func (m Model) cmdVoteIssue(number int, content string) tea.Cmd {
	client := m.client
	full := m.repoFull()
	username := m.username
	return func() tea.Msg {
		added, err := client.ToggleIssueVote(full, number, content, username)
		if err != nil {
			return errMsg{err}
		}
		return doneMsg{text: voteStatus(content, added), reloadList: true, reloadDetail: true}
	}
}

func voteStatus(content string, added bool) string {
	up := content != "-1"
	switch {
	case up && added:
		return "👍 Vote up added"
	case up:
		return "👍 Vote up removed"
	case added:
		return "👎 Vote down added"
	default:
		return "👎 Vote down removed"
	}
}

func (m Model) cmdMerge(number int) tea.Cmd {
	return m.cmdMutate(func(client *gh.Client, full string) error {
		return client.MergePull(full, number)
	}, "Merged pull request #"+itoa(number))
}

func (m Model) cmdCancel(id int64) tea.Cmd {
	return m.cmdMutate(func(client *gh.Client, full string) error {
		return client.CancelRun(full, id)
	}, "Cancel requested")
}

func (m Model) cmdRerun(id int64) tea.Cmd {
	return m.cmdMutate(func(client *gh.Client, full string) error {
		return client.Rerun(full, id)
	}, "Re-run requested")
}

func (m Model) cmdMutate(fn func(*gh.Client, string) error, okText string) tea.Cmd {
	client := m.client
	full := m.repoFull()
	reloadDetail := m.returnState == stateDetail || m.state == stateDetail
	return func() tea.Msg {
		if err := fn(client, full); err != nil {
			return errMsg{err}
		}
		return doneMsg{text: okText, reloadList: true, reloadDetail: reloadDetail}
	}
}

func (m Model) cmdSwitchServer(idx int, srv config.Server) tea.Cmd {
	return func() tea.Msg {
		client, err := gh.NewClient(srv.URL, srv.ResolvedToken())
		if err != nil {
			return errMsg{err}
		}
		return serverReadyMsg{idx: idx, client: client}
	}
}

func (m Model) openSelectedLink() (tea.Model, tea.Cmd) {
	var raw string
	switch m.tab {
	case tabPRs:
		if m.prCursor >= 0 && m.prCursor < len(m.prs) {
			raw = m.prs[m.prCursor].HTMLURL
		}
	case tabIssues:
		if m.issueCursor >= 0 && m.issueCursor < len(m.issues) {
			raw = m.issues[m.issueCursor].HTMLURL
		}
	case tabActions:
		if m.runCursor >= 0 && m.runCursor < len(m.runs) {
			raw = m.runs[m.runCursor].HTMLURL
		}
	case tabBranches, tabTags:
		raw = m.branchTagURL()
	case tabRepos:
		if repo := m.selectedRepo(); repo != nil {
			raw = repo.HTMLURL
		}
	}
	if raw == "" {
		return m, nil
	}
	return m.openURL(raw)
}

func (m Model) openDetailLinks() (tea.Model, tea.Cmd) {
	links := m.collectLinks()
	if len(links) == 0 {
		return m, nil
	}
	if len(links) == 1 {
		return m.openURL(links[0].URL)
	}
	m.links = links
	m.linkCursor = 0
	m.returnState = stateDetail
	m.state = stateLinkSelect
	return m, nil
}

func (m Model) yankSelected() (tea.Model, tea.Cmd) {
	var raw string
	switch m.tab {
	case tabPRs:
		if m.prCursor >= 0 && m.prCursor < len(m.prs) {
			raw = m.prs[m.prCursor].HTMLURL
		}
	case tabIssues:
		if m.issueCursor >= 0 && m.issueCursor < len(m.issues) {
			raw = m.issues[m.issueCursor].HTMLURL
		}
	case tabActions:
		if m.runCursor >= 0 && m.runCursor < len(m.runs) {
			raw = m.runs[m.runCursor].HTMLURL
		}
	case tabBranches, tabTags:
		raw = m.branchTagURL()
	case tabRepos:
		if repo := m.selectedRepo(); repo != nil {
			raw = repo.HTMLURL
		}
	}
	return m.yank(raw)
}

func (m Model) yank(raw string) (tea.Model, tea.Cmd) {
	if raw == "" {
		return m, nil
	}
	if err := clipboardWriteAll(raw); err != nil {
		m.setStatus(err.Error())
		return m, m.scheduleClear()
	}
	m.setStatus("Copied " + raw)
	return m, m.scheduleClear()
}

func (m Model) openURL(raw string) (tea.Model, tea.Cmd) {
	command := m.cfg.BrowserCommand
	if command == "" {
		command = "xdg-open"
	}
	suspend := false
	if m.cfg.IsYouTrackURL(raw) && m.cfg.YouTrackCommand != "" {
		command = m.cfg.YouTrackCommand
		suspend = true
	}
	cmd := exec.Command(command, raw)
	if suspend {
		return m.track(tea.ExecProcess(cmd, func(err error) tea.Msg {
			if err != nil {
				return errMsg{err}
			}
			return doneMsg{text: "Opened " + raw}
		}))
	}
	return m.track(func() tea.Msg {
		if err := cmd.Start(); err != nil {
			return errMsg{err}
		}
		return doneMsg{text: "Opened " + raw}
	})
}

func (m Model) collectLinks() []linkItem {
	var blobs []string
	var primary string
	switch {
	case m.detailPR != nil:
		primary = m.detailPR.HTMLURL
		blobs = append(blobs, m.detailPR.Title, m.detailPR.Body)
	case m.detailIssue != nil:
		primary = m.detailIssue.HTMLURL
		blobs = append(blobs, m.detailIssue.Title, m.detailIssue.Body)
	case m.detailRun != nil:
		primary = m.detailRun.HTMLURL
	}
	for _, c := range m.comments {
		blobs = append(blobs, c.Body)
	}
	seen := map[string]bool{}
	var links []linkItem
	add := func(label, raw string) {
		if raw == "" || seen[raw] {
			return
		}
		seen[raw] = true
		links = append(links, linkItem{Label: label, URL: raw})
	}
	if primary != "" {
		add("Open in browser", primary)
	}
	for _, blob := range blobs {
		for _, raw := range findURLs(blob) {
			add(raw, raw)
		}
		for _, key := range findIssueKeys(blob) {
			if u, ok := m.cfg.GetYouTrackURL(key); ok {
				add(key, u)
			}
		}
	}
	return links
}

type stringError string

func (e stringError) Error() string { return string(e) }

func errString(text string) error { return stringError(text) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
