package tui

import (
	"strings"
	"testing"
)

func TestParseUnifiedDiff_LineNumbersAndCounts(t *testing.T) {
	raw := `diff --git a/main.go b/main.go
index 1111111..2222222 100644
--- a/main.go
+++ b/main.go
@@ -10,4 +10,5 @@ func hello() {
 	fmt.Println("keep")
-	fmt.Println("old")
+	fmt.Println("new1")
+	fmt.Println("new2")
 	return
`

	parsed := parseUnifiedDiff(raw)
	if parsed.added != 2 {
		t.Errorf("added = %d, want 2", parsed.added)
	}
	if parsed.removed != 1 {
		t.Errorf("removed = %d, want 1", parsed.removed)
	}

	var foundOld, foundNew1, foundNew2 bool
	for _, dl := range parsed.lines {
		if dl.kind == lineDel && strings.Contains(dl.text, `"old"`) {
			foundOld = true
			if dl.oldNum != 11 {
				t.Errorf("oldNum = %d, want 11", dl.oldNum)
			}
		}
		if dl.kind == lineAdd && strings.Contains(dl.text, `"new1"`) {
			foundNew1 = true
			if dl.newNum != 11 {
				t.Errorf("newNum = %d, want 11", dl.newNum)
			}
		}
		if dl.kind == lineAdd && strings.Contains(dl.text, `"new2"`) {
			foundNew2 = true
			if dl.newNum != 12 {
				t.Errorf("newNum = %d, want 12", dl.newNum)
			}
		}
	}

	if !foundOld || !foundNew1 || !foundNew2 {
		t.Fatalf("missing expected diff lines: old=%v new1=%v new2=%v", foundOld, foundNew1, foundNew2)
	}
}

func TestRenderVSCodeDiff_UnifiedAndSideBySide(t *testing.T) {
	raw := `diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,3 +1,3 @@
 alpha
-beta
+BETA
 gamma
`
	st := defaultStyles()

	unified := renderVSCodeUnifiedDiff(raw, 80, st)
	for _, want := range []string{"+1", "-1", "┃-", "┃+", "beta", "BETA", "Visualizacao Unificada"} {
		if !strings.Contains(unified, want) {
			t.Errorf("unified diff missing %q:\n%s", want, unified)
		}
	}

	sideBySide := renderVSCodeSideBySideDiff(raw, 100, st)
	for _, want := range []string{"+1", "-1", "Original (Antes)", "Modificado (Depois)", "beta", "BETA", "┃-", "┃+"} {
		if !strings.Contains(sideBySide, want) {
			t.Errorf("side-by-side diff missing %q:\n%s", want, sideBySide)
		}
	}
}

func TestParseUnifiedDiff_HunkPatchesAndStagedSections(t *testing.T) {
	raw := "=== Staged ===\n" +
		"diff --git a/app.go b/app.go\n" +
		"index 1111111..2222222 100644\n" +
		"--- a/app.go\n" +
		"+++ b/app.go\n" +
		"@@ -1,2 +1,2 @@\n" +
		"-\tprintln(\"a\")\n" +
		"+\tprintln(\"b\")\n" +
		" keep\n" +
		"\n" +
		"=== Unstaged ===\n" +
		"diff --git a/app.go b/app.go\n" +
		"index 2222222..3333333 100644\n" +
		"--- a/app.go\n" +
		"+++ b/app.go\n" +
		"@@ -10,2 +10,2 @@\n" +
		"-\tprintln(\"x\")\n" +
		"+\tprintln(\"y\")\n" +
		" end\n"

	parsed := parseUnifiedDiff(raw)
	if len(parsed.hunks) != 2 {
		t.Fatalf("got %d hunks, want 2", len(parsed.hunks))
	}
	if !parsed.hunks[0].staged {
		t.Errorf("expected hunk 0 to be staged=true")
	}
	if parsed.hunks[1].staged {
		t.Errorf("expected hunk 1 to be staged=false")
	}
	if !strings.Contains(parsed.hunks[0].patch, "-\tprintln(\"a\")\n") {
		t.Errorf("expected original tab preserved in hunk 0 patch:\n%q", parsed.hunks[0].patch)
	}
	if !strings.HasPrefix(parsed.hunks[0].patch, "diff --git a/app.go b/app.go\n") {
		t.Errorf("expected file header in hunk 0 patch:\n%s", parsed.hunks[0].patch)
	}
	if strings.Contains(parsed.hunks[0].patch, "=== Unstaged ===") {
		t.Errorf("hunk 0 patch leaked unstaged header:\n%s", parsed.hunks[0].patch)
	}
}

func TestRenderVSCodeDiff_IntraLineHighlightAndActiveHunk(t *testing.T) {
	prefix, oe, ne, ok := computeIntraLineSpan("value := 100", "value := 250")
	if !ok || prefix != 9 || oe != 11 || ne != 11 {
		t.Fatalf("computeIntraLineSpan = (%d, %d, %d, %v), want (9, 11, 11, true)", prefix, oe, ne, ok)
	}

	raw := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,2 @@
-value := 100
+value := 250
 keep
@@ -10,2 +10,2 @@
-flag := false
+flag := true
 end
`
	st := defaultStyles()
	rendered, offset := renderVSCodeSideBySideDiffHunk(raw, 100, 1, st)
	if offset <= 3 {
		t.Errorf("expected offset > 3 for second hunk, got %d", offset)
	}
	if !strings.Contains(rendered, "▶ Hunk 2/2") {
		t.Errorf("expected active hunk marker in rendered diff:\n%s", rendered)
	}
	if !strings.Contains(rendered, "10") || !strings.Contains(rendered, "25") {
		t.Errorf("expected intra-line content preserved:\n%s", rendered)
	}
}
