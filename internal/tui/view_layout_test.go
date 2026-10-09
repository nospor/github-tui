package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github-tui/internal/config"
	gh "github-tui/internal/github"
)

func TestViewLineCountWithEmptyPRList(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabPRs
	m.username = "nospor"
	m.prs = []*gh.PullInfo{} // loaded, empty

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("empty PR list: rendered %d lines, terminal height %d (title may scroll off)", len(lines), m.height)
	}
	if !strings.Contains(got, "github-tui") {
		t.Fatal("expected title bar in view")
	}
}

func TestViewTitleBarHugsText(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	m := New(cfg, 0, nil, nil, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabRepos
	m.username = "nospor"

	title := m.viewTitle()
	w := lipgloss.Width(title)
	if w >= m.width {
		t.Fatalf("title bar width %d spanned the full terminal (%d)", w, m.width)
	}
	if !strings.Contains(title, "github-tui") {
		t.Fatalf("expected title text, got %q", title)
	}
}

func TestViewLineCountReposTab(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	m := New(cfg, 0, nil, nil, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabRepos
	m.username = "nospor"
	m.repos = make([]*gh.RepoInfo, 25)
	for i := range m.repos {
		m.repos[i] = &gh.RepoInfo{FullName: "nospor/teams-tui-go", Description: "x"}
	}
	m.layoutInputs()

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("repos tab many: rendered %d lines, terminal height %d", len(lines), m.height)
	}
	if !strings.Contains(got, "github-tui") {
		t.Fatal("expected title bar in view")
	}
}

func TestViewLineCountReposTabEmpty(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	m := New(cfg, 0, nil, nil, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabRepos
	m.username = "nospor"
	m.repos = []*gh.RepoInfo{}
	m.layoutInputs()

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("repos tab empty: rendered %d lines, terminal height %d", len(lines), m.height)
	}
}

func TestViewLineCountReposTabFitsSmallTerminal(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	for _, height := range []int{11, 12, 15, 20, 24} {
		m := New(cfg, 0, nil, nil, "", "", 0)
		m.width = 80
		m.height = height
		m.tab = tabRepos
		m.username = "nospor"
		m.repos = nil
		m.layoutInputs()
		got := m.View()
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
		if len(lines) > height {
			t.Fatalf("height=%d rendered %d lines", height, len(lines))
		}
	}
}

func TestViewLineCountReposTabManyReposNarrow(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 40
	m.height = 24
	m.tab = tabRepos
	m.username = "nospor"
	m.repos = make([]*gh.RepoInfo, 25)
	for i := range m.repos {
		m.repos[i] = &gh.RepoInfo{FullName: "org/some-repository-name", Description: "description text"}
	}
	m.layoutInputs()

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("narrow repos tab: rendered %d lines, height %d", len(lines), m.height)
	}
}

func TestViewLineCountReposTabWithStatus(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	m := New(cfg, 0, nil, nil, "token warning", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabRepos
	m.username = "nospor"
	m.repos = []*gh.RepoInfo{{FullName: "a/b"}}
	m.layoutInputs()

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("repos+status: rendered %d lines, height %d", len(lines), m.height)
	}
}

func TestRepoRowLongDescriptionDoesNotWrap(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	m := New(cfg, 0, nil, nil, "", "", 0)
	m.width = 120
	m.height = 24
	m.tab = tabRepos
	m.username = "nospor"
	m.repos = []*gh.RepoInfo{
		{
			FullName:    "nospor/noodle",
			Description: "A lightweight, fully keyboard-operated feed reader built in Go with Bubble Tea. Noodle brings a responsive and elegant TUI, complete with vim-style navigation",
		},
		{
			FullName:    "nospor/dbx",
			Description: "Fast multi-OS TUI client",
		},
	}
	m.layoutInputs()

	rowWidth := max(20, m.width-2)
	row := m.renderRepoRow(m.repos[0], false, rowWidth)
	if strings.Contains(row, "\n") {
		t.Fatalf("long description wrapped onto a second line:\n%s", row)
	}
	if w := lipgloss.Width(row); w > rowWidth {
		t.Fatalf("repo row width %d exceeds %d", w, rowWidth)
	}

	got := plain(m.View())
	var dbxLine string
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "nospor/dbx") {
			dbxLine = line
			break
		}
	}
	if dbxLine == "" {
		t.Fatal("expected dbx row in view")
	}
	if strings.Contains(dbxLine, "complete") || strings.Contains(dbxLine, "comple") {
		t.Fatalf("previous description leaked into next repo row: %q", dbxLine)
	}
	if !strings.Contains(dbxLine, "public") {
		t.Fatalf("expected visibility on dbx row, got %q", dbxLine)
	}
}

func TestViewLineCountWithPRs(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabPRs
	m.username = "nospor"
	m.prs = []*gh.PullInfo{{Number: 1, Title: "Fix things", State: "open"}}

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("with PRs: rendered %d lines, terminal height %d", len(lines), m.height)
	}
}

func TestPRListTableLayout(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 140
	m.height = 24
	m.tab = tabPRs
	m.username = "nospor"
	m.prs = []*gh.PullInfo{{
		Number:    553,
		Title:     "J-ET Reporting menu rebrand and reorganisation",
		State:     "open",
		Draft:     true,
		Author:    "carmermn",
		UpdatedAt: time.Date(2026, 10, 2, 8, 30, 0, 0, time.Local),
	}, {
		Number:    534,
		Title:     "build(docker): bake application code into image",
		State:     "open",
		Author:    "simonl",
		UpdatedAt: time.Date(2026, 9, 4, 13, 23, 0, 0, time.Local),
	}}

	got := plain(m.View())
	for _, want := range []string{"Title", "State", "Author", "Updated", "#553", "#534", "[DRAFT]", "carmermn", "simonl", "2026-10-02 08:30", "2026-09-04 13:23"} {
		if !strings.Contains(got, want) {
			t.Fatalf("PR table missing %q in:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "DRAFT") {
		t.Fatal("expected DRAFT marker for draft PR")
	}
	if strings.Contains(got, "head →") {
		t.Fatal("PR list should use columns, not head → base inline rest")
	}
}

func TestViewLineCountWithIssues(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabIssues
	m.username = "nospor"
	m.issues = []*gh.IssueInfo{{Number: 1, Title: "test issue", State: "closed", Author: "robertn"}}

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("with Issues: rendered %d lines, terminal height %d", len(lines), m.height)
	}
}

func TestIssuesListTableLayout(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 140
	m.height = 24
	m.tab = tabIssues
	m.username = "nospor"
	m.issues = []*gh.IssueInfo{{
		Number:    1,
		Title:     "test issue kkkk",
		State:     "closed",
		Author:    "robertn",
		UpdatedAt: time.Date(2026, 7, 20, 9, 26, 0, 0, time.Local),
	}, {
		Number:    2,
		Title:     "new tst2 222",
		State:     "closed",
		Author:    "robertn",
		UpdatedAt: time.Date(2026, 7, 20, 8, 56, 0, 0, time.Local),
	}}

	got := plain(m.View())
	for _, want := range []string{
		"Title", "State", "Author", "Updated",
		"#1", "#2", "test issue kkkk", "new tst2 222", "robertn",
		"2026-07-20 09:26", "2026-07-20 08:56", "closed",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Issues table missing %q in:\n%s", want, got)
		}
	}
}

func TestViewLineCountWithActions(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabActions
	m.username = "nospor"
	m.runs = []*gh.RunInfo{{ID: 38459, Name: "CI", Branch: "develop", Event: "push", Status: "completed", Conclusion: "success"}}

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("with Actions: rendered %d lines, terminal height %d", len(lines), m.height)
	}
}

func TestActionsListTableLayout(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 140
	m.height = 24
	m.tab = tabActions
	m.username = "nospor"
	m.runs = []*gh.RunInfo{{
		ID:         38459,
		Name:       "CI",
		Branch:     "refs/merge-requests/5/head",
		Event:      "pull_request",
		Actor:      "carmermn",
		Status:     "completed",
		Conclusion: "success",
		UpdatedAt:  time.Date(2026, 10, 2, 7, 25, 0, 0, time.Local),
	}, {
		ID:         38318,
		Name:       "CI",
		Branch:     "develop",
		Event:      "push",
		Actor:      "simonl",
		Status:     "completed",
		Conclusion: "failure",
		UpdatedAt:  time.Date(2026, 9, 29, 15, 7, 0, 0, time.Local),
	}}

	got := plain(m.View())
	for _, want := range []string{
		"ID", "Workflow", "Ref", "Status", "Triggered by", "Source", "Updated",
		"#38459", "#38318", "CI", "develop", "carmermn", "simonl",
		"push", "pull_request", "2026-10-02 07:25", "2026-09-29 15:07",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Actions table missing %q in:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "success") {
		t.Fatal("expected success status badge")
	}
	if !strings.Contains(got, "failed") {
		t.Fatal("expected failed status badge")
	}
}

func footerAtBottom(t *testing.T, m Model, got string) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != m.height {
		t.Fatalf("expected %d lines to pin footer, got %d", m.height, len(lines))
	}
	last := lines[len(lines)-1]
	if m.status != "" {
		if !strings.Contains(last, m.status) {
			t.Fatalf("expected status on last line, got %q", last)
		}
		last = lines[len(lines)-2]
	}
	if !strings.Contains(last, "j/k") && !strings.Contains(last, "tabs") {
		t.Fatalf("expected footer on last content line, got %q", last)
	}
}

func TestViewFooterPinnedOnBranchesTab(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabBranches
	m.username = "nospor"
	m.branches = []string{"main"}

	got := m.View()
	if !strings.Contains(got, "main") {
		t.Fatal("expected branch name in view")
	}
	footerAtBottom(t, m, got)
}

func TestViewFooterPinnedOnTagsTab(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabTags
	m.username = "nospor"
	m.tags = []*gh.TagInfo{{Name: "v1.0.0", ShortID: "abc1234"}}

	got := m.View()
	if !strings.Contains(got, "v1.0.0") {
		t.Fatal("expected tag name in view")
	}
	footerAtBottom(t, m, got)
}

func hasTabBar(got string) bool {
	plainView := plain(got)
	for _, label := range tabLabels {
		if strings.Contains(plainView, label) {
			return true
		}
	}
	return false
}

func TestViewShowsTabsOnList(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabPRs
	m.username = "nospor"
	m.prs = []*gh.PullInfo{{Number: 1, Title: "Fix things", State: "open"}}

	if !hasTabBar(m.View()) {
		t.Fatal("expected tab bar on list view")
	}
}

func TestViewHidesTabsWhenInaccessible(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	base := func() Model {
		m := New(cfg, 0, nil, repo, "", "", 0)
		m.width = 80
		m.height = 24
		m.username = "nospor"
		return m
	}

	t.Run("pr detail", func(t *testing.T) {
		m := base()
		m.tab = tabPRs
		m.state = stateDetail
		m.detailPR = &gh.PullInfo{Number: 1, Title: "Add diffs", State: "open", Head: "feat", Base: "main"}
		m.rebuildDetail()
		if hasTabBar(m.View()) {
			t.Fatal("expected no tab bar on PR detail")
		}
	})
	t.Run("issue detail", func(t *testing.T) {
		m := base()
		m.tab = tabIssues
		m.state = stateDetail
		m.detailIssue = &gh.IssueInfo{Number: 1, Title: "test issue", State: "open", Author: "robertn"}
		m.rebuildDetail()
		if hasTabBar(m.View()) {
			t.Fatal("expected no tab bar on issue detail")
		}
	})
	t.Run("actions detail", func(t *testing.T) {
		m := base()
		m.tab = tabActions
		m.state = stateDetail
		m.detailRun = &gh.RunInfo{ID: 38459, Name: "CI", Branch: "develop", Event: "push", Status: "completed"}
		if hasTabBar(m.View()) {
			t.Fatal("expected no tab bar on Actions run detail")
		}
	})
	t.Run("branch detail", func(t *testing.T) {
		m := base()
		m.tab = tabBranches
		m.state = stateDetail
		m.branchDetailName = "feat"
		m.branchCommits = []*gh.CommitInfo{{ShortID: "abc1234", Title: "wip"}}
		if hasTabBar(m.View()) {
			t.Fatal("expected no tab bar on branch detail")
		}
	})
	t.Run("job log", func(t *testing.T) {
		m := base()
		m.tab = tabActions
		m.state = stateJobLog
		m.logName = "test"
		m.logLines = []string{"ok"}
		if hasTabBar(m.View()) {
			t.Fatal("expected no tab bar on job log")
		}
	})
}

func TestViewPRDiffSplitFitsTerminal(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "nospor/teams-tui-go"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabPRs
	m.state = stateDetail
	m.username = "nospor"
	m.detailPR = &gh.PullInfo{Number: 1, Title: "Add diffs", State: "open", Head: "feat", Base: "main"}
	m.prDiffPanelOpen = true
	m.prDiffFiles = []*gh.DiffFile{{
		NewPath: "internal/tui/view.go",
		Added:   1,
		Deleted: 1,
		Lines: []gh.DiffLine{
			{Type: "hunk", Content: "@@ -1,2 +1,2 @@"},
			{Type: "removed", Content: "-old"},
			{Type: "added", Content: "+new"},
		},
	}}
	m.rebuildDetail()

	got := m.View()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("PR diff split: rendered %d lines, height %d", len(lines), m.height)
	}
	if !strings.Contains(got, "Changes") {
		t.Fatal("expected diff panel in view")
	}
}

func TestIssueDetailLayout(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "org/app"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 120
	m.height = 32
	m.tab = tabIssues
	m.state = stateDetail
	m.username = "robertn"
	m.detailIssue = &gh.IssueInfo{
		Number:    1,
		Title:     "test issue kkkk",
		State:     "closed",
		Author:    "robertn",
		Body:      "test **ddd**",
		HTMLURL:   "https://github.com/org/app/issues/1",
		Comments:  1,
		CreatedAt: time.Date(2026, 7, 20, 8, 12, 0, 0, time.Local),
		UpdatedAt: time.Date(2026, 7, 20, 9, 26, 0, 0, time.Local),
	}
	m.comments = []*gh.CommentInfo{{
		Author:    "robertn",
		Body:      "test comment",
		UpdatedAt: time.Date(2026, 7, 20, 8, 13, 0, 0, time.Local),
	}, {
		Author:    "robertn",
		Body:      "changed title from **test issue** to **test issue kkkk**",
		System:    true,
		UpdatedAt: time.Date(2026, 7, 20, 8, 46, 0, 0, time.Local),
	}}
	m.rebuildDetail()

	got := plain(m.View())
	for _, want := range []string{
		"#1", "test issue kkkk", "Author:", "robertn", "closed",
		"Updated: 2026-07-20 09:26", "Created: 2026-07-20 08:12",
		"ddd", "Discussions & Comments", "General Thread",
		"@robertn", "test comment", "changed title from",
		"https://github.com/org/app/issues/1",
		"vote up", "vote down",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("issue detail missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "**ddd**") {
		t.Fatal("expected markdown bold markers to be parsed")
	}
	if strings.Contains(got, "**test issue**") {
		t.Fatal("expected system note markdown to be parsed")
	}
}

func TestPRDetailShowsVotes(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "org/app"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 120
	m.height = 32
	m.tab = tabPRs
	m.state = stateDetail
	m.username = "robertn"
	m.detailPR = &gh.PullInfo{
		Number:   2,
		Title:    "Add diffs",
		State:    "open",
		Head:     "feat",
		Base:     "main",
		Author:   "robertn",
		Body:     "please review",
		PlusOne:  4,
		MinusOne: 1,
	}
	m.rebuildDetail()

	got := plain(m.View())
	for _, want := range []string{
		"#2", "Add diffs", "feat → main", "please review",
		"👍 4", "👎 1", "vote up", "vote down",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("PR detail missing %q in:\n%s", want, got)
		}
	}
}

func TestVoteStatus(t *testing.T) {
	if got := voteStatus("+1", true); got != "👍 Vote up added" {
		t.Fatalf("got %q", got)
	}
	if got := voteStatus("+1", false); got != "👍 Vote up removed" {
		t.Fatalf("got %q", got)
	}
	if got := voteStatus("-1", true); got != "👎 Vote down added" {
		t.Fatalf("got %q", got)
	}
	if got := voteStatus("-1", false); got != "👎 Vote down removed" {
		t.Fatalf("got %q", got)
	}
}

func TestConfirmDialogShowsYNHint(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "org/app"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	m.tab = tabPRs
	m.state = stateConfirm
	m.confirmMsg = "Merge pull request #1?"
	m.username = "robertn"

	got := plain(m.viewBody())
	for _, want := range []string{
		"Confirm",
		"Merge pull request #1?",
		"y confirm",
		"n cancel",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("confirm dialog missing %q in:\n%s", want, got)
		}
	}
}
