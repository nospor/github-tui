package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github-tui/internal/config"
	gh "github-tui/internal/github"
)

type branchDetailView int

const (
	branchViewCommits branchDetailView = iota
	branchViewCompare
)

type tabID int

const (
	tabPRs tabID = iota
	tabBranches
	tabTags
	tabActions
	tabIssues
	tabRepos
	tabCount
)

var tabLabels = [tabCount]string{
	"1 Pull requests",
	"2 Branches",
	"3 Tags",
	"4 Actions",
	"5 Issues",
	"6 Repos",
}

type appState int

const (
	stateMain appState = iota
	stateDetail
	stateJobLog
	stateConfirm
	stateComment
	stateCreate
	stateServerSelect
	stateLinkSelect
	stateCompareBranchSelect
	stateCreateTag
	stateEditTag
	stateCreateIssueBranch
)

type (
	bootMsg   struct{}
	whoAmIMsg struct{ name string }
	reposMsg  struct {
		items   []*gh.RepoInfo
		hasNext bool
		page    int
	}
	pullsMsg struct {
		items   []*gh.PullInfo
		hasNext bool
		page    int
	}
	issuesMsg struct {
		items   []*gh.IssueInfo
		hasNext bool
		page    int
	}
	runsMsg struct {
		items   []*gh.RunInfo
		hasNext bool
		page    int
	}
	pullDetailMsg struct {
		item       *gh.PullInfo
		comments   []*gh.CommentInfo
		truncated  bool
		commentErr string
		files      []*gh.DiffFile
		diffErr    string
	}
	issueDetailMsg struct {
		item       *gh.IssueInfo
		comments   []*gh.CommentInfo
		truncated  bool
		commentErr string
	}
	runDetailMsg struct {
		item   *gh.RunInfo
		jobs   []*gh.JobInfo
		jobErr string
	}
	logMsg struct {
		name string
		text string
	}
	doneMsg struct {
		text         string
		reloadList   bool
		reloadDetail bool
	}
	errMsg         struct{ err error }
	statusClearMsg struct{ id int }
	serverReadyMsg struct {
		idx    int
		client *gh.Client
	}
	branchesLoadedMsg struct {
		branches []string
	}
	branchCommitsLoadedMsg struct {
		branch  string
		commits []*gh.CommitInfo
	}
	branchCompareLoadedMsg struct {
		targetBranch string
		sourceBranch string
		compare      *gh.CompareInfo
	}
	commitDiffsLoadedMsg struct {
		sha   string
		files []*gh.DiffFile
	}
	tagsLoadedMsg struct {
		tags []*gh.TagInfo
	}
	tagCommitsLoadedMsg struct {
		tag     string
		commits []*gh.CommitInfo
	}
)

type linkItem struct {
	Label string
	URL   string
}

// Model is the Bubble Tea root model.
type Model struct {
	width, height int
	cfg           *config.Config
	serverIdx     int
	client        *gh.Client
	repo          *gh.RepoInfo
	username      string

	tab   tabID
	state appState

	inflight int
	loading  bool
	status   string
	statusID int

	openKind   string
	openNumber int64

	prState   string
	prPage    int
	prHasNext bool
	prs       []*gh.PullInfo
	prCursor  int
	prOffset  int

	issueState   string
	issuePage    int
	issueHasNext bool
	issues       []*gh.IssueInfo
	issueCursor  int
	issueOffset  int

	runPage    int
	runHasNext bool
	runs       []*gh.RunInfo
	runCursor  int
	runOffset  int

	repoPage    int
	repoHasNext bool
	repos       []*gh.RepoInfo
	repoCursor  int
	repoOffset  int
	repoInput   textinput.Model

	branches                     []string
	branchCursor                 int
	branchOffset                 int
	branchDetailView             branchDetailView
	branchDetailName             string
	branchCommits                []*gh.CommitInfo
	branchCommitCursor           int
	branchCompare                *gh.CompareInfo
	branchCompareTarget          string
	branchCompareCursor          int
	compareSelectCursor          int
	branchCommitDiffFiles        []*gh.DiffFile
	branchCommitDiffFileIdx      int
	branchCommitDiffLineCursor   int
	branchCommitDiffScrollOffset int
	branchCommitDiffPanelOpen    bool
	branchCommitDiffLoading      bool
	branchCommitDiffSHA          string

	tags                      []*gh.TagInfo
	tagCursor                 int
	tagOffset                 int
	tagDetailName             string
	tagCommits                []*gh.CommitInfo
	tagCommitCursor           int
	tagCommitDiffFiles        []*gh.DiffFile
	tagCommitDiffFileIdx      int
	tagCommitDiffLineCursor   int
	tagCommitDiffScrollOffset int
	tagCommitDiffPanelOpen    bool
	tagCommitDiffLoading      bool
	tagCommitDiffSHA          string

	createTagName         textinput.Model
	createTagMessage      textarea.Model
	createTagBranchCursor int
	createTagField        int

	editTagName        string
	editTagDescription textarea.Model

	createIssueBranchIssue *gh.IssueInfo
	createIssueBranchName  textinput.Model
	createIssueBranchRef   textinput.Model
	createIssueBranchField int

	detailPR           *gh.PullInfo
	prDiffFiles        []*gh.DiffFile
	prDiffFileIdx      int
	prDiffLineCursor   int
	prDiffScrollOffset int
	prDiffPanelOpen    bool
	detailIssue        *gh.IssueInfo
	detailRun          *gh.RunInfo
	comments           []*gh.CommentInfo
	commentNote        string
	jobs               []*gh.JobInfo
	jobCursor          int
	jobOffset          int
	detailScroll       int
	detailLines        []string
	logName            string
	logLines           []string
	logScroll          int

	formKind   string
	formFocus  int
	titleInput textinput.Model
	headInput  textinput.Model
	baseInput  textinput.Model
	bodyInput  textarea.Model

	confirmMsg   string
	confirmCmd   tea.Cmd
	returnState  appState
	links        []linkItem
	linkCursor   int
	serverCursor int
}

// New builds the initial model. repo may be nil when nothing was auto-detected.
func New(cfg *config.Config, serverIdx int, client *gh.Client, repo *gh.RepoInfo, startupWarn string, openKind string, openNumber int64) Model {
	title := textinput.New()
	title.Placeholder = "Title"
	title.CharLimit = 256

	head := textinput.New()
	head.Placeholder = "Head branch (or user:branch)"
	head.CharLimit = 256

	base := textinput.New()
	base.Placeholder = "Base branch"
	base.CharLimit = 256

	body := textarea.New()
	body.Placeholder = "Description"
	body.CharLimit = 0
	body.ShowLineNumbers = false

	filter := textinput.New()
	filter.Placeholder = "Search repositories"
	filter.Prompt = ""
	filter.CharLimit = 128

	tagName := textinput.New()
	tagName.Placeholder = "Tag name"
	tagName.CharLimit = 128

	tagMsg := textarea.New()
	tagMsg.Placeholder = "Tag message / release notes"
	tagMsg.ShowLineNumbers = false

	editTagDesc := textarea.New()
	editTagDesc.ShowLineNumbers = false

	issueBranchName := textinput.New()
	issueBranchName.CharLimit = 256
	issueBranchRef := textinput.New()
	issueBranchRef.CharLimit = 256

	m := Model{
		cfg:                   cfg,
		serverIdx:             serverIdx,
		client:                client,
		repo:                  repo,
		status:                startupWarn,
		openKind:              openKind,
		openNumber:            openNumber,
		prState:               "open",
		prPage:                1,
		issueState:            "open",
		issuePage:             1,
		runPage:               1,
		repoPage:              1,
		titleInput:            title,
		headInput:             head,
		baseInput:             base,
		bodyInput:             body,
		repoInput:             filter,
		createTagName:         tagName,
		createTagMessage:      tagMsg,
		editTagDescription:    editTagDesc,
		createIssueBranchName: issueBranchName,
		createIssueBranchRef:  issueBranchRef,
	}
	if repo == nil {
		m.tab = tabRepos
	}
	switch openKind {
	case "pr":
		m.tab = tabPRs
	case "issue":
		m.tab = tabIssues
	case "run":
		m.tab = tabActions
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return bootMsg{} }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layoutInputs()
		m.rebuildDetail()
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	case bootMsg:
		cmds := []tea.Cmd{m.cmdWhoAmI(), m.cmdRepos()}
		if m.repo != nil {
			cmds = append(cmds, m.reloadCurrent())
			if m.openKind != "" && m.openNumber > 0 {
				cmds = append(cmds, m.cmdOpenInitial())
			}
		}
		nm, cmd := m.track(cmds...)
		if nm.tab == tabRepos {
			return nm, tea.Batch(cmd, nm.repoInput.Focus())
		}
		return nm, cmd
	case whoAmIMsg:
		m = m.finish()
		m.username = msg.name
		return m, nil
	case reposMsg:
		m = m.finish()
		if msg.page == m.repoPage {
			m.repos = msg.items
			m.repoHasNext = msg.hasNext
			m.clampList()
		}
		return m, nil
	case pullsMsg:
		m = m.finish()
		if msg.page == m.prPage {
			m.prs = msg.items
			m.prHasNext = msg.hasNext
			m.clampList()
		}
		return m, nil
	case issuesMsg:
		m = m.finish()
		if msg.page == m.issuePage {
			m.issues = msg.items
			m.issueHasNext = msg.hasNext
			m.clampList()
		}
		return m, nil
	case runsMsg:
		m = m.finish()
		if msg.page == m.runPage {
			m.runs = msg.items
			m.runHasNext = msg.hasNext
			m.clampList()
		}
		return m, nil
	case branchesLoadedMsg:
		m = m.finish()
		m.branches = msg.branches
		if m.tab == tabBranches {
			m.clampList()
		}
		if m.state == stateCreateTag && m.repo != nil {
			for i, b := range m.branches {
				if b == m.repo.DefaultBranch {
					m.createTagBranchCursor = i
					break
				}
			}
		}
		return m, nil
	case branchCommitsLoadedMsg:
		m = m.finish()
		m.clearItemDetail()
		m.branchCommits = msg.commits
		m.branchDetailName = msg.branch
		m.branchCommitCursor = 0
		m.state = stateDetail
		return m, nil
	case branchCompareLoadedMsg:
		m = m.finish()
		m.clearItemDetail()
		m.branchCompare = msg.compare
		m.branchCompareTarget = msg.targetBranch
		m.branchDetailName = msg.sourceBranch
		m.branchCompareCursor = 0
		m.branchDetailView = branchViewCompare
		m.state = stateDetail
		return m, nil
	case commitDiffsLoadedMsg:
		m = m.finish()
		m.branchCommitDiffLoading = false
		m.tagCommitDiffLoading = false
		var currentSHA string
		if m.tab == tabTags {
			if len(m.tagCommits) > 0 && m.tagCommitCursor < len(m.tagCommits) {
				currentSHA = m.tagCommits[m.tagCommitCursor].ID
			}
			if currentSHA != "" && msg.sha == currentSHA {
				m.tagCommitDiffFiles = msg.files
				m.tagCommitDiffSHA = msg.sha
				m.tagCommitDiffFileIdx = 0
				m.tagCommitDiffLineCursor = 0
				m.tagCommitDiffScrollOffset = 0
			}
		} else if m.branchDetailView == branchViewCompare {
			if m.branchCompare != nil && m.branchCompareCursor < len(m.branchCompare.Commits) {
				currentSHA = m.branchCompare.Commits[m.branchCompareCursor].ID
			}
			if currentSHA != "" && msg.sha == currentSHA {
				m.branchCommitDiffFiles = msg.files
				m.branchCommitDiffSHA = msg.sha
				m.branchCommitDiffFileIdx = 0
				m.branchCommitDiffLineCursor = 0
				m.branchCommitDiffScrollOffset = 0
			}
		} else if len(m.branchCommits) > 0 && m.branchCommitCursor < len(m.branchCommits) {
			currentSHA = m.branchCommits[m.branchCommitCursor].ID
			if currentSHA == msg.sha {
				m.branchCommitDiffFiles = msg.files
				m.branchCommitDiffSHA = msg.sha
				m.branchCommitDiffFileIdx = 0
				m.branchCommitDiffLineCursor = 0
				m.branchCommitDiffScrollOffset = 0
			}
		}
		return m, nil
	case tagsLoadedMsg:
		m = m.finish()
		m.tags = msg.tags
		if m.tab == tabTags {
			m.clampList()
		}
		return m, nil
	case tagCommitsLoadedMsg:
		m = m.finish()
		m.clearItemDetail()
		m.tagCommits = msg.commits
		m.tagDetailName = msg.tag
		m.tagCommitCursor = 0
		m.tagCommitDiffPanelOpen = false
		m.state = stateDetail
		return m, nil
	case pullDetailMsg:
		m = m.finish()
		m.clearRefDetail()
		m.detailPR = msg.item
		m.detailIssue = nil
		m.detailRun = nil
		m.comments = msg.comments
		m.commentNote = commentNote(msg.truncated, msg.commentErr)
		m.detailScroll = 0
		m.prDiffFiles = msg.files
		m.prDiffFileIdx = 0
		m.prDiffLineCursor = 0
		m.prDiffScrollOffset = 0
		m.state = stateDetail
		m.rebuildDetail()
		if msg.diffErr != "" {
			m.setStatus(msg.diffErr)
			return m, m.scheduleClear()
		}
		return m, nil
	case issueDetailMsg:
		m = m.finish()
		m.clearRefDetail()
		m.detailIssue = msg.item
		m.detailPR = nil
		m.detailRun = nil
		m.clearPRDiff()
		m.comments = msg.comments
		m.commentNote = commentNote(msg.truncated, msg.commentErr)
		m.detailScroll = 0
		m.state = stateDetail
		m.rebuildDetail()
		return m, nil
	case runDetailMsg:
		m = m.finish()
		m.clearRefDetail()
		m.detailRun = msg.item
		m.jobs = msg.jobs
		m.detailPR = nil
		m.detailIssue = nil
		m.clearPRDiff()
		m.jobCursor = 0
		m.jobOffset = 0
		m.state = stateDetail
		if msg.jobErr != "" {
			m.setStatus(msg.jobErr)
		}
		return m, m.scheduleClear()
	case logMsg:
		m = m.finish()
		m.logName = msg.name
		m.logLines = strings.Split(strings.ReplaceAll(msg.text, "\r\n", "\n"), "\n")
		m.logScroll = 0
		m.state = stateJobLog
		return m, nil
	case doneMsg:
		m = m.finish()
		m.setStatus(msg.text)
		var loads []tea.Cmd
		if msg.reloadList {
			if cmd := m.reloadCurrent(); cmd != nil {
				loads = append(loads, cmd)
			}
		}
		if msg.reloadDetail {
			if cmd := m.reloadDetail(); cmd != nil {
				loads = append(loads, cmd)
			}
		}
		nm, cmd := m.track(loads...)
		return nm, tea.Batch(nm.scheduleClear(), cmd)
	case serverReadyMsg:
		m = m.finish()
		m.serverIdx = msg.idx
		m.client = msg.client
		m.repo = nil
		m.username = ""
		m.prs, m.issues, m.runs, m.repos = nil, nil, nil, nil
		m.branches, m.tags = nil, nil
		m.clearAllDetail()
		m.tab = tabRepos
		m.state = stateMain
		m.repoPage = 1
		m.repoCursor = 0
		m.repoInput.SetValue("")
		nm, cmd := m.track(m.cmdWhoAmI(), m.cmdRepos())
		return nm, tea.Batch(cmd, nm.repoInput.Focus())
	case errMsg:
		m = m.finish()
		if msg.err != nil {
			m.setStatus(msg.err.Error())
		}
		return m, m.scheduleClear()
	case statusClearMsg:
		if int(msg.id) == m.statusID {
			m.status = ""
		}
		return m, nil
	}
	return m, nil
}

func commentNote(truncated bool, commentErr string) string {
	switch {
	case commentErr != "":
		return commentErr
	case truncated:
		return "Showing the first 100 comments."
	default:
		return ""
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	onRepoSearch := m.state == stateMain && m.tab == tabRepos
	if m.loading && m.state != stateComment && m.state != stateCreate && !onRepoSearch {
		if msg.String() == "q" {
			return m, tea.Quit
		}
		return m, nil
	}
	if onRepoSearch {
		var focus tea.Cmd
		if !m.repoInput.Focused() {
			focus = m.repoInput.Focus()
		}
		nm, cmd := m.handleRepoSearch(msg)
		return nm, tea.Batch(focus, cmd)
	}
	switch m.state {
	case stateConfirm:
		return m.handleConfirm(msg.String())
	case stateServerSelect:
		return m.handleServerSelect(msg.String())
	case stateLinkSelect:
		return m.handleLinks(msg.String())
	case stateComment:
		return m.handleComment(msg)
	case stateCreate:
		return m.handleCreate(msg)
	case stateCreateTag:
		return m.handleCreateTagKey(msg)
	case stateEditTag:
		return m.handleEditTagKey(msg)
	case stateCreateIssueBranch:
		return m.handleCreateIssueBranchKey(msg)
	case stateCompareBranchSelect:
		return m.handleCompareBranchSelectKey(msg.String())
	case stateJobLog:
		return m.handleLog(msg.String())
	case stateDetail:
		if m.inRefDetail() {
			return m.handleRefDetail(msg.String())
		}
		return m.handleDetail(msg.String())
	default:
		return m.handleMain(msg.String())
	}
}

func (m Model) handleMain(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "tab", "right":
		m.blurRepoSearch()
		m.tab = (m.tab + 1) % tabCount
		m.clampList()
		return m.afterTabChange()
	case "shift+tab", "left":
		m.blurRepoSearch()
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.clampList()
		return m.afterTabChange()
	case "1":
		return m.jumpTab(tabPRs)
	case "2":
		return m.jumpTab(tabBranches)
	case "3":
		return m.jumpTab(tabTags)
	case "4":
		return m.jumpTab(tabActions)
	case "5":
		return m.jumpTab(tabIssues)
	case "6":
		return m.jumpTab(tabRepos)
	case "j", "down":
		m.move(1)
		return m, nil
	case "k", "up":
		m.move(-1)
		return m, nil
	case "g":
		m.setCursor(0)
		return m, nil
	case "G":
		m.setCursor(m.listLen() - 1)
		return m, nil
	case "enter":
		if m.tab == tabBranches || m.tab == tabTags {
			return m.openBranchOrTagDetail()
		}
		return m.openSelected()
	case "r":
		return m.track(m.reloadCurrent())
	case "n":
		return m.nextPage()
	case "p":
		return m.prevPage()
	case "s":
		return m.cycleState()
	case "c":
		if m.tab == tabBranches || m.tab == tabTags {
			return m.handleMainBranchTag("c")
		}
		return m.createOrCancel()
	case "R":
		return m.rerunSelected()
	case "x":
		return m.closeSelected()
	case "O":
		return m.reopenSelected()
	case "m":
		return m.mergeSelected()
	case "o":
		return m.openSelectedLink()
	case "y":
		return m.yankSelected()
	case "P":
		return m.jumpTab(tabRepos)
	case "S":
		m.returnState = stateMain
		m.state = stateServerSelect
		m.serverCursor = m.serverIdx
		return m, nil
	case "/":
		if m.tab == tabRepos {
			return m, m.repoInput.Focus()
		}
	}
	return m.handleMainBranchTag(key)
}

func (m *Model) blurRepoSearch() {
	m.repoInput.Blur()
}

func (m Model) afterTabChange() (tea.Model, tea.Cmd) {
	m.clearAllDetail()
	var cmds []tea.Cmd
	if m.tab == tabRepos {
		cmds = append(cmds, m.repoInput.Focus())
	}
	if m.needsLoad() {
		nm, cmd := m.track(m.reloadCurrent())
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if len(cmds) == 0 {
			return nm, nil
		}
		return nm, tea.Batch(cmds...)
	}
	if len(cmds) == 0 {
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleRepoSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.blurRepoSearch()
		m.tab = (m.tab + 1) % tabCount
		m.clampList()
		return m.afterTabChange()
	case "shift+tab":
		m.blurRepoSearch()
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.clampList()
		return m.afterTabChange()
	case "up":
		m.move(-1)
		return m, nil
	case "down":
		m.move(1)
		return m, nil
	case "pgdown":
		return m.nextPage()
	case "pgup":
		return m.prevPage()
	case "ctrl+r":
		return m.track(m.cmdRepos())
	case "enter":
		return m.useSelectedRepo()
	case "esc":
		if strings.TrimSpace(m.repoInput.Value()) == "" {
			return m, nil
		}
		m.repoInput.SetValue("")
		m.repoCursor = 0
		m.repoOffset = 0
		m.clampList()
		return m, nil
	default:
		prev := m.repoInput.Value()
		var cmd tea.Cmd
		m.repoInput, cmd = m.repoInput.Update(msg)
		if m.repoInput.Value() != prev {
			m.repoCursor = 0
			m.repoOffset = 0
			m.clampList()
		}
		return m, cmd
	}
}

func (m Model) useSelectedRepo() (tea.Model, tea.Cmd) {
	repo := m.selectedRepo()
	if repo == nil {
		return m, nil
	}
	m.repo = repo
	m.repoInput.Blur()
	m.prs, m.issues, m.runs = nil, nil, nil
	m.branches, m.tags = nil, nil
	m.clearAllDetail()
	m.prPage, m.issuePage, m.runPage = 1, 1, 1
	m.tab = tabPRs
	m.state = stateMain
	return m.track(m.cmdPulls())
}

func (m Model) jumpTab(tab tabID) (tea.Model, tea.Cmd) {
	if m.tab == tabRepos && tab != tabRepos {
		m.blurRepoSearch()
	}
	m.tab = tab
	m.state = stateMain
	m.clearAllDetail()
	m.clampList()
	var cmds []tea.Cmd
	if tab == tabRepos {
		cmds = append(cmds, m.repoInput.Focus())
	}
	if m.needsLoad() {
		nm, cmd := m.track(m.reloadCurrent())
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if len(cmds) == 0 {
			return nm, nil
		}
		return nm, tea.Batch(cmds...)
	}
	if len(cmds) == 0 {
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleDetail(key string) (tea.Model, tea.Cmd) {
	if m.prDiffPanelOpen {
		switch key {
		case "q":
			return m, tea.Quit
		case "esc", "tab":
			m.prDiffPanelOpen = false
			m.rebuildDetail()
			return m, nil
		case "j", "down":
			m.prDiffLineCursorDown()
			m.updatePRDiffScroll()
			return m, nil
		case "k", "up":
			m.prDiffLineCursorUp()
			m.updatePRDiffScroll()
			return m, nil
		case "J":
			m.prDiffNextHunk()
			m.updatePRDiffScroll()
			return m, nil
		case "K":
			m.prDiffPrevHunk()
			m.updatePRDiffScroll()
			return m, nil
		case "n":
			if m.prDiffFileIdx < len(m.prDiffFiles)-1 {
				m.prDiffFileIdx++
				m.prDiffLineCursor = 0
				m.prDiffScrollOffset = 0
				m.updatePRDiffScroll()
			}
			return m, nil
		case "p":
			if m.prDiffFileIdx > 0 {
				m.prDiffFileIdx--
				m.prDiffLineCursor = 0
				m.prDiffScrollOffset = 0
				m.updatePRDiffScroll()
			}
			return m, nil
		}
	}
	switch key {
	case "q":
		return m, tea.Quit
	case "esc":
		m.state = stateMain
		m.clearAllDetail()
		return m, nil
	case "tab":
		if m.detailPR != nil {
			m.prDiffPanelOpen = true
			m.rebuildDetail()
			return m, nil
		}
	case "b":
		if m.detailIssue != nil && m.tab == tabIssues {
			return m.startCreateBranchForIssue()
		}
	case "j", "down":
		if m.detailRun != nil {
			m.moveJob(1)
			return m, nil
		}
		m.detailScroll++
		m.clampDetail()
		return m, nil
	case "k", "up":
		if m.detailRun != nil {
			m.moveJob(-1)
			return m, nil
		}
		m.detailScroll--
		m.clampDetail()
		return m, nil
	case "g":
		if m.detailRun != nil {
			m.jobCursor = 0
			m.jobOffset = 0
		} else {
			m.detailScroll = 0
		}
		return m, nil
	case "G":
		if m.detailRun != nil {
			m.jobCursor = max(0, len(m.jobs)-1)
			m.clampJobs()
		} else {
			m.detailScroll = max(0, len(m.detailLines)-m.detailHeight())
		}
		return m, nil
	case "enter":
		if m.detailRun != nil {
			return m.openJobLog()
		}
	case "C":
		if m.detailPR != nil || m.detailIssue != nil {
			return m.startComment()
		}
	case "+":
		if m.detailIssue != nil {
			return m.track(m.cmdVoteIssue(m.detailIssue.Number, "+1"))
		}
	case "-":
		if m.detailIssue != nil {
			return m.track(m.cmdVoteIssue(m.detailIssue.Number, "-1"))
		}
	case "x":
		return m.closeDetail()
	case "O":
		return m.reopenDetail()
	case "m":
		return m.mergeDetail()
	case "c":
		return m.cancelDetail()
	case "R":
		return m.rerunDetail()
	case "o":
		return m.openDetailLinks()
	case "y":
		return m.yankDetail()
	case "r":
		if cmd := m.reloadDetail(); cmd != nil {
			return m.track(cmd)
		}
	}
	return m, nil
}

func (m Model) handleLog(key string) (tea.Model, tea.Cmd) {
	page := max(1, m.logHeight())
	switch key {
	case "q":
		return m, tea.Quit
	case "esc", "enter":
		m.state = stateDetail
		return m, nil
	case "j", "down":
		m.logScroll++
	case "k", "up":
		m.logScroll--
	case "ctrl+d", "pgdown":
		m.logScroll += page / 2
	case "ctrl+u", "pgup":
		m.logScroll -= page / 2
	case "g":
		m.logScroll = 0
	case "G":
		m.logScroll = max(0, len(m.logLines)-page)
	case "o":
		if m.jobCursor >= 0 && m.jobCursor < len(m.jobs) && m.jobs[m.jobCursor].HTMLURL != "" {
			return m.openURL(m.jobs[m.jobCursor].HTMLURL)
		}
	default:
		return m, nil
	}
	m.clampLog()
	return m, nil
}

func (m Model) handleConfirm(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y", "enter":
		m.state = m.returnState
		cmd := m.confirmCmd
		m.confirmCmd = nil
		return m.track(cmd)
	case "n", "N", "esc", "q":
		m.state = m.returnState
		m.confirmCmd = nil
		return m, nil
	}
	return m, nil
}

func (m Model) handleServerSelect(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.serverCursor < len(m.cfg.Servers)-1 {
			m.serverCursor++
		}
	case "k", "up":
		if m.serverCursor > 0 {
			m.serverCursor--
		}
	case "enter":
		if m.serverCursor == m.serverIdx {
			m.state = m.returnState
			return m, nil
		}
		srv := m.cfg.Servers[m.serverCursor]
		m.state = stateMain
		return m.track(m.cmdSwitchServer(m.serverCursor, srv))
	case "esc", "q":
		m.state = m.returnState
	}
	return m, nil
}

func (m Model) handleLinks(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.linkCursor < len(m.links)-1 {
			m.linkCursor++
		}
	case "k", "up":
		if m.linkCursor > 0 {
			m.linkCursor--
		}
	case "enter":
		if m.linkCursor >= 0 && m.linkCursor < len(m.links) {
			url := m.links[m.linkCursor].URL
			m.state = m.returnState
			return m.openURL(url)
		}
	case "esc", "q":
		m.state = m.returnState
	}
	return m, nil
}

func (m Model) handleComment(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = m.returnState
		m.bodyInput.Blur()
		return m, nil
	case "ctrl+s":
		body := strings.TrimSpace(m.bodyInput.Value())
		if body == "" {
			m.setStatus("Comment is empty")
			return m, m.scheduleClear()
		}
		m.state = m.returnState
		m.bodyInput.Blur()
		return m.track(m.cmdComment(body))
	default:
		var cmd tea.Cmd
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	n := m.formFieldCount()
	switch key {
	case "esc":
		m.state = m.returnState
		m.blurForm()
		return m, nil
	case "tab":
		m.formFocus = (m.formFocus + 1) % n
		return m.focusCurrent()
	case "shift+tab":
		m.formFocus = (m.formFocus + n - 1) % n
		return m.focusCurrent()
	case "enter":
		if !m.bodyFocused() {
			m.formFocus = (m.formFocus + 1) % n
			return m.focusCurrent()
		}
	case "ctrl+s":
		return m.submitCreate()
	}
	var cmd tea.Cmd
	switch {
	case m.formKind == "pr" && m.formFocus == 0:
		m.headInput, cmd = m.headInput.Update(msg)
	case m.formKind == "pr" && m.formFocus == 1:
		m.baseInput, cmd = m.baseInput.Update(msg)
	case m.bodyFocused():
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	default:
		m.titleInput, cmd = m.titleInput.Update(msg)
	}
	return m, cmd
}

func (m Model) formFieldCount() int {
	if m.formKind == "pr" {
		return 4
	}
	return 2
}

func (m Model) bodyFocused() bool {
	if m.formKind == "pr" {
		return m.formFocus == 3
	}
	return m.formFocus == 1
}

func (m Model) blurForm() {
	m.titleInput.Blur()
	m.headInput.Blur()
	m.baseInput.Blur()
	m.bodyInput.Blur()
}

func (m Model) focusCurrent() (Model, tea.Cmd) {
	m.blurForm()
	if m.formKind == "pr" {
		switch m.formFocus {
		case 0:
			return m, m.headInput.Focus()
		case 1:
			return m, m.baseInput.Focus()
		case 2:
			return m, m.titleInput.Focus()
		default:
			return m, m.bodyInput.Focus()
		}
	}
	if m.formFocus == 0 {
		return m, m.titleInput.Focus()
	}
	return m, m.bodyInput.Focus()
}

func (m Model) layoutInputs() {
	w := m.width - 10
	if w < 20 {
		w = 20
	}
	if w > 88 {
		w = 88
	}
	repoW := m.width - 6
	if repoW < 20 {
		repoW = 20
	}
	m.titleInput.Width = w
	m.headInput.Width = w
	m.baseInput.Width = w
	m.repoInput.Width = repoW
	m.bodyInput.SetWidth(w)
	h := m.height / 4
	if h < 5 {
		h = 5
	}
	if h > 14 {
		h = 14
	}
	m.bodyInput.SetHeight(h)
}

func (m *Model) setStatus(text string) {
	m.status = text
	m.statusID++
}

func (m Model) scheduleClear() tea.Cmd {
	id := m.statusID
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return statusClearMsg{id: id}
	})
}

func (m Model) track(cmds ...tea.Cmd) (Model, tea.Cmd) {
	var real []tea.Cmd
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		m.inflight++
		real = append(real, cmd)
	}
	m.loading = m.inflight > 0
	if len(real) == 0 {
		return m, nil
	}
	return m, tea.Batch(real...)
}

func (m Model) finish() Model {
	if m.inflight > 0 {
		m.inflight--
	}
	m.loading = m.inflight > 0
	return m
}

func (m Model) repoFull() string {
	if m.repo == nil {
		return ""
	}
	return m.repo.FullName
}

func (m Model) needsLoad() bool {
	if m.tab == tabRepos {
		return m.repos == nil || (len(m.repos) == 0 && strings.TrimSpace(m.repoInput.Value()) == "")
	}
	if m.repo == nil {
		return false
	}
	switch m.tab {
	case tabPRs:
		return m.prs == nil
	case tabBranches:
		return m.branches == nil
	case tabTags:
		return m.tags == nil
	case tabIssues:
		return m.issues == nil
	case tabActions:
		return m.runs == nil
	default:
		return false
	}
}

func (m Model) reloadCurrent() tea.Cmd {
	switch m.tab {
	case tabPRs:
		if m.repo == nil {
			return nil
		}
		return m.cmdPulls()
	case tabBranches:
		if m.repo == nil {
			return nil
		}
		return m.cmdLoadBranches()
	case tabTags:
		if m.repo == nil {
			return nil
		}
		return m.cmdLoadTags()
	case tabIssues:
		if m.repo == nil {
			return nil
		}
		return m.cmdIssues()
	case tabActions:
		if m.repo == nil {
			return nil
		}
		return m.cmdRuns()
	case tabRepos:
		return m.cmdRepos()
	default:
		return nil
	}
}

func (m Model) reloadDetail() tea.Cmd {
	switch {
	case m.detailPR != nil:
		return m.cmdPull(m.detailPR.Number)
	case m.detailIssue != nil:
		return m.cmdIssue(m.detailIssue.Number)
	case m.detailRun != nil:
		return m.cmdRun(m.detailRun.ID)
	default:
		return nil
	}
}

func (m *Model) listLen() int {
	switch m.tab {
	case tabPRs:
		return len(m.prs)
	case tabBranches:
		return len(m.branches)
	case tabTags:
		return len(m.tags)
	case tabIssues:
		return len(m.issues)
	case tabActions:
		return len(m.runs)
	case tabRepos:
		return len(m.visibleRepos())
	default:
		return 0
	}
}

func (m *Model) cursorPair() (cursor *int, offset *int) {
	switch m.tab {
	case tabPRs:
		return &m.prCursor, &m.prOffset
	case tabBranches:
		return &m.branchCursor, &m.branchOffset
	case tabTags:
		return &m.tagCursor, &m.tagOffset
	case tabIssues:
		return &m.issueCursor, &m.issueOffset
	case tabActions:
		return &m.runCursor, &m.runOffset
	default:
		return &m.repoCursor, &m.repoOffset
	}
}

func (m *Model) move(delta int) {
	n := m.listLen()
	if n == 0 {
		return
	}
	cur, _ := m.cursorPair()
	m.setCursor(*cur + delta)
}

func (m *Model) setCursor(i int) {
	n := m.listLen()
	cur, off := m.cursorPair()
	if n <= 0 {
		*cur, *off = 0, 0
		return
	}
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	*cur = i
	h := m.listHeight()
	if *cur < *off {
		*off = *cur
	}
	if *cur >= *off+h {
		*off = *cur - h + 1
	}
}

func (m *Model) clampList() {
	cur, _ := m.cursorPair()
	m.setCursor(*cur)
}

func (m Model) visibleRepos() []*gh.RepoInfo {
	q := strings.ToLower(strings.TrimSpace(m.repoInput.Value()))
	if q == "" {
		return m.repos
	}
	var out []*gh.RepoInfo
	for _, repo := range m.repos {
		if repo == nil {
			continue
		}
		if strings.Contains(strings.ToLower(repo.FullName), q) || strings.Contains(strings.ToLower(repo.Description), q) {
			out = append(out, repo)
		}
	}
	return out
}

func (m Model) selectedRepo() *gh.RepoInfo {
	items := m.visibleRepos()
	if m.repoCursor < 0 || m.repoCursor >= len(items) {
		return nil
	}
	return items[m.repoCursor]
}

func (m Model) nextPage() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabPRs:
		if m.prHasNext {
			m.prPage++
			m.prCursor, m.prOffset = 0, 0
			return m.track(m.cmdPulls())
		}
	case tabIssues:
		if m.issueHasNext {
			m.issuePage++
			m.issueCursor, m.issueOffset = 0, 0
			return m.track(m.cmdIssues())
		}
	case tabActions:
		if m.runHasNext {
			m.runPage++
			m.runCursor, m.runOffset = 0, 0
			return m.track(m.cmdRuns())
		}
	case tabRepos:
		if m.repoHasNext {
			m.repoPage++
			m.repoCursor, m.repoOffset = 0, 0
			return m.track(m.cmdRepos())
		}
	}
	return m, nil
}

func (m Model) prevPage() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabPRs:
		if m.prPage > 1 {
			m.prPage--
			m.prCursor, m.prOffset = 0, 0
			return m.track(m.cmdPulls())
		}
	case tabIssues:
		if m.issuePage > 1 {
			m.issuePage--
			m.issueCursor, m.issueOffset = 0, 0
			return m.track(m.cmdIssues())
		}
	case tabActions:
		if m.runPage > 1 {
			m.runPage--
			m.runCursor, m.runOffset = 0, 0
			return m.track(m.cmdRuns())
		}
	case tabRepos:
		if m.repoPage > 1 {
			m.repoPage--
			m.repoCursor, m.repoOffset = 0, 0
			return m.track(m.cmdRepos())
		}
	}
	return m, nil
}

func (m Model) cycleState() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabPRs:
		m.prState = nextFilter(m.prState)
		m.prPage, m.prCursor, m.prOffset = 1, 0, 0
		m.prs = nil
		return m.track(m.cmdPulls())
	case tabIssues:
		m.issueState = nextFilter(m.issueState)
		m.issuePage, m.issueCursor, m.issueOffset = 1, 0, 0
		m.issues = nil
		return m.track(m.cmdIssues())
	default:
		return m, nil
	}
}

func nextFilter(state string) string {
	switch state {
	case "open":
		return "closed"
	case "closed":
		return "all"
	default:
		return "open"
	}
}

func (m Model) prompt(message string, cmd tea.Cmd) (Model, tea.Cmd) {
	m.returnState = m.state
	if m.state == stateConfirm {
		m.returnState = stateMain
	}
	m.state = stateConfirm
	m.confirmMsg = message
	m.confirmCmd = cmd
	return m, nil
}

func (m Model) openSelected() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabPRs:
		if m.prCursor < 0 || m.prCursor >= len(m.prs) {
			return m, nil
		}
		return m.track(m.cmdPull(m.prs[m.prCursor].Number))
	case tabIssues:
		if m.issueCursor < 0 || m.issueCursor >= len(m.issues) {
			return m, nil
		}
		return m.track(m.cmdIssue(m.issues[m.issueCursor].Number))
	case tabActions:
		if m.runCursor < 0 || m.runCursor >= len(m.runs) {
			return m, nil
		}
		return m.track(m.cmdRun(m.runs[m.runCursor].ID))
	case tabRepos:
		repo := m.selectedRepo()
		if repo == nil {
			return m, nil
		}
		m.repo = repo
		m.repoInput.Blur()
		m.prs, m.issues, m.runs = nil, nil, nil
		m.branches, m.tags = nil, nil
		m.clearAllDetail()
		m.prPage, m.issuePage, m.runPage = 1, 1, 1
		m.tab = tabPRs
		return m.track(m.cmdPulls())
	default:
		return m, nil
	}
}

func (m Model) createOrCancel() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabPRs:
		return m.startCreatePR()
	case tabIssues:
		return m.startCreateIssue()
	case tabActions:
		return m.cancelSelected()
	default:
		return m, nil
	}
}

func (m Model) startCreateIssue() (tea.Model, tea.Cmd) {
	if m.repo == nil {
		m.setStatus("Select a repository first")
		return m, m.scheduleClear()
	}
	m.formKind = "issue"
	m.formFocus = 0
	m.titleInput.SetValue("")
	m.bodyInput.SetValue("")
	m.returnState = m.state
	m.state = stateCreate
	m.layoutInputs()
	return m.focusCurrent()
}

func (m Model) startCreatePR() (tea.Model, tea.Cmd) {
	if m.repo == nil {
		m.setStatus("Select a repository first")
		return m, m.scheduleClear()
	}
	m.formKind = "pr"
	m.formFocus = 0
	m.headInput.SetValue("")
	m.baseInput.SetValue(m.repo.DefaultBranch)
	m.titleInput.SetValue("")
	m.bodyInput.SetValue("")
	m.returnState = m.state
	m.state = stateCreate
	m.layoutInputs()
	return m.focusCurrent()
}

func (m Model) startComment() (tea.Model, tea.Cmd) {
	m.bodyInput.SetValue("")
	m.returnState = m.state
	m.state = stateComment
	m.layoutInputs()
	return m, m.bodyInput.Focus()
}

func (m Model) submitCreate() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.titleInput.Value())
	body := m.bodyInput.Value()
	if m.formKind == "pr" {
		head := strings.TrimSpace(m.headInput.Value())
		base := strings.TrimSpace(m.baseInput.Value())
		if head == "" || base == "" || title == "" {
			m.setStatus("Head, base, and title are required")
			return m, m.scheduleClear()
		}
		m.state = m.returnState
		m.blurForm()
		return m.track(m.cmdCreatePull(head, base, title, body))
	}
	if title == "" {
		m.setStatus("Title is required")
		return m, m.scheduleClear()
	}
	m.state = m.returnState
	m.blurForm()
	return m.track(m.cmdCreateIssue(title, body))
}

func (m Model) closeSelected() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabPRs:
		if m.prCursor < 0 || m.prCursor >= len(m.prs) {
			return m, nil
		}
		pr := m.prs[m.prCursor]
		if pr.State != "open" {
			return m, nil
		}
		return m.prompt(fmt.Sprintf("Close pull request #%d?", pr.Number), m.cmdSetPull(pr.Number, "closed"))
	case tabIssues:
		if m.issueCursor < 0 || m.issueCursor >= len(m.issues) {
			return m, nil
		}
		issue := m.issues[m.issueCursor]
		if issue.State != "open" {
			return m, nil
		}
		return m.prompt(fmt.Sprintf("Close issue #%d?", issue.Number), m.cmdSetIssue(issue.Number, "closed"))
	default:
		return m, nil
	}
}

func (m Model) reopenSelected() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabPRs:
		if m.prCursor < 0 || m.prCursor >= len(m.prs) {
			return m, nil
		}
		pr := m.prs[m.prCursor]
		if pr.State == "open" || pr.State == "merged" {
			return m, nil
		}
		return m.prompt(fmt.Sprintf("Reopen pull request #%d?", pr.Number), m.cmdSetPull(pr.Number, "open"))
	case tabIssues:
		if m.issueCursor < 0 || m.issueCursor >= len(m.issues) {
			return m, nil
		}
		issue := m.issues[m.issueCursor]
		if issue.State == "open" {
			return m, nil
		}
		return m.prompt(fmt.Sprintf("Reopen issue #%d?", issue.Number), m.cmdSetIssue(issue.Number, "open"))
	default:
		return m, nil
	}
}

func (m Model) mergeSelected() (tea.Model, tea.Cmd) {
	if m.tab != tabPRs || m.prCursor < 0 || m.prCursor >= len(m.prs) {
		return m, nil
	}
	pr := m.prs[m.prCursor]
	if pr.State != "open" {
		return m, nil
	}
	return m.prompt(fmt.Sprintf("Merge pull request #%d?", pr.Number), m.cmdMerge(pr.Number))
}

func (m Model) cancelSelected() (tea.Model, tea.Cmd) {
	if m.tab != tabActions || m.runCursor < 0 || m.runCursor >= len(m.runs) {
		return m, nil
	}
	run := m.runs[m.runCursor]
	if run.Status == "completed" {
		m.setStatus("That workflow run has already finished")
		return m, m.scheduleClear()
	}
	return m.prompt(fmt.Sprintf("Cancel workflow run %d?", run.ID), m.cmdCancel(run.ID))
}

func (m Model) rerunSelected() (tea.Model, tea.Cmd) {
	if m.tab != tabActions || m.runCursor < 0 || m.runCursor >= len(m.runs) {
		return m, nil
	}
	run := m.runs[m.runCursor]
	return m.prompt(fmt.Sprintf("Re-run workflow %d?", run.ID), m.cmdRerun(run.ID))
}

func (m Model) closeDetail() (tea.Model, tea.Cmd) {
	switch {
	case m.detailPR != nil && m.detailPR.State == "open":
		return m.prompt(fmt.Sprintf("Close pull request #%d?", m.detailPR.Number), m.cmdSetPull(m.detailPR.Number, "closed"))
	case m.detailIssue != nil && m.detailIssue.State == "open":
		return m.prompt(fmt.Sprintf("Close issue #%d?", m.detailIssue.Number), m.cmdSetIssue(m.detailIssue.Number, "closed"))
	default:
		return m, nil
	}
}

func (m Model) reopenDetail() (tea.Model, tea.Cmd) {
	switch {
	case m.detailPR != nil && m.detailPR.State == "closed":
		return m.prompt(fmt.Sprintf("Reopen pull request #%d?", m.detailPR.Number), m.cmdSetPull(m.detailPR.Number, "open"))
	case m.detailIssue != nil && m.detailIssue.State == "closed":
		return m.prompt(fmt.Sprintf("Reopen issue #%d?", m.detailIssue.Number), m.cmdSetIssue(m.detailIssue.Number, "open"))
	default:
		return m, nil
	}
}

func (m Model) mergeDetail() (tea.Model, tea.Cmd) {
	if m.detailPR == nil || m.detailPR.State != "open" {
		return m, nil
	}
	return m.prompt(fmt.Sprintf("Merge pull request #%d?", m.detailPR.Number), m.cmdMerge(m.detailPR.Number))
}

func (m Model) cancelDetail() (tea.Model, tea.Cmd) {
	if m.detailRun == nil || m.detailRun.Status == "completed" {
		return m, nil
	}
	return m.prompt(fmt.Sprintf("Cancel workflow run %d?", m.detailRun.ID), m.cmdCancel(m.detailRun.ID))
}

func (m Model) rerunDetail() (tea.Model, tea.Cmd) {
	if m.detailRun == nil {
		return m, nil
	}
	return m.prompt(fmt.Sprintf("Re-run workflow %d?", m.detailRun.ID), m.cmdRerun(m.detailRun.ID))
}

func (m Model) openJobLog() (tea.Model, tea.Cmd) {
	if m.jobCursor < 0 || m.jobCursor >= len(m.jobs) {
		return m, nil
	}
	job := m.jobs[m.jobCursor]
	return m.track(m.cmdJobLog(job))
}

func (m *Model) moveJob(delta int) {
	if len(m.jobs) == 0 {
		return
	}
	m.jobCursor += delta
	if m.jobCursor < 0 {
		m.jobCursor = 0
	}
	if m.jobCursor >= len(m.jobs) {
		m.jobCursor = len(m.jobs) - 1
	}
	m.clampJobs()
}

func (m *Model) clampJobs() {
	h := m.jobListHeight()
	if m.jobCursor < m.jobOffset {
		m.jobOffset = m.jobCursor
	}
	if m.jobCursor >= m.jobOffset+h {
		m.jobOffset = m.jobCursor - h + 1
	}
}

func (m *Model) clampDetail() {
	maxScroll := max(0, len(m.detailLines)-m.detailHeight())
	if m.detailScroll < 0 {
		m.detailScroll = 0
	}
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
}

func (m *Model) clampLog() {
	maxScroll := max(0, len(m.logLines)-m.logHeight())
	if m.logScroll < 0 {
		m.logScroll = 0
	}
	if m.logScroll > maxScroll {
		m.logScroll = maxScroll
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
