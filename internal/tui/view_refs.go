package tui

func (m Model) viewRefDetail() string {
	h := m.listHeight() + 2
	switch m.tab {
	case tabBranches:
		if m.branchDetailView == branchViewCompare {
			return m.viewBranchCompare(h)
		}
		return m.viewBranchCommits(h)
	case tabTags:
		return m.viewTagCommits(h)
	default:
		return ""
	}
}
