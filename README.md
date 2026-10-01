# github-tui

A terminal UI for GitHub pull requests, issues, and Actions. It talks to the GitHub REST API with a personal access token. No GitHub App is required.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), in the same shape as [gitlab-tui](https://github.com/nospor/gitlab-tui).

## Features

- **Pull requests** — list, filter (`open` / `closed` / `all`), read the description and comments, comment, open, merge, close, reopen, and create
- **Branches** — list, delete, open a branch to browse commits with diff pane (`Tab`), compare two branches (`C`), create a pull request from a branch (`c`)
- **Tags** — list, create (name, ref branch, message), edit GitHub release notes (`e` in tag detail), delete, browse commits with diff pane
- **Issues** — list, filter, read, comment, create, close, reopen, and create a git branch for an issue (`b`)
- **Actions** — list workflow runs, inspect jobs, read job logs, re-run, and cancel
- **Repositories** — browse repos the token can access, search the list as you type, and switch the working repo
- **Servers** — github.com and GitHub Enterprise, selected from config
- **Auto-detect** — picks the server and `owner/repo` from the current directory's git remote
- **Open and yank** — `o` opens the GitHub page in a browser, `y` copies the URL
- **YouTrack** — optional issue-key links (`PROJ-123`) in descriptions and comments
- **Themes** — `catppuccin` (default) and `teams`

Pull request and issue detail views do not include an inline diff viewer (branch/tag commit views do). There is no container registry tab. GitHub fine-grained tokens also cannot call the Packages API.

## Install

```bash
make build
./github-tui

# or
make install PREFIX=$HOME/.local
```

## Configuration

On first run a sample file is created at `~/.config/github-tui/config.json`:

```json
{
  "servers": [
    {
      "name": "github.com",
      "url": "https://github.com",
      "token": "YOUR_PERSONAL_ACCESS_TOKEN",
      "default": true
    }
  ],
  "youtrack_servers": [
    {
      "name": "company-youtrack",
      "url": "https://youtrack.company.com",
      "projects": ["PROJ"]
    }
  ],
  "browser_command": "xdg-open",
  "youtrack_command": "yt-tui",
  "theme": "catppuccin"
}
```

Put the token you already created in `token`. If the file still contains `YOUR_PERSONAL_ACCESS_TOKEN`, the app uses `GITHUB_TOKEN` or `GH_TOKEN` from the environment.

For a fine-grained token, grant **Read and write** on Contents, Pull requests, Issues, and Actions, and include the repositories you want to manage. A classic token with the `repo` scope covers the same features.

GitHub Enterprise uses the same file with the server's web URL, for example `https://ghe.company.com`.

## Usage

```bash
github-tui
github-tui https://github.com/owner/repo
github-tui https://github.com/owner/repo/pull/123
github-tui https://github.com/owner/repo/issues/9
github-tui https://github.com/owner/repo/actions/runs/42
```

From inside a clone, the `origin` remote selects the repository when its host matches a configured server.

### Keys

| Key | Action |
| --- | --- |
| `1`–`6` / `Tab` | Pull requests, Branches, Tags, Actions, Issues, Repos |
| `j` / `k` | Move |
| `Enter` | Open the item, or select the highlighted repository |
| `s` | Cycle open / closed / all |
| `c` | Create a pull request or issue. On Branches, open create PR with head prefilled. On Tags, create a tag. On Actions, cancel the run |
| `C` | On Branches (list), compare with another branch |
| `d` | On Branches or Tags (list), delete the selected branch or tag |
| `b` | On Issues (list or detail), create a git branch for the issue |
| `m` | Merge the pull request |
| `x` / `O` | Close / reopen |
| `R` | Re-run a workflow |
| `C` | Comment on the open pull request or issue |
| `r` | Refresh |
| `n` / `p` | Next / previous page |
| Repos tab | Type to search (letters including j/k), `↑`/`↓` to move, `Enter` to use the repo and open Pull requests, `Esc` to clear, `PgUp`/`PgDn` to page |
| `o` | Open in the browser |
| `y` | Copy the URL |
| `S` | Switch server |
| `q` | Quit |

Create forms use `Tab` to move between fields and `Ctrl+S` to save. In a pull request, **Head** is the branch name, or `user:branch` when the branch is on a fork. **Base** starts as the repository's default branch.

## Project layout

```
cmd/github-tui/main.go
internal/config/     config file, git remote detection, URL parsing
internal/github/     GitHub REST client
internal/tui/        Bubble Tea model and views
```
