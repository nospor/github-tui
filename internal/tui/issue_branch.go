package tui

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	gh "github-tui/internal/github"
)

func slugifyIssueBranch(number int, title string) string {
	base := fmt.Sprintf("%d-%s", number, title)
	var sb strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(base) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen {
			sb.WriteRune('-')
			lastHyphen = true
		}
	}
	return strings.TrimSuffix(sb.String(), "-")
}

func (m Model) startCreateBranchForIssue() (Model, tea.Cmd) {
	var target *gh.IssueInfo
	if m.state == stateDetail && m.tab == tabIssues {
		target = m.detailIssue
	} else if m.state == stateMain && m.tab == tabIssues && m.issueCursor < len(m.issues) {
		target = m.issues[m.issueCursor]
	}
	if target == nil || m.repo == nil {
		return m, nil
	}
	ref := m.repo.DefaultBranch
	if ref == "" {
		ref = "main"
	}
	m.createIssueBranchIssue = target
	m.createIssueBranchName.SetValue(slugifyIssueBranch(target.Number, target.Title))
	m.createIssueBranchRef.SetValue(ref)
	m.createIssueBranchField = createIssueBranchFieldName
	m.createIssueBranchRef.Blur()
	m.returnState = m.state
	m.state = stateCreateIssueBranch
	cmd := m.createIssueBranchName.Focus()
	m.createIssueBranchName.CursorEnd()
	return m, cmd
}

func (m Model) handleCreateIssueBranchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.createIssueBranchName.Blur()
		m.createIssueBranchRef.Blur()
		m.state = m.returnState
		return m, nil
	case "tab":
		m.createIssueBranchField = (m.createIssueBranchField + 1) % createIssueBranchFieldCount
		return m.focusCreateIssueBranchField()
	case "shift+tab":
		m.createIssueBranchField = (m.createIssueBranchField - 1 + createIssueBranchFieldCount) % createIssueBranchFieldCount
		return m.focusCreateIssueBranchField()
	case "enter", "ctrl+s":
		return m.submitCreateBranchForIssue()
	}
	var cmd tea.Cmd
	if m.createIssueBranchField == createIssueBranchFieldName {
		m.createIssueBranchName, cmd = m.createIssueBranchName.Update(msg)
	} else {
		m.createIssueBranchRef, cmd = m.createIssueBranchRef.Update(msg)
	}
	return m, cmd
}

func (m Model) focusCreateIssueBranchField() (Model, tea.Cmd) {
	m.createIssueBranchName.Blur()
	m.createIssueBranchRef.Blur()
	if m.createIssueBranchField == createIssueBranchFieldName {
		return m, m.createIssueBranchName.Focus()
	}
	return m, m.createIssueBranchRef.Focus()
}

func (m Model) submitCreateBranchForIssue() (Model, tea.Cmd) {
	branchName := strings.TrimSpace(m.createIssueBranchName.Value())
	ref := strings.TrimSpace(m.createIssueBranchRef.Value())
	if branchName == "" || m.repo == nil || m.createIssueBranchIssue == nil {
		return m, nil
	}
	if ref == "" {
		ref = m.repo.DefaultBranch
	}
	if ref == "" {
		ref = "main"
	}
	m.createIssueBranchName.Blur()
	m.createIssueBranchRef.Blur()
	m.state = m.returnState
	return m.track(m.cmdCreateBranchForIssue(branchName, ref, m.createIssueBranchIssue.Number))
}

func (m Model) cmdCreateBranchForIssue(branch, ref string, issueNumber int) tea.Cmd {
	full := m.repoFull()
	return func() tea.Msg {
		if err := m.client.CreateBranch(full, branch, ref); err != nil {
			return errMsg{err}
		}
		return doneMsg{text: fmt.Sprintf("Branch '%s' created for issue #%d", branch, issueNumber), reloadList: true}
	}
}

func (m Model) viewCreateIssueBranch() string {
	return m.placeDialog(
		subtitleStyle.Render("Create branch for issue"),
		"tab between fields · ctrl+s save",
		strings.Join([]string{
			dimStyle.Render("Branch name"),
			m.createIssueBranchName.View(),
			"",
			dimStyle.Render("Create from ref"),
			m.createIssueBranchRef.View(),
		}, "\n"),
	)
}
