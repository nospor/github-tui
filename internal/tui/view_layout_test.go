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
		"ID", "Ref", "Status", "Triggered by", "Source", "Updated",
		"#38459", "#38318", "develop", "carmermn", "simonl",
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
