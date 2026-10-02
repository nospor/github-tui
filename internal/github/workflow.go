package github

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	gh "github.com/google/go-github/v68/github"
	"gopkg.in/yaml.v3"
)

// WorkflowInfo is a repository Actions workflow definition.
type WorkflowInfo struct {
	ID    int64
	Name  string
	Path  string
	State string
}

// Dispatchable reports whether GitHub will accept a workflow_dispatch for this workflow.
func (w *WorkflowInfo) Dispatchable() bool {
	return w != nil && w.State == "active"
}

// DispatchInput is one workflow_dispatch input from a workflow YAML file.
type DispatchInput struct {
	Name        string
	Description string
	Required    bool
	Type        string
	Default     string
	Options     []string
}

// IsChoice reports whether the input is selected from a fixed list (choice or boolean).
func (in *DispatchInput) IsChoice() bool {
	if in == nil {
		return false
	}
	switch in.Type {
	case "choice", "boolean":
		return true
	default:
		return len(in.Options) > 0
	}
}

// ListWorkflows lists Actions workflows, active first then by name.
func (c *Client) ListWorkflows(full string) ([]*WorkflowInfo, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	var out []*WorkflowInfo
	page := 1
	for {
		result, resp, err := c.raw.Actions.ListWorkflows(context.Background(), owner, name, &gh.ListOptions{
			Page:    page,
			PerPage: 100,
		})
		if err != nil {
			return nil, apiErr("list workflows", err)
		}
		if result != nil {
			for _, wf := range result.Workflows {
				if wf == nil || wf.GetState() == "deleted" {
					continue
				}
				item := &WorkflowInfo{
					ID:    wf.GetID(),
					Name:  wf.GetName(),
					Path:  wf.GetPath(),
					State: wf.GetState(),
				}
				if item.Name == "" {
					item.Name = item.Path
				}
				if item.Name == "" {
					continue
				}
				out = append(out, item)
			}
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}
	sort.Slice(out, func(i, j int) bool {
		ai, aj := out[i].State == "active", out[j].State == "active"
		if ai != aj {
			return ai
		}
		ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if ni != nj {
			return ni < nj
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// GetDispatchSpec loads a workflow file and returns its workflow_dispatch inputs.
func (c *Client) GetDispatchSpec(full, path, ref string) (ok bool, inputs []*DispatchInput, err error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return false, nil, err
	}
	if strings.TrimSpace(path) == "" {
		return false, nil, fmt.Errorf("workflow has no file path")
	}
	opts := &gh.RepositoryContentGetOptions{}
	if ref != "" {
		opts.Ref = ref
	}
	file, _, _, err := c.raw.Repositories.GetContents(context.Background(), owner, name, path, opts)
	if err != nil {
		return false, nil, apiErr("get workflow file "+path, err)
	}
	if file == nil {
		return false, nil, fmt.Errorf("workflow file %s is not a file", path)
	}
	body, err := file.GetContent()
	if err != nil {
		return false, nil, fmt.Errorf("decode workflow file %s: %w", path, err)
	}
	return ParseWorkflowDispatch([]byte(body))
}

// DispatchWorkflow starts a workflow_dispatch run on ref.
func (c *Client) DispatchWorkflow(full string, workflowID int64, ref string, inputs map[string]interface{}) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	if strings.TrimSpace(ref) == "" {
		return fmt.Errorf("ref is required to run a workflow")
	}
	_, err = c.raw.Actions.CreateWorkflowDispatchEventByID(context.Background(), owner, name, workflowID, gh.CreateWorkflowDispatchEventRequest{
		Ref:    ref,
		Inputs: inputs,
	})
	return apiErr(fmt.Sprintf("dispatch workflow %d", workflowID), err)
}

// ParseWorkflowDispatch reports whether YAML defines on.workflow_dispatch and its inputs.
func ParseWorkflowDispatch(src []byte) (ok bool, inputs []*DispatchInput, err error) {
	var root yaml.Node
	if err := yaml.Unmarshal(src, &root); err != nil {
		return false, nil, fmt.Errorf("parse workflow yaml: %w", err)
	}
	on := mappingValue(documentMapping(&root), "on")
	if on == nil {
		return false, nil, nil
	}
	dispatch, has := workflowDispatchNode(on)
	if !has {
		return false, nil, nil
	}
	if dispatch == nil || dispatch.Kind == yaml.ScalarNode || dispatch.Tag == "!!null" {
		return true, nil, nil
	}
	if dispatch.Kind != yaml.MappingNode {
		return true, nil, nil
	}
	inputsNode := mappingValue(dispatch, "inputs")
	if inputsNode == nil || inputsNode.Kind != yaml.MappingNode {
		return true, nil, nil
	}
	for i := 0; i+1 < len(inputsNode.Content); i += 2 {
		name := strings.TrimSpace(inputsNode.Content[i].Value)
		if name == "" {
			continue
		}
		in, derr := decodeDispatchInput(name, inputsNode.Content[i+1])
		if derr != nil {
			return true, nil, derr
		}
		inputs = append(inputs, in)
	}
	return true, inputs, nil
}

func documentMapping(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}
	if n.Kind != yaml.MappingNode {
		return nil
	}
	return n
}

func mappingValue(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func workflowDispatchNode(on *yaml.Node) (*yaml.Node, bool) {
	if on == nil {
		return nil, false
	}
	switch on.Kind {
	case yaml.ScalarNode:
		return nil, on.Value == "workflow_dispatch"
	case yaml.SequenceNode:
		for _, item := range on.Content {
			if item != nil && item.Kind == yaml.ScalarNode && item.Value == "workflow_dispatch" {
				return nil, true
			}
		}
		return nil, false
	case yaml.MappingNode:
		node := mappingValue(on, "workflow_dispatch")
		return node, node != nil
	default:
		return nil, false
	}
}

func decodeDispatchInput(name string, spec *yaml.Node) (*DispatchInput, error) {
	in := &DispatchInput{Name: name, Type: "string"}
	if spec == nil || spec.Kind == yaml.ScalarNode {
		if spec != nil {
			in.Description = spec.Value
		}
		return in, nil
	}
	var raw struct {
		Description string `yaml:"description"`
		Required    bool   `yaml:"required"`
		Type        string `yaml:"type"`
		Default     any    `yaml:"default"`
		Options     []any  `yaml:"options"`
	}
	if err := spec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("workflow input %q: %w", name, err)
	}
	in.Description = strings.TrimSpace(raw.Description)
	in.Required = raw.Required
	if t := strings.ToLower(strings.TrimSpace(raw.Type)); t != "" {
		in.Type = t
	}
	in.Default = formatDispatchValue(raw.Default)
	for _, opt := range raw.Options {
		if s := formatDispatchValue(opt); s != "" {
			in.Options = append(in.Options, s)
		}
	}
	if in.Type == "boolean" && len(in.Options) == 0 {
		in.Options = []string{"false", "true"}
		if in.Default == "" {
			in.Default = "false"
		}
	}
	return in, nil
}

func formatDispatchValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}
