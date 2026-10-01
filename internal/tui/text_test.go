package tui

import (
	"testing"

	"github-tui/internal/config"
)

func TestFit(t *testing.T) {
	if got := fit("hello", 10); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if got := fit("hello", 4); got != "hel…" {
		t.Fatalf("got %q", got)
	}
}

func TestWrapAndFind(t *testing.T) {
	lines := wrapBlock("one two three", 8)
	if len(lines) < 2 {
		t.Fatalf("lines %#v", lines)
	}
	if got := findURLs("see https://github.com/cli/cli/pull/1."); len(got) != 1 || got[0] != "https://github.com/cli/cli/pull/1" {
		t.Fatalf("%#v", got)
	}
	if got := findIssueKeys("fixes PROJ-15"); len(got) != 1 || got[0] != "PROJ-15" {
		t.Fatalf("%#v", got)
	}
}

func TestViewEmptyRepoList(t *testing.T) {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	m := New(cfg, 0, nil, nil, "", "", 0)
	m.width = 100
	m.height = 30
	if got := m.View(); got == "" {
		t.Fatal("expected a rendered frame")
	}
}
