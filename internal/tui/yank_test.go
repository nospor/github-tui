package tui

import (
	"strings"
	"testing"

	gh "github-tui/internal/github"
)

func TestYankOptionsPerDetail(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Model)
		wantLen int
		hasURLs bool
	}{
		{"PR detail", func(m *Model) { m.detailPR = &gh.PullInfo{Number: 1} }, 5, true},
		{"Issue detail", func(m *Model) { m.detailIssue = &gh.IssueInfo{Number: 1} }, 4, true},
		{"Run detail", func(m *Model) { m.detailRun = &gh.RunInfo{ID: 1} }, 3, true},
		{"No detail", func(*Model) {}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testModel()
			tt.setup(&m)
			opts := m.currentYankOptions()
			if len(opts) != tt.wantLen {
				t.Fatalf("expected %d options, got %d", tt.wantLen, len(opts))
			}
			found := false
			for _, o := range opts {
				if o.Key == "u" {
					found = true
				}
			}
			if found != tt.hasURLs {
				t.Fatalf("expected URLs option presence %v, got %v", tt.hasURLs, found)
			}
		})
	}
}

func TestYankTextPR(t *testing.T) {
	m := testModel()
	m.detailPR = &gh.PullInfo{
		Number: 42,
		Title:  "Add yank support",
		Body:   "Copies things",
		Head:   "feature/yank",
	}
	tests := []struct {
		option string
		want   string
	}{
		{"i", "#42"},
		{"t", "Add yank support"},
		{"d", "Copies things"},
		{"b", "feature/yank"},
		{"z", ""},
	}
	for _, tt := range tests {
		if got := m.yankText(tt.option); got != tt.want {
			t.Errorf("yankText(%q) = %q, want %q", tt.option, got, tt.want)
		}
	}
}

func TestYankTextIssueAndRun(t *testing.T) {
	issue := testModel()
	issue.detailIssue = &gh.IssueInfo{Number: 7, Title: "Broken link", Body: "It broke"}
	if got := issue.yankText("i"); got != "#7" {
		t.Errorf("issue ID = %q, want %q", got, "#7")
	}
	if got := issue.yankText("t"); got != "Broken link" {
		t.Errorf("issue title = %q", got)
	}
	if got := issue.yankText("d"); got != "It broke" {
		t.Errorf("issue body = %q", got)
	}

	run := testModel()
	run.detailRun = &gh.RunInfo{ID: 991, Name: "CI"}
	if got := run.yankText("i"); got != "#991" {
		t.Errorf("run ID = %q, want %q", got, "#991")
	}
	if got := run.yankText("t"); got != "CI" {
		t.Errorf("run name = %q", got)
	}
	if got := run.yankText("d"); got != "" {
		t.Errorf("run description should be empty, got %q", got)
	}
}

func TestOpenYankPopupWithYKey(t *testing.T) {
	m := testModel()
	m.state = stateDetail
	m.tab = tabIssues
	m.detailIssue = &gh.IssueInfo{Number: 1, Title: "T"}

	next, _ := m.handleDetail("y")
	got := asModel(t, next)
	if !got.yankOpen {
		t.Fatal("expected yank popup to open after 'y'")
	}

	next, _ = got.handleDetail("x")
	got = asModel(t, next)
	if got.yankOpen {
		t.Fatal("expected yank popup to close on unrelated key")
	}
	if got.state != stateDetail {
		t.Fatalf("dismissing yank should stay in detail, got state %v", got.state)
	}
	if got.detailIssue == nil {
		t.Fatal("dismissing yank with 'x' should not close the issue")
	}
}

func TestHandleYankPopupKeyCopies(t *testing.T) {
	var captured string
	var calls int
	orig := clipboardWriteAll
	clipboardWriteAll = func(s string) error { captured = s; calls++; return nil }
	defer func() { clipboardWriteAll = orig }()

	m := testModel()
	m.detailPR = &gh.PullInfo{Number: 5, Title: "My PR"}
	m.yankOpen = true

	next, _ := m.handleDetail("i")
	got := asModel(t, next)
	if got.yankOpen {
		t.Fatal("popup should close after copying")
	}
	if calls != 1 || captured != "#5" {
		t.Fatalf("expected clipboard to receive %q once, got %q (%d calls)", "#5", captured, calls)
	}
	if !strings.Contains(got.status, "Copied") {
		t.Fatalf("expected status message about copy, got %q", got.status)
	}
}

func TestCopyFailureShowsErrorStatus(t *testing.T) {
	orig := clipboardWriteAll
	clipboardWriteAll = func(string) error { return errTestClipboard }
	defer func() { clipboardWriteAll = orig }()

	m := testModel()
	m.detailPR = &gh.PullInfo{Number: 5}
	m.yankOpen = true
	next, _ := m.handleDetail("i")
	got := asModel(t, next)
	if !strings.Contains(got.status, "failed") {
		t.Fatalf("expected failure status message, got %q", got.status)
	}
}

type testErr struct{}

func (testErr) Error() string { return "no clipboard tool" }

var errTestClipboard = testErr{}

func TestYankURLSelectFlow(t *testing.T) {
	var captured []string
	orig := clipboardWriteAll
	clipboardWriteAll = func(s string) error { captured = append(captured, s); return nil }
	defer func() { clipboardWriteAll = orig }()

	m := testModel()
	m.detailPR = &gh.PullInfo{
		Number:  3,
		HTMLURL: "https://github.com/owner/repo/pull/3",
		Body:    "See https://example.com/a and https://example.com/b",
	}

	next, _ := m.performYank("u")
	got := asModel(t, next)
	if !got.yankURLSelect {
		t.Fatal("expected URL select mode with multiple URLs")
	}
	if len(got.yankItems) < 2 {
		t.Fatalf("expected at least 2 items in select list, got %d", len(got.yankItems))
	}

	next, _ = got.handleYankURLSelectKey("j")
	got = asModel(t, next)
	next, _ = got.handleYankURLSelectKey("j")
	got = asModel(t, next)
	next, _ = got.handleYankURLSelectKey("enter")
	got = asModel(t, next)
	if len(captured) != 1 {
		t.Fatalf("expected one copy after Enter, got %d", len(captured))
	}
	if captured[0] != "https://example.com/b" {
		t.Fatalf("expected second URL copied, got %q", captured[0])
	}
	if got.yankURLSelect {
		t.Fatal("URL select should close after Enter")
	}
}

func TestYankSingleURLCopiesDirectly(t *testing.T) {
	var captured []string
	orig := clipboardWriteAll
	clipboardWriteAll = func(s string) error { captured = append(captured, s); return nil }
	defer func() { clipboardWriteAll = orig }()

	m := testModel()
	m.detailRun = &gh.RunInfo{ID: 9, HTMLURL: "https://github.com/owner/repo/actions/runs/9"}
	next, _ := m.performYank("u")
	got := asModel(t, next)
	if got.yankURLSelect {
		t.Fatal("single URL should not open the select list")
	}
	if len(captured) != 1 || captured[0] != "https://github.com/owner/repo/actions/runs/9" {
		t.Fatalf("expected run URL copied, got %v", captured)
	}
}

func TestEscClosesYankBeforeLeavingDetail(t *testing.T) {
	m := testModel()
	m.state = stateDetail
	m.tab = tabIssues
	m.detailIssue = &gh.IssueInfo{Number: 1}
	m.yankOpen = true

	next, _ := m.handleDetail("esc")
	got := asModel(t, next)
	if got.yankOpen {
		t.Fatal("esc should close yank popup")
	}
	if got.state != stateDetail {
		t.Fatalf("esc should stay in detail view, got state %v", got.state)
	}
	if got.detailIssue == nil {
		t.Fatal("esc should not leave issue detail while yank is open")
	}
}
