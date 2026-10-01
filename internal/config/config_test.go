package config

import "testing"

func sampleConfig() *Config {
	return &Config{
		Servers: []Server{{
			Name: "github.com",
			URL:  "https://github.com",
		}},
	}
}

func TestMatchRemote(t *testing.T) {
	cfg := sampleConfig()
	cases := []struct {
		remote string
		want   string
	}{
		{"git@github.com:cli/cli.git", "cli/cli"},
		{"https://github.com/cli/cli.git", "cli/cli"},
		{"ssh://git@github.com/cli/cli.git", "cli/cli"},
		{"git@gitlab.com:group/proj.git", ""},
	}
	for _, tc := range cases {
		got := matchRemote(tc.remote, cfg)
		var full string
		if got != nil {
			full = got.FullName
		}
		if full != tc.want {
			t.Errorf("matchRemote(%q) = %q, want %q", tc.remote, full, tc.want)
		}
	}
}

func TestParseGitHubURL(t *testing.T) {
	cfg := sampleConfig()
	idx, name, res, err := ParseGitHubURL(cfg, "https://github.com/cli/cli/pull/1234")
	if err != nil {
		t.Fatal(err)
	}
	if idx != 0 || name != "cli/cli" || res.Kind != "pr" || res.Number != 1234 {
		t.Fatalf("got idx=%d name=%s res=%+v", idx, name, res)
	}

	_, _, res, err = ParseGitHubURL(cfg, "https://github.com/cli/cli/issues/9")
	if err != nil || res.Kind != "issue" || res.Number != 9 {
		t.Fatalf("issue: %+v %v", res, err)
	}

	_, _, res, err = ParseGitHubURL(cfg, "https://github.com/cli/cli/actions/runs/42")
	if err != nil || res.Kind != "run" || res.Number != 42 {
		t.Fatalf("run: %+v %v", res, err)
	}

	_, name, res, err = ParseGitHubURL(cfg, "https://github.com/cli/cli")
	if err != nil || name != "cli/cli" || res.Kind != "" {
		t.Fatalf("repo: name=%s res=%+v err=%v", name, res, err)
	}

	if _, _, _, err := ParseGitHubURL(cfg, "https://gitlab.com/group/proj/-/merge_requests/1"); err == nil {
		t.Fatal("expected unknown host error")
	}
}

func TestYouTrackURL(t *testing.T) {
	cfg := &Config{YouTrackServers: []YouTrackServer{{
		URL:      "https://youtrack.example.com",
		Projects: []string{"PROJ"},
	}}}
	got, ok := cfg.GetYouTrackURL("PROJ-15")
	if !ok || got != "https://youtrack.example.com/issue/PROJ-15" {
		t.Fatalf("got %q %v", got, ok)
	}
	if _, ok := cfg.GetYouTrackURL("OTHER-1"); ok {
		t.Fatal("unexpected match")
	}
	if !cfg.IsYouTrackURL("https://youtrack.example.com/issue/PROJ-15") {
		t.Fatal("expected youtrack url")
	}
}
