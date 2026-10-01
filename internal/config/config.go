package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const placeholderToken = "YOUR_PERSONAL_ACCESS_TOKEN"

// Server is one GitHub host (github.com or GitHub Enterprise).
type Server struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Token   string `json:"token"`
	Default bool   `json:"default,omitempty"`
}

// YouTrackServer resolves issue keys such as PROJ-123 to YouTrack URLs.
type YouTrackServer struct {
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Projects []string `json:"projects"`
}

// Config is the on-disk configuration.
type Config struct {
	Servers         []Server         `json:"servers"`
	YouTrackServers []YouTrackServer `json:"youtrack_servers,omitempty"`
	BrowserCommand  string           `json:"browser_command,omitempty"`
	YouTrackCommand string           `json:"youtrack_command,omitempty"`
	Theme           string           `json:"theme,omitempty"`
}

const defaultBrowserCommand = "xdg-open"

// ConfigPath returns ~/.config/github-tui/config.json.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "github-tui", "config.json")
}

// Load reads config from disk. A missing file yields an empty config.
func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	dirty := false
	if cfg.BrowserCommand == "" {
		cfg.BrowserCommand = defaultBrowserCommand
		dirty = true
	}
	if cfg.Theme == "" {
		cfg.Theme = "catppuccin"
		dirty = true
	}
	if dirty {
		if err := Save(&cfg); err != nil {
			return nil, fmt.Errorf("saving defaults: %w", err)
		}
	}
	return &cfg, nil
}

// Save writes config to disk with permissions that keep the token private.
func Save(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// EnsureDefaultConfig creates a sample config when none exists.
func EnsureDefaultConfig() error {
	path := ConfigPath()
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	sample := Config{
		Servers: []Server{{
			Name:    "github.com",
			URL:     "https://github.com",
			Token:   placeholderToken,
			Default: true,
		}},
		BrowserCommand: defaultBrowserCommand,
		Theme:          "catppuccin",
	}
	return Save(&sample)
}

// ResolvedToken returns the configured token, or GITHUB_TOKEN / GH_TOKEN
// when the config still has the sample placeholder.
func (s Server) ResolvedToken() string {
	token := strings.TrimSpace(s.Token)
	if token != "" && token != placeholderToken {
		return token
	}
	if env := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); env != "" {
		return env
	}
	if env := strings.TrimSpace(os.Getenv("GH_TOKEN")); env != "" {
		return env
	}
	return token
}

// DetectedRepo is a repository matched from a git remote.
type DetectedRepo struct {
	ServerName string
	ServerURL  string
	FullName   string // owner/repo
}

// DetectFromGit matches the current directory's git remotes to a configured server.
// The second return value is the raw remote URL that was considered, for error text.
func DetectFromGit(cfg *Config) (*DetectedRepo, string) {
	out, err := exec.Command("git", "remote").Output()
	if err != nil {
		return nil, ""
	}
	remotes := strings.Fields(strings.TrimSpace(string(out)))
	sort.SliceStable(remotes, func(i, _ int) bool { return remotes[i] == "origin" })

	var last string
	for _, name := range remotes {
		urlOut, err := exec.Command("git", "remote", "get-url", name).Output()
		if err != nil {
			continue
		}
		raw := strings.TrimSpace(string(urlOut))
		if raw == "" {
			continue
		}
		last = raw
		if repo := matchRemote(raw, cfg); repo != nil {
			return repo, raw
		}
	}
	return nil, last
}

func matchRemote(remote string, cfg *Config) *DetectedRepo {
	parsed, _ := url.Parse(remote)
	for _, srv := range cfg.Servers {
		host := serverHostname(srv.URL)
		if host == "" {
			continue
		}
		var projectPath string

		if parsed != nil && parsed.Scheme == "ssh" {
			if strings.EqualFold(parsed.Hostname(), host) {
				projectPath = strings.TrimPrefix(strings.TrimSuffix(parsed.Path, ".git"), "/")
			}
		}
		if strings.Contains(remote, "@") && strings.Contains(remote, ":") && !strings.Contains(remote, "://") {
			parts := strings.SplitN(remote, "@", 2)
			if len(parts) == 2 {
				hostAndPath := parts[1]
				colon := strings.Index(hostAndPath, ":")
				if colon >= 0 {
					remoteHost := hostAndPath[:colon]
					path := strings.TrimSuffix(hostAndPath[colon+1:], ".git")
					if strings.EqualFold(remoteHost, host) {
						projectPath = path
					}
				}
			}
		}
		if parsed != nil && (parsed.Scheme == "https" || parsed.Scheme == "http") {
			h := parsed.Hostname()
			if strings.EqualFold(h, host) {
				projectPath = strings.TrimPrefix(strings.TrimSuffix(parsed.Path, ".git"), "/")
			}
		}
		if projectPath != "" {
			return &DetectedRepo{
				ServerName: srv.Name,
				ServerURL:  srv.URL,
				FullName:   strings.Trim(projectPath, "/"),
			}
		}
	}
	return nil
}

func serverHostname(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// Resource is a pull request, issue, or Actions run referenced by a GitHub URL.
type Resource struct {
	Kind   string // "", "pr", "issue", "run"
	Number int64
}

// ParseGitHubURL matches a GitHub web URL to a configured server.
// A repository URL has an empty resource kind.
func ParseGitHubURL(cfg *Config, rawURL string) (serverIdx int, fullName string, res Resource, err error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return -1, "", Resource{}, fmt.Errorf("parsing URL: %w", err)
	}

	serverIdx = -1
	host := strings.ToLower(u.Hostname())
	for i, s := range cfg.Servers {
		if serverHostname(s.URL) == host {
			serverIdx = i
			break
		}
	}
	if serverIdx == -1 {
		return -1, "", Resource{}, fmt.Errorf("no configured server matches host %q", u.Host)
	}

	parts := splitPath(u.Path)
	if len(parts) < 2 {
		return -1, "", Resource{}, fmt.Errorf("URL %q has no owner/repo", rawURL)
	}
	fullName = parts[0] + "/" + parts[1]
	rest := parts[2:]
	if len(rest) == 0 {
		return serverIdx, fullName, Resource{}, nil
	}

	switch rest[0] {
	case "pull":
		n, nerr := pathNumber(rest, 1)
		if nerr != nil {
			return -1, "", Resource{}, nerr
		}
		return serverIdx, fullName, Resource{Kind: "pr", Number: n}, nil
	case "issues":
		n, nerr := pathNumber(rest, 1)
		if nerr != nil {
			return -1, "", Resource{}, nerr
		}
		return serverIdx, fullName, Resource{Kind: "issue", Number: n}, nil
	case "actions":
		if len(rest) >= 3 && rest[1] == "runs" {
			n, nerr := pathNumber(rest, 2)
			if nerr != nil {
				return -1, "", Resource{}, nerr
			}
			return serverIdx, fullName, Resource{Kind: "run", Number: n}, nil
		}
	}
	return -1, "", Resource{}, fmt.Errorf("URL is not a repository, pull request, issue, or Actions run")
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func pathNumber(parts []string, idx int) (int64, error) {
	if idx >= len(parts) {
		return 0, fmt.Errorf("URL is missing a numeric id")
	}
	part := parts[idx]
	if q := strings.IndexAny(part, "?#"); q >= 0 {
		part = part[:q]
	}
	n, err := strconv.ParseInt(part, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q", parts[idx])
	}
	return n, nil
}

// GetYouTrackURL returns a YouTrack issue URL when the key's project is configured.
func (c *Config) GetYouTrackURL(issueKey string) (string, bool) {
	if c == nil {
		return "", false
	}
	parts := strings.Split(issueKey, "-")
	if len(parts) != 2 || parts[1] == "" {
		return "", false
	}
	for _, ch := range parts[1] {
		if ch < '0' || ch > '9' {
			return "", false
		}
	}
	proj := strings.ToUpper(parts[0])
	for _, srv := range c.YouTrackServers {
		for _, p := range srv.Projects {
			if strings.EqualFold(p, proj) {
				base := strings.TrimRight(srv.URL, "/")
				return fmt.Sprintf("%s/issue/%s-%s", base, proj, parts[1]), true
			}
		}
	}
	return "", false
}

// IsYouTrackURL reports whether rawURL belongs to a configured YouTrack server.
func (c *Config) IsYouTrackURL(rawURL string) bool {
	if c == nil {
		return false
	}
	rawURL = strings.ToLower(rawURL)
	for _, srv := range c.YouTrackServers {
		base := strings.ToLower(strings.TrimRight(srv.URL, "/"))
		if base != "" && (rawURL == base || strings.HasPrefix(rawURL, base+"/")) {
			return true
		}
	}
	return false
}
