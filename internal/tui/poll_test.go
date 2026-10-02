package tui

import (
	"strings"
	"testing"

	gh "github-tui/internal/github"
)

func TestIsActionsStatusActive(t *testing.T) {
	active := []string{"queued", "in_progress", "waiting", "pending", "requested", "QUEUED", " in_progress "}
	for _, status := range active {
		if !isActionsStatusActive(status) {
			t.Fatalf("%q should be active", status)
		}
	}
	inactive := []string{"completed", "success", "failure", "", "cancelled"}
	for _, status := range inactive {
		if isActionsStatusActive(status) {
			t.Fatalf("%q should not be active", status)
		}
	}
}

func TestIsRunOrJobsActive(t *testing.T) {
	if isRunOrJobsActive(&gh.RunInfo{Status: "completed"}, nil) {
		t.Fatal("completed run should not poll")
	}
	if !isRunOrJobsActive(&gh.RunInfo{Status: "in_progress"}, nil) {
		t.Fatal("running run should poll")
	}
	if !isRunOrJobsActive(&gh.RunInfo{Status: "completed"}, []*gh.JobInfo{{Status: "queued"}}) {
		t.Fatal("completed run with queued job should poll")
	}
}

func TestRunDetailStartsPollWhenActive(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.state = stateMain
	got, cmd := m.Update(runDetailMsg{
		item: &gh.RunInfo{ID: 7, Name: "CI", Status: "in_progress"},
		jobs: []*gh.JobInfo{{ID: 1, Name: "build", Status: "queued"}},
	})
	m = asModel(t, got)
	if m.state != stateDetail {
		t.Fatalf("state=%v, want detail", m.state)
	}
	if cmd == nil {
		t.Fatal("expected poll tick for an active run")
	}
}

func TestRunDetailSkipsPollWhenComplete(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	got, cmd := m.Update(runDetailMsg{
		item: &gh.RunInfo{ID: 7, Name: "CI", Status: "completed", Conclusion: "success"},
		jobs: []*gh.JobInfo{{ID: 1, Name: "build", Status: "completed", Conclusion: "success"}},
	})
	m = asModel(t, got)
	if m.state != stateDetail {
		t.Fatalf("state=%v, want detail", m.state)
	}
	if cmd != nil {
		t.Fatal("completed run should not schedule a poll")
	}
}

func TestQuietRunRefreshKeepsJobsOnPartialError(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.state = stateDetail
	m.detailRun = &gh.RunInfo{ID: 7, Status: "in_progress"}
	m.jobs = []*gh.JobInfo{{ID: 10, Name: "build", Status: "in_progress"}}
	m.jobCursor = 0

	got, _ := m.Update(runDetailMsg{
		item:   &gh.RunInfo{ID: 7, Status: "in_progress"},
		jobs:   nil,
		jobErr: "list workflow jobs: timeout",
		quiet:  true,
	})
	m = asModel(t, got)
	if len(m.jobs) != 1 || m.jobs[0].Name != "build" {
		t.Fatal("quiet poll should keep jobs when the job list fails")
	}
}

func TestQuietRunRefreshKeepsCursorAndLog(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.state = stateJobLog
	m.detailRun = &gh.RunInfo{ID: 7, Status: "in_progress"}
	m.jobs = []*gh.JobInfo{{ID: 10, Name: "a"}, {ID: 11, Name: "b", Status: "in_progress"}}
	m.jobCursor = 1
	m.logJobID = 11
	m.logName = "b"
	m.logLines = []string{"old"}

	got, cmd := m.Update(runDetailMsg{
		item:  &gh.RunInfo{ID: 7, Status: "in_progress"},
		jobs:  []*gh.JobInfo{{ID: 10, Name: "a", Status: "completed"}, {ID: 11, Name: "b", Status: "in_progress"}},
		quiet: true,
	})
	m = asModel(t, got)
	if m.state != stateJobLog {
		t.Fatal("quiet poll should leave the job log open")
	}
	if m.jobCursor != 1 {
		t.Fatalf("jobCursor=%d, want 1", m.jobCursor)
	}
	if m.loading {
		t.Fatal("quiet poll should not show the loading spinner")
	}
	if cmd == nil {
		t.Fatal("still-active run should schedule another tick")
	}
}

func TestTickReloadsActiveRunAndOpenLog(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.state = stateJobLog
	m.detailRun = &gh.RunInfo{ID: 9, Status: "in_progress"}
	m.jobs = []*gh.JobInfo{{ID: 3, Name: "test", Status: "in_progress"}}
	m.logJobID = 3

	got, cmd := m.Update(tickMsg{})
	m = asModel(t, got)
	if cmd == nil {
		t.Fatal("expected silent reload cmds")
	}
	if m.loading {
		t.Fatal("tick should not use tracked loading")
	}
}

func TestTickStopsWhenRunFinished(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.state = stateDetail
	m.detailRun = &gh.RunInfo{ID: 9, Status: "completed", Conclusion: "success"}
	m.jobs = []*gh.JobInfo{{ID: 3, Status: "completed", Conclusion: "success"}}

	_, cmd := m.Update(tickMsg{})
	if cmd != nil {
		t.Fatal("finished run should not keep polling")
	}
}

func TestTickIgnoredAwayFromRun(t *testing.T) {
	m := testModel()
	m.tab = tabPRs
	m.state = stateMain
	m.detailRun = &gh.RunInfo{ID: 9, Status: "in_progress"}

	_, cmd := m.Update(tickMsg{})
	if cmd != nil {
		t.Fatal("should not poll after leaving the run")
	}
}

func TestQuietLogFollowsBottom(t *testing.T) {
	m := testModel()
	m.width = 80
	m.height = 8
	m.state = stateJobLog
	m.logJobID = 4
	m.logName = "build"
	m.logLines = []string{"a", "b", "c", "d", "e", "f"}
	m.logScroll = max(0, len(m.logLines)-m.logHeight())

	got, _ := m.Update(logMsg{
		name:  "build",
		text:  "a\nb\nc\nd\ne\nf\ng\nh\ni\nj",
		jobID: 4,
		quiet: true,
	})
	m = asModel(t, got)
	if m.state != stateJobLog {
		t.Fatal("quiet log should stay on the log view")
	}
	if want := max(0, len(m.logLines)-m.logHeight()); m.logScroll != want {
		t.Fatalf("logScroll=%d, want bottom %d (height=%d)", m.logScroll, want, m.logHeight())
	}
	if got := strings.Join(m.logLines, "\n"); got != "a\nb\nc\nd\ne\nf\ng\nh\ni\nj" {
		t.Fatalf("lines=%q", got)
	}
}

func TestQuietLogPreservesScrollWhenNotAtBottom(t *testing.T) {
	m := testModel()
	m.width = 80
	m.height = 8
	m.state = stateJobLog
	m.logJobID = 4
	m.logLines = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	m.logScroll = 1

	got, _ := m.Update(logMsg{
		name:  "build",
		text:  "a\nb\nc\nd\ne\nf\ng\nh\ni\nj",
		jobID: 4,
		quiet: true,
	})
	m = asModel(t, got)
	if m.logScroll != 1 {
		t.Fatalf("logScroll=%d, want 1", m.logScroll)
	}
}

func TestInitialLogResetsScroll(t *testing.T) {
	m := testModel()
	m.state = stateDetail
	m.detailRun = &gh.RunInfo{ID: 1}
	m.logScroll = 9

	got, _ := m.Update(logMsg{name: "build", text: "line\nline2", jobID: 4})
	m = asModel(t, got)
	if m.state != stateJobLog {
		t.Fatalf("state=%v", m.state)
	}
	if m.logScroll != 0 {
		t.Fatalf("logScroll=%d, want 0", m.logScroll)
	}
	if m.logJobID != 4 {
		t.Fatalf("logJobID=%d", m.logJobID)
	}
}

func TestHandleLogRefresh(t *testing.T) {
	m := testModel()
	m.state = stateJobLog
	m.logJobID = 4
	m.jobs = []*gh.JobInfo{{ID: 4, Name: "build"}}

	got, cmd := m.handleLog("r")
	m = asModel(t, got)
	if cmd == nil {
		t.Fatal("expected log reload")
	}
	if !m.loading {
		t.Fatal("manual refresh should show loading")
	}
}

func TestPollDoesNotBatchTickAsTrackedLoad(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.state = stateDetail
	m.detailRun = &gh.RunInfo{ID: 1, Status: "queued"}
	cmds := m.pollRunCmds()
	if cmds == nil {
		t.Fatal("expected reload cmd")
	}
	if m.inflight != 0 || m.loading {
		t.Fatal("poll cmds must stay untracked")
	}
}

func TestFooterMentionsRefresh(t *testing.T) {
	m := testModel()
	m.tab = tabActions
	m.state = stateDetail
	m.detailRun = &gh.RunInfo{ID: 1, Name: "CI", Status: "in_progress"}
	if !strings.Contains(m.viewFooter(), "refresh") {
		t.Fatalf("footer=%q", m.viewFooter())
	}
	m.state = stateJobLog
	if !strings.Contains(m.viewFooter(), "refresh") {
		t.Fatalf("log footer=%q", m.viewFooter())
	}
}
