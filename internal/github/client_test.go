package github

import (
	"net/http"
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

func TestReactionCounts(t *testing.T) {
	plus, minus := reactionCounts(nil)
	if plus != 0 || minus != 0 {
		t.Fatalf("nil: got %d %d", plus, minus)
	}
	plus, minus = reactionCounts(&gh.Reactions{PlusOne: gh.Ptr(3), MinusOne: gh.Ptr(1)})
	if plus != 3 || minus != 1 {
		t.Fatalf("got %d %d", plus, minus)
	}
}

func TestUserReactionID(t *testing.T) {
	reactions := []*gh.Reaction{
		nil,
		{ID: gh.Ptr(int64(11)), Content: gh.Ptr("+1"), User: &gh.User{Login: gh.Ptr("alice")}},
		{ID: gh.Ptr(int64(22)), Content: gh.Ptr("-1"), User: &gh.User{Login: gh.Ptr("Bob")}},
	}
	if got := userReactionID(reactions, "+1", "alice"); got != 11 {
		t.Fatalf("+1 alice: got %d", got)
	}
	if got := userReactionID(reactions, "-1", "bob"); got != 22 {
		t.Fatalf("-1 bob: got %d", got)
	}
	if got := userReactionID(reactions, "-1", "alice"); got != 0 {
		t.Fatalf("alice has no -1, got %d", got)
	}
}

func TestToggleIssueVoteValidation(t *testing.T) {
	c := &Client{}
	if _, err := c.ToggleIssueVote("owner/repo", 1, "heart", "alice"); err == nil {
		t.Fatal("expected unsupported vote error")
	}
	if _, err := c.ToggleIssueVote("owner/repo", 1, "+1", ""); err == nil {
		t.Fatal("expected username required error")
	}
	if _, err := c.ToggleIssueVote("bad", 1, "+1", "alice"); err == nil {
		t.Fatal("expected split repo error")
	}
}

func TestIsNotFound(t *testing.T) {
	if isNotFound(nil) {
		t.Fatal("nil should not be not-found")
	}
	notFound := &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}
	if !isNotFound(notFound) {
		t.Fatal("expected 404 to be not-found")
	}
	if isNotFound(&gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}) {
		t.Fatal("500 should not be not-found")
	}
}
