package github

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v68/github"
)

// CommitInfo is a repository commit shown in branch/tag views.
type CommitInfo struct {
	ID         string
	ShortID    string
	Title      string
	AuthorName string
	Date       string
}

// TagInfo is an annotated or lightweight tag.
type TagInfo struct {
	Name        string
	Target      string
	Message     string
	CommitTitle string
	CommitID    string
	ShortID     string
	AuthorName  string
	Date        string
	ReleaseDesc string
}

// CompareInfo is the result of comparing two refs.
type CompareInfo struct {
	Commits []*CommitInfo
	Diffs   []*DiffFile
}

// ListBranches returns branch names for a repository.
func (c *Client) ListBranches(full string) ([]string, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	var names []string
	page := 1
	for {
		opts := &gh.BranchListOptions{
			ListOptions: gh.ListOptions{Page: page, PerPage: 100},
		}
		branches, resp, err := c.raw.Repositories.ListBranches(context.Background(), owner, name, opts)
		if err != nil {
			return nil, apiErr("list branches", err)
		}
		for _, b := range branches {
			if b != nil && b.GetName() != "" {
				names = append(names, b.GetName())
			}
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}
	return names, nil
}

// CreateBranch creates branch from ref (branch name or tag).
func (c *Client) CreateBranch(full, branch, ref string) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	sha, err := c.resolveRefSHA(owner, name, ref)
	if err != nil {
		return err
	}
	refPath := "refs/heads/" + branch
	_, _, err = c.raw.Git.CreateRef(context.Background(), owner, name, &gh.Reference{
		Ref:    gh.Ptr(refPath),
		Object: &gh.GitObject{SHA: gh.Ptr(sha)},
	})
	return apiErr(fmt.Sprintf("create branch %q", branch), err)
}

// DeleteBranch deletes a branch ref.
func (c *Client) DeleteBranch(full, branch string) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	ref, _, err := c.raw.Git.GetRef(context.Background(), owner, name, "heads/"+branch)
	if err != nil {
		return apiErr(fmt.Sprintf("delete branch %q", branch), err)
	}
	_, err = c.raw.Git.DeleteRef(context.Background(), owner, name, ref.GetRef())
	return apiErr(fmt.Sprintf("delete branch %q", branch), err)
}

// ListCommits lists commits reachable from ref.
func (c *Client) ListCommits(full, ref string) ([]*CommitInfo, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	opts := &gh.CommitsListOptions{SHA: ref, ListOptions: gh.ListOptions{PerPage: 100}}
	commits, _, err := c.raw.Repositories.ListCommits(context.Background(), owner, name, opts)
	if err != nil {
		return nil, apiErr("list commits", err)
	}
	out := make([]*CommitInfo, 0, len(commits))
	for _, comm := range commits {
		out = append(out, mapCommit(comm))
	}
	return out, nil
}

// Compare compares base...head (commits on head not in base).
func (c *Client) Compare(full, base, head string) (*CompareInfo, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	comp, _, err := c.raw.Repositories.CompareCommits(context.Background(), owner, name, base, head, nil)
	if err != nil {
		return nil, apiErr("compare branches", err)
	}
	info := &CompareInfo{}
	for _, comm := range comp.Commits {
		info.Commits = append(info.Commits, mapCommit(comm))
	}
	for _, f := range comp.Files {
		if f == nil {
			continue
		}
		patch := f.GetPatch()
		df := mapPatchFile(f.GetFilename(), patch)
		df.OldPath = f.GetPreviousFilename()
		if df.OldPath == "" {
			df.OldPath = f.GetFilename()
		}
		df.NewPath = f.GetFilename()
		df.NewFile = f.GetStatus() == "added"
		df.DeletedFile = f.GetStatus() == "removed"
		df.RenamedFile = f.GetStatus() == "renamed"
		info.Diffs = append(info.Diffs, df)
	}
	return info, nil
}

// GetCommitDiffs returns per-file diffs for a commit SHA.
func (c *Client) GetCommitDiffs(full, sha string) ([]*DiffFile, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	commit, _, err := c.raw.Repositories.GetCommit(context.Background(), owner, name, sha, nil)
	if err != nil {
		return nil, apiErr("get commit", err)
	}
	var out []*DiffFile
	for _, f := range commit.Files {
		if f == nil {
			continue
		}
		df := mapPatchFile(f.GetFilename(), f.GetPatch())
		df.OldPath = f.GetPreviousFilename()
		if df.OldPath == "" {
			df.OldPath = f.GetFilename()
		}
		df.NewPath = f.GetFilename()
		df.NewFile = f.GetStatus() == "added"
		df.DeletedFile = f.GetStatus() == "removed"
		df.RenamedFile = f.GetStatus() == "renamed"
		out = append(out, df)
	}
	return out, nil
}

// GetPullDiffs returns per-file diffs for a pull request.
func (c *Client) GetPullDiffs(full string, number int) ([]*DiffFile, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	var out []*DiffFile
	page := 1
	for {
		opts := &gh.ListOptions{Page: page, PerPage: 100}
		files, resp, err := c.raw.PullRequests.ListFiles(context.Background(), owner, name, number, opts)
		if err != nil {
			return nil, apiErr(fmt.Sprintf("list pull request #%d files", number), err)
		}
		for _, f := range files {
			if f == nil {
				continue
			}
			df := mapPatchFile(f.GetFilename(), f.GetPatch())
			df.OldPath = f.GetPreviousFilename()
			if df.OldPath == "" {
				df.OldPath = f.GetFilename()
			}
			df.NewPath = f.GetFilename()
			df.NewFile = f.GetStatus() == "added"
			df.DeletedFile = f.GetStatus() == "removed"
			df.RenamedFile = f.GetStatus() == "renamed"
			out = append(out, df)
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}
	return out, nil
}

// ListTags lists repository tags.
func (c *Client) ListTags(full string) ([]*TagInfo, error) {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return nil, err
	}
	var tags []*TagInfo
	page := 1
	for {
		opts := &gh.ListOptions{Page: page, PerPage: 100}
		res, resp, err := c.raw.Repositories.ListTags(context.Background(), owner, name, opts)
		if err != nil {
			return nil, apiErr("list tags", err)
		}
		for _, t := range res {
			tags = append(tags, mapTag(t))
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}
	// Attach GitHub release notes when a release exists for the tag.
	for _, ti := range tags {
		rel, _, err := c.raw.Repositories.GetReleaseByTag(context.Background(), owner, name, ti.Name)
		if err == nil && rel != nil {
			ti.ReleaseDesc = rel.GetBody()
		}
	}
	return tags, nil
}

// CreateTag creates an annotated tag (or lightweight when message is empty).
func (c *Client) CreateTag(full, tagName, ref, message string) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	sha, err := c.resolveRefSHA(owner, name, ref)
	if err != nil {
		return err
	}
	if strings.TrimSpace(message) == "" {
		refPath := "refs/tags/" + tagName
		_, _, err = c.raw.Git.CreateRef(context.Background(), owner, name, &gh.Reference{
			Ref:    gh.Ptr(refPath),
			Object: &gh.GitObject{SHA: gh.Ptr(sha)},
		})
		return apiErr(fmt.Sprintf("create tag %q", tagName), err)
	}
	_, _, err = c.raw.Git.CreateTag(context.Background(), owner, name, &gh.Tag{
		Tag:     gh.Ptr(tagName),
		Message: gh.Ptr(message),
		Object:  &gh.GitObject{SHA: gh.Ptr(sha), Type: gh.Ptr("commit")},
	})
	return apiErr(fmt.Sprintf("create tag %q", tagName), err)
}

// DeleteTag deletes a tag ref.
func (c *Client) DeleteTag(full, tag string) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	ref, _, err := c.raw.Git.GetRef(context.Background(), owner, name, "tags/"+tag)
	if err != nil {
		return apiErr(fmt.Sprintf("delete tag %q", tag), err)
	}
	_, err = c.raw.Git.DeleteRef(context.Background(), owner, name, ref.GetRef())
	return apiErr(fmt.Sprintf("delete tag %q", tag), err)
}

// UpdateTagRelease creates or updates a GitHub release for a tag.
func (c *Client) UpdateTagRelease(full, tagName, description string) error {
	owner, name, err := SplitRepo(full)
	if err != nil {
		return err
	}
	rel, _, err := c.raw.Repositories.GetReleaseByTag(context.Background(), owner, name, tagName)
	if err == nil && rel != nil {
		rel.Body = &description
		_, _, err = c.raw.Repositories.EditRelease(context.Background(), owner, name, rel.GetID(), rel)
		return apiErr("update release", err)
	}
	_, _, err = c.raw.Repositories.CreateRelease(context.Background(), owner, name, &gh.RepositoryRelease{
		TagName: gh.Ptr(tagName),
		Name:    gh.Ptr(tagName),
		Body:    gh.Ptr(description),
	})
	return apiErr("create release", err)
}

func (c *Client) resolveRefSHA(owner, repo, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty ref")
	}
	// Try branch/tag ref first.
	for _, prefix := range []string{"heads/", "tags/"} {
		gitRef, _, err := c.raw.Git.GetRef(context.Background(), owner, repo, prefix+ref)
		if err == nil && gitRef != nil && gitRef.Object != nil {
			return gitRef.Object.GetSHA(), nil
		}
	}
	// Fall back to commit SHA.
	if len(ref) >= 7 {
		commit, _, err := c.raw.Repositories.GetCommit(context.Background(), owner, repo, ref, nil)
		if err == nil && commit != nil {
			return commit.GetSHA(), nil
		}
	}
	return "", fmt.Errorf("could not resolve ref %q", ref)
}

func mapCommit(comm *gh.RepositoryCommit) *CommitInfo {
	if comm == nil {
		return &CommitInfo{}
	}
	title := ""
	if comm.Commit != nil {
		title = comm.Commit.GetMessage()
	}
	if idx := strings.Index(title, "\n"); idx >= 0 {
		title = title[:idx]
	}
	date := ""
	if comm.Commit != nil && comm.Commit.Author != nil && comm.Commit.Author.Date != nil {
		date = comm.Commit.Author.Date.Format("2006-01-02 15:04")
	}
	author := ""
	if comm.Author != nil {
		author = comm.Author.GetLogin()
	} else if comm.Commit != nil && comm.Commit.Author != nil {
		author = comm.Commit.Author.GetName()
	}
	sha := comm.GetSHA()
	short := sha
	if len(short) > 10 {
		short = short[:10]
	}
	return &CommitInfo{
		ID:         sha,
		ShortID:    short,
		Title:      title,
		AuthorName: author,
		Date:       date,
	}
}

func mapTag(t *gh.RepositoryTag) *TagInfo {
	if t == nil {
		return &TagInfo{}
	}
	ti := &TagInfo{Name: t.GetName()}
	if t.Commit != nil {
		ti.CommitID = t.Commit.GetSHA()
		if len(ti.CommitID) > 10 {
			ti.ShortID = ti.CommitID[:10]
		}
		msg := t.Commit.GetMessage()
		if idx := strings.Index(msg, "\n"); idx >= 0 {
			ti.CommitTitle = msg[:idx]
		} else {
			ti.CommitTitle = msg
		}
	}
	return ti
}
