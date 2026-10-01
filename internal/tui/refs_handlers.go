package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) inRefDetail() bool {
	return m.state == stateDetail && m.detailPR == nil && m.detailIssue == nil && m.detailRun == nil &&
		(m.tab == tabBranches || m.tab == tabTags)
}

func (m *Model) clearRefDetail() {
	m.branchDetailName = ""
	m.branchCommits = nil
	m.branchCommitCursor = 0
	m.branchCompare = nil
	m.branchCompareTarget = ""
	m.branchCompareCursor = 0
	m.branchCommitDiffFiles = nil
	m.branchCommitDiffFileIdx = 0
	m.branchCommitDiffLineCursor = 0
	m.branchCommitDiffScrollOffset = 0
	m.branchCommitDiffPanelOpen = false
	m.branchCommitDiffLoading = false
	m.branchCommitDiffSHA = ""
	m.tagDetailName = ""
	m.tagCommits = nil
	m.tagCommitCursor = 0
	m.tagCommitDiffFiles = nil
	m.tagCommitDiffFileIdx = 0
	m.tagCommitDiffLineCursor = 0
	m.tagCommitDiffScrollOffset = 0
	m.tagCommitDiffPanelOpen = false
	m.tagCommitDiffLoading = false
	m.tagCommitDiffSHA = ""
}

func (m Model) handleMainBranchTag(key string) (tea.Model, tea.Cmd) {
	if m.repo == nil {
		return m, nil
	}
	switch key {
	case "c":
		switch m.tab {
		case tabBranches:
			if m.branchCursor < len(m.branches) {
				return m.startCreatePRFromBranch(m.branches[m.branchCursor])
			}
		case tabTags:
			return m.startCreateTag()
		}
	case "C":
		if m.tab == tabBranches && m.branchCursor < len(m.branches) {
			m.compareSelectCursor = 0
			m.returnState = m.state
			m.state = stateCompareBranchSelect
			return m, nil
		}
	case "d":
		switch m.tab {
		case tabBranches:
			if m.branchCursor < len(m.branches) {
				branch := m.branches[m.branchCursor]
				return m.prompt(fmt.Sprintf("Delete branch '%s'?", branch), m.cmdDeleteBranch(branch))
			}
		case tabTags:
			if m.tagCursor < len(m.tags) {
				tag := m.tags[m.tagCursor]
				return m.prompt(fmt.Sprintf("Delete tag '%s'?", tag.Name), m.cmdDeleteTag(tag.Name))
			}
		}
	case "b":
		if m.tab == tabIssues {
			return m.startCreateBranchForIssue()
		}
	}
	return m, nil
}

func (m Model) openBranchOrTagDetail() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabBranches:
		if m.branchCursor >= len(m.branches) {
			return m, nil
		}
		branch := m.branches[m.branchCursor]
		m.branchDetailView = branchViewCommits
		m.branchDetailName = branch
		m.branchCommits = nil
		m.branchCommitCursor = 0
		m.returnState = stateMain
		m.state = stateDetail
		return m.track(m.cmdLoadBranchCommits(branch))
	case tabTags:
		if m.tagCursor >= len(m.tags) {
			return m, nil
		}
		tag := m.tags[m.tagCursor]
		m.tagDetailName = tag.Name
		m.tagCommits = nil
		m.tagCommitCursor = 0
		m.returnState = stateMain
		m.state = stateDetail
		return m.track(m.cmdLoadTagCommits(tag.Name))
	default:
		return m, nil
	}
}

func (m Model) handleRefDetail(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "esc":
		if m.returnState != stateMain && m.returnState != stateDetail {
			m.state = m.returnState
		} else {
			m.state = stateMain
		}
		m.clearRefDetail()
		return m, nil
	}
	switch m.tab {
	case tabBranches:
		return m.handleBranchDetailKey(key)
	case tabTags:
		return m.handleTagDetailKey(key)
	}
	return m, nil
}

func (m Model) handleBranchDetailKey(key string) (tea.Model, tea.Cmd) {
	if m.branchDetailView == branchViewCommits {
		if m.branchCommitDiffPanelOpen {
			return m.handleCommitDiffKey(key, true)
		}
		switch key {
		case "j", "down":
			if m.branchCommitCursor < len(m.branchCommits)-1 {
				m.branchCommitCursor++
			}
		case "k", "up":
			if m.branchCommitCursor > 0 {
				m.branchCommitCursor--
			}
		case "tab":
			m.branchCommitDiffPanelOpen = true
			if len(m.branchCommits) > 0 && m.branchCommitCursor < len(m.branchCommits) {
				c := m.branchCommits[m.branchCommitCursor]
				if m.branchCommitDiffSHA != c.ID {
					m.branchCommitDiffLoading = true
					m.branchCommitDiffFiles = nil
					return m.track(m.cmdLoadCommitDiff(c.ID))
				}
			}
		}
		return m, nil
	}
	if m.branchCompare == nil {
		return m, nil
	}
	if m.branchCommitDiffPanelOpen {
		return m.handleCommitDiffKey(key, true)
	}
	switch key {
	case "j", "down":
		if m.branchCompareCursor < len(m.branchCompare.Commits)-1 {
			m.branchCompareCursor++
		}
	case "k", "up":
		if m.branchCompareCursor > 0 {
			m.branchCompareCursor--
		}
	case "tab":
		m.branchCommitDiffPanelOpen = true
		if len(m.branchCompare.Commits) > 0 && m.branchCompareCursor < len(m.branchCompare.Commits) {
			c := m.branchCompare.Commits[m.branchCompareCursor]
			if m.branchCommitDiffSHA != c.ID {
				m.branchCommitDiffLoading = true
				m.branchCommitDiffFiles = nil
				return m.track(m.cmdLoadCommitDiff(c.ID))
			}
		}
	}
	return m, nil
}

func (m Model) handleTagDetailKey(key string) (tea.Model, tea.Cmd) {
	if m.tagCommitDiffPanelOpen {
		return m.handleCommitDiffKey(key, false)
	}
	switch key {
	case "j", "down":
		if m.tagCommitCursor < len(m.tagCommits)-1 {
			m.tagCommitCursor++
		}
	case "k", "up":
		if m.tagCommitCursor > 0 {
			m.tagCommitCursor--
		}
	case "tab":
		m.tagCommitDiffPanelOpen = true
		if len(m.tagCommits) > 0 && m.tagCommitCursor < len(m.tagCommits) {
			c := m.tagCommits[m.tagCommitCursor]
			if m.tagCommitDiffSHA != c.ID {
				m.tagCommitDiffLoading = true
				m.tagCommitDiffFiles = nil
				return m.track(m.cmdLoadCommitDiff(c.ID))
			}
		}
	case "e":
		for _, t := range m.tags {
			if t.Name == m.tagDetailName {
				return m.startEditTag(t)
			}
		}
	}
	return m, nil
}

func (m Model) handleCommitDiffKey(key string, branchPanel bool) (tea.Model, tea.Cmd) {
	switch key {
	case "tab", "esc":
		if branchPanel {
			m.branchCommitDiffPanelOpen = false
		} else {
			m.tagCommitDiffPanelOpen = false
		}
	case "j", "down":
		if branchPanel {
			m.branchCommitDiffLineCursorDown()
			m.updateBranchCommitDiffScroll()
		} else {
			m.tagCommitDiffLineCursorDown()
			m.updateTagCommitDiffScroll()
		}
	case "k", "up":
		if branchPanel {
			m.branchCommitDiffLineCursorUp()
			m.updateBranchCommitDiffScroll()
		} else {
			m.tagCommitDiffLineCursorUp()
			m.updateTagCommitDiffScroll()
		}
	case "J":
		if branchPanel {
			m.branchCommitDiffNextHunk()
			m.updateBranchCommitDiffScroll()
		} else {
			m.tagCommitDiffNextHunk()
			m.updateTagCommitDiffScroll()
		}
	case "K":
		if branchPanel {
			m.branchCommitDiffPrevHunk()
			m.updateBranchCommitDiffScroll()
		} else {
			m.tagCommitDiffPrevHunk()
			m.updateTagCommitDiffScroll()
		}
	case "n":
		if branchPanel {
			if m.branchCommitDiffFileIdx < len(m.branchCommitDiffFiles)-1 {
				m.branchCommitDiffFileIdx++
				m.branchCommitDiffLineCursor = 0
				m.branchCommitDiffScrollOffset = 0
			}
		} else if m.tagCommitDiffFileIdx < len(m.tagCommitDiffFiles)-1 {
			m.tagCommitDiffFileIdx++
			m.tagCommitDiffLineCursor = 0
			m.tagCommitDiffScrollOffset = 0
		}
	case "p":
		if branchPanel {
			if m.branchCommitDiffFileIdx > 0 {
				m.branchCommitDiffFileIdx--
				m.branchCommitDiffLineCursor = 0
				m.branchCommitDiffScrollOffset = 0
			}
		} else if m.tagCommitDiffFileIdx > 0 {
			m.tagCommitDiffFileIdx--
			m.tagCommitDiffLineCursor = 0
			m.tagCommitDiffScrollOffset = 0
		}
	}
	return m, nil
}

func (m Model) branchTagURL() string {
	if m.repo == nil {
		return ""
	}
	base := m.repo.HTMLURL
	switch m.tab {
	case tabBranches:
		if m.branchCursor >= 0 && m.branchCursor < len(m.branches) {
			return base + "/tree/" + m.branches[m.branchCursor]
		}
	case tabTags:
		if m.tagCursor >= 0 && m.tagCursor < len(m.tags) {
			return base + "/releases/tag/" + m.tags[m.tagCursor].Name
		}
	}
	return ""
}
