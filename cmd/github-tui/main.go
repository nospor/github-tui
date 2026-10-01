package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github-tui/internal/config"
	gh "github-tui/internal/github"
	"github-tui/internal/tui"
)

func main() {
	if err := config.EnsureDefaultConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create default config: %v\n", err)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.Servers) == 0 {
		fmt.Fprintln(os.Stderr, "No servers configured.")
		fmt.Fprintf(os.Stderr, "Edit %s to add a GitHub server and token.\n", config.ConfigPath())
		os.Exit(1)
	}

	serverIdx := 0
	for i, s := range cfg.Servers {
		if s.Default {
			serverIdx = i
			break
		}
	}

	var fullName string
	var openKind string
	var openNumber int64
	var startupWarn string
	var detectedRemote string

	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--help" || arg == "-h" || arg == "help" {
			printHelp()
			os.Exit(0)
		}
		if arg[0] == '-' {
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", arg)
			printHelp()
			os.Exit(1)
		}
		var res config.Resource
		serverIdx, fullName, res, err = config.ParseGitHubURL(cfg, arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		openKind = res.Kind
		openNumber = res.Number
	} else {
		var detected *config.DetectedRepo
		detected, detectedRemote = config.DetectFromGit(cfg)
		if detected != nil {
			for i, s := range cfg.Servers {
				if s.Name == detected.ServerName || s.URL == detected.ServerURL {
					serverIdx = i
					break
				}
			}
			fullName = detected.FullName
		}
	}

	srv := cfg.Servers[serverIdx]
	client, err := gh.NewClient(srv.URL, srv.ResolvedToken())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		fmt.Fprintf(os.Stderr, "Set the token in %s, or export GITHUB_TOKEN.\n", config.ConfigPath())
		os.Exit(1)
	}

	var repo *gh.RepoInfo
	if fullName != "" {
		repo, err = client.GetRepo(fullName)
		if err != nil {
			if openNumber > 0 {
				fmt.Fprintf(os.Stderr, "Error: could not load %s: %v\n", fullName, err)
				os.Exit(1)
			}
			startupWarn = fmt.Sprintf("Detected %q (%s) but could not load it: %v", detectedRemote, fullName, err)
			repo = nil
		}
	} else if detectedRemote != "" {
		startupWarn = fmt.Sprintf("Git remote %q did not match a configured GitHub server", detectedRemote)
	}

	tui.InitTheme(cfg.Theme)
	model := tui.New(cfg, serverIdx, client, repo, startupWarn, openKind, openNumber)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("github-tui is a terminal UI for GitHub pull requests, issues, and Actions.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  github-tui")
	fmt.Println("  github-tui <repository, pull request, issue, or Actions URL>")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  github-tui")
	fmt.Println("  github-tui https://github.com/cli/cli")
	fmt.Println("  github-tui https://github.com/cli/cli/pull/123")
	fmt.Println("  github-tui https://github.com/cli/cli/issues/9")
	fmt.Println("  github-tui https://github.com/cli/cli/actions/runs/42")
	fmt.Println()
	fmt.Println("Config:")
	fmt.Printf("  %s\n", config.ConfigPath())
}
