package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/FelipeSrutkoske/Ygg/internal/ai"
	"github.com/FelipeSrutkoske/Ygg/internal/config"
	"github.com/FelipeSrutkoske/Ygg/internal/git"
)

var ansiSeqRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func keyRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func keySpecial(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func setupLoadedTestModel() Model {
	m := NewModel(git.NewClient("."))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = updated.(Model)

	updated, _ = m.Update(repoLoadedMsg{
		state: git.RepoState{
			RepoName:     "ygg",
			Kind:         git.KindNormal,
			Branch:       "main",
			Upstream:     "origin/main",
			Ahead:        2,
			Behind:       1,
			ChangedCount: 2,
		},
		branches: []git.Branch{
			{Name: "main", IsCurrent: true, Upstream: "origin/main", Ahead: 2, Behind: 1},
			{Name: "feat/diff", IsCurrent: false},
		},
		status: []git.FileStatus{
			{Path: "main.go", IsUnstaged: true, WorkStatus: 'M'},
			{Path: "README.md", IsStaged: true, IndexStatus: 'M'},
		},
		lastCmd: "git status --porcelain=v1 --branch",
	})
	m = updated.(Model)

	updated, _ = m.Update(commitsLoadedMsg{
		ref: "",
		commits: []git.Commit{
			{Hash: "abc1234", Subject: "feat: add push", Author: "Felipe", RelativeDate: "2 hours ago"},
			{Hash: "def5678", Subject: "fix: parser", Author: "Ana", RelativeDate: "1 day ago"},
		},
	})
	return updated.(Model)
}

func TestTabSwitching_AndHistoryTab(t *testing.T) {
	m := setupLoadedTestModel()
	if m.activeTab != tabBranches {
		t.Fatalf("expected initial tabBranches, got %v", m.activeTab)
	}

	// Tab -> Changes
	updated, _ := m.Update(keySpecial(tea.KeyTab))
	m = updated.(Model)
	if m.activeTab != tabChanges {
		t.Fatalf("expected tabChanges after Tab, got %v", m.activeTab)
	}

	// Tab -> Commits (Histórico)
	updated, cmd := m.Update(keySpecial(tea.KeyTab))
	m = updated.(Model)
	if m.activeTab != tabCommits {
		t.Fatalf("expected tabCommits after second Tab, got %v", m.activeTab)
	}
	if cmd == nil {
		t.Errorf("expected loadCommitsCmd when entering tabCommits")
	}

	// Direct jump with '1', '2', '3'
	updated, _ = m.Update(keyRune('1'))
	m = updated.(Model)
	if m.activeTab != tabBranches {
		t.Fatalf("expected tabBranches after '1', got %v", m.activeTab)
	}

	updated, _ = m.Update(keyRune('3'))
	m = updated.(Model)
	if m.activeTab != tabCommits {
		t.Fatalf("expected tabCommits after '3', got %v", m.activeTab)
	}
}

func TestQuickFilter_AcrossTabs(t *testing.T) {
	m := setupLoadedTestModel()

	// Start filtering in Branches tab
	updated, _ := m.Update(keyRune('/'))
	m = updated.(Model)
	if !m.filtering {
		t.Fatalf("expected filtering=true after '/'")
	}

	for _, r := range "feat" {
		updated, _ = m.Update(keyRune(r))
		m = updated.(Model)
	}
	if m.filterQuery != "feat" {
		t.Fatalf("filterQuery = %q, want feat", m.filterQuery)
	}
	visBranches := m.visibleBranches()
	if len(visBranches) != 1 || visBranches[0].Name != "feat/diff" {
		t.Fatalf("unexpected visibleBranches: %+v", visBranches)
	}

	// Enter keeps filterQuery active while leaving typing mode
	updated, _ = m.Update(keySpecial(tea.KeyEnter))
	m = updated.(Model)
	if m.filtering {
		t.Fatalf("expected filtering=false after Enter")
	}
	if m.filterQuery != "feat" {
		t.Fatalf("expected filterQuery to stay 'feat' after Enter, got %q", m.filterQuery)
	}

	// Esc clears the filter
	updated, _ = m.Update(keySpecial(tea.KeyEscape))
	m = updated.(Model)
	if m.filterQuery != "" {
		t.Fatalf("expected filterQuery cleared after Esc, got %q", m.filterQuery)
	}
	if len(m.visibleBranches()) != 2 {
		t.Fatalf("expected all 2 branches after clearing filter, got %d", len(m.visibleBranches()))
	}
}

func TestDiffModal_VSCodeToggleAndClose(t *testing.T) {
	m := setupLoadedTestModel()

	updated, _ := m.Update(keyRune('2'))
	m = updated.(Model)

	updated, _ = m.Update(diffLoadedMsg{
		path: "main.go",
		diff: "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1,2 +1,2 @@\n-oldLine\n+newLine\n keep\n",
	})
	m = updated.(Model)

	// Open VS Code diff modal with Enter
	updated, _ = m.Update(keySpecial(tea.KeyEnter))
	m = updated.(Model)
	if m.modal != modalDiffView {
		t.Fatalf("expected modalDiffView after Enter, got %v", m.modal)
	}
	if !m.sideBySideDiff {
		t.Fatalf("expected sideBySideDiff=true on 120-col terminal")
	}

	viewSideBySide := m.renderDiffModal()
	if !strings.Contains(viewSideBySide, "[Lado a Lado]") || !strings.Contains(viewSideBySide, "Original (Antes)") {
		t.Errorf("expected side-by-side diff modal content, got:\n%s", viewSideBySide)
	}

	// Toggle to unified with 'l'
	updated, _ = m.Update(keyRune('l'))
	m = updated.(Model)
	if m.sideBySideDiff {
		t.Fatalf("expected sideBySideDiff=false after 'l'")
	}
	viewUnified := m.renderDiffModal()
	if !strings.Contains(viewUnified, "[Unificado]") || !strings.Contains(viewUnified, "Visualizacao Unificada") {
		t.Errorf("expected unified diff modal content, got:\n%s", viewUnified)
	}

	// Close with Esc
	updated, _ = m.Update(keySpecial(tea.KeyEscape))
	m = updated.(Model)
	if m.modal != modalNone {
		t.Fatalf("expected modalNone after Esc, got %v", m.modal)
	}
}

func TestHeaderAndFooter_UpstreamAndCommandPreview(t *testing.T) {
	m := setupLoadedTestModel()
	updated, _ := m.Update(actionDoneMsg{
		message: "Push concluido",
		lastCmd: "git push -u origin main",
	})
	m = updated.(Model)

	header := m.renderHeader()
	for _, want := range []string{"origin/main", "↑2 push (P)", "↓1 pull (p)"} {
		if !strings.Contains(header, want) {
			t.Errorf("header missing %q:\n%s", want, header)
		}
	}

	footer := m.renderFooter()
	if !strings.Contains(footer, "Executado: git push -u origin main") {
		t.Errorf("footer missing command preview:\n%s", footer)
	}
}

func assertClosedBox(t *testing.T, name, rendered string, wantW, wantH int) {
	t.Helper()
	if gotW := lipgloss.Width(rendered); gotW != wantW {
		t.Errorf("%s width = %d, want %d\n%s", name, gotW, wantW, rendered)
	}
	if gotH := lipgloss.Height(rendered); gotH != wantH {
		t.Errorf("%s height = %d, want %d\n%s", name, gotH, wantH, rendered)
	}
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		t.Fatalf("%s rendered empty", name)
	}
	first := ansiSeqRe.ReplaceAllString(lines[0], "")
	last := ansiSeqRe.ReplaceAllString(lines[len(lines)-1], "")
	if !strings.HasPrefix(first, "╭") || !strings.HasSuffix(first, "╮") {
		t.Errorf("%s top border malformed: %q", name, first)
	}
	if !strings.HasPrefix(last, "╰") || !strings.HasSuffix(last, "╯") {
		t.Errorf("%s bottom border malformed: %q", name, last)
	}
}

func TestPaneAlignment_AllTabsAndSizes(t *testing.T) {
	longSubj := "✨ feat: " + strings.Repeat("extremely_long_commit_subject_with_unicode_çã_and_tabs\t", 3) + " 🚀"
	longAuthor := "Felipe Srutkoske With A Very Long Author Name For Testing Overflow"

	var commits []git.Commit
	for i := 0; i < 40; i++ {
		commits = append(commits, git.Commit{
			Hash:         fmt.Sprintf("abc%04d", i),
			Subject:      fmt.Sprintf("[%02d] %s", i, longSubj),
			Author:       longAuthor,
			RelativeDate: "2 days ago",
		})
	}

	var branches []git.Branch
	for i := 0; i < 30; i++ {
		branches = append(branches, git.Branch{
			Name:     fmt.Sprintf("feature/very-long-branch-name-for-alignment-testing-%02d", i),
			IsRemote: i%3 == 0,
			Ahead:    i,
			Behind:   i % 5,
		})
	}
	branches[0].IsCurrent = true
	branches[0].Upstream = "origin/main"

	var files []git.FileStatus
	files = append(files, git.FileStatus{
		Path:         "internal/tui/very/deep/nested/conflicted_file_with_super_long_name.go",
		IsConflicted: true,
	})
	for i := 1; i < 25; i++ {
		files = append(files, git.FileStatus{
			Path:       fmt.Sprintf("internal/pkg/module_%02d/subpackage/extremely_long_file_name_to_test_wrapping.go", i),
			IsStaged:   i%2 == 0,
			IsUnstaged: i%2 == 1,
		})
	}

	longCodeLine := "\t\treturn fmt.Sprintf(\"%s\", " + strings.Repeat("veryLongVariableName+", 10) + "\"end\")"
	stressDiff := fmt.Sprintf(`diff --git a/internal/tui/views.go b/internal/tui/views.go
index 1111111..2222222 100644
--- a/internal/tui/views.go
+++ b/internal/tui/views.go
@@ -12000,4 +12000,5 @@ func veryLongFunctionSignatureWithManyParametersThatExceedsPaneWidth(ctx context.Context, cfg Config) error {
 %s
-%s
+%s
+%s
 %s
`, longCodeLine, longCodeLine, longCodeLine, longCodeLine, longCodeLine)

	conflictDiff := strings.Join([]string{
		"<<<<<<< HEAD",
		longCodeLine,
		"=======",
		longCodeLine,
		">>>>>>> feature/branch",
	}, "\n")

	sizes := []struct {
		w, h int
	}{
		{80, 24},
		{120, 36},
		{190, 48},
		{260, 65},
	}

	for _, sz := range sizes {
		for _, filtering := range []bool{false, true} {
			for _, tab := range []tabKind{tabBranches, tabChanges, tabCommits, tabStashes} {
				m := NewModel(git.NewClient("."))
				updated, _ := m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
				m = updated.(Model)

				updated, _ = m.Update(repoLoadedMsg{
					state: git.RepoState{
						RepoName:      "ygg-stress-repo-name",
						Kind:          git.KindNormal,
						ActiveOp:      git.OpMerge,
						Branch:        branches[0].Name,
						Upstream:      "origin/very-long-upstream-branch-name",
						Ahead:         3,
						Behind:        2,
						ChangedCount:  len(files),
						ConflictCount: 1,
					},
					branches: branches,
					status:   files,
					stashes: []git.StashEntry{
						{Index: 0, Ref: "stash@{0}", Message: "WIP on " + longSubj, RelativeDate: "5 minutes ago"},
						{Index: 1, Ref: "stash@{1}", Message: "very-long stash message for filter alignment", RelativeDate: "2 days ago"},
					},
					lastCmd: "git status --porcelain=v1 --branch --verbose",
				})
				m = updated.(Model)
				m.commits = commits
				m.commitFiles = []git.CommitFile{
					{Status: "M", Path: "internal/tui/views.go"},
					{Status: "A", Path: "internal/git/new_file.go"},
				}
				m.commitFileIdx = 1
				m.activeTab = tab
				m.filtering = filtering
				m.filterQuery = "very-long"

				cases := []struct {
					label   string
					fileIdx int
					diff    string
				}{
					{label: "normal", fileIdx: 1, diff: stressDiff},
				}
				if tab == tabChanges {
					cases = append(cases, struct {
						label   string
						fileIdx int
						diff    string
					}{label: "conflict", fileIdx: 0, diff: conflictDiff})
				}

				for _, tc := range cases {
					m.fileIdx = tc.fileIdx
					m.diffText = tc.diff
					m.commitDiffText = stressDiff
					m.stashDiffText = stressDiff
					m.syncRightPane()

					t.Run(fmt.Sprintf("%dx%d_tab%d_filter%v_%s", sz.w, sz.h, tab, filtering, tc.label), func(t *testing.T) {
						header := m.renderHeader()
						if lipgloss.Height(header) != 3 {
							t.Errorf("header height = %d, want 3\n%s", lipgloss.Height(header), header)
						}
						if lipgloss.Width(header) != sz.w {
							t.Errorf("header width = %d, want %d\n%s", lipgloss.Width(header), sz.w, header)
						}

						leftW, leftH, rightW, rightH := m.paneDimensions()
						left := m.renderLeftPane(leftW, leftH)
						right := m.renderRightPane(rightW, rightH)

						assertClosedBox(t, "left", left, leftW, leftH)
						assertClosedBox(t, "right", right, rightW, rightH)

						if sz.w >= 90 && lipgloss.Height(left) != lipgloss.Height(right) {
							t.Errorf("left height (%d) != right height (%d)", lipgloss.Height(left), lipgloss.Height(right))
						}

						mainView := m.renderMain()
						if gotH := lipgloss.Height(mainView); gotH != sz.h {
							t.Errorf("main view height = %d, want %d", gotH, sz.h)
						}
					})
				}
			}
		}
	}
}

func TestCommitList_AllRowsVisibleAndTabLabel(t *testing.T) {
	m := setupLoadedTestModel()
	m.commits = []git.Commit{
		{Hash: "aaa1111", Subject: "first", Author: "Alice", RelativeDate: "1m ago"},
		{Hash: "\nbbb2222", Subject: "second", Author: "Bob", RelativeDate: "2m ago"},
		{Hash: "\r\nccc3333", Subject: "third", Author: "Carol", RelativeDate: "3m ago"},
	}
	m.activeTab = tabCommits
	m.commitIdx = 2

	renderedList := ansiSeqRe.ReplaceAllString(m.renderCommitList(60, 10), "")
	for _, want := range []string{"aaa1111", "bbb2222", "ccc3333"} {
		if !strings.Contains(renderedList, want) {
			t.Errorf("renderCommitList missing hash %q:\n%s", want, renderedList)
		}
	}

	tabsRow := ansiSeqRe.ReplaceAllString(m.renderTabsRow(40), "")
	if !strings.Contains(tabsRow, "[2] Change") {
		t.Errorf("expected '[2] Change' in tabs row, got %q", tabsRow)
	}
	if strings.Contains(tabsRow, "[2] Alt") {
		t.Errorf("unexpected '[2] Alt' in tabs row, got %q", tabsRow)
	}
}

func TestTab4_StashNavigationAndModals(t *testing.T) {
	m := setupLoadedTestModel()
	m.stashes = []git.StashEntry{
		{Index: 0, Ref: "stash@{0}", Message: "wip first", RelativeDate: "2 mins ago"},
		{Index: 1, Ref: "stash@{1}", Message: "wip second", RelativeDate: "1 hour ago"},
	}

	// Jump to tab 4 (Stashes)
	updated, _ := m.Update(keyRune('4'))
	m = updated.(Model)
	if m.activeTab != tabStashes {
		t.Fatalf("expected tabStashes after '4', got %v", m.activeTab)
	}

	// Move down to second stash
	updated, _ = m.Update(keyRune('j'))
	m = updated.(Model)
	if m.stashIdx != 1 {
		t.Fatalf("expected stashIdx=1 after 'j', got %d", m.stashIdx)
	}

	// Open drop stash modal with 'x'
	updated, _ = m.Update(keyRune('x'))
	m = updated.(Model)
	if m.modal != modalDropStash || m.modalTarget != "stash@{1}" {
		t.Fatalf("expected modalDropStash for stash@{1}, got modal=%v target=%q", m.modal, m.modalTarget)
	}
	updated, _ = m.Update(keySpecial(tea.KeyEscape))
	m = updated.(Model)
	if m.modal != modalNone {
		t.Fatalf("expected modalNone after Esc")
	}

	// Switch to Changes tab and press 'z' -> modalCreateStash
	updated, _ = m.Update(keyRune('2'))
	m = updated.(Model)
	updated, _ = m.Update(keyRune('z'))
	m = updated.(Model)
	if m.modal != modalCreateStash {
		t.Fatalf("expected modalCreateStash on dirty repo, got %v", m.modal)
	}
}

func TestDiffModal_HunkNavigationAndDiscardModal(t *testing.T) {
	m := setupLoadedTestModel()
	updated, _ := m.Update(keyRune('2'))
	m = updated.(Model)

	twoHunkDiff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,2 @@
-old1
+new1
 keep
@@ -10,2 +10,2 @@
-old2
+new2
 end
`
	updated, _ = m.Update(diffLoadedMsg{path: "main.go", diff: twoHunkDiff})
	m = updated.(Model)

	updated, _ = m.Update(keySpecial(tea.KeyEnter))
	m = updated.(Model)
	if m.modal != modalDiffView || m.activeHunkIdx != 0 {
		t.Fatalf("expected modalDiffView with hunk 0, got modal=%v hunk=%d", m.modal, m.activeHunkIdx)
	}

	// Next hunk with 'n'
	updated, _ = m.Update(keyRune('n'))
	m = updated.(Model)
	if m.activeHunkIdx != 1 {
		t.Fatalf("expected activeHunkIdx=1 after 'n', got %d", m.activeHunkIdx)
	}

	// Open discard hunk confirmation with 'x'
	updated, _ = m.Update(keyRune('x'))
	m = updated.(Model)
	if m.modal != modalDiscardHunk || !strings.Contains(m.modalTarget, "Hunk 2/2") {
		t.Fatalf("expected modalDiscardHunk for Hunk 2/2, got modal=%v target=%q", m.modal, m.modalTarget)
	}

	// Esc cancels back to diff modal
	updated, _ = m.Update(keySpecial(tea.KeyEscape))
	m = updated.(Model)
	if m.modal != modalDiffView {
		t.Fatalf("expected return to modalDiffView after Esc, got %v", m.modal)
	}
}

func TestCommitsTab_FileBarDecorationsAndActiveOp(t *testing.T) {
	m := setupLoadedTestModel()
	m.state.ActiveOp = git.OpRebase

	header := ansiSeqRe.ReplaceAllString(m.renderHeader(), "")
	if !strings.Contains(header, "REBASE EM ANDAMENTO") {
		t.Errorf("expected REBASE EM ANDAMENTO in header, got:\n%s", header)
	}

	// Press 'A' to open abort confirmation modal
	updated, _ := m.Update(keyRune('A'))
	m = updated.(Model)
	if m.modal != modalAbortOp {
		t.Fatalf("expected modalAbortOp after 'A', got %v", m.modal)
	}
	updated, _ = m.Update(keySpecial(tea.KeyEscape))
	m = updated.(Model)

	// Switch to tabCommits and load commit files + decorations
	updated, _ = m.Update(keyRune('3'))
	m = updated.(Model)
	m.commits[0].Decorations = "HEAD -> main, tag: v1.2.0"

	commitList := ansiSeqRe.ReplaceAllString(m.renderCommitList(90, 12), "")
	if !strings.Contains(commitList, "tag: v1.2.0") {
		t.Errorf("expected commit decoration in commit list:\n%s", commitList)
	}

	updated, _ = m.Update(commitDiffLoadedMsg{
		hash: "abc1234",
		files: []git.CommitFile{
			{Status: "M", Path: "main.go"},
			{Status: "A", Path: "helper.go"},
		},
		diff: "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-a\n+b\n",
	})
	m = updated.(Model)

	updated, cmd := m.Update(keySpecial(tea.KeyRight))
	m = updated.(Model)
	if m.commitFileIdx != 1 || cmd == nil {
		t.Fatalf("expected commitFileIdx=1 and cmd != nil after Right, got idx=%d cmd=%v", m.commitFileIdx, cmd)
	}

	right := ansiSeqRe.ReplaceAllString(m.renderRightPane(70, 20), "")
	if !strings.Contains(right, "Arq 1/2: M main.go") {
		t.Errorf("expected commit file bar in right pane:\n%s", right)
	}
}

func TestBranchesTab_RenameModal(t *testing.T) {
	m := setupLoadedTestModel()
	updated, _ := m.Update(keyRune('M'))
	m = updated.(Model)
	if m.modal != modalRenameBranch || m.modalTarget != "main" {
		t.Fatalf("expected modalRenameBranch for main, got modal=%v target=%q", m.modal, m.modalTarget)
	}
	if m.input.Value() != "main" {
		t.Errorf("expected input pre-filled with 'main', got %q", m.input.Value())
	}
}

func TestAutoRefresh_TickAndFocus_PreservesScrollAndSkipsDuringInputModal(t *testing.T) {
	m := setupLoadedTestModel()
	updated, _ := m.Update(keyRune('2'))
	m = updated.(Model)

	// Load a long diff so viewport can scroll
	var lines []string
	lines = append(lines, "diff --git a/main.go b/main.go", "--- a/main.go", "+++ b/main.go", "@@ -1,40 +1,40 @@")
	for i := 0; i < 40; i++ {
		lines = append(lines, fmt.Sprintf("+line %d", i))
	}
	longDiff := strings.Join(lines, "\n") + "\n"
	updated, _ = m.Update(diffLoadedMsg{path: "main.go", diff: longDiff})
	m = updated.(Model)
	m.viewport.SetYOffset(5)

	// Silent diffLoadedMsg with same content should preserve scroll
	updated, _ = m.Update(diffLoadedMsg{path: "main.go", diff: longDiff, silent: true})
	m = updated.(Model)
	if m.viewport.YOffset() != 5 {
		t.Fatalf("expected YOffset=5 preserved on unchanged silent diff, got %d", m.viewport.YOffset())
	}

	// Silent diffLoadedMsg with updated content should preserve scroll
	updated, _ = m.Update(diffLoadedMsg{path: "main.go", diff: longDiff + "+extra\n", silent: true})
	m = updated.(Model)
	if m.viewport.YOffset() != 5 {
		t.Fatalf("expected YOffset=5 preserved on updated silent diff, got %d", m.viewport.YOffset())
	}

	// FocusMsg triggers refresh cmd when idle
	_, focusCmd := m.Update(tea.FocusMsg{})
	if focusCmd == nil {
		t.Fatalf("expected FocusMsg to trigger silent refresh cmd")
	}

	// AutoRefreshTickMsg triggers refresh cmd when idle
	_, tickCmd := m.Update(autoRefreshTickMsg{})
	if tickCmd == nil {
		t.Fatalf("expected autoRefreshTickMsg to return batch cmd")
	}

	// Open commit modal: shouldAutoRefresh becomes false
	m.state.StagedCount = 1
	updated, _ = m.Update(keyRune('c'))
	m = updated.(Model)
	if m.shouldAutoRefresh() {
		t.Fatalf("expected shouldAutoRefresh()=false when modalCommit is open")
	}
	_, focusWhileModal := m.Update(tea.FocusMsg{})
	if focusWhileModal != nil {
		t.Fatalf("expected nil cmd on FocusMsg while modalCommit is open")
	}
}

func TestAPIKeyModal_CtrlK_Paste_AndUnauthorizedRePrompt(t *testing.T) {
	tmpCfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpCfg)
	t.Setenv("AI_API_KEY", "")

	m := setupLoadedTestModel()

	// Ctrl+K opens modalAPIKey from main screen
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = updated.(Model)
	if m.modal != modalAPIKey {
		t.Fatalf("expected modalAPIKey after Ctrl+K, got %v", m.modal)
	}

	// PasteMsg trims whitespace and populates input
	updated, _ = m.Update(tea.PasteMsg{Content: "  nvapi-test-12345678  \n"})
	m = updated.(Model)
	if m.input.Value() != "nvapi-test-12345678" {
		t.Fatalf("expected trimmed pasted key, got %q", m.input.Value())
	}

	// Enter saves key and returns to modalNone
	updated, _ = m.Update(keySpecial(tea.KeyEnter))
	m = updated.(Model)
	if m.modal != modalNone {
		t.Fatalf("expected modalNone after saving key, got %v", m.modal)
	}
	cfg, err := config.Load()
	if err != nil || cfg.AIAPIKey != "nvapi-test-12345678" {
		t.Fatalf("expected saved key in config, got %q (err=%v)", cfg.AIAPIKey, err)
	}

	// Unauthorized error from AI generation re-opens modalAPIKey
	m.modal = modalCommit
	updated, _ = m.Update(commitMsgGeneratedMsg{err: ai.ErrUnauthorized})
	m = updated.(Model)
	if m.modal != modalAPIKey || m.apiKeyReturnModal != modalCommit {
		t.Fatalf("expected modalAPIKey returning to modalCommit on 401/403, got modal=%v return=%v", m.modal, m.apiKeyReturnModal)
	}
	if !strings.Contains(m.renderModal(), "401/403") {
		t.Errorf("expected 401/403 error rendered in modalAPIKey")
	}
}
