package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

type diffLineKind int

const (
	lineHeader diffLineKind = iota
	lineMeta
	lineHunk
	lineContext
	lineAdd
	lineDel
)

type diffLine struct {
	kind   diffLineKind
	oldNum int
	newNum int
	text   string
}

type parsedDiff struct {
	added   int
	removed int
	lines   []diffLine
}

func parseUnifiedDiff(raw string) parsedDiff {
	if strings.TrimSpace(raw) == "" {
		return parsedDiff{}
	}

	rawLines := strings.Split(raw, "\n")
	if len(rawLines) > 0 && rawLines[len(rawLines)-1] == "" {
		rawLines = rawLines[:len(rawLines)-1]
	}

	var res parsedDiff
	inHunk := false
	oldLine := 0
	newLine := 0

	for _, rawLine := range rawLines {
		line := strings.ReplaceAll(rawLine, "\t", "    ")
		switch {
		case isDiffMetaLine(line):
			inHunk = false
			res.lines = append(res.lines, diffLine{kind: lineMeta, text: line})

		case strings.HasPrefix(line, "@@"):
			if o, n, ok := parseHunkHeader(line); ok {
				oldLine = o
				newLine = n
				inHunk = true
			} else {
				inHunk = false
			}
			res.lines = append(res.lines, diffLine{kind: lineHunk, text: line})

		case inHunk:
			switch {
			case strings.HasPrefix(line, "+"):
				res.added++
				res.lines = append(res.lines, diffLine{
					kind:   lineAdd,
					newNum: newLine,
					text:   line[1:],
				})
				newLine++
			case strings.HasPrefix(line, "-"):
				res.removed++
				res.lines = append(res.lines, diffLine{
					kind:   lineDel,
					oldNum: oldLine,
					text:   line[1:],
				})
				oldLine++
			case strings.HasPrefix(line, `\`):
				res.lines = append(res.lines, diffLine{kind: lineMeta, text: line})
			default:
				text := line
				if strings.HasPrefix(line, " ") {
					text = line[1:]
				}
				res.lines = append(res.lines, diffLine{
					kind:   lineContext,
					oldNum: oldLine,
					newNum: newLine,
					text:   text,
				})
				oldLine++
				newLine++
			}

		default:
			res.lines = append(res.lines, diffLine{kind: lineHeader, text: line})
		}
	}

	return res
}

func isDiffMetaLine(line string) bool {
	prefixes := []string{
		"diff --git ",
		"index ",
		"--- ",
		"+++ ",
		"new file mode ",
		"deleted file mode ",
		"old mode ",
		"new mode ",
		"similarity index ",
		"rename from ",
		"rename to ",
		"copy from ",
		"copy to ",
		"Binary files ",
		"=== ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

func parseHunkHeader(line string) (oldStart, newStart int, ok bool) {
	trimmed := strings.TrimPrefix(line, "@@ ")
	rangePart, _, found := strings.Cut(trimmed, " @@")
	if !found {
		return 0, 0, false
	}

	var gotOld, gotNew bool
	for _, field := range strings.Fields(rangePart) {
		if strings.HasPrefix(field, "-") {
			numStr, _, _ := strings.Cut(strings.TrimPrefix(field, "-"), ",")
			if n, err := strconv.Atoi(numStr); err == nil {
				oldStart = n
				gotOld = true
			}
		} else if strings.HasPrefix(field, "+") {
			numStr, _, _ := strings.Cut(strings.TrimPrefix(field, "+"), ",")
			if n, err := strconv.Atoi(numStr); err == nil {
				newStart = n
				gotNew = true
			}
		}
	}
	return oldStart, newStart, gotOld && gotNew
}

func sanitizeDiffText(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\t", "    ")
}

func renderDiffSummaryBar(added, removed int, modeLabel string, st styles) string {
	addBadge := st.diffStatAdd.Render(fmt.Sprintf("+%d", added))
	delBadge := st.diffStatDel.Render(fmt.Sprintf("-%d", removed))
	if modeLabel != "" {
		return addBadge + " " + delBadge + "  " + st.itemMuted.Render(modeLabel)
	}
	return addBadge + " " + delBadge
}

func renderVSCodeUnifiedDiff(raw string, width int, st styles) string {
	if strings.TrimSpace(raw) == "" {
		return st.emptyState.Render("Sem alteracoes para exibir.")
	}

	width = max(16, width)
	parsed := parseUnifiedDiff(raw)
	out := make([]string, 0, len(parsed.lines)+2)
	summary := lipgloss.NewStyle().MaxWidth(width).MaxHeight(1).Render(
		renderDiffSummaryBar(parsed.added, parsed.removed, "Visualizacao Unificada", st),
	)
	out = append(out, summary)
	out = append(out, st.itemMuted.Render(strings.Repeat("-", width)))

	const gutterWidth = 2

	for _, dl := range parsed.lines {
		text := sanitizeDiffText(dl.text)
		switch dl.kind {
		case lineMeta:
			out = append(out, st.diffMeta.Render(fitRunesWidth(text, width)))
		case lineHeader:
			out = append(out, renderDiffHeaderLine(text, width, st))
		case lineHunk:
			out = append(out, st.diffHunk.Render(fitRunesWidth(text, width)))
		case lineAdd:
			numStr := fmt.Sprintf("     %4d ", dl.newNum)
			nums := st.diffLineNum.Render(numStr)
			gutter := st.diffAddGutter.Render("┃+")
			lineCodeW := max(1, width-lipgloss.Width(numStr)-gutterWidth)
			code := st.diffAddLine.Render(padRightDisplayWidth(fitRunesWidth(" "+text, lineCodeW), lineCodeW))
			out = append(out, nums+gutter+code)
		case lineDel:
			numStr := fmt.Sprintf("%4d      ", dl.oldNum)
			nums := st.diffLineNum.Render(numStr)
			gutter := st.diffDelGutter.Render("┃-")
			lineCodeW := max(1, width-lipgloss.Width(numStr)-gutterWidth)
			code := st.diffDelLine.Render(padRightDisplayWidth(fitRunesWidth(" "+text, lineCodeW), lineCodeW))
			out = append(out, nums+gutter+code)
		case lineContext:
			numStr := fmt.Sprintf("%4d %4d ", dl.oldNum, dl.newNum)
			nums := st.diffLineNum.Render(numStr)
			gutter := st.diffLineNum.Render("│ ")
			lineCodeW := max(1, width-lipgloss.Width(numStr)-gutterWidth)
			code := st.itemNormal.Render(fitRunesWidth(" "+text, lineCodeW))
			out = append(out, nums+gutter+code)
		}
	}

	return strings.Join(out, "\n")
}

func renderVSCodeSideBySideDiff(raw string, width int, st styles) string {
	if strings.TrimSpace(raw) == "" {
		return st.emptyState.Render("Sem alteracoes para exibir.")
	}

	width = max(24, width)
	colW := max(10, (width-3)/2)
	totalW := min(width, colW*2+3)

	parsed := parseUnifiedDiff(raw)
	out := make([]string, 0, len(parsed.lines)+4)
	summary := lipgloss.NewStyle().MaxWidth(width).MaxHeight(1).Render(
		renderDiffSummaryBar(parsed.added, parsed.removed, "Visualizacao Lado a Lado (VS Code)", st),
	)
	out = append(out, summary)

	leftHdr := st.diffMeta.Render(padRightDisplayWidth("Original (Antes)", colW))
	rightHdr := st.diffMeta.Render(padRightDisplayWidth("Modificado (Depois)", colW))
	sep := st.diffLineNum.Render(" │ ")
	out = append(out, leftHdr+sep+rightHdr)
	out = append(out, st.itemMuted.Render(strings.Repeat("-", totalW)))

	renderLeftCell := func(dl *diffLine) string {
		if dl == nil {
			return st.diffLineNum.Render(strings.Repeat(" ", colW))
		}
		text := sanitizeDiffText(dl.text)
		numStr := fmt.Sprintf("%4d ", dl.oldNum)
		codeW := max(1, colW-lipgloss.Width(numStr)-2)
		num := st.diffLineNum.Render(numStr)
		switch dl.kind {
		case lineDel:
			gutter := st.diffDelGutter.Render("┃-")
			code := st.diffDelLine.Render(padRightDisplayWidth(" "+text, codeW))
			return num + gutter + code
		default:
			gutter := st.diffLineNum.Render("│ ")
			code := st.itemNormal.Render(padRightDisplayWidth(" "+text, codeW))
			return num + gutter + code
		}
	}

	renderRightCell := func(dl *diffLine) string {
		if dl == nil {
			return st.diffLineNum.Render(strings.Repeat(" ", colW))
		}
		text := sanitizeDiffText(dl.text)
		numStr := fmt.Sprintf("%4d ", dl.newNum)
		codeW := max(1, colW-lipgloss.Width(numStr)-2)
		num := st.diffLineNum.Render(numStr)
		switch dl.kind {
		case lineAdd:
			gutter := st.diffAddGutter.Render("┃+")
			code := st.diffAddLine.Render(padRightDisplayWidth(" "+text, codeW))
			return num + gutter + code
		default:
			gutter := st.diffLineNum.Render("│ ")
			code := st.itemNormal.Render(padRightDisplayWidth(" "+text, codeW))
			return num + gutter + code
		}
	}

	lines := parsed.lines
	for i := 0; i < len(lines); {
		dl := lines[i]
		switch dl.kind {
		case lineMeta:
			out = append(out, st.diffMeta.Render(fitRunesWidth(sanitizeDiffText(dl.text), totalW)))
			i++
		case lineHeader:
			out = append(out, renderDiffHeaderLine(dl.text, totalW, st))
			i++
		case lineHunk:
			out = append(out, st.diffHunk.Render(fitRunesWidth(sanitizeDiffText(dl.text), totalW)))
			i++
		case lineContext:
			c := dl
			out = append(out, renderLeftCell(&c)+sep+renderRightCell(&c))
			i++
		case lineDel, lineAdd:
			var dels []diffLine
			for i < len(lines) && lines[i].kind == lineDel {
				dels = append(dels, lines[i])
				i++
			}
			var adds []diffLine
			for i < len(lines) && lines[i].kind == lineAdd {
				adds = append(adds, lines[i])
				i++
			}
			pairs := max(len(dels), len(adds))
			for k := 0; k < pairs; k++ {
				var lPtr, rPtr *diffLine
				if k < len(dels) {
					lPtr = &dels[k]
				}
				if k < len(adds) {
					rPtr = &adds[k]
				}
				out = append(out, renderLeftCell(lPtr)+sep+renderRightCell(rPtr))
			}
		default:
			i++
		}
	}

	return strings.Join(out, "\n")
}

func renderDiffHeaderLine(line string, width int, st styles) string {
	trimmed := fitRunesWidth(sanitizeDiffText(line), width)
	switch {
	case strings.HasPrefix(line, "commit "):
		return st.commitHash.Render(trimmed)
	case strings.HasPrefix(line, "Autor:"), strings.HasPrefix(line, "Data:"):
		return st.commitMeta.Render(trimmed)
	default:
		return st.itemNormal.Render(trimmed)
	}
}

func fitRunesWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	var b strings.Builder
	curW := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if curW+rw > maxWidth {
			break
		}
		b.WriteRune(r)
		curW += rw
	}
	return b.String()
}

func padRightDisplayWidth(s string, targetWidth int) string {
	fitted := fitRunesWidth(s, targetWidth)
	w := lipgloss.Width(fitted)
	if w >= targetWidth {
		return fitted
	}
	return fitted + strings.Repeat(" ", targetWidth-w)
}
