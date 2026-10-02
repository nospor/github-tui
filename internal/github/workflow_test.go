package github

import (
	"strings"
	"testing"
)

func TestParseWorkflowDispatchScalar(t *testing.T) {
	ok, inputs, err := ParseWorkflowDispatch([]byte("on: workflow_dispatch\njobs: {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected workflow_dispatch")
	}
	if len(inputs) != 0 {
		t.Fatalf("inputs = %#v", inputs)
	}
}

func TestParseWorkflowDispatchSequence(t *testing.T) {
	ok, _, err := ParseWorkflowDispatch([]byte("on: [push, workflow_dispatch]\n"))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	ok, _, err = ParseWorkflowDispatch([]byte("on: [push, pull_request]\n"))
	if err != nil || ok {
		t.Fatalf("push-only should not dispatch, ok=%v err=%v", ok, err)
	}
}

func TestParseWorkflowDispatchInputsOrderAndTypes(t *testing.T) {
	src := `
on:
  push:
    branches: [develop]
  workflow_dispatch:
    inputs:
      environment:
        description: Environment
        required: true
        type: choice
        options:
          - test
          - staging
          - prod
        default: test
      image_tag:
        description: "Image tag (test only, optional)"
        required: false
        type: string
      dry_run:
        description: Skip apply
        type: boolean
        default: false
jobs: {}
`
	ok, inputs, err := ParseWorkflowDispatch([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected workflow_dispatch")
	}
	if len(inputs) != 3 {
		t.Fatalf("len=%d", len(inputs))
	}
	if inputs[0].Name != "environment" || !inputs[0].Required || inputs[0].Type != "choice" {
		t.Fatalf("environment: %#v", inputs[0])
	}
	if got := strings.Join(inputs[0].Options, ","); got != "test,staging,prod" {
		t.Fatalf("options %q", got)
	}
	if inputs[0].Default != "test" {
		t.Fatalf("default %q", inputs[0].Default)
	}
	if inputs[1].Name != "image_tag" || inputs[1].Required || inputs[1].Type != "string" {
		t.Fatalf("image_tag: %#v", inputs[1])
	}
	if inputs[2].Name != "dry_run" || !inputs[2].IsChoice() {
		t.Fatalf("dry_run: %#v", inputs[2])
	}
	if got := strings.Join(inputs[2].Options, ","); got != "false,true" {
		t.Fatalf("boolean options %q", got)
	}
}

func TestParseWorkflowDispatchAbsent(t *testing.T) {
	ok, inputs, err := ParseWorkflowDispatch([]byte("on: push\n"))
	if err != nil || ok || inputs != nil {
		t.Fatalf("ok=%v inputs=%#v err=%v", ok, inputs, err)
	}
}

func TestParseWorkflowDispatchEmptyMapping(t *testing.T) {
	ok, inputs, err := ParseWorkflowDispatch([]byte("on:\n  workflow_dispatch:\n"))
	if err != nil || !ok || len(inputs) != 0 {
		t.Fatalf("ok=%v inputs=%#v err=%v", ok, inputs, err)
	}
}

func TestParseWorkflowDispatchInvalidYAML(t *testing.T) {
	_, _, err := ParseWorkflowDispatch([]byte("on: [\n"))
	if err == nil {
		t.Fatal("expected yaml error")
	}
}

func TestDispatchable(t *testing.T) {
	if (&WorkflowInfo{State: "active"}).Dispatchable() != true {
		t.Fatal("active should dispatch")
	}
	if (&WorkflowInfo{State: "disabled_manually"}).Dispatchable() {
		t.Fatal("disabled should not dispatch")
	}
	if (*WorkflowInfo)(nil).Dispatchable() {
		t.Fatal("nil should not dispatch")
	}
}
