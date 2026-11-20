package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
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
			RepoName:     "go-vit",
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
			for _, tab := range []tabKind{tabBranches, tabChanges, tabCommits} {
				m := NewModel(git.NewClient("."))
				updated, _ := m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
				m = updated.(Model)

				updated, _ = m.Update(repoLoadedMsg{
					state: git.RepoState{
						RepoName:      "go-vit-stress-repo-name",
						Kind:          git.KindNormal,
						Branch:        branches[0].Name,
						Upstream:      "origin/very-long-upstream-branch-name",
						Ahead:         3,
						Behind:        2,
						ChangedCount:  len(files),
						ConflictCount: 1,
					},
					branches: branches,
					status:   files,
					lastCmd:  "git status --porcelain=v1 --branch --verbose",
				})
				m = updated.(Model)
				m.commits = commits
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
