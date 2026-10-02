package tui

import (
	"testing"

	"github-tui/internal/config"
	"strings"
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

func TestMarkdownToStyled(t *testing.T) {
	got := plain(markdownToStyled("hello **world** and `code`"))
	if strings.Contains(got, "**") {
		t.Fatalf("expected bold markers stripped, got %q", got)
	}
	if !strings.Contains(got, "world") || !strings.Contains(got, "code") {
		t.Fatalf("got %q", got)
	}
	got = plain(markdownToStyled("## Heading\n- item one\nSee [docs](https://example.com)"))
	if strings.Contains(got, "##") {
		t.Fatalf("expected heading markers stripped, got %q", got)
	}
	if !strings.Contains(got, "• item one") {
		t.Fatalf("expected list bullet, got %q", got)
	}
	if !strings.Contains(got, "docs") || !strings.Contains(got, "https://example.com") {
		t.Fatalf("expected link, got %q", got)
	}
	got = plain(markdownToStyled("- ~~Atomic attachment downloads — yes, please.~~"))
	if strings.Contains(got, "~~") {
		t.Fatalf("expected strikethrough markers stripped, got %q", got)
	}
	if !strings.Contains(got, "• Atomic attachment downloads") {
		t.Fatalf("expected list+strike text, got %q", got)
	}
	got = plain(markdownToStyled("this is *italic* text"))
	if strings.Contains(got, "*italic*") {
		t.Fatalf("expected italic markers stripped, got %q", got)
	}
}

func TestStyleSystemNote(t *testing.T) {
	got := plain(styleSystemNote("changed title from **old** to **new**"))
	if strings.Contains(got, "**") {
		t.Fatalf("expected bold markers stripped, got %q", got)
	}
	if !strings.Contains(got, "old") || !strings.Contains(got, "new") {
		t.Fatalf("got %q", got)
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
