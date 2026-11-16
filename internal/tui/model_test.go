package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
)

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
	for _, want := range []string{"origin/main", "↑2", "↓1"} {
		if !strings.Contains(header, want) {
			t.Errorf("header missing %q:\n%s", want, header)
		}
	}

	footer := m.renderFooter()
	if !strings.Contains(footer, "Executado: git push -u origin main") {
		t.Errorf("footer missing command preview:\n%s", footer)
	}
}
