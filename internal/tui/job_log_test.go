package tui

import (
	"os"
	"strings"
	"testing"

	gh "github-tui/internal/github"
)

func TestJobLogPagerStartLineSkipsTruncationMarker(t *testing.T) {
	lines := []string{"a", "b", "c", "", gh.JobLogTruncatedMarker}
	if got := jobLogPagerStartLine(lines); got != 3 {
		t.Fatalf("got %d, want 3", got)
	}
	if got := jobLogPagerStartLine(nil); got != 1 {
		t.Fatalf("empty: got %d", got)
	}
}

func TestJobLogPagerCmdDefaultLess(t *testing.T) {
	t.Setenv("PAGER", "")
	cmd := jobLogPagerCmd("/tmp/log.txt", 12)
	if cmd.Args[0] != "less" {
		t.Fatalf("args=%v", cmd.Args)
	}
	want := []string{"less", "-R", "+12", "--", "/tmp/log.txt"}
	if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
		t.Fatalf("args=%v, want %v", cmd.Args, want)
	}
}

func TestJobLogPagerCmdCustom(t *testing.T) {
	t.Setenv("PAGER", "more")
	cmd := jobLogPagerCmd("log.txt", 4)
	if strings.Join(cmd.Args, " ") != "more log.txt" {
		t.Fatalf("args=%v", cmd.Args)
	}

	t.Setenv("PAGER", "less -S")
	cmd = jobLogPagerCmd("log.txt", 4)
	if cmd.Args[0] != "sh" || !strings.Contains(cmd.Args[2], "less -S") {
		t.Fatalf("args=%v", cmd.Args)
	}
}

func TestApplyLogSetsTruncated(t *testing.T) {
	m := testModel()
	got, _ := m.Update(logMsg{name: "build", text: "x", jobID: 1, truncated: true})
	m = asModel(t, got)
	if !m.logTruncated {
		t.Fatal("expected truncated")
	}
	if m.logName != "build" {
		t.Fatalf("name=%q", m.logName)
	}

	got, _ = m.Update(logMsg{name: "build", text: "x", jobID: 1, quiet: true})
	m = asModel(t, got)
	if m.logTruncated {
		t.Fatal("quiet full log should clear truncated")
	}
}

func TestHandleLogLoadRest(t *testing.T) {
	m := testModel()
	m.state = stateJobLog
	m.logTruncated = true
	m.logJobID = 4
	m.jobs = []*gh.JobInfo{{ID: 4, Name: "build"}}

	got, cmd := m.handleLog("L")
	m = asModel(t, got)
	if cmd == nil {
		t.Fatal("expected pager download")
	}
	if !m.loading {
		t.Fatal("load rest should show loading")
	}
	if !strings.Contains(m.status, "Downloading full log") {
		t.Fatalf("status=%q", m.status)
	}

	m.logTruncated = false
	_, cmd = m.handleLog("L")
	if cmd != nil {
		t.Fatal("should ignore load rest when the log is complete")
	}
}

func TestJobLogFooterLoadRest(t *testing.T) {
	m := testModel()
	m.state = stateJobLog
	m.logTruncated = true
	if !strings.Contains(m.viewFooter(), "load rest") {
		t.Fatalf("footer=%q", m.viewFooter())
	}
	m.logTruncated = false
	if strings.Contains(m.viewFooter(), "load rest") {
		t.Fatalf("complete log should hide load rest: %q", m.viewFooter())
	}
}

func TestJobLogPagerMsgClearsTempHint(t *testing.T) {
	m := testModel()
	m.inflight = 1
	m.loading = true
	m.status = "Downloading full log…"
	f, err := os.CreateTemp("", "github-tui-job-log-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	got, cmd := m.Update(jobLogPagerMsg{path: path, name: "build", line: 1})
	m = asModel(t, got)
	if cmd == nil {
		t.Fatal("expected pager exec")
	}
	if m.status != "" {
		t.Fatalf("status=%q", m.status)
	}
}
