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
