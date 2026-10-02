package github

import (
	"testing"

	gh "github.com/google/go-github/v68/github"
)

func TestSplitRepo(t *testing.T) {
	owner, name, err := SplitRepo("cli/cli")
	if err != nil || owner != "cli" || name != "cli" {
		t.Fatalf("got %s %s %v", owner, name, err)
	}
	if _, _, err := SplitRepo("onlyone"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeServer(t *testing.T) {
	web, api, enterprise, err := normalizeServer("https://github.com")
	if err != nil || enterprise || web != "https://github.com" || api != "https://api.github.com" {
		t.Fatalf("github.com: %s %s %v %v", web, api, enterprise, err)
	}
	web, api, enterprise, err = normalizeServer("https://ghe.example.com")
	if err != nil || !enterprise || web != "https://ghe.example.com" || api != "https://ghe.example.com/api/v3" {
		t.Fatalf("ghe: %s %s %v %v", web, api, enterprise, err)
	}
}

func TestRunBadge(t *testing.T) {
	if got := (&RunInfo{Status: "in_progress"}).Badge(); got != "running" {
		t.Fatalf("got %s", got)
	}
	if got := (&RunInfo{Status: "completed", Conclusion: "failure"}).Badge(); got != "failed" {
		t.Fatalf("got %s", got)
	}
}

func TestMapTimeline(t *testing.T) {
	from, to := "old title", "new title"
	ev := &gh.Timeline{
		Event:  gh.Ptr("renamed"),
		Actor:  &gh.User{Login: gh.Ptr("robertn")},
		Rename: &gh.Rename{From: &from, To: &to},
	}
	got := mapTimeline(ev)
	if got == nil || !got.System || got.Author != "robertn" {
		t.Fatalf("renamed: %#v", got)
	}
	if got.Body != "changed title from **old title** to **new title**" {
		t.Fatalf("body %q", got.Body)
	}

	comment := mapTimeline(&gh.Timeline{
		Event: gh.Ptr("commented"),
		User:  &gh.User{Login: gh.Ptr("alice")},
		Body:  gh.Ptr("hello **there**"),
	})
	if comment == nil || comment.System || comment.Author != "alice" || comment.Body != "hello **there**" {
		t.Fatalf("comment: %#v", comment)
	}

	if mapTimeline(&gh.Timeline{Event: gh.Ptr("subscribed")}) != nil {
		t.Fatal("expected noisy events to be skipped")
	}
}
