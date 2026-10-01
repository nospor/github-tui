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
