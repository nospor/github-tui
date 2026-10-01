package tui

import (
	"strings"
	"testing"

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
