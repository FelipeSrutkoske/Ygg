package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitTest(t, dir, "init", "-b", "main")
	runGitTest(t, dir, "config", "user.name", "Test User")
	runGitTest(t, dir, "config", "user.email", "test@example.com")
	return dir
}

func writeTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", full, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func TestCheckGitInstalled(t *testing.T) {
	if err := CheckGitInstalled(); err != nil {
		t.Fatalf("CheckGitInstalled failed: %v", err)
	}
}

func TestInspectRepo_NotARepo(t *testing.T) {
	dir := t.TempDir()
	client := NewClient(dir)
	state, err := client.InspectRepo(context.Background())
	if err != nil {
		t.Fatalf("InspectRepo error: %v", err)
	}
	if state.Kind != KindNotARepo {
		t.Errorf("Kind = %v, want %v", state.Kind, KindNotARepo)
	}
}

func TestInspectRepo_UnbornStageUnstageCommit(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	client := NewClient(dir)

	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if state.Kind != KindUnborn {
		t.Errorf("Kind = %v, want %v", state.Kind, KindUnborn)
	}
	if state.Branch != "main" {
		t.Errorf("Branch = %q, want main", state.Branch)
	}

	commits, err := client.ListCommits(ctx, "", 10)
	if err != nil {
		t.Fatalf("ListCommits on unborn: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits on unborn, got %d", len(commits))
	}

	writeTestFile(t, dir, "hello.txt", "world\n")
	if err := client.StageFile(ctx, "hello.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}

	if err := client.UnstageFile(ctx, "hello.txt"); err != nil {
		t.Fatalf("UnstageFile on unborn: %v", err)
	}
	statuses, err := client.ListStatus(ctx)
	if err != nil || len(statuses) != 1 || !statuses[0].IsUntracked {
		t.Fatalf("expected untracked hello.txt after unstage, got %+v (err=%v)", statuses, err)
	}

	if err := client.StageFile(ctx, "hello.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}
	if err := client.Commit(ctx, "initial commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	state, err = client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo after commit: %v", err)
	}
	if state.Kind != KindNormal {
		t.Errorf("Kind = %v, want %v", state.Kind, KindNormal)
	}
	if state.HeadHash == "" || state.LastCommit != "initial commit" {
		t.Errorf("unexpected HeadHash=%q LastCommit=%q", state.HeadHash, state.LastCommit)
	}
	if state.ChangedCount != 0 {
		t.Errorf("ChangedCount = %d, want 0", state.ChangedCount)
	}
}

func TestInspectRepo_Subdir(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "sub/nested/file.txt", "v1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	writeTestFile(t, dir, "sub/nested/file.txt", "v2\n")
	subdir := filepath.Join(dir, "sub", "nested")
	client := NewClient(subdir)

	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo from subdir: %v", err)
	}
	if state.RootDir != dir {
		t.Errorf("RootDir = %q, want %q", state.RootDir, dir)
	}
	if state.ChangedCount != 1 {
		t.Errorf("ChangedCount = %d, want 1", state.ChangedCount)
	}

	statuses, err := client.ListStatus(ctx)
	if err != nil || len(statuses) != 1 {
		t.Fatalf("ListStatus from subdir: %+v (err=%v)", statuses, err)
	}
	diff, err := client.DiffFile(ctx, statuses[0])
	if err != nil {
		t.Fatalf("DiffFile from subdir: %v", err)
	}
	if !strings.Contains(diff, "+v2") || !strings.Contains(diff, "-v1") {
		t.Errorf("unexpected diff:\n%s", diff)
	}
}

func TestInspectRepo_DetachedHEAD(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.txt", "1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "c1")
	writeTestFile(t, dir, "a.txt", "2\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "c2")

	runGitTest(t, dir, "checkout", "HEAD~1")

	client := NewClient(dir)
	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo detached: %v", err)
	}
	if state.Kind != KindDetached {
		t.Errorf("Kind = %v, want %v", state.Kind, KindDetached)
	}
	if state.HeadHash == "" || state.LastCommit != "c1" {
		t.Errorf("HeadHash=%q LastCommit=%q, want c1", state.HeadHash, state.LastCommit)
	}
}

func TestBranches_CreateSwitchDeleteAndCommits(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "base.txt", "base\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "base commit")

	client := NewClient(dir)
	if err := client.CreateBranch(ctx, "feature/test"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	writeTestFile(t, dir, "feat.txt", "feat\n")
	if err := client.StageFile(ctx, "feat.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}
	if err := client.Commit(ctx, "feat commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	branches, err := client.ListBranches(ctx, false)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("got %d branches, want 2", len(branches))
	}

	commits, err := client.ListCommits(ctx, "feature/test", 10)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 2 || commits[0].Subject != "feat commit" {
		t.Errorf("unexpected commits: %+v", commits)
	}

	if err := client.SwitchBranch(ctx, "main"); err != nil {
		t.Fatalf("SwitchBranch: %v", err)
	}
	if err := client.DeleteBranch(ctx, "feature/test", false); err == nil {
		t.Error("expected non-force delete of unmerged branch to fail")
	}
	if err := client.DeleteBranch(ctx, "feature/test", true); err != nil {
		t.Fatalf("force DeleteBranch: %v", err)
	}
}

func TestMergeConflictAndResolution(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "conflict.txt", "line 1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "base")

	client := NewClient(dir)
	if err := client.CreateBranch(ctx, "theirs"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	writeTestFile(t, dir, "conflict.txt", "from theirs\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "theirs change")

	if err := client.SwitchBranch(ctx, "main"); err != nil {
		t.Fatalf("SwitchBranch main: %v", err)
	}
	writeTestFile(t, dir, "conflict.txt", "from ours\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "ours change")

	res, err := client.MergeBranch(ctx, "theirs")
	if err != nil {
		t.Fatalf("MergeBranch returned unexpected error: %v", err)
	}
	if !res.Conflicted {
		t.Fatalf("expected Conflicted=true, got false (output=%q)", res.Output)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0] != "conflict.txt" {
		t.Fatalf("unexpected Conflicts: %v", res.Conflicts)
	}

	statuses, err := client.ListStatus(ctx)
	if err != nil || len(statuses) != 1 || !statuses[0].IsConflicted {
		t.Fatalf("expected conflicted status, got %+v (err=%v)", statuses, err)
	}

	diff, err := client.DiffFile(ctx, statuses[0])
	if err != nil {
		t.Fatalf("DiffFile on conflict: %v", err)
	}
	if !strings.Contains(diff, "<<<<<<<") || !strings.Contains(diff, ">>>>>>>") {
		t.Errorf("expected conflict markers in diff, got:\n%s", diff)
	}

	if err := client.ResolveConflict(ctx, "conflict.txt", SideTheirs); err != nil {
		t.Fatalf("ResolveConflict: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "conflict.txt"))
	if err != nil {
		t.Fatalf("read resolved file: %v", err)
	}
	if string(data) != "from theirs\n" {
		t.Errorf("resolved content = %q, want %q", string(data), "from theirs\n")
	}

	if err := client.Commit(ctx, "resolve conflict"); err != nil {
		t.Fatalf("Commit after resolution: %v", err)
	}

	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if state.ConflictCount != 0 || state.ChangedCount != 0 {
		t.Errorf("expected clean repo after resolution, got %+v", state)
	}
}

func TestDiffFile_StagedUnstagedUntrackedUnicode(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "tracked file.txt", "one\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	writeTestFile(t, dir, "tracked file.txt", "two\n")
	runGitTest(t, dir, "add", "tracked file.txt")
	writeTestFile(t, dir, "tracked file.txt", "three\n")

	writeTestFile(t, dir, "café/notas finais.txt", "olá mundo\n")

	client := NewClient(dir)
	statuses, err := client.ListStatus(ctx)
	if err != nil {
		t.Fatalf("ListStatus: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("got %d statuses, want 2: %+v", len(statuses), statuses)
	}

	for _, st := range statuses {
		diff, err := client.DiffFile(ctx, st)
		if err != nil {
			t.Fatalf("DiffFile(%s): %v", st.Path, err)
		}
		if st.IsUntracked {
			if !strings.Contains(diff, "+olá mundo") {
				t.Errorf("untracked diff missing content:\n%s", diff)
			}
		} else {
			if !strings.Contains(diff, "=== Staged ===") || !strings.Contains(diff, "=== Unstaged ===") {
				t.Errorf("MM diff missing staged/unstaged sections:\n%s", diff)
			}
		}
	}
}

func TestStashPushAndPop(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "f.txt", "v1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	writeTestFile(t, dir, "f.txt", "v2\n")
	writeTestFile(t, dir, "untracked.txt", "temp\n")

	client := NewClient(dir)
	if err := client.StashPush(ctx, "wip"); err != nil {
		t.Fatalf("StashPush: %v", err)
	}
	state, err := client.InspectRepo(ctx)
	if err != nil || state.ChangedCount != 0 {
		t.Fatalf("expected clean repo after stash, got %+v (err=%v)", state, err)
	}

	if err := client.StashPop(ctx); err != nil {
		t.Fatalf("StashPop: %v", err)
	}
	state, err = client.InspectRepo(ctx)
	if err != nil || state.ChangedCount != 2 {
		t.Fatalf("expected 2 changed files after stash pop, got %+v (err=%v)", state, err)
	}
}

func TestDiscardFile(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.txt", "original\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	writeTestFile(t, dir, "a.txt", "modified\n")
	writeTestFile(t, dir, "new.txt", "untracked\n")

	client := NewClient(dir)
	statuses, err := client.ListStatus(ctx)
	if err != nil {
		t.Fatalf("ListStatus: %v", err)
	}
	for _, st := range statuses {
		if err := client.DiscardFile(ctx, st); err != nil {
			t.Fatalf("DiscardFile(%s): %v", st.Path, err)
		}
	}

	state, err := client.InspectRepo(ctx)
	if err != nil || state.ChangedCount != 0 {
		t.Fatalf("expected 0 changes after discard, got %+v (err=%v)", state, err)
	}
}

func TestFetchAndPull_LocalBareRemote(t *testing.T) {
	ctx := context.Background()
	bare := t.TempDir()
	runGitTest(t, bare, "init", "--bare", "-b", "main")

	repoA := initTestRepo(t)
	runGitTest(t, repoA, "remote", "add", "origin", bare)
	writeTestFile(t, repoA, "readme.txt", "v1\n")
	runGitTest(t, repoA, "add", ".")
	runGitTest(t, repoA, "commit", "-m", "c1")
	runGitTest(t, repoA, "push", "-u", "origin", "main")

	repoBParent := t.TempDir()
	repoB := filepath.Join(repoBParent, "clone")
	cmd := exec.Command("git", "clone", bare, repoB)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone failed: %v\n%s", err, string(out))
	}
	runGitTest(t, repoB, "config", "user.name", "User B")
	runGitTest(t, repoB, "config", "user.email", "b@example.com")

	writeTestFile(t, repoA, "readme.txt", "v2\n")
	runGitTest(t, repoA, "add", ".")
	runGitTest(t, repoA, "commit", "-m", "c2")
	runGitTest(t, repoA, "push")

	clientB := NewClient(repoB)
	if err := clientB.Fetch(ctx); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	state, err := clientB.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if state.Behind != 1 {
		t.Errorf("Behind = %d, want 1", state.Behind)
	}

	branches, err := clientB.ListBranches(ctx, true)
	if err != nil {
		t.Fatalf("ListBranches(includeRemote=true): %v", err)
	}
	hasRemote := false
	for _, b := range branches {
		if b.IsRemote && b.Name == "origin/main" {
			hasRemote = true
		}
	}
	if !hasRemote {
		t.Errorf("expected origin/main in remote branches: %+v", branches)
	}

	if err := clientB.Pull(ctx); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	state, err = clientB.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo after pull: %v", err)
	}
	if state.Behind != 0 || state.LastCommit != "c2" {
		t.Errorf("unexpected state after pull: %+v", state)
	}
	if state.Upstream != "origin/main" {
		t.Errorf("Upstream = %q, want origin/main", state.Upstream)
	}
}

func TestPush_WithAndWithoutUpstream_AndLastCommand(t *testing.T) {
	ctx := context.Background()
	bare := t.TempDir()
	runGitTest(t, bare, "init", "--bare", "-b", "main")

	repo := initTestRepo(t)
	runGitTest(t, repo, "remote", "add", "origin", bare)
	writeTestFile(t, repo, "a.txt", "v1\n")
	runGitTest(t, repo, "add", ".")
	runGitTest(t, repo, "commit", "-m", "c1")

	client := NewClient(repo)
	if _, err := client.InspectRepo(ctx); err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if got := client.LastCommand(); got != "git status --porcelain=v1 --branch" {
		t.Errorf("initial LastCommand = %q", got)
	}

	// First push without upstream -> should push -u origin main
	msg, err := client.Push(ctx)
	if err != nil {
		t.Fatalf("Push without upstream: %v", err)
	}
	if !strings.Contains(msg, "origin/main") {
		t.Errorf("expected upstream message, got %q", msg)
	}
	if got := client.LastCommand(); got != "git push -u origin main" {
		t.Errorf("LastCommand after first push = %q", got)
	}

	// Second commit + push with upstream already configured
	writeTestFile(t, repo, "a.txt", "v2\n")
	if err := client.StageFile(ctx, "a.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}
	if got := client.LastCommand(); got != "git add -- a.txt" {
		t.Errorf("LastCommand after stage = %q", got)
	}
	if err := client.Commit(ctx, "second commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if got := client.LastCommand(); got != `git commit -m "second commit"` {
		t.Errorf("LastCommand after commit = %q", got)
	}

	state, err := client.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if state.Ahead != 1 || state.Upstream != "origin/main" {
		t.Errorf("expected Ahead=1 Upstream=origin/main, got %+v", state)
	}

	msg, err = client.Push(ctx)
	if err != nil {
		t.Fatalf("Push with upstream: %v", err)
	}
	if msg != "Push concluído" {
		t.Errorf("unexpected push message: %q", msg)
	}
	if got := client.LastCommand(); got != "git push" {
		t.Errorf("LastCommand after push = %q", got)
	}
}

func TestShowCommit_RevertAndCherryPick(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "app.txt", "line1\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "initial")

	client := NewClient(dir)
	if err := client.CreateBranch(ctx, "feat"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	writeTestFile(t, dir, "feat.txt", "hello feat\n")
	if err := client.StageFile(ctx, "feat.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}
	if err := client.Commit(ctx, "add feat"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	featCommits, err := client.ListCommits(ctx, "feat", 5)
	if err != nil || len(featCommits) == 0 {
		t.Fatalf("ListCommits feat: %v", err)
	}
	featHash := featCommits[0].Hash

	diff, err := client.ShowCommit(ctx, featHash)
	if err != nil {
		t.Fatalf("ShowCommit: %v", err)
	}
	if !strings.Contains(diff, "add feat") || !strings.Contains(diff, "+hello feat") {
		t.Errorf("unexpected ShowCommit output:\n%s", diff)
	}

	if err := client.SwitchBranch(ctx, "main"); err != nil {
		t.Fatalf("SwitchBranch main: %v", err)
	}
	if err := client.CherryPick(ctx, featHash); err != nil {
		t.Fatalf("CherryPick: %v", err)
	}
	if got := client.LastCommand(); got != "git cherry-pick "+featHash {
		t.Errorf("LastCommand after cherry-pick = %q", got)
	}

	mainCommits, err := client.ListCommits(ctx, "main", 5)
	if err != nil || len(mainCommits) < 2 || mainCommits[0].Subject != "add feat" {
		t.Fatalf("expected cherry-picked commit on main, got %+v (err=%v)", mainCommits, err)
	}

	if err := client.RevertCommit(ctx, mainCommits[0].Hash); err != nil {
		t.Fatalf("RevertCommit: %v", err)
	}
	if got := client.LastCommand(); got != "git revert --no-edit "+mainCommits[0].Hash {
		t.Errorf("LastCommand after revert = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "feat.txt")); !os.IsNotExist(err) {
		t.Errorf("expected feat.txt to be removed after revert")
	}
}

func TestStashListShowApplyPopDrop(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "a.txt", "base\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	client := NewClient(dir)

	writeTestFile(t, dir, "a.txt", "first change\n")
	if err := client.StashPush(ctx, "stash one"); err != nil {
		t.Fatalf("StashPush 1: %v", err)
	}
	writeTestFile(t, dir, "a.txt", "second change\n")
	if err := client.StashPush(ctx, "stash two"); err != nil {
		t.Fatalf("StashPush 2: %v", err)
	}

	stashes, err := client.ListStashes(ctx)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 2 {
		t.Fatalf("got %d stashes, want 2: %+v", len(stashes), stashes)
	}
	if !strings.Contains(stashes[0].Message, "stash two") || !strings.Contains(stashes[1].Message, "stash one") {
		t.Errorf("unexpected stash messages: %+v", stashes)
	}

	diff, err := client.ShowStash(ctx, "stash@{1}")
	if err != nil {
		t.Fatalf("ShowStash: %v", err)
	}
	if !strings.Contains(diff, "+first change") {
		t.Errorf("ShowStash missing expected diff:\n%s", diff)
	}

	if err := client.StashApply(ctx, "stash@{1}"); err != nil {
		t.Fatalf("StashApply: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil || string(data) != "first change\n" {
		t.Fatalf("expected first change after StashApply, got %q (err=%v)", string(data), err)
	}

	// Clean working tree before dropping & popping
	runGitTest(t, dir, "checkout", "--", "a.txt")

	if err := client.StashDrop(ctx, "stash@{1}"); err != nil {
		t.Fatalf("StashDrop: %v", err)
	}
	stashes, err = client.ListStashes(ctx)
	if err != nil || len(stashes) != 1 {
		t.Fatalf("expected 1 stash after drop, got %+v (err=%v)", stashes, err)
	}

	if err := client.StashPopRef(ctx, "stash@{0}"); err != nil {
		t.Fatalf("StashPopRef: %v", err)
	}
	stashes, err = client.ListStashes(ctx)
	if err != nil || len(stashes) != 0 {
		t.Fatalf("expected 0 stashes after pop, got %+v (err=%v)", stashes, err)
	}
	data, _ = os.ReadFile(filepath.Join(dir, "a.txt"))
	if string(data) != "second change\n" {
		t.Errorf("expected second change after pop, got %q", string(data))
	}
}

func TestApplyAndDiscardHunk(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)

	var origLines []string
	for i := 1; i <= 25; i++ {
		origLines = append(origLines, "line")
	}
	origLines[1] = "alpha"
	origLines[23] = "omega"
	writeTestFile(t, dir, "multi.txt", strings.Join(origLines, "\n")+"\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "init")

	modLines := append([]string(nil), origLines...)
	modLines[1] = "ALPHA_MOD"
	modLines[23] = "OMEGA_MOD"
	writeTestFile(t, dir, "multi.txt", strings.Join(modLines, "\n")+"\n")

	client := NewClient(dir)
	rawDiff := runGitTest(t, dir, "diff", "--", "multi.txt")

	// Extract headers + individual hunks
	hunk1Idx := strings.Index(rawDiff, "@@")
	hunk2Idx := strings.LastIndex(rawDiff, "@@ -")
	if hunk1Idx < 0 || hunk2Idx <= hunk1Idx {
		t.Fatalf("expected 2 hunks in diff:\n%s", rawDiff)
	}
	headers := rawDiff[:hunk1Idx]
	hunk1Patch := headers + rawDiff[hunk1Idx:hunk2Idx]
	hunk2Patch := headers + rawDiff[hunk2Idx:] + "\n"

	if err := client.ApplyHunkCached(ctx, hunk1Patch, false); err != nil {
		t.Fatalf("ApplyHunkCached stage hunk1: %v", err)
	}

	stagedDiff, err := client.DiffStaged(ctx)
	if err != nil {
		t.Fatalf("DiffStaged: %v", err)
	}
	if !strings.Contains(stagedDiff, "+ALPHA_MOD") || strings.Contains(stagedDiff, "+OMEGA_MOD") {
		t.Errorf("expected only hunk1 staged, got:\n%s", stagedDiff)
	}

	// Discard unstaged hunk2 while keeping staged hunk1
	if err := client.DiscardHunk(ctx, hunk2Patch, false); err != nil {
		t.Fatalf("DiscardHunk hunk2: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "multi.txt"))
	if !strings.Contains(string(data), "ALPHA_MOD") || strings.Contains(string(data), "OMEGA_MOD") {
		t.Errorf("expected ALPHA_MOD preserved and OMEGA_MOD discarded, got:\n%s", string(data))
	}

	// Unstage hunk1
	if err := client.ApplyHunkCached(ctx, hunk1Patch, true); err != nil {
		t.Fatalf("ApplyHunkCached reverse hunk1: %v", err)
	}
	stagedDiff, _ = client.DiffStaged(ctx)
	if strings.TrimSpace(stagedDiff) != "" {
		t.Errorf("expected empty staged diff after unstage hunk1, got:\n%s", stagedDiff)
	}
}

func TestCommitFilesDiffAndPagination(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)

	for i := 1; i <= 5; i++ {
		writeTestFile(t, dir, "a.txt", strings.Repeat("a", i)+"\n")
		writeTestFile(t, dir, "b.txt", strings.Repeat("b", i)+"\n")
		runGitTest(t, dir, "add", ".")
		runGitTest(t, dir, "commit", "-m", "commit "+string(rune('0'+i)))
	}
	runGitTest(t, dir, "tag", "v1.0.0")

	client := NewClient(dir)
	page1, err := client.ListCommitsPage(ctx, "", 2, 0)
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1 got %d (err=%v)", len(page1), err)
	}
	if !strings.Contains(page1[0].Decorations, "v1.0.0") {
		t.Errorf("expected tag decoration on HEAD commit, got %q", page1[0].Decorations)
	}

	page2, err := client.ListCommitsPage(ctx, "", 2, 2)
	if err != nil || len(page2) != 2 {
		t.Fatalf("page2 got %d (err=%v)", len(page2), err)
	}
	if page1[0].Hash == page2[0].Hash || page2[0].Subject != "commit 3" {
		t.Errorf("unexpected page2: %+v", page2)
	}

	files, err := client.ListCommitFiles(ctx, page1[0].Hash)
	if err != nil || len(files) != 2 {
		t.Fatalf("ListCommitFiles got %+v (err=%v)", files, err)
	}

	fileDiff, err := client.DiffCommitFile(ctx, page1[0].Hash, "a.txt")
	if err != nil {
		t.Fatalf("DiffCommitFile: %v", err)
	}
	if !strings.Contains(fileDiff, "a.txt") || strings.Contains(fileDiff, "b.txt") {
		t.Errorf("expected diff scoped to a.txt only, got:\n%s", fileDiff)
	}
}

func TestActiveOp_MergeRebaseContinueAbort(t *testing.T) {
	ctx := context.Background()
	dir := initTestRepo(t)
	writeTestFile(t, dir, "f.txt", "base\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "base")

	client := NewClient(dir)
	if err := client.CreateBranch(ctx, "topic"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	writeTestFile(t, dir, "f.txt", "topic\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "topic change")

	if err := client.SwitchBranch(ctx, "main"); err != nil {
		t.Fatalf("SwitchBranch: %v", err)
	}
	writeTestFile(t, dir, "f.txt", "main\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "main change")

	// Merge conflict -> OpMerge -> Abort
	res, err := client.MergeBranch(ctx, "topic")
	if err != nil || !res.Conflicted {
		t.Fatalf("expected conflicted merge, got res=%+v err=%v", res, err)
	}
	state, err := client.InspectRepo(ctx)
	if err != nil || state.ActiveOp != OpMerge {
		t.Fatalf("expected ActiveOp=OpMerge, got %v (err=%v)", state.ActiveOp, err)
	}
	if err := client.AbortActiveOp(ctx, OpMerge); err != nil {
		t.Fatalf("AbortActiveOp merge: %v", err)
	}
	state, _ = client.InspectRepo(ctx)
	if state.ActiveOp != OpNone {
		t.Fatalf("expected OpNone after abort, got %v", state.ActiveOp)
	}

	// Merge conflict -> Resolve -> Continue
	res, err = client.MergeBranch(ctx, "topic")
	if err != nil || !res.Conflicted {
		t.Fatalf("expected conflicted merge 2, got res=%+v err=%v", res, err)
	}
	if err := client.ResolveConflict(ctx, "f.txt", SideTheirs); err != nil {
		t.Fatalf("ResolveConflict: %v", err)
	}
	if err := client.ContinueActiveOp(ctx, OpMerge); err != nil {
		t.Fatalf("ContinueActiveOp merge: %v", err)
	}
	state, _ = client.InspectRepo(ctx)
	if state.ActiveOp != OpNone || state.ChangedCount != 0 {
		t.Fatalf("expected clean OpNone after continue, got %+v", state)
	}

	// Rebase conflict on topic -> OpRebase -> Abort
	if err := client.SwitchBranch(ctx, "topic"); err != nil {
		t.Fatalf("SwitchBranch topic: %v", err)
	}
	writeTestFile(t, dir, "f.txt", "topic2\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "topic2")

	// Switch back to create divergence with main
	runGitTest(t, dir, "checkout", "main")
	writeTestFile(t, dir, "f.txt", "main2\n")
	runGitTest(t, dir, "add", ".")
	runGitTest(t, dir, "commit", "-m", "main2")
	runGitTest(t, dir, "checkout", "topic")

	rebaseRes, err := client.RebaseBranch(ctx, "main")
	if err != nil || !rebaseRes.Conflicted {
		t.Fatalf("expected rebase conflict, got res=%+v err=%v", rebaseRes, err)
	}
	state, _ = client.InspectRepo(ctx)
	if state.ActiveOp != OpRebase {
		t.Fatalf("expected ActiveOp=OpRebase, got %v", state.ActiveOp)
	}
	if err := client.AbortActiveOp(ctx, OpRebase); err != nil {
		t.Fatalf("AbortActiveOp rebase: %v", err)
	}
	state, _ = client.InspectRepo(ctx)
	if state.ActiveOp != OpNone {
		t.Fatalf("expected OpNone after rebase abort, got %v", state.ActiveOp)
	}
}

func TestBranches_RemoteTrackRenameAndRebase(t *testing.T) {
	ctx := context.Background()
	bare := t.TempDir()
	runGitTest(t, bare, "init", "--bare", "-b", "main")

	repoA := initTestRepo(t)
	runGitTest(t, repoA, "remote", "add", "origin", bare)
	writeTestFile(t, repoA, "a.txt", "base\n")
	runGitTest(t, repoA, "add", ".")
	runGitTest(t, repoA, "commit", "-m", "base")
	runGitTest(t, repoA, "push", "-u", "origin", "main")

	runGitTest(t, repoA, "checkout", "-b", "remote-feat")
	writeTestFile(t, repoA, "r.txt", "remote\n")
	runGitTest(t, repoA, "add", ".")
	runGitTest(t, repoA, "commit", "-m", "remote feat")
	runGitTest(t, repoA, "push", "-u", "origin", "remote-feat")

	repoBParent := t.TempDir()
	repoB := filepath.Join(repoBParent, "clone")
	cmd := exec.Command("git", "clone", bare, repoB)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone failed: %v\n%s", err, string(out))
	}
	runGitTest(t, repoB, "config", "user.name", "User B")
	runGitTest(t, repoB, "config", "user.email", "b@example.com")

	clientB := NewClient(repoB)
	if err := clientB.SwitchBranch(ctx, "origin/remote-feat"); err != nil {
		t.Fatalf("SwitchBranch remote-feat: %v", err)
	}
	state, err := clientB.InspectRepo(ctx)
	if err != nil {
		t.Fatalf("InspectRepo: %v", err)
	}
	if state.Branch != "remote-feat" || state.Upstream != "origin/remote-feat" {
		t.Errorf("expected tracked local branch remote-feat -> origin/remote-feat, got %+v", state)
	}

	if err := clientB.RenameBranch(ctx, "remote-feat", "local-renamed"); err != nil {
		t.Fatalf("RenameBranch: %v", err)
	}
	state, _ = clientB.InspectRepo(ctx)
	if state.Branch != "local-renamed" {
		t.Errorf("expected renamed branch local-renamed, got %q", state.Branch)
	}

	// Clean rebase on main
	runGitTest(t, repoB, "checkout", "main")
	writeTestFile(t, repoB, "m.txt", "main new\n")
	runGitTest(t, repoB, "add", ".")
	runGitTest(t, repoB, "commit", "-m", "main new")
	runGitTest(t, repoB, "checkout", "local-renamed")

	rebaseRes, err := clientB.RebaseBranch(ctx, "main")
	if err != nil || rebaseRes.Conflicted {
		t.Fatalf("expected clean rebase, got res=%+v err=%v", rebaseRes, err)
	}
	if _, err := os.Stat(filepath.Join(repoB, "m.txt")); err != nil {
		t.Errorf("expected m.txt after rebase: %v", err)
	}
}
