package github

import "testing"

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
