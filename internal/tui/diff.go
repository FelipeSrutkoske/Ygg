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
	kind    diffLineKind
	oldNum  int
	newNum  int
	text    string
	hunkIdx int
}

type diffHunk struct {
	index     int
	header    string
	patch     string
	staged    bool
	startLine int
}

type parsedDiff struct {
	added   int
	removed int
	lines   []diffLine
	hunks   []diffHunk
}

type intraSpan struct {
	start int
	end   int
	ok    bool
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
	inStagedSection := false
	oldLine := 0
	newLine := 0
	oldRem := 0
	newRem := 0

	var fileHeaders []string
	var curHunk *diffHunk
	var hunkBuf strings.Builder

	flushHunk := func() {
		if curHunk != nil {
			curHunk.patch = hunkBuf.String()
			res.hunks = append(res.hunks, *curHunk)
			curHunk = nil
			hunkBuf.Reset()
		}
	}

	for _, rawLine := range rawLines {
		rawNoCR := strings.TrimRight(rawLine, "\r")
		line := strings.ReplaceAll(rawNoCR, "\t", "    ")

		if inHunk && rawNoCR == "" && oldRem <= 0 && newRem <= 0 {
			inHunk = false
			flushHunk()
			continue
		}

		switch {
		case isDiffMetaLine(line):
			inHunk = false
			flushHunk()
			switch {
			case line == "=== Staged ===":
				inStagedSection = true
				fileHeaders = nil
			case line == "=== Unstaged ===":
				inStagedSection = false
				fileHeaders = nil
			case strings.HasPrefix(rawNoCR, "diff --git "):
				fileHeaders = []string{rawNoCR}
			case isPatchHeaderLine(rawNoCR):
				fileHeaders = append(fileHeaders, rawNoCR)
			}
			res.lines = append(res.lines, diffLine{kind: lineMeta, text: line, hunkIdx: -1})

		case strings.HasPrefix(line, "@@"):
			flushHunk()
			if o, oc, n, nc, ok := parseHunkHeaderFull(line); ok {
				oldLine = o
				newLine = n
				oldRem = oc
				newRem = nc
				inHunk = true

				idx := len(res.hunks)
				curHunk = &diffHunk{
					index:     idx,
					header:    rawNoCR,
					staged:    inStagedSection,
					startLine: len(res.lines),
				}
				if len(fileHeaders) > 0 {
					hunkBuf.WriteString(strings.Join(fileHeaders, "\n"))
					hunkBuf.WriteByte('\n')
				}
				hunkBuf.WriteString(rawNoCR)
				hunkBuf.WriteByte('\n')

				res.lines = append(res.lines, diffLine{kind: lineHunk, text: line, hunkIdx: idx})
			} else {
				inHunk = false
				res.lines = append(res.lines, diffLine{kind: lineHunk, text: line, hunkIdx: -1})
			}

		case inHunk:
			hIdx := -1
			if curHunk != nil {
				hIdx = curHunk.index
				hunkBuf.WriteString(rawNoCR)
				hunkBuf.WriteByte('\n')
			}
			switch {
			case strings.HasPrefix(line, "+"):
				res.added++
				newRem--
				res.lines = append(res.lines, diffLine{
					kind:    lineAdd,
					newNum:  newLine,
					text:    line[1:],
					hunkIdx: hIdx,
				})
				newLine++
			case strings.HasPrefix(line, "-"):
				res.removed++
				oldRem--
				res.lines = append(res.lines, diffLine{
					kind:    lineDel,
					oldNum:  oldLine,
					text:    line[1:],
					hunkIdx: hIdx,
				})
				oldLine++
			case strings.HasPrefix(line, `\`):
				res.lines = append(res.lines, diffLine{kind: lineMeta, text: line, hunkIdx: hIdx})
			default:
				text := line
				if strings.HasPrefix(line, " ") {
					text = line[1:]
				}
				oldRem--
				newRem--
				res.lines = append(res.lines, diffLine{
					kind:    lineContext,
					oldNum:  oldLine,
					newNum:  newLine,
					text:    text,
					hunkIdx: hIdx,
				})
				oldLine++
				newLine++
			}

		default:
			res.lines = append(res.lines, diffLine{kind: lineHeader, text: line, hunkIdx: -1})
		}
	}

	flushHunk()
	return res
}

func isPatchHeaderLine(line string) bool {
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
	}
	for _, p := range prefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
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
	o, _, n, _, ok := parseHunkHeaderFull(line)
	return o, n, ok
}

func parseHunkHeaderFull(line string) (oldStart, oldCount, newStart, newCount int, ok bool) {
	trimmed := strings.TrimPrefix(line, "@@ ")
	rangePart, _, found := strings.Cut(trimmed, " @@")
	if !found {
		return 0, 0, 0, 0, false
	}

	var gotOld, gotNew bool
	for _, field := range strings.Fields(rangePart) {
		if strings.HasPrefix(field, "-") {
			numStr, cntStr, hasComma := strings.Cut(strings.TrimPrefix(field, "-"), ",")
			if n, err := strconv.Atoi(numStr); err == nil {
				oldStart = n
				oldCount = 1
				if hasComma {
					if c, err := strconv.Atoi(cntStr); err == nil {
						oldCount = c
					}
				}
				gotOld = true
			}
		} else if strings.HasPrefix(field, "+") {
			numStr, cntStr, hasComma := strings.Cut(strings.TrimPrefix(field, "+"), ",")
			if n, err := strconv.Atoi(numStr); err == nil {
				newStart = n
				newCount = 1
				if hasComma {
					if c, err := strconv.Atoi(cntStr); err == nil {
						newCount = c
					}
				}
				gotNew = true
			}
		}
	}
	return oldStart, oldCount, newStart, newCount, gotOld && gotNew
}

func sanitizeDiffText(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\t", "    ")
}

func computeIntraLineSpan(oldStr, newStr string) (prefixLen, oldEnd, newEnd int, ok bool) {
	oldRunes := []rune(oldStr)
	newRunes := []rune(newStr)
	minLen := min(len(oldRunes), len(newRunes))

	for prefixLen < minLen && oldRunes[prefixLen] == newRunes[prefixLen] {
		prefixLen++
	}

	suffixLen := 0
	for suffixLen < len(oldRunes)-prefixLen &&
		suffixLen < len(newRunes)-prefixLen &&
		oldRunes[len(oldRunes)-1-suffixLen] == newRunes[len(newRunes)-1-suffixLen] {
		suffixLen++
	}

	oldEnd = len(oldRunes) - suffixLen
	newEnd = len(newRunes) - suffixLen

	if prefixLen+suffixLen < 2 {
		return 0, 0, 0, false
	}
	if prefixLen >= oldEnd && prefixLen >= newEnd {
		return 0, 0, 0, false
	}
	return prefixLen, oldEnd, newEnd, true
}

func renderCodeWithEmph(text string, codeW int, baseStyle, emphStyle lipgloss.Style, span intraSpan) string {
	padded := padRightDisplayWidth(fitRunesWidth(" "+text, codeW), codeW)
	if !span.ok {
		return baseStyle.Render(padded)
	}
	runes := []rune(padded)
	s := min(max(0, span.start+1), len(runes))
	e := min(max(s, span.end+1), len(runes))
	if s >= e {
		return baseStyle.Render(padded)
	}
	var b strings.Builder
	if s > 0 {
		b.WriteString(baseStyle.Render(string(runes[:s])))
	}
	b.WriteString(emphStyle.Render(string(runes[s:e])))
	if e < len(runes) {
		b.WriteString(baseStyle.Render(string(runes[e:])))
	}
	return b.String()
}

func computeBlockSpans(dels, adds []diffLine) ([]intraSpan, []intraSpan) {
	delSpans := make([]intraSpan, len(dels))
	addSpans := make([]intraSpan, len(adds))
	pairs := min(len(dels), len(adds))
	for k := 0; k < pairs; k++ {
		oldText := sanitizeDiffText(dels[k].text)
		newText := sanitizeDiffText(adds[k].text)
		if p, oe, ne, ok := computeIntraLineSpan(oldText, newText); ok {
			if p < oe {
				delSpans[k] = intraSpan{start: p, end: oe, ok: true}
			}
			if p < ne {
				addSpans[k] = intraSpan{start: p, end: ne, ok: true}
			}
		}
	}
	return delSpans, addSpans
}

func renderDiffSummaryBar(added, removed int, modeLabel string, st styles) string {
	addBadge := st.diffStatAdd.Render(fmt.Sprintf("+%d", added))
	delBadge := st.diffStatDel.Render(fmt.Sprintf("-%d", removed))
	if modeLabel != "" {
		return addBadge + " " + delBadge + "  " + st.itemMuted.Render(modeLabel)
	}
	return addBadge + " " + delBadge
}

func renderHunkHeaderLine(dl diffLine, width, activeHunk, totalHunks int, st styles) string {
	text := sanitizeDiffText(dl.text)
	if activeHunk >= 0 && dl.hunkIdx == activeHunk && totalHunks > 0 {
		badge := st.hunkActive.Render(fmt.Sprintf("▶ Hunk %d/%d", activeHunk+1, totalHunks))
		badgeW := lipgloss.Width(badge)
		rem := width - badgeW - 1
		if rem > 0 {
			return badge + " " + st.diffHunk.Render(fitRunesWidth(text, rem))
		}
		return fitRunesWidth(badge, width)
	}
	return st.diffHunk.Render(fitRunesWidth(text, width))
}

func renderVSCodeUnifiedDiff(raw string, width int, st styles) string {
	rendered, _ := renderVSCodeUnifiedDiffHunk(raw, width, -1, st)
	return rendered
}

func renderVSCodeUnifiedDiffHunk(raw string, width int, activeHunk int, st styles) (string, int) {
	if strings.TrimSpace(raw) == "" {
		return st.emptyState.Render("Sem alteracoes para exibir."), 0
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
	activeOffset := -1

	renderDel := func(dl diffLine, span intraSpan) string {
		text := sanitizeDiffText(dl.text)
		numStr := fmt.Sprintf("%4d      ", dl.oldNum)
		nums := st.diffLineNum.Render(numStr)
		gutter := st.diffDelGutter.Render("┃-")
		lineCodeW := max(1, width-lipgloss.Width(numStr)-gutterWidth)
		code := renderCodeWithEmph(text, lineCodeW, st.diffDelLine, st.diffDelEmph, span)
		return nums + gutter + code
	}

	renderAdd := func(dl diffLine, span intraSpan) string {
		text := sanitizeDiffText(dl.text)
		numStr := fmt.Sprintf("     %4d ", dl.newNum)
		nums := st.diffLineNum.Render(numStr)
		gutter := st.diffAddGutter.Render("┃+")
		lineCodeW := max(1, width-lipgloss.Width(numStr)-gutterWidth)
		code := renderCodeWithEmph(text, lineCodeW, st.diffAddLine, st.diffAddEmph, span)
		return nums + gutter + code
	}

	lines := parsed.lines
	for i := 0; i < len(lines); {
		dl := lines[i]
		text := sanitizeDiffText(dl.text)
		switch dl.kind {
		case lineMeta:
			out = append(out, st.diffMeta.Render(fitRunesWidth(text, width)))
			i++
		case lineHeader:
			out = append(out, renderDiffHeaderLine(text, width, st))
			i++
		case lineHunk:
			if activeHunk >= 0 && dl.hunkIdx == activeHunk && activeOffset < 0 {
				activeOffset = len(out)
			}
			out = append(out, renderHunkHeaderLine(dl, width, activeHunk, len(parsed.hunks), st))
			i++
		case lineContext:
			numStr := fmt.Sprintf("%4d %4d ", dl.oldNum, dl.newNum)
			nums := st.diffLineNum.Render(numStr)
			gutter := st.diffLineNum.Render("│ ")
			lineCodeW := max(1, width-lipgloss.Width(numStr)-gutterWidth)
			code := st.itemNormal.Render(fitRunesWidth(" "+text, lineCodeW))
			out = append(out, nums+gutter+code)
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
			delSpans, addSpans := computeBlockSpans(dels, adds)
			for k, d := range dels {
				out = append(out, renderDel(d, delSpans[k]))
			}
			for k, a := range adds {
				out = append(out, renderAdd(a, addSpans[k]))
			}
		default:
			i++
		}
	}

	if activeOffset < 0 {
		activeOffset = 0
	}
	return strings.Join(out, "\n"), activeOffset
}

func renderVSCodeSideBySideDiff(raw string, width int, st styles) string {
	rendered, _ := renderVSCodeSideBySideDiffHunk(raw, width, -1, st)
	return rendered
}

func renderVSCodeSideBySideDiffHunk(raw string, width int, activeHunk int, st styles) (string, int) {
	if strings.TrimSpace(raw) == "" {
		return st.emptyState.Render("Sem alteracoes para exibir."), 0
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

	renderLeftCell := func(dl *diffLine, span intraSpan) string {
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
			code := renderCodeWithEmph(text, codeW, st.diffDelLine, st.diffDelEmph, span)
			return num + gutter + code
		default:
			gutter := st.diffLineNum.Render("│ ")
			code := st.itemNormal.Render(padRightDisplayWidth(" "+text, codeW))
			return num + gutter + code
		}
	}

	renderRightCell := func(dl *diffLine, span intraSpan) string {
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
			code := renderCodeWithEmph(text, codeW, st.diffAddLine, st.diffAddEmph, span)
			return num + gutter + code
		default:
			gutter := st.diffLineNum.Render("│ ")
			code := st.itemNormal.Render(padRightDisplayWidth(" "+text, codeW))
			return num + gutter + code
		}
	}

	activeOffset := -1
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
			if activeHunk >= 0 && dl.hunkIdx == activeHunk && activeOffset < 0 {
				activeOffset = len(out)
			}
			out = append(out, renderHunkHeaderLine(dl, totalW, activeHunk, len(parsed.hunks), st))
			i++
		case lineContext:
			c := dl
			out = append(out, renderLeftCell(&c, intraSpan{})+sep+renderRightCell(&c, intraSpan{}))
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
			delSpans, addSpans := computeBlockSpans(dels, adds)
			pairs := max(len(dels), len(adds))
			for k := 0; k < pairs; k++ {
				var lPtr, rPtr *diffLine
				var lSpan, rSpan intraSpan
				if k < len(dels) {
					lPtr = &dels[k]
					lSpan = delSpans[k]
				}
				if k < len(adds) {
					rPtr = &adds[k]
					rSpan = addSpans[k]
				}
				out = append(out, renderLeftCell(lPtr, lSpan)+sep+renderRightCell(rPtr, rSpan))
			}
		default:
			i++
		}
	}

	if activeOffset < 0 {
		activeOffset = 0
	}
	return strings.Join(out, "\n"), activeOffset
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
