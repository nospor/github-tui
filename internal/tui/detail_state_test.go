package tui

import (
	"testing"

	"github-tui/internal/config"
	gh "github-tui/internal/github"
)

func testModel() Model {
	cfg := &config.Config{Servers: []config.Server{{
		Name: "github.com",
		URL:  "https://github.com",
	}}}
	repo := &gh.RepoInfo{FullName: "owner/repo"}
	m := New(cfg, 0, nil, repo, "", "", 0)
	m.width = 80
	m.height = 24
	return m
}

func asModel(t *testing.T, got any) Model {
	t.Helper()
	m, ok := got.(Model)
	if !ok {
		t.Fatalf("got %T, want Model", got)
	}
	return m
}

func TestEscFromIssueClearsItemDetail(t *testing.T) {
	m := testModel()
	m.tab = tabIssues
	m.state = stateDetail
	m.detailIssue = &gh.IssueInfo{Number: 7, Title: "Broken", State: "open"}
	m.comments = []*gh.CommentInfo{{Author: "a", Body: "hi"}}
	m.rebuildDetail()

	got, _ := m.handleDetail("esc")
	m = asModel(t, got)
	if m.state != stateMain {
		t.Fatalf("state = %v, want main", m.state)
	}
	if m.detailIssue != nil {
		t.Fatal("esc from issue detail should clear detailIssue")
	}
	if m.comments != nil || m.detailLines != nil {
		t.Fatal("esc from issue detail should clear comments and rendered lines")
	}
}

func TestOpenBranchClearsLeftoverIssue(t *testing.T) {
	m := testModel()
	m.tab = tabBranches
	m.state = stateMain
	m.detailIssue = &gh.IssueInfo{Number: 7, Title: "Broken", State: "open"}
	m.comments = []*gh.CommentInfo{{Author: "a", Body: "hi"}}
	m.rebuildDetail()
	m.branches = []string{"main", "feat"}
	m.branchCursor = 1

	got, _ := m.openBranchOrTagDetail()
	m = asModel(t, got)
	if m.detailIssue != nil {
		t.Fatal("opening a branch should clear leftover issue detail")
	}
	if !m.inRefDetail() {
		t.Fatal("expected branch detail, not leftover issue detail")
	}
	if m.branchDetailName != "feat" {
		t.Fatalf("branch name = %q, want feat", m.branchDetailName)
	}
}

func TestOpenTagClearsLeftoverPR(t *testing.T) {
	m := testModel()
	m.tab = tabTags
	m.state = stateMain
	m.detailPR = &gh.PullInfo{Number: 3, Title: "Add diffs", State: "open"}
	m.rebuildDetail()
	m.tags = []*gh.TagInfo{{Name: "v1.0.0"}}
	m.tagCursor = 0

	got, _ := m.openBranchOrTagDetail()
	m = asModel(t, got)
	if m.detailPR != nil {
		t.Fatal("opening a tag should clear leftover PR detail")
	}
	if !m.inRefDetail() {
		t.Fatal("expected tag detail, not leftover PR detail")
	}
	if m.tagDetailName != "v1.0.0" {
		t.Fatalf("tag name = %q, want v1.0.0", m.tagDetailName)
	}
}

func TestBranchCompareLoadedClearsLeftoverRun(t *testing.T) {
	m := testModel()
	m.tab = tabBranches
	m.state = stateCompareBranchSelect
	m.detailRun = &gh.RunInfo{ID: 99, Name: "CI"}
	m.jobs = []*gh.JobInfo{{Name: "test"}}
	m.inflight = 1

	got, _ := m.Update(branchCompareLoadedMsg{
		targetBranch: "main",
		sourceBranch: "feat",
		compare:      &gh.CompareInfo{Commits: []*gh.CommitInfo{{ID: "abc", Title: "wip"}}},
	})
	m = asModel(t, got)
	if m.detailRun != nil {
		t.Fatal("compare result should clear leftover run detail")
	}
	if !m.inRefDetail() {
		t.Fatal("expected branch compare detail, not leftover run detail")
	}
	if m.branchDetailView != branchViewCompare {
		t.Fatal("expected compare view")
	}
}

func TestJumpTabClearsLeftoverIssue(t *testing.T) {
	m := testModel()
	m.tab = tabIssues
	m.state = stateMain
	m.detailIssue = &gh.IssueInfo{Number: 7, Title: "Broken", State: "open"}
	m.branches = []string{"main"}

	got, _ := m.jumpTab(tabBranches)
	m = asModel(t, got)
	if m.detailIssue != nil {
		t.Fatal("switching tabs should clear leftover issue detail")
	}
	if m.tab != tabBranches {
		t.Fatalf("tab = %v, want branches", m.tab)
	}
}

func TestAfterTabChangeClearsLeftoverPR(t *testing.T) {
	m := testModel()
	m.tab = tabBranches
	m.state = stateMain
	m.detailPR = &gh.PullInfo{Number: 3, Title: "Add diffs", State: "open"}
	m.branches = []string{"main"}

	got, _ := m.afterTabChange()
	m = asModel(t, got)
	if m.detailPR != nil {
		t.Fatal("cycling tabs should clear leftover PR detail")
	}
}
