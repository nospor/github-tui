package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	gh "github-tui/internal/github"
)

type workflowsMsg struct {
	items []*gh.WorkflowInfo
}

type dispatchSpecMsg struct {
	workflow *gh.WorkflowInfo
	ok       bool
	inputs   []*gh.DispatchInput
}

func (m Model) cmdWorkflows() tea.Cmd {
	client := m.client
	full := m.repoFull()
	return func() tea.Msg {
		items, err := client.ListWorkflows(full)
		if err != nil {
			return errMsg{err}
		}
		return workflowsMsg{items: items}
	}
}

func (m Model) cmdDispatchSpec(wf *gh.WorkflowInfo) tea.Cmd {
	client := m.client
	full := m.repoFull()
	path := wf.Path
	ref := ""
	if m.repo != nil {
		ref = m.repo.DefaultBranch
	}
	return func() tea.Msg {
		ok, inputs, err := client.GetDispatchSpec(full, path, ref)
		if err != nil {
			return errMsg{err}
		}
		return dispatchSpecMsg{workflow: wf, ok: ok, inputs: inputs}
	}
}

func (m Model) cmdDispatchWorkflow(id int64, ref string, inputs map[string]interface{}) tea.Cmd {
	return m.cmdMutate(func(client *gh.Client, full string) error {
		return client.DispatchWorkflow(full, id, ref, inputs)
	}, "Workflow dispatched")
}

func (m Model) filteredWorkflow() *gh.WorkflowInfo {
	if m.runWorkflowID == 0 {
		return nil
	}
	for _, wf := range m.workflows {
		if wf != nil && wf.ID == m.runWorkflowID {
			return wf
		}
	}
	return nil
}

func (m *Model) cycleWorkflowFilter() {
	if len(m.workflows) == 0 {
		m.runWorkflowID = 0
		return
	}
	if m.runWorkflowID == 0 {
		m.runWorkflowID = m.workflows[0].ID
		return
	}
	for i, wf := range m.workflows {
		if wf != nil && wf.ID == m.runWorkflowID {
			if i+1 < len(m.workflows) {
				m.runWorkflowID = m.workflows[i+1].ID
			} else {
				m.runWorkflowID = 0
			}
			return
		}
	}
	m.runWorkflowID = 0
}

func (m *Model) clampWorkflowFilter() {
	if m.runWorkflowID == 0 {
		return
	}
	if m.filteredWorkflow() == nil {
		m.runWorkflowID = 0
	}
}

func (m Model) startRunWorkflow() (tea.Model, tea.Cmd) {
	if m.tab != tabActions {
		return m, nil
	}
	if m.repo == nil {
		m.setStatus("Select a repository first")
		return m, m.scheduleClear()
	}
	if m.workflows == nil {
		m.pendingDispatch = true
		return m.track(m.cmdWorkflows())
	}
	return m.openDispatchChooser()
}

func (m Model) startRunWorkflowFromDetail() (tea.Model, tea.Cmd) {
	if m.detailRun == nil {
		return m, nil
	}
	if m.workflows == nil {
		m.pendingDispatch = true
		m.pendingDispatchID = m.detailRun.WorkflowID
		return m.track(m.cmdWorkflows())
	}
	if wf := m.workflowByID(m.detailRun.WorkflowID); wf != nil {
		return m.beginDispatch(wf)
	}
	m.setStatus("Could not find that workflow")
	return m, m.scheduleClear()
}

func (m Model) openDispatchChooser() (tea.Model, tea.Cmd) {
	if id := m.pendingDispatchID; id != 0 {
		m.pendingDispatchID = 0
		if wf := m.workflowByID(id); wf != nil {
			return m.beginDispatch(wf)
		}
	}
	if wf := m.filteredWorkflow(); wf != nil {
		return m.beginDispatch(wf)
	}
	switch len(m.workflows) {
	case 0:
		m.setStatus("This repository has no workflows")
		return m, m.scheduleClear()
	case 1:
		return m.beginDispatch(m.workflows[0])
	default:
		m.dispatchCursor = 0
		m.returnState = m.state
		m.state = stateDispatchSelect
		return m, nil
	}
}

func (m Model) workflowByID(id int64) *gh.WorkflowInfo {
	if id == 0 {
		return nil
	}
	for _, wf := range m.workflows {
		if wf != nil && wf.ID == id {
			return wf
		}
	}
	return nil
}

func (m Model) beginDispatch(wf *gh.WorkflowInfo) (tea.Model, tea.Cmd) {
	if wf == nil {
		return m, nil
	}
	if !wf.Dispatchable() {
		m.setStatus(fmt.Sprintf("Workflow %s is %s", wf.Name, wf.State))
		return m, m.scheduleClear()
	}
	m.dispatchWorkflow = wf
	return m.track(m.cmdDispatchSpec(wf))
}

func (m Model) openDispatchForm(wf *gh.WorkflowInfo, inputs []*gh.DispatchInput) (tea.Model, tea.Cmd) {
	m.dispatchWorkflow = wf
	m.dispatchInputs = inputs
	m.dispatchField = 0
	m.dispatchTexts = make([]textinput.Model, len(inputs))
	m.dispatchChoices = make([]int, len(inputs))
	for i, in := range inputs {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Placeholder = in.Name
		if in.Description != "" {
			ti.Placeholder = in.Description
		}
		ti.SetValue(in.Default)
		m.dispatchTexts[i] = ti
		m.dispatchChoices[i] = choiceIndex(in, in.Default)
	}
	m.dispatchBranchCursor = 0
	if m.repo != nil && m.repo.DefaultBranch != "" {
		for i, b := range m.branches {
			if b == m.repo.DefaultBranch {
				m.dispatchBranchCursor = i
				break
			}
		}
	}
	if m.state == stateMain || m.state == stateDetail {
		m.returnState = m.state
	} else {
		m.returnState = stateMain
	}
	m.state = stateDispatch
	m.layoutInputs()
	nm, cmd := m.focusDispatchField()
	if len(m.branches) == 0 {
		nm, load := nm.track(m.cmdLoadBranches())
		return nm, tea.Batch(cmd, load)
	}
	return nm, cmd
}

func choiceIndex(in *gh.DispatchInput, value string) int {
	if in == nil || len(in.Options) == 0 {
		return 0
	}
	for i, opt := range in.Options {
		if opt == value {
			return i
		}
	}
	return 0
}

func (m Model) dispatchFieldCount() int {
	return 1 + len(m.dispatchInputs)
}

func (m Model) handleDispatchSelect(key string) (tea.Model, tea.Cmd) {
	n := len(m.workflows)
	switch key {
	case "esc", "q":
		m.state = m.returnState
		return m, nil
	case "j", "down":
		if m.dispatchCursor < n-1 {
			m.dispatchCursor++
		}
	case "k", "up":
		if m.dispatchCursor > 0 {
			m.dispatchCursor--
		}
	case "enter":
		if m.dispatchCursor >= 0 && m.dispatchCursor < n {
			wf := m.workflows[m.dispatchCursor]
			m.state = m.returnState
			return m.beginDispatch(wf)
		}
	}
	return m, nil
}

func (m Model) handleDispatch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := m.dispatchFieldCount()
	if n < 1 {
		n = 1
	}
	key := msg.String()
	switch key {
	case "esc":
		m.blurDispatch()
		m.state = m.returnState
		return m, nil
	case "tab":
		m.dispatchField = (m.dispatchField + 1) % n
		return m.focusDispatchField()
	case "shift+tab":
		m.dispatchField = (m.dispatchField + n - 1) % n
		return m.focusDispatchField()
	case "ctrl+s":
		return m.submitDispatch()
	case "enter":
		if m.dispatchField != 0 && !m.dispatchInputIsChoice(m.dispatchField-1) {
			m.dispatchField = (m.dispatchField + 1) % n
			return m.focusDispatchField()
		}
	}
	if m.dispatchField == 0 {
		switch key {
		case "j", "down":
			if m.dispatchBranchCursor < len(m.branches)-1 {
				m.dispatchBranchCursor++
			}
			return m, nil
		case "k", "up":
			if m.dispatchBranchCursor > 0 {
				m.dispatchBranchCursor--
			}
			return m, nil
		case "enter":
			m.dispatchField = (m.dispatchField + 1) % n
			return m.focusDispatchField()
		}
		return m, nil
	}
	idx := m.dispatchField - 1
	if m.dispatchInputIsChoice(idx) {
		opts := m.dispatchOptions(idx)
		switch key {
		case "j", "down":
			if len(opts) > 0 {
				m.dispatchChoices[idx] = (m.dispatchChoices[idx] + 1) % len(opts)
			}
			return m, nil
		case "k", "up":
			if len(opts) > 0 {
				m.dispatchChoices[idx] = (m.dispatchChoices[idx] + len(opts) - 1) % len(opts)
			}
			return m, nil
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.dispatchTexts[idx], cmd = m.dispatchTexts[idx].Update(msg)
	return m, cmd
}

func (m Model) dispatchInputIsChoice(idx int) bool {
	return idx >= 0 && idx < len(m.dispatchInputs) && m.dispatchInputs[idx].IsChoice()
}

func (m Model) dispatchOptions(idx int) []string {
	if idx < 0 || idx >= len(m.dispatchInputs) {
		return nil
	}
	return m.dispatchInputs[idx].Options
}

func (m Model) blurDispatch() {
	for i := range m.dispatchTexts {
		m.dispatchTexts[i].Blur()
	}
}

func (m Model) focusDispatchField() (Model, tea.Cmd) {
	m.blurDispatch()
	if m.dispatchField <= 0 {
		return m, nil
	}
	idx := m.dispatchField - 1
	if idx < 0 || idx >= len(m.dispatchTexts) || m.dispatchInputIsChoice(idx) {
		return m, nil
	}
	return m, m.dispatchTexts[idx].Focus()
}

func (m Model) dispatchRef() string {
	if len(m.branches) > 0 && m.dispatchBranchCursor >= 0 && m.dispatchBranchCursor < len(m.branches) {
		return m.branches[m.dispatchBranchCursor]
	}
	if m.repo != nil && m.repo.DefaultBranch != "" {
		return m.repo.DefaultBranch
	}
	return ""
}

func (m Model) submitDispatch() (tea.Model, tea.Cmd) {
	if m.dispatchWorkflow == nil {
		return m, nil
	}
	ref := strings.TrimSpace(m.dispatchRef())
	if ref == "" {
		m.setStatus("Branch is required")
		return m, m.scheduleClear()
	}
	values := map[string]interface{}{}
	for i, in := range m.dispatchInputs {
		val := m.dispatchValue(i)
		if strings.TrimSpace(val) == "" {
			if in.Required {
				label := in.Name
				if in.Description != "" {
					label = in.Description
				}
				m.setStatus(label + " is required")
				return m, m.scheduleClear()
			}
			continue
		}
		if in.Type == "boolean" {
			values[in.Name] = val == "true"
		} else {
			values[in.Name] = val
		}
	}
	m.blurDispatch()
	m.state = m.returnState
	return m.track(m.cmdDispatchWorkflow(m.dispatchWorkflow.ID, ref, values))
}

func (m Model) dispatchValue(idx int) string {
	if idx < 0 || idx >= len(m.dispatchInputs) {
		return ""
	}
	in := m.dispatchInputs[idx]
	if in.IsChoice() {
		opts := in.Options
		if len(opts) == 0 {
			return in.Default
		}
		i := m.dispatchChoices[idx]
		if i < 0 || i >= len(opts) {
			i = 0
		}
		return opts[i]
	}
	if idx < len(m.dispatchTexts) {
		return strings.TrimSpace(m.dispatchTexts[idx].Value())
	}
	return in.Default
}

func (m Model) viewDispatchSelect() string {
	var lines []string
	width := min(m.width-12, 72)
	for i, wf := range m.workflows {
		label := wf.Name
		if wf.State != "active" {
			label += "  (" + wf.State + ")"
		}
		lines = append(lines, m.renderChoice(label, i == m.dispatchCursor, width))
	}
	if len(lines) == 0 {
		lines = []string{dimStyle.Render("No workflows.")}
	}
	return m.placeDialog(subtitleStyle.Render("Run workflow"), "enter selects a workflow", strings.Join(lines, "\n"))
}

func (m Model) viewDispatch() string {
	var b strings.Builder
	title := "Run workflow"
	if m.dispatchWorkflow != nil {
		title += " · " + m.dispatchWorkflow.Name
	}
	b.WriteString(subtitleStyle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(m.fieldLabel("Use workflow from", m.dispatchField == 0))
	b.WriteString("\n")
	b.WriteString(m.viewDispatchBranch())
	for i, in := range m.dispatchInputs {
		b.WriteString("\n\n")
		label := in.Name
		if in.Description != "" {
			label = in.Description
		}
		if in.Required {
			label += " *"
		}
		b.WriteString(m.fieldLabel(label, m.dispatchField == i+1))
		b.WriteString("\n")
		if in.IsChoice() {
			b.WriteString(m.viewDispatchChoice(i))
		} else if i < len(m.dispatchTexts) {
			b.WriteString(m.dispatchTexts[i].View())
		}
	}
	return m.placeDialog("", "j/k changes branch or choice · ctrl+s runs the workflow", strings.TrimPrefix(b.String(), "\n"))
}

func (m Model) viewDispatchBranch() string {
	ref := m.dispatchRef()
	if ref == "" {
		if len(m.branches) == 0 {
			return dimStyle.Render("loading branches…")
		}
		return dimStyle.Render("(none)")
	}
	if m.dispatchField == 0 && len(m.branches) > 1 {
		return accentStyle.Render("◀ " + ref + " ▶")
	}
	return ref
}

func (m Model) viewDispatchChoice(idx int) string {
	opts := m.dispatchOptions(idx)
	if len(opts) == 0 {
		return dimStyle.Render("(no options)")
	}
	val := m.dispatchValue(idx)
	if m.dispatchField == idx+1 && len(opts) > 1 {
		return accentStyle.Render("◀ " + val + " ▶")
	}
	return val
}
