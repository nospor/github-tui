package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	gh "github-tui/internal/github"
)

const runPollInterval = 5 * time.Second

func tickCmd() tea.Cmd {
	return tea.Tick(runPollInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) viewingSameRun(item *gh.RunInfo) bool {
	return item != nil && m.detailRun != nil && m.detailRun.ID == item.ID && m.viewingRun()
}

func (m Model) viewingRun() bool {
	return m.detailRun != nil && (m.state == stateDetail || m.state == stateJobLog)
}

func (m Model) scheduleRunPoll() tea.Cmd {
	if m.viewingRun() && isRunOrJobsActive(m.detailRun, m.jobs) {
		return tickCmd()
	}
	return nil
}

func (m Model) pollRunCmds() tea.Cmd {
	if !m.viewingRun() || !isRunOrJobsActive(m.detailRun, m.jobs) {
		return nil
	}
	cmds := []tea.Cmd{m.cmdLoadRun(m.detailRun.ID, true)}
	if m.state == stateJobLog {
		if job := m.jobByID(m.logJobID); job != nil {
			cmds = append(cmds, m.cmdLoadJobLog(job, true))
		}
	}
	return tea.Batch(cmds...)
}

func (m Model) jobByID(id int64) *gh.JobInfo {
	if id <= 0 {
		return nil
	}
	for _, job := range m.jobs {
		if job != nil && job.ID == id {
			return job
		}
	}
	return nil
}

func isRunOrJobsActive(run *gh.RunInfo, jobs []*gh.JobInfo) bool {
	if run != nil && isActionsStatusActive(run.Status) {
		return true
	}
	for _, job := range jobs {
		if job != nil && isActionsStatusActive(job.Status) {
			return true
		}
	}
	return false
}

func isActionsStatusActive(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "in_progress", "waiting", "pending", "requested":
		return true
	default:
		return false
	}
}

func (m *Model) applyLog(msg logMsg) {
	lines := strings.Split(strings.ReplaceAll(msg.text, "\r\n", "\n"), "\n")
	same := m.state == stateJobLog && msg.jobID > 0 && msg.jobID == m.logJobID
	atBottom := m.logScroll >= max(0, len(m.logLines)-m.logHeight())
	m.logName = msg.name
	m.logLines = lines
	m.logJobID = msg.jobID
	if !same {
		m.logScroll = 0
		m.state = stateJobLog
		return
	}
	if atBottom {
		m.logScroll = max(0, len(m.logLines)-m.logHeight())
	}
	m.clampLog()
}
