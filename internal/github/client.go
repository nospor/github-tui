package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	gh "github.com/google/go-github/v68/github"
)

const pageSize = 25

// Client wraps the GitHub REST API.
type Client struct {
	raw    *gh.Client
	http   *http.Client
	webURL string
}

// RepoInfo is a repository shown in the repo list and used as the working target.
type RepoInfo struct {
	FullName      string
	Description   string
	DefaultBranch string
	Private       bool
	HTMLURL       string
	UpdatedAt     time.Time
}

// OwnerRepo splits FullName into owner and name.
func (r *RepoInfo) OwnerRepo() (string, string, error) {
	return SplitRepo(r.FullName)
}

// PullInfo is a pull request.
type PullInfo struct {
	Number    int
	Title     string
	State     string
	Draft     bool
	Author    string
	Body      string
	HTMLURL   string
	Head      string
	Base      string
	UpdatedAt time.Time
}

// IssueInfo is an issue (pull requests are filtered out of issue lists).
type IssueInfo struct {
	Number    int
	Title     string
	State     string
	Author    string
	Body      string
	HTMLURL   string
	Comments  int
	PlusOne   int
	MinusOne  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CommentInfo is an issue or pull request conversation comment or timeline event.
type CommentInfo struct {
	Author    string
	Body      string
	System    bool
	UpdatedAt time.Time
}

// RunInfo is a GitHub Actions workflow run.
type RunInfo struct {
	ID         int64
	WorkflowID int64
	Name       string
	Status     string
	Conclusion string
	Branch     string
	Event      string
	Actor      string
	HTMLURL    string
	UpdatedAt  time.Time
}

// Badge returns the status word used by the TUI badge.
func (r *RunInfo) Badge() string {
	if r.Status != "" && r.Status != "completed" {
		switch r.Status {
		case "in_progress":
			return "running"
		case "queued", "waiting", "pending", "requested":
			return "pending"
		default:
			return r.Status
		}
	}
	switch r.Conclusion {
	case "success":
		return "success"
	case "failure":
		return "failed"
	case "cancelled":
		return "canceled"
	case "skipped":
		return "skipped"
	case "":
		return "pending"
	default:
		return r.Conclusion
	}
}

// JobInfo is one job inside a workflow run.
type JobInfo struct {
	ID         int64
	Name       string
	Status     string
	Conclusion string
	HTMLURL    string
}

// Badge returns the status word used by the TUI badge.
func (j *JobInfo) Badge() string {
	run := &RunInfo{Status: j.Status, Conclusion: j.Conclusion}
	return run.Badge()
}

// NewClient authenticates with a personal access token.
// serverURL is the web host, for example https://github.com or a GitHub Enterprise URL.
func NewClient(serverURL, token string) (*Client, error) {
	token = strings.TrimSpace(token)
	if token == "" || token == "YOUR_PERSONAL_ACCESS_TOKEN" {
		return nil, fmt.Errorf("missing token for %s", serverURL)
	}
	web, api, enterprise, err := normalizeServer(serverURL)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	raw := gh.NewClient(httpClient).WithAuthToken(token)
	if enterprise {
		raw, err = raw.WithEnterpriseURLs(api, api)
		if err != nil {
			return nil, fmt.Errorf("enterprise URL: %w", err)
		}
	}
	return &Client{raw: raw, http: httpClient, webURL: web}, nil
}

// WebURL is the browser host for this server, without a trailing slash.
func (c *Client) WebURL() string { return c.webURL }

func normalizeServer(raw string) (web, api string, enterprise bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "https://github.com"
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", false, fmt.Errorf("server URL: %w", err)
	}
	if u.Host == "" {
		return "", "", false, fmt.Errorf("server URL %q has no host", raw)
	}
	host := strings.ToLower(u.Hostname())
	if host == "github.com" || host == "www.github.com" || host == "api.github.com" {
		return "https://github.com", "https://api.github.com", false, nil
	}
	base := u.Scheme + "://" + u.Host
	path := strings.TrimRight(u.Path, "/")
	if strings.HasSuffix(path, "/api/v3") {
		return base, base + path, true, nil
	}
	return base, base + "/api/v3", true, nil
}

// SplitRepo splits "owner/repo" into its two parts.
func SplitRepo(full string) (owner, name string, err error) {
	full = strings.TrimSuffix(strings.Trim(full, "/"), ".git")
	parts := strings.Split(full, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected owner/repo, got %q", full)
	}
	return parts[0], parts[1], nil
}

// WhoAmI returns the authenticated login.
func (c *Client) WhoAmI() (string, error) {
	user, _, err := c.raw.Users.Get(context.Background(), "")
	if err != nil {
		return "", apiErr("whoami", err)
	}
	return user.GetLogin(), nil
}

// GetRepo loads one repository by "owner/repo".
func (c *Client) GetRepo(full string) (*RepoInfo, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	repo, _, err := c.raw.Repositories.Get(context.Background(), owner, name)
	if err != nil {
		return nil, apiErr("get repository "+full, err)
	}
	return mapRepo(repo), nil
}

// ListRepos lists repositories the token can access, newest activity first.
func (c *Client) ListRepos(page int) ([]*RepoInfo, bool, error) {
	if page < 1 {
		page = 1
	}
	opts := &gh.RepositoryListByAuthenticatedUserOptions{
		Visibility:  "all",
		Affiliation: "owner,collaborator,organization_member",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{Page: page, PerPage: 50},
	}
	repos, resp, err := c.raw.Repositories.ListByAuthenticatedUser(context.Background(), opts)
	if err != nil {
		return nil, false, apiErr("list repositories", err)
	}
	out := make([]*RepoInfo, 0, len(repos))
	for _, repo := range repos {
		out = append(out, mapRepo(repo))
	}
	return out, hasNext(resp), nil
}

// ListPulls lists pull requests. state is open, closed, or all.
func (c *Client) ListPulls(full, state string, page int) ([]*PullInfo, bool, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, false, err
	}
	if page < 1 {
		page = 1
	}
	if state == "" {
		state = "open"
	}
	opts := &gh.PullRequestListOptions{
		State:     state,
		Sort:      "updated",
		Direction: "desc",
		ListOptions: gh.ListOptions{
			Page:    page,
			PerPage: pageSize,
		},
	}
	prs, resp, err := c.raw.PullRequests.List(context.Background(), owner, name, opts)
	if err != nil {
		return nil, false, apiErr("list pull requests", err)
	}
	out := make([]*PullInfo, 0, len(prs))
	for _, pr := range prs {
		out = append(out, mapPull(pr))
	}
	return out, hasNext(resp), nil
}

// GetPull loads a pull request and its conversation comments.
func (c *Client) GetPull(full string, number int) (*PullInfo, []*CommentInfo, bool, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, nil, false, err
	}
	pr, _, err := c.raw.PullRequests.Get(context.Background(), owner, name, number)
	if err != nil {
		return nil, nil, false, apiErr(fmt.Sprintf("get pull request #%d", number), err)
	}
	comments, truncated, cerr := c.listComments(owner, name, number)
	if cerr != nil {
		return mapPull(pr), nil, false, cerr
	}
	return mapPull(pr), comments, truncated, nil
}

// CreatePull opens a pull request. head is a branch name, or "user:branch" for a fork.
func (c *Client) CreatePull(full, head, base, title, body string) (*PullInfo, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	pr, _, err := c.raw.PullRequests.Create(context.Background(), owner, name, &gh.NewPullRequest{
		Title: gh.Ptr(title),
		Head:  gh.Ptr(head),
		Base:  gh.Ptr(base),
		Body:  gh.Ptr(body),
	})
	if err != nil {
		return nil, apiErr("create pull request", err)
	}
	return mapPull(pr), nil
}

// MergePull merges with a merge commit.
func (c *Client) MergePull(full string, number int) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	_, _, err = c.raw.PullRequests.Merge(context.Background(), owner, name, number, "", &gh.PullRequestOptions{
		MergeMethod: "merge",
	})
	return apiErr(fmt.Sprintf("merge pull request #%d", number), err)
}

// SetPullState sets a pull request to "open" or "closed".
func (c *Client) SetPullState(full string, number int, state string) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	_, _, err = c.raw.PullRequests.Edit(context.Background(), owner, name, number, &gh.PullRequest{
		State: gh.Ptr(state),
	})
	return apiErr(fmt.Sprintf("%s pull request #%d", state, number), err)
}

// ListIssues lists issues. Pull requests are omitted. state is open, closed, or all.
func (c *Client) ListIssues(full, state string, page int) ([]*IssueInfo, bool, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, false, err
	}
	if page < 1 {
		page = 1
	}
	if state == "" {
		state = "open"
	}
	opts := &gh.IssueListByRepoOptions{
		State:     state,
		Sort:      "updated",
		Direction: "desc",
		ListOptions: gh.ListOptions{
			Page:    page,
			PerPage: pageSize,
		},
	}
	issues, resp, err := c.raw.Issues.ListByRepo(context.Background(), owner, name, opts)
	if err != nil {
		return nil, false, apiErr("list issues", err)
	}
	out := make([]*IssueInfo, 0, len(issues))
	for _, issue := range issues {
		if issue.IsPullRequest() {
			continue
		}
		out = append(out, mapIssue(issue))
	}
	return out, hasNext(resp), nil
}

// GetIssue loads an issue and its comments.
func (c *Client) GetIssue(full string, number int) (*IssueInfo, []*CommentInfo, bool, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, nil, false, err
	}
	issue, _, err := c.raw.Issues.Get(context.Background(), owner, name, number)
	if err != nil {
		return nil, nil, false, apiErr(fmt.Sprintf("get issue #%d", number), err)
	}
	if issue.IsPullRequest() {
		return nil, nil, false, fmt.Errorf("#%d is a pull request", number)
	}
	comments, truncated, cerr := c.listIssueActivity(owner, name, number)
	if cerr != nil {
		return mapIssue(issue), nil, false, cerr
	}
	return mapIssue(issue), comments, truncated, nil
}

// CreateIssue opens an issue.
func (c *Client) CreateIssue(full, title, body string) (*IssueInfo, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	issue, _, err := c.raw.Issues.Create(context.Background(), owner, name, &gh.IssueRequest{
		Title: gh.Ptr(title),
		Body:  gh.Ptr(body),
	})
	if err != nil {
		return nil, apiErr("create issue", err)
	}
	return mapIssue(issue), nil
}

// ToggleIssueVote adds or removes a +1/-1 reaction on an issue.
// content must be "+1" or "-1". If the authenticated user already has that
// reaction, it is deleted and the function returns false. Otherwise it is
// created and the function returns true.
func (c *Client) ToggleIssueVote(full string, number int, content, username string) (added bool, err error) {
	if content != "+1" && content != "-1" {
		return false, fmt.Errorf("unsupported vote %q", content)
	}
	if username == "" {
		return false, fmt.Errorf("username is required to toggle a vote")
	}
	owner, name, err := SplitRepo(full)
	if err != nil {
		return false, err
	}
	opts := &gh.ListOptions{PerPage: 100}
	for {
		reactions, resp, err := c.raw.Reactions.ListIssueReactions(context.Background(), owner, name, number, opts)
		if err != nil {
			return false, apiErr(fmt.Sprintf("list reactions on issue #%d", number), err)
		}
		if id := userReactionID(reactions, content, username); id != 0 {
			_, err = c.raw.Reactions.DeleteIssueReaction(context.Background(), owner, name, number, id)
			return false, apiErr(fmt.Sprintf("remove reaction on issue #%d", number), err)
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	_, _, err = c.raw.Reactions.CreateIssueReaction(context.Background(), owner, name, number, content)
	if err != nil {
		return false, apiErr(fmt.Sprintf("add reaction on issue #%d", number), err)
	}
	return true, nil
}

func userReactionID(reactions []*gh.Reaction, content, username string) int64 {
	for _, r := range reactions {
		if r == nil {
			continue
		}
		if r.GetContent() == content && strings.EqualFold(login(r.User), username) {
			return r.GetID()
		}
	}
	return 0
}

// SetIssueState sets an issue to "open" or "closed".
func (c *Client) SetIssueState(full string, number int, state string) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	_, _, err = c.raw.Issues.Edit(context.Background(), owner, name, number, &gh.IssueRequest{
		State: gh.Ptr(state),
	})
	return apiErr(fmt.Sprintf("%s issue #%d", state, number), err)
}

// Comment posts a conversation comment on an issue or pull request.
func (c *Client) Comment(full string, number int, body string) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	_, _, err = c.raw.Issues.CreateComment(context.Background(), owner, name, number, &gh.IssueComment{
		Body: gh.Ptr(body),
	})
	return apiErr("post comment", err)
}

// ListRuns lists Actions workflow runs, newest first.
// If workflowID is non-zero, only runs for that workflow are returned.
func (c *Client) ListRuns(full string, page int, workflowID int64) ([]*RunInfo, bool, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, false, err
	}
	if page < 1 {
		page = 1
	}
	opts := &gh.ListWorkflowRunsOptions{
		ListOptions: gh.ListOptions{Page: page, PerPage: pageSize},
	}
	var (
		result *gh.WorkflowRuns
		resp   *gh.Response
	)
	if workflowID > 0 {
		result, resp, err = c.raw.Actions.ListWorkflowRunsByID(context.Background(), owner, name, workflowID, opts)
	} else {
		result, resp, err = c.raw.Actions.ListRepositoryWorkflowRuns(context.Background(), owner, name, opts)
	}
	if err != nil {
		return nil, false, apiErr("list workflow runs", err)
	}
	out := make([]*RunInfo, 0, len(result.WorkflowRuns))
	for _, run := range result.WorkflowRuns {
		out = append(out, mapRun(run))
	}
	return out, hasNext(resp), nil
}

// GetRun loads a workflow run and its latest jobs.
func (c *Client) GetRun(full string, id int64) (*RunInfo, []*JobInfo, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, nil, err
	}
	run, _, err := c.raw.Actions.GetWorkflowRunByID(context.Background(), owner, name, id)
	if err != nil {
		return nil, nil, apiErr(fmt.Sprintf("get workflow run %d", id), err)
	}
	jobs, _, err := c.raw.Actions.ListWorkflowJobs(context.Background(), owner, name, id, &gh.ListWorkflowJobsOptions{
		Filter:      "latest",
		ListOptions: gh.ListOptions{PerPage: 100},
	})
	if err != nil {
		return mapRun(run), nil, apiErr("list workflow jobs", err)
	}
	out := make([]*JobInfo, 0, len(jobs.Jobs))
	for _, job := range jobs.Jobs {
		out = append(out, &JobInfo{
			ID:         job.GetID(),
			Name:       job.GetName(),
			Status:     job.GetStatus(),
			Conclusion: job.GetConclusion(),
			HTMLURL:    job.GetHTMLURL(),
		})
	}
	return mapRun(run), out, nil
}

// JobLog downloads the plain-text log of one job.
func (c *Client) JobLog(full string, jobID int64) (string, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return "", err
	}
	logURL, _, err := c.raw.Actions.GetWorkflowJobLogs(context.Background(), owner, name, jobID, 3)
	if err != nil {
		return "", apiErr("get job log", err)
	}
	if logURL == nil {
		return "", fmt.Errorf("job %d has no log URL", jobID)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, logURL.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("download job log: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("download job log: %s", resp.Status)
	}
	const maxLog = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLog+1))
	if err != nil {
		return "", fmt.Errorf("read job log: %w", err)
	}
	text := string(body)
	if len(body) > maxLog {
		text = text[:maxLog] + "\n\n… log truncated …"
	}
	return text, nil
}

// Rerun starts a workflow run again.
func (c *Client) Rerun(full string, id int64) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	_, err = c.raw.Actions.RerunWorkflowByID(context.Background(), owner, name, id)
	return apiErr(fmt.Sprintf("rerun workflow %d", id), err)
}

// CancelRun cancels a workflow run.
func (c *Client) CancelRun(full string, id int64) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	_, err = c.raw.Actions.CancelWorkflowRunByID(context.Background(), owner, name, id)
	return apiErr(fmt.Sprintf("cancel workflow %d", id), err)
}

func (c *Client) listComments(owner, name string, number int) ([]*CommentInfo, bool, error) {
	opts := &gh.IssueListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	comments, resp, err := c.raw.Issues.ListComments(context.Background(), owner, name, number, opts)
	if err != nil {
		return nil, false, apiErr("list comments", err)
	}
	out := make([]*CommentInfo, 0, len(comments))
	for _, comment := range comments {
		out = append(out, &CommentInfo{
			Author:    login(comment.User),
			Body:      comment.GetBody(),
			UpdatedAt: comment.GetUpdatedAt().Time,
		})
	}
	return out, hasNext(resp), nil
}

func (c *Client) listIssueActivity(owner, name string, number int) ([]*CommentInfo, bool, error) {
	opts := &gh.ListOptions{PerPage: 100}
	events, resp, err := c.raw.Issues.ListIssueTimeline(context.Background(), owner, name, number, opts)
	if err != nil {
		return c.listComments(owner, name, number)
	}
	out := make([]*CommentInfo, 0, len(events))
	for _, ev := range events {
		if item := mapTimeline(ev); item != nil {
			out = append(out, item)
		}
	}
	return out, hasNext(resp), nil
}

func mapTimeline(ev *gh.Timeline) *CommentInfo {
	if ev == nil {
		return nil
	}
	author := login(ev.GetActor())
	if author == "" {
		author = login(ev.GetUser())
	}
	item := &CommentInfo{
		Author:    author,
		UpdatedAt: ev.GetCreatedAt().Time,
		System:    true,
	}
	switch ev.GetEvent() {
	case "commented":
		item.System = false
		item.Author = login(ev.GetUser())
		if item.Author == "" {
			item.Author = author
		}
		item.Body = ev.GetBody()
		return item
	case "renamed":
		from, to := "", ""
		if r := ev.GetRename(); r != nil {
			from, to = r.GetFrom(), r.GetTo()
		}
		item.Body = fmt.Sprintf("changed title from **%s** to **%s**", from, to)
		return item
	case "closed":
		item.Body = "closed this"
		return item
	case "reopened":
		item.Body = "reopened this"
		return item
	case "labeled":
		item.Body = fmt.Sprintf("added label **%s**", labelName(ev))
		return item
	case "unlabeled":
		item.Body = fmt.Sprintf("removed label **%s**", labelName(ev))
		return item
	case "assigned":
		who := login(ev.GetAssignee())
		if who == "" {
			who = author
		}
		item.Body = fmt.Sprintf("assigned **%s**", who)
		return item
	case "unassigned":
		who := login(ev.GetAssignee())
		if who == "" {
			who = author
		}
		item.Body = fmt.Sprintf("unassigned **%s**", who)
		return item
	case "referenced", "cross-referenced":
		item.Body = "referenced this"
		return item
	default:
		return nil
	}
}

func labelName(ev *gh.Timeline) string {
	if ev == nil || ev.GetLabel() == nil {
		return ""
	}
	return ev.GetLabel().GetName()
}

func mapRepo(repo *gh.Repository) *RepoInfo {
	if repo == nil {
		return &RepoInfo{}
	}
	return &RepoInfo{
		FullName:      repo.GetFullName(),
		Description:   repo.GetDescription(),
		DefaultBranch: repo.GetDefaultBranch(),
		Private:       repo.GetPrivate(),
		HTMLURL:       repo.GetHTMLURL(),
		UpdatedAt:     repo.GetUpdatedAt().Time,
	}
}

func mapPull(pr *gh.PullRequest) *PullInfo {
	if pr == nil {
		return &PullInfo{}
	}
	state := pr.GetState()
	if pr.GetMerged() || !pr.GetMergedAt().IsZero() {
		state = "merged"
	}
	head, base := "", ""
	if pr.Head != nil {
		head = pr.Head.GetRef()
	}
	if pr.Base != nil {
		base = pr.Base.GetRef()
	}
	return &PullInfo{
		Number:    pr.GetNumber(),
		Title:     pr.GetTitle(),
		State:     state,
		Draft:     pr.GetDraft(),
		Author:    login(pr.User),
		Body:      pr.GetBody(),
		HTMLURL:   pr.GetHTMLURL(),
		Head:      head,
		Base:      base,
		UpdatedAt: pr.GetUpdatedAt().Time,
	}
}

func mapIssue(issue *gh.Issue) *IssueInfo {
	if issue == nil {
		return &IssueInfo{}
	}
	plus, minus := 0, 0
	if r := issue.GetReactions(); r != nil {
		plus, minus = r.GetPlusOne(), r.GetMinusOne()
	}
	return &IssueInfo{
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		State:     issue.GetState(),
		Author:    login(issue.User),
		Body:      issue.GetBody(),
		HTMLURL:   issue.GetHTMLURL(),
		Comments:  issue.GetComments(),
		PlusOne:   plus,
		MinusOne:  minus,
		CreatedAt: issue.GetCreatedAt().Time,
		UpdatedAt: issue.GetUpdatedAt().Time,
	}
}

func mapRun(run *gh.WorkflowRun) *RunInfo {
	if run == nil {
		return &RunInfo{}
	}
	name := run.GetName()
	if name == "" {
		name = run.GetDisplayTitle()
	}
	actor := login(run.GetTriggeringActor())
	if actor == "" {
		actor = login(run.GetActor())
	}
	return &RunInfo{
		ID:         run.GetID(),
		WorkflowID: run.GetWorkflowID(),
		Name:       name,
		Status:     run.GetStatus(),
		Conclusion: run.GetConclusion(),
		Branch:     run.GetHeadBranch(),
		Event:      run.GetEvent(),
		Actor:      actor,
		HTMLURL:    run.GetHTMLURL(),
		UpdatedAt:  run.GetUpdatedAt().Time,
	}
}

func login(user *gh.User) string {
	if user == nil {
		return ""
	}
	return user.GetLogin()
}

func hasNext(resp *gh.Response) bool {
	return resp != nil && resp.NextPage > 0
}

func isNotFound(err error) bool {
	var ge *gh.ErrorResponse
	if errors.As(err, &ge) && ge.Response != nil {
		return ge.Response.StatusCode == http.StatusNotFound
	}
	return false
}

func apiErr(action string, err error) error {
	if err == nil {
		return nil
	}
	var rate *gh.RateLimitError
	if errors.As(err, &rate) {
		return fmt.Errorf("%s: rate limited until %s", action, rate.Rate.Reset.Time.Format(time.RFC3339))
	}
	var ge *gh.ErrorResponse
	if errors.As(err, &ge) {
		msg := strings.TrimSpace(ge.Message)
		if msg == "" && ge.Response != nil {
			msg = ge.Response.Status
		}
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s: %s", action, msg)
	}
	return fmt.Errorf("%s: %w", action, err)
}
