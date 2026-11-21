package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultLocalTimeout   = 15 * time.Second
	defaultNetworkTimeout = 60 * time.Second
	maxPreviewBytes       = 256 * 1024
)

type Client struct {
	dir     string
	mu      sync.Mutex
	lastCmd string
}

func NewClient(dir string) *Client {
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	return &Client{dir: filepath.Clean(dir)}
}

func (c *Client) Dir() string {
	return c.dir
}

func (c *Client) LastCommand() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastCmd
}

func (c *Client) setLastCommand(args ...string) {
	if len(args) == 0 {
		return
	}
	formatted := make([]string, len(args))
	for i, arg := range args {
		if strings.ContainsAny(arg, " \t\n\"'") {
			formatted[i] = strconv.Quote(arg)
		} else {
			formatted[i] = arg
		}
	}
	c.mu.Lock()
	c.lastCmd = "git " + strings.Join(formatted, " ")
	c.mu.Unlock()
}

func CheckGitInstalled() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "--version")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git --version failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *Client) InspectRepo(ctx context.Context) (RepoState, error) {
	rootDir, isRepo, err := c.resolveRoot(ctx)
	if err != nil {
		return RepoState{}, err
	}
	if !isRepo {
		return RepoState{Kind: KindNotARepo}, nil
	}

	if c.LastCommand() == "" {
		c.setLastCommand("status", "--porcelain=v1", "--branch")
	}

	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "status", "--porcelain=v1", "--branch", "-z", "-uall")
	if err != nil {
		return RepoState{}, err
	}

	header, files, err := parseStatusPorcelainZ(stdout)
	if err != nil {
		return RepoState{}, err
	}

	state := RepoState{
		RootDir:      rootDir,
		RepoName:     filepath.Base(rootDir),
		Kind:         header.Kind,
		ActiveOp:     c.detectActiveOp(ctx, rootDir),
		Branch:       header.Branch,
		Upstream:     header.Upstream,
		Ahead:        header.Ahead,
		Behind:       header.Behind,
		ChangedCount: len(files),
	}

	for _, f := range files {
		if f.IsConflicted {
			state.ConflictCount++
		}
		if f.IsStaged {
			state.StagedCount++
		}
	}

	if state.Kind != KindUnborn {
		if hashOut, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "rev-parse", "--short", "HEAD"); err == nil {
			state.HeadHash = strings.TrimSpace(string(hashOut))
		}
		if subjOut, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "log", "-1", "--format=%s", "HEAD"); err == nil {
			state.LastCommit = strings.TrimSpace(string(subjOut))
		}
	}

	return state, nil
}

func (c *Client) ListBranches(ctx context.Context, includeRemote bool) ([]Branch, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return nil, err
	}

	args := []string{
		"for-each-ref",
		"--format=%(HEAD)%1f%(refname)%1f%(refname:short)%1f%(upstream:short)%1f%(upstream:track)%1f%(objectname:short)%1f%(contents:subject)%1f%(committerdate:relative)%00",
		"refs/heads",
	}
	if includeRemote {
		args = append(args, "refs/remotes")
	}

	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, args...)
	if err != nil {
		return nil, err
	}
	return parseBranches(stdout), nil
}

func (c *Client) ListCommits(ctx context.Context, ref string, limit int) ([]Commit, error) {
	return c.ListCommitsPage(ctx, ref, limit, 0)
}

// ListCommitsPage returns a paginated slice of commits for the given ref.
func (c *Client) ListCommitsPage(ctx context.Context, ref string, limit, skip int) ([]Commit, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	args := []string{
		"log",
		"-z",
		"-n", strconv.Itoa(limit),
		"--format=%h%x1f%s%x1f%an%x1f%ar%x1f%D",
	}
	if skip > 0 {
		args = append(args, "--skip="+strconv.Itoa(skip))
	}
	if strings.TrimSpace(ref) != "" {
		args = append(args, ref)
	}

	stdout, stderr, err := c.runGit(ctx, rootDir, defaultLocalTimeout, args...)
	if err != nil {
		if strings.Contains(string(stderr), "does not have any commits yet") {
			return nil, nil
		}
		return nil, err
	}
	return parseCommits(stdout), nil
}

// ListCommitFiles returns the list of files touched by the given commit hash.
func (c *Client) ListCommitFiles(ctx context.Context, hash string) ([]CommitFile, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil, errors.New("commit hash cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return nil, err
	}
	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "show", "--name-status", "--format=", "-z", hash)
	if err != nil {
		return nil, err
	}
	return parseCommitFiles(stdout), nil
}

// DiffCommitFile returns the patch diff of a single file within a commit.
func (c *Client) DiffCommitFile(ctx context.Context, hash, path string) (string, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return "", errors.New("commit hash cannot be empty")
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return "", err
	}
	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "show", "--format=", "--patch", hash, "--", path)
	if err != nil {
		return "", err
	}
	return truncateOutput(string(stdout), maxPreviewBytes), nil
}

func (c *Client) ListStatus(ctx context.Context) ([]FileStatus, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return nil, err
	}

	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "status", "--porcelain=v1", "--branch", "-z", "-uall")
	if err != nil {
		return nil, err
	}
	_, files, err := parseStatusPorcelainZ(stdout)
	return files, err
}

// DiffStaged returns the unified diff of all staged changes.
func (c *Client) DiffStaged(ctx context.Context) (string, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return "", err
	}
	out, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--cached")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Client) DiffFile(ctx context.Context, file FileStatus) (string, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return "", err
	}

	if file.IsConflicted {
		return readPreviewFile(rootDir, file.Path)
	}

	if file.IsUntracked {
		return c.diffUntracked(ctx, rootDir, file.Path)
	}

	if file.IsStaged && file.IsUnstaged {
		stagedOut, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--cached", "--", file.Path)
		if err != nil {
			return "", err
		}
		unstagedOut, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--", file.Path)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		if len(stagedOut) > 0 {
			b.WriteString("=== Staged ===\n")
			b.Write(stagedOut)
		}
		if len(unstagedOut) > 0 {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("=== Unstaged ===\n")
			b.Write(unstagedOut)
		}
		return b.String(), nil
	}

	if file.IsStaged {
		stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--cached", "--", file.Path)
		if err != nil {
			return "", err
		}
		return string(stdout), nil
	}

	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "diff", "--", file.Path)
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

func (c *Client) diffUntracked(ctx context.Context, rootDir, relPath string) (string, error) {
	absPath, err := safeJoin(rootDir, relPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Untracked directory: %s (%d entries)\n", relPath, len(entries))
		for _, e := range entries {
			b.WriteString("  ")
			b.WriteString(e.Name())
			if e.IsDir() {
				b.WriteByte('/')
			}
			b.WriteByte('\n')
		}
		return b.String(), nil
	}

	// git diff --no-index retorna exit code 1 quando ha diferencas
	tctx, cancel := withTimeout(ctx, defaultLocalTimeout)
	defer cancel()

	cmd := exec.CommandContext(tctx, "git", "-C", rootDir, "diff", "--no-index", "--", "/dev/null", relPath)
	cmd.Env = gitEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && stdout.Len() > 0 {
			return truncateOutput(stdout.String(), maxPreviewBytes), nil
		}
		return "", fmt.Errorf("git diff --no-index: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return truncateOutput(stdout.String(), maxPreviewBytes), nil
}

func (c *Client) SwitchBranch(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	if !c.refExists(ctx, rootDir, "refs/heads/"+name) && c.refExists(ctx, rootDir, "refs/remotes/"+name) {
		if _, localName, ok := strings.Cut(name, "/"); ok && localName != "" {
			if c.refExists(ctx, rootDir, "refs/heads/"+localName) {
				c.setLastCommand("switch", localName)
				_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "switch", localName)
				return err
			}
			c.setLastCommand("switch", "-c", localName, "--track", name)
			_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "switch", "-c", localName, "--track", name)
			return err
		}
	}
	c.setLastCommand("switch", name)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "switch", name)
	return err
}

// RenameBranch renames a local branch from oldName to newName.
func (c *Client) RenameBranch(ctx context.Context, oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return errors.New("branch names cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("branch", "-m", oldName, newName)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "branch", "-m", oldName, newName)
	return err
}

// RebaseBranch rebases the current branch onto the target branch or ref.
func (c *Client) RebaseBranch(ctx context.Context, onto string) (MergeResult, error) {
	onto = strings.TrimSpace(onto)
	if onto == "" {
		return MergeResult{}, errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return MergeResult{}, err
	}

	c.setLastCommand("rebase", onto)
	stdout, stderr, rebaseErr := c.runGit(ctx, rootDir, defaultLocalTimeout, "rebase", onto)
	combined := combineOutputs(stdout, stderr)
	if rebaseErr == nil {
		return MergeResult{Conflicted: false, Output: combined}, nil
	}

	var conflicts []string
	if statuses, statusErr := c.ListStatus(ctx); statusErr == nil {
		for _, s := range statuses {
			if s.IsConflicted {
				conflicts = append(conflicts, s.Path)
			}
		}
	}
	if len(conflicts) > 0 || c.detectActiveOp(ctx, rootDir) == OpRebase {
		return MergeResult{
			Conflicted: true,
			Conflicts:  conflicts,
			Output:     combined,
		}, nil
	}

	return MergeResult{Output: combined}, rebaseErr
}

func (c *Client) CreateBranch(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("switch", "-c", name)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "switch", "-c", name)
	return err
}

func (c *Client) DeleteBranch(ctx context.Context, name string, force bool) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	c.setLastCommand("branch", flag, name)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "branch", flag, name)
	return err
}

func (c *Client) MergeBranch(ctx context.Context, name string) (MergeResult, error) {
	if strings.TrimSpace(name) == "" {
		return MergeResult{}, errors.New("branch name cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return MergeResult{}, err
	}

	c.setLastCommand("merge", "--no-edit", name)
	stdout, stderr, mergeErr := c.runGit(ctx, rootDir, defaultLocalTimeout, "merge", "--no-edit", name)
	combined := combineOutputs(stdout, stderr)
	if mergeErr == nil {
		return MergeResult{Conflicted: false, Output: combined}, nil
	}

	statuses, statusErr := c.ListStatus(ctx)
	if statusErr == nil {
		var conflicts []string
		for _, s := range statuses {
			if s.IsConflicted {
				conflicts = append(conflicts, s.Path)
			}
		}
		if len(conflicts) > 0 {
			return MergeResult{
				Conflicted: true,
				Conflicts:  conflicts,
				Output:     combined,
			}, nil
		}
	}

	return MergeResult{Output: combined}, mergeErr
}

func (c *Client) StageFile(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("add", "--", path)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "add", "--", path)
	return err
}

func (c *Client) StageAll(ctx context.Context) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("add", "-A")
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "add", "-A")
	return err
}

func (c *Client) UnstageAll(ctx context.Context) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	if c.hasHead(ctx, rootDir) {
		c.setLastCommand("reset", "HEAD")
		_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "reset", "HEAD")
		return err
	}
	c.setLastCommand("rm", "-r", "--cached", ".")
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "rm", "-r", "--cached", ".")
	return err
}

func (c *Client) UnstageFile(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}

	if c.hasHead(ctx, rootDir) {
		c.setLastCommand("reset", "HEAD", "--", path)
		_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "reset", "HEAD", "--", path)
		return err
	}
	c.setLastCommand("rm", "--cached", "--", path)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "rm", "--cached", "--", path)
	return err
}

func (c *Client) DiscardFile(ctx context.Context, file FileStatus) error {
	if strings.TrimSpace(file.Path) == "" {
		return errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}

	if file.IsUntracked {
		absPath, err := safeJoin(rootDir, file.Path)
		if err != nil {
			return err
		}
		c.setLastCommand("clean", "-f", "--", file.Path)
		return os.RemoveAll(absPath)
	}

	if file.IsStaged && !file.IsUnstaged && file.IndexStatus == 'A' {
		if err := c.UnstageFile(ctx, file.Path); err != nil {
			return err
		}
		absPath, err := safeJoin(rootDir, file.Path)
		if err != nil {
			return err
		}
		return os.RemoveAll(absPath)
	}

	c.setLastCommand("checkout", "--", file.Path)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "checkout", "--", file.Path)
	return err
}

func (c *Client) Commit(ctx context.Context, message string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("commit message cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("commit", "-m", message)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "commit", "-m", message)
	return err
}

func (c *Client) StashPush(ctx context.Context, message string) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	args := []string{"stash", "push", "-u"}
	if strings.TrimSpace(message) != "" {
		args = append(args, "-m", message)
	}
	c.setLastCommand(args...)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, args...)
	return err
}

func (c *Client) StashPop(ctx context.Context) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("stash", "pop")
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "stash", "pop")
	return err
}

// ListStashes returns all saved stashes in the repository.
func (c *Client) ListStashes(ctx context.Context) ([]StashEntry, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return nil, err
	}
	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "stash", "list", "-z", "--format=%gd%x1f%gs%x1f%cr")
	if err != nil {
		return nil, err
	}
	return parseStashes(stdout), nil
}

// ShowStash returns the stat and patch diff of the specified stash ref.
func (c *Client) ShowStash(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("stash ref cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return "", err
	}
	stdout, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "stash", "show", "--stat", "--patch", ref)
	if err != nil {
		return "", err
	}
	return truncateOutput(string(stdout), maxPreviewBytes), nil
}

// StashApply applies a stash entry without removing it from the stash list.
func (c *Client) StashApply(ctx context.Context, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return errors.New("stash ref cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("stash", "apply", ref)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "stash", "apply", ref)
	return err
}

// StashPopRef pops the specified stash entry.
func (c *Client) StashPopRef(ctx context.Context, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return errors.New("stash ref cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("stash", "pop", ref)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "stash", "pop", ref)
	return err
}

// StashDrop drops the specified stash entry.
func (c *Client) StashDrop(ctx context.Context, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return errors.New("stash ref cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("stash", "drop", ref)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "stash", "drop", ref)
	return err
}

// ApplyHunkCached stages or unstages a single hunk patch in the index.
func (c *Client) ApplyHunkCached(ctx context.Context, patch string, reverse bool) error {
	if strings.TrimSpace(patch) == "" {
		return errors.New("patch cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	args := []string{"apply", "--cached"}
	if reverse {
		args = append(args, "--reverse")
	}
	args = append(args, "--unidiff-zero", "-")
	c.setLastCommand(args...)
	_, _, err = c.runGitStdin(ctx, rootDir, defaultLocalTimeout, patch, args...)
	return err
}

// DiscardHunk reverses a single hunk patch from the working tree or index.
func (c *Client) DiscardHunk(ctx context.Context, patch string, staged bool) error {
	if strings.TrimSpace(patch) == "" {
		return errors.New("patch cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	args := []string{"apply", "--reverse"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--unidiff-zero", "-")
	c.setLastCommand(args...)
	_, _, err = c.runGitStdin(ctx, rootDir, defaultLocalTimeout, patch, args...)
	return err
}

// ContinueActiveOp continues an active merge, rebase, cherry-pick, or revert.
func (c *Client) ContinueActiveOp(ctx context.Context, op ActiveOpKind) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	var subcmd string
	switch op {
	case OpMerge:
		subcmd = "merge"
	case OpRebase:
		subcmd = "rebase"
	case OpCherryPick:
		subcmd = "cherry-pick"
	case OpRevert:
		subcmd = "revert"
	default:
		return errors.New("nenhuma operacao ativa para continuar")
	}
	c.setLastCommand(subcmd, "--continue")
	_, _, err = c.runGitWithExtraEnv(ctx, rootDir, defaultLocalTimeout, "", []string{"GIT_EDITOR=true"}, subcmd, "--continue")
	return err
}

// AbortActiveOp aborts an active merge, rebase, cherry-pick, or revert.
func (c *Client) AbortActiveOp(ctx context.Context, op ActiveOpKind) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	switch op {
	case OpMerge:
		c.setLastCommand("merge", "--abort")
		_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "merge", "--abort")
	case OpRebase:
		c.setLastCommand("rebase", "--abort")
		_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "rebase", "--abort")
	case OpCherryPick:
		c.setLastCommand("cherry-pick", "--abort")
		_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "cherry-pick", "--abort")
	case OpRevert:
		c.setLastCommand("revert", "--abort")
		_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "revert", "--abort")
	default:
		return errors.New("nenhuma operacao ativa para abortar")
	}
	return err
}

func (c *Client) ResolveConflict(ctx context.Context, path string, side ConflictSide) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("file path cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}

	var sideFlag string
	switch side {
	case SideOurs:
		sideFlag = "--ours"
	case SideTheirs:
		sideFlag = "--theirs"
	default:
		return fmt.Errorf("invalid conflict side: %d", side)
	}

	c.setLastCommand("checkout", sideFlag, "--", path)
	if _, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "checkout", sideFlag, "--", path); err != nil {
		return err
	}
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "add", "--", path)
	return err
}

func (c *Client) Fetch(ctx context.Context) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("fetch", "--prune")
	_, _, err = c.runGit(ctx, rootDir, defaultNetworkTimeout, "fetch", "--prune")
	return err
}

func (c *Client) Pull(ctx context.Context) error {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("pull")
	_, _, err = c.runGit(ctx, rootDir, defaultNetworkTimeout, "pull")
	return err
}

func (c *Client) Push(ctx context.Context) (string, error) {
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return "", err
	}

	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err == nil {
		c.setLastCommand("push")
		if _, _, pushErr := c.runGit(ctx, rootDir, defaultNetworkTimeout, "push"); pushErr != nil {
			return "", pushErr
		}
		return "Push concluído", nil
	}

	branchOut, _, branchErr := c.runGit(ctx, rootDir, defaultLocalTimeout, "rev-parse", "--abbrev-ref", "HEAD")
	if branchErr != nil {
		return "", branchErr
	}
	branch := strings.TrimSpace(string(branchOut))
	if branch == "" || branch == "HEAD" {
		return "", errors.New("HEAD destacado: não é possível fazer push sem uma branch")
	}

	c.setLastCommand("push", "-u", "origin", branch)
	if _, _, pushErr := c.runGit(ctx, rootDir, defaultNetworkTimeout, "push", "-u", "origin", branch); pushErr != nil {
		return "", pushErr
	}
	return fmt.Sprintf("Push concluído (upstream configurado: origin/%s)", branch), nil
}

func (c *Client) ShowCommit(ctx context.Context, hash string) (string, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return "", errors.New("commit hash cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return "", err
	}
	stdout, _, err := c.runGit(
		ctx,
		rootDir,
		defaultLocalTimeout,
		"show",
		"--stat",
		"--patch",
		"--format=commit %h (%H)%nAutor: %an <%ae>%nData:  %ar%n%n    %s%n",
		hash,
	)
	if err != nil {
		return "", err
	}
	return truncateOutput(string(stdout), maxPreviewBytes), nil
}

func (c *Client) RevertCommit(ctx context.Context, hash string) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return errors.New("commit hash cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("revert", "--no-edit", hash)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "revert", "--no-edit", hash)
	return err
}

func (c *Client) CherryPick(ctx context.Context, hash string) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return errors.New("commit hash cannot be empty")
	}
	rootDir, err := c.requireRoot(ctx)
	if err != nil {
		return err
	}
	c.setLastCommand("cherry-pick", hash)
	_, _, err = c.runGit(ctx, rootDir, defaultLocalTimeout, "cherry-pick", hash)
	return err
}

func (c *Client) resolveRoot(ctx context.Context) (string, bool, error) {
	stdout, stderr, err := c.runGit(ctx, c.dir, defaultLocalTimeout, "rev-parse", "--show-toplevel")
	if err != nil {
		if isNotARepoError(string(stderr)) {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(string(stdout)), true, nil
}

func (c *Client) requireRoot(ctx context.Context) (string, error) {
	rootDir, isRepo, err := c.resolveRoot(ctx)
	if err != nil {
		return "", err
	}
	if !isRepo {
		return "", fmt.Errorf("%s is not a git repository", c.dir)
	}
	return rootDir, nil
}

func (c *Client) hasHead(ctx context.Context, rootDir string) bool {
	_, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "rev-parse", "--verify", "HEAD")
	return err == nil
}

func (c *Client) refExists(ctx context.Context, rootDir, fullRef string) bool {
	_, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "show-ref", "--verify", "--quiet", fullRef)
	return err == nil
}

func (c *Client) detectActiveOp(ctx context.Context, rootDir string) ActiveOpKind {
	gitDir := filepath.Join(rootDir, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		out, _, err := c.runGit(ctx, rootDir, defaultLocalTimeout, "rev-parse", "--git-dir")
		if err != nil {
			return OpNone
		}
		resolved := strings.TrimSpace(string(out))
		if resolved == "" {
			return OpNone
		}
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(rootDir, resolved)
		}
		gitDir = resolved
	}

	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(gitDir, name))
		return err == nil
	}

	switch {
	case exists("rebase-merge") || exists("rebase-apply"):
		return OpRebase
	case exists("MERGE_HEAD"):
		return OpMerge
	case exists("CHERRY_PICK_HEAD"):
		return OpCherryPick
	case exists("REVERT_HEAD"):
		return OpRevert
	default:
		return OpNone
	}
}

func (c *Client) runGit(ctx context.Context, dir string, timeout time.Duration, args ...string) ([]byte, []byte, error) {
	return c.runGitStdin(ctx, dir, timeout, "", args...)
}

func (c *Client) runGitStdin(ctx context.Context, dir string, timeout time.Duration, stdin string, args ...string) ([]byte, []byte, error) {
	return c.runGitWithExtraEnv(ctx, dir, timeout, stdin, nil, args...)
}

func (c *Client) runGitWithExtraEnv(ctx context.Context, dir string, timeout time.Duration, stdin string, extraEnv []string, args ...string) ([]byte, []byte, error) {
	tctx, cancel := withTimeout(ctx, timeout)
	defer cancel()

	fullArgs := make([]string, 0, len(args)+2)
	fullArgs = append(fullArgs, "-C", dir)
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(tctx, "git", fullArgs...)
	env := gitEnv()
	if len(extraEnv) > 0 {
		env = append(env, extraEnv...)
	}
	cmd.Env = env
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("git %s: %w: %s", args[0], err, errMsg)
		}
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("git %s: %w", args[0], err)
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

func gitEnv() []string {
	return append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
}

func isNotARepoError(stderr string) bool {
	lower := strings.ToLower(stderr)
	return strings.Contains(lower, "not a git repository")
}

func combineOutputs(stdout, stderr []byte) string {
	out := strings.TrimSpace(string(stdout))
	errStr := strings.TrimSpace(string(stderr))
	switch {
	case out != "" && errStr != "":
		return out + "\n" + errStr
	case out != "":
		return out
	default:
		return errStr
	}
}

func safeJoin(rootDir, relPath string) (string, error) {
	cleanRel := filepath.Clean(relPath)
	if filepath.IsAbs(cleanRel) || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid relative path: %q", relPath)
	}
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(absRoot, cleanRel)
	if joined != absRoot && !strings.HasPrefix(joined, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository root: %q", relPath)
	}
	return joined, nil
}

func readPreviewFile(rootDir, relPath string) (string, error) {
	absPath, err := safeJoin(rootDir, relPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	if len(data) > maxPreviewBytes {
		data = data[:maxPreviewBytes]
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return "(binary file)", nil
	}
	return string(data), nil
}

func truncateOutput(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "\n... (truncated)\n"
}
