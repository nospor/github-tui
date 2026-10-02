package tui

import (
	"strings"
	"testing"

	gh "github-tui/internal/github"
)

func TestCycleWorkflowFilter(t *testing.T) {
	m := testModel()
	m.workflows = []*gh.WorkflowInfo{
		{ID: 1, Name: "Build"},
		{ID: 2, Name: "CI"},
		{ID: 3, Name: "Deploy"},
	}
	if m.filteredWorkflow() != nil {
		t.Fatal("all should have no filter")
	}
	m.cycleWorkflowFilter()
	if got := m.filteredWorkflow(); got == nil || got.Name != "Build" {
		t.Fatalf("first cycle: %#v", got)
	}
	m.cycleWorkflowFilter()
	m.cycleWorkflowFilter()
	if got := m.filteredWorkflow(); got == nil || got.Name != "Deploy" {
		t.Fatalf("third cycle: %#v", got)
	}
	m.cycleWorkflowFilter()
	if m.runWorkflowID != 0 {
		t.Fatalf("wrap to all, id=%d", m.runWorkflowID)
	}
}

func TestActionsTitleIncludesWorkflow(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.workflows = []*gh.WorkflowInfo{{ID: 9, Name: "Deploy"}}
	m.runWorkflowID = 9
	got := m.listTitle()
	if !strings.Contains(got, "Deploy") {
		t.Fatalf("title %q", got)
	}
}

func TestOpenDispatchChooserSingleWorkflow(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.workflows = []*gh.WorkflowInfo{{ID: 4, Name: "Deploy", Path: ".github/workflows/deploy.yml", State: "active"}}
	got, cmd := m.openDispatchChooser()
	m = asModel(t, got)
	if m.dispatchWorkflow == nil || m.dispatchWorkflow.Name != "Deploy" {
		t.Fatal("expected to target Deploy")
	}
	if cmd == nil {
		t.Fatal("expected dispatch spec load")
	}
}

func TestOpenDispatchChooserManyWorkflows(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.state = stateMain
	m.workflows = []*gh.WorkflowInfo{
		{ID: 1, Name: "Build", State: "active"},
		{ID: 2, Name: "Deploy", State: "active"},
	}
	got, _ := m.openDispatchChooser()
	m = asModel(t, got)
	if m.state != stateDispatchSelect {
		t.Fatalf("state=%v", m.state)
	}
}

func TestOpenDispatchFormRendersInputs(t *testing.T) {
	m := testModel()
	m.width = 80
	m.height = 24
	m.tab = tabActions
	m.branches = []string{"main", "develop"}
	m.repo.DefaultBranch = "develop"
	wf := &gh.WorkflowInfo{ID: 2, Name: "Deploy", State: "active"}
	inputs := []*gh.DispatchInput{
		{Name: "environment", Description: "Environment", Required: true, Type: "choice", Default: "test", Options: []string{"test", "staging"}},
		{Name: "image_tag", Description: "Image tag (test only, optional)", Type: "string"},
	}
	got, _ := m.openDispatchForm(wf, inputs)
	m = asModel(t, got)
	if m.state != stateDispatch {
		t.Fatalf("state=%v", m.state)
	}
	if m.dispatchRef() != "develop" {
		t.Fatalf("ref=%q", m.dispatchRef())
	}
	view := m.viewDispatch()
	for _, want := range []string{"Deploy", "Environment", "test", "Image tag"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in:\n%s", want, view)
		}
	}
}

func TestSubmitDispatchRequiresInput(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.branches = []string{"develop"}
	m.dispatchWorkflow = &gh.WorkflowInfo{ID: 2, Name: "Deploy", State: "active"}
	m.dispatchInputs = []*gh.DispatchInput{
		{Name: "environment", Description: "Environment", Required: true, Type: "string"},
	}
	m.dispatchTexts = nil
	m.dispatchChoices = []int{0}
	got, _ := m.submitDispatch()
	m = asModel(t, got)
	if m.status == "" {
		t.Fatal("expected required-field status")
	}
}
