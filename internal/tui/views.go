package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
)

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	if m.width == 0 || m.height == 0 {
		v := tea.NewView("Carregando go-vit...")
		v.AltScreen = true
		return v
	}

	var content string
	if m.modal == modalDiffView {
		content = m.renderDiffModal()
	} else if m.modal != modalNone {
		content = m.renderModal()
	} else if m.state.Kind == git.KindNotARepo && !m.loading {
		content = m.renderNotARepo()
	} else {
		content = m.renderMain()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m *Model) resizeViewport() {
	if m.width == 0 || m.height == 0 {
		return
	}
	_, _, rightW, rightH := m.paneDimensions()
	vpW := max(16, rightW-4)
	innerH := max(3, rightH-2)
	vpH := max(1, innerH-1)
	files := m.visibleFiles()
	if m.activeTab == tabChanges && len(files) > 0 && files[m.fileIdx].IsConflicted {
		vpH = max(1, innerH-2)
	} else if m.activeTab == tabCommits && len(m.commitFiles) > 0 {
		vpH = max(1, innerH-2)
	}
	m.viewport.SetWidth(vpW)
	m.viewport.SetHeight(vpH)

	modalW := max(40, m.width-4)
	modalH := max(10, m.height-2)
	extraModalLines := 0
	if m.activeTab == tabCommits && len(m.commitFiles) > 0 {
		extraModalLines = 1
	}
	m.diffModalViewport.SetWidth(max(32, modalW-4))
	m.diffModalViewport.SetHeight(max(4, modalH-6-extraModalLines))
}

func (m *Model) paneDimensions() (leftW, leftH, rightW, rightH int) {
	const headerH = 3
	const footerH = 2
	bodyH := max(8, m.height-headerH-footerH)

	if m.width < 90 {
		leftW = max(20, m.width)
		rightW = leftW
		leftH = max(5, bodyH*45/100)
		rightH = max(5, bodyH-leftH)
		return
	}

	leftW = max(34, m.width*38/100)
	rightW = max(30, m.width-leftW)
	leftH = bodyH
	rightH = bodyH
	return
}

func (m *Model) syncRightPane() {
	m.resizeViewport()
	vpW := max(16, m.viewport.Width())
	switch m.activeTab {
	case tabBranches:
		m.viewport.SetContent(m.renderCommitLog(vpW))
		m.viewport.GotoTop()

	case tabChanges:
		files := m.visibleFiles()
		if len(files) == 0 {
			if m.filterQuery != "" && len(m.files) > 0 {
				m.viewport.SetContent(m.styles.emptyState.Width(vpW).Render("Nenhum arquivo corresponde ao filtro atual."))
			} else {
				m.viewport.SetContent(m.styles.emptyState.Width(vpW).Render("Nenhuma alteracao pendente. O diretorio de trabalho esta limpo."))
			}
			m.viewport.GotoTop()
			return
		}
		f := files[m.fileIdx]
		if f.IsConflicted {
			m.viewport.SetContent(m.renderConflictDiff(m.diffText, vpW))
		} else {
			m.viewport.SetContent(renderVSCodeUnifiedDiff(m.diffText, vpW, m.styles))
		}
		m.viewport.GotoTop()

	case tabCommits:
		commits := m.visibleCommits()
		if len(commits) == 0 {
			if m.filterQuery != "" && len(m.commits) > 0 {
				m.viewport.SetContent(m.styles.emptyState.Width(vpW).Render("Nenhum commit corresponde ao filtro atual."))
			} else {
				m.viewport.SetContent(m.styles.emptyState.Width(vpW).Render("Nenhum commit encontrado nesta branch."))
			}
			m.viewport.GotoTop()
			return
		}
		m.viewport.SetContent(renderVSCodeUnifiedDiff(m.commitDiffText, vpW, m.styles))
		m.viewport.GotoTop()

	case tabStashes:
		stashes := m.visibleStashes()
		if len(stashes) == 0 {
			if m.filterQuery != "" && len(m.stashes) > 0 {
				m.viewport.SetContent(m.styles.emptyState.Width(vpW).Render("Nenhum stash corresponde ao filtro atual."))
			} else {
				m.viewport.SetContent(m.styles.emptyState.Width(vpW).Render("Nenhum stash salvo no repositorio."))
			}
			m.viewport.GotoTop()
			return
		}
		m.viewport.SetContent(renderVSCodeUnifiedDiff(m.stashDiffText, vpW, m.styles))
		m.viewport.GotoTop()
	}
}

func (m *Model) syncDiffModal() {
	m.syncDiffModalWithScroll(false)
}

func (m *Model) syncDiffModalWithScroll(scrollToHunk bool) {
	m.resizeViewport()
	vpW := max(16, m.diffModalViewport.Width())

	switch m.activeTab {
	case tabChanges:
		files := m.visibleFiles()
		if len(files) == 0 {
			m.diffModalViewport.SetContent(m.styles.emptyState.Render("Nenhum arquivo selecionado."))
			return
		}
		f := files[m.fileIdx]
		if f.IsConflicted {
			m.diffModalViewport.SetContent(m.renderConflictDiff(m.diffText, vpW))
			return
		}
		parsed := parseUnifiedDiff(m.diffText)
		if len(parsed.hunks) == 0 {
			m.activeHunkIdx = 0
		} else if m.activeHunkIdx >= len(parsed.hunks) {
			m.activeHunkIdx = len(parsed.hunks) - 1
		} else if m.activeHunkIdx < 0 {
			m.activeHunkIdx = 0
		}
		activeHunk := m.activeHunkIdx
		if len(parsed.hunks) == 0 || f.IsUntracked {
			activeHunk = -1
		}
		var rendered string
		var offset int
		if m.sideBySideDiff {
			rendered, offset = renderVSCodeSideBySideDiffHunk(m.diffText, vpW, activeHunk, m.styles)
		} else {
			rendered, offset = renderVSCodeUnifiedDiffHunk(m.diffText, vpW, activeHunk, m.styles)
		}
		m.diffModalViewport.SetContent(rendered)
		if scrollToHunk && activeHunk >= 0 {
			m.diffModalViewport.SetYOffset(max(0, offset-1))
		}

	case tabCommits:
		commits := m.visibleCommits()
		if len(commits) == 0 {
			m.diffModalViewport.SetContent(m.styles.emptyState.Render("Nenhum commit selecionado."))
			return
		}
		if m.sideBySideDiff {
			m.diffModalViewport.SetContent(renderVSCodeSideBySideDiff(m.commitDiffText, vpW, m.styles))
		} else {
			m.diffModalViewport.SetContent(renderVSCodeUnifiedDiff(m.commitDiffText, vpW, m.styles))
		}

	case tabStashes:
		stashes := m.visibleStashes()
		if len(stashes) == 0 {
			m.diffModalViewport.SetContent(m.styles.emptyState.Render("Nenhum stash selecionado."))
			return
		}
		if m.sideBySideDiff {
			m.diffModalViewport.SetContent(renderVSCodeSideBySideDiff(m.stashDiffText, vpW, m.styles))
		} else {
			m.diffModalViewport.SetContent(renderVSCodeUnifiedDiff(m.stashDiffText, vpW, m.styles))
		}
	}
}

func (m Model) renderMain() string {
	header := m.renderHeader()
	leftW, leftH, rightW, rightH := m.paneDimensions()
	left := m.renderLeftPane(leftW, leftH)
	right := m.renderRightPane(rightW, rightH)

	var body string
	if m.width < 90 {
		body = lipgloss.JoinVertical(lipgloss.Left, left, right)
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) renderHeader() string {
	repo := m.state.RepoName
	if repo == "" {
		repo = m.client.Dir()
	}

	var branchBadge string
	switch m.state.Kind {
	case git.KindDetached:
		label := "HEAD destacado"
		if m.state.HeadHash != "" {
			label += " @" + m.state.HeadHash
		}
		branchBadge = m.styles.detachedBadge.Render(label)
	case git.KindUnborn:
		label := m.state.Branch
		if label == "" {
			label = "main"
		}
		branchBadge = m.styles.unbornBadge.Render(label + " (sem commits)")
	default:
		label := m.state.Branch
		if label == "" {
			label = "..."
		}
		branchBadge = m.styles.branchBadge.Render(label)
	}

	upstreamStr := ""
	if m.state.Kind == git.KindNormal {
		if m.state.Upstream != "" {
			upstreamStr = " " + m.styles.itemMuted.Render("-> "+m.state.Upstream)
		} else {
			upstreamStr = " " + m.styles.itemMuted.Render("[sem upstream]")
		}
	}

	var syncParts []string
	if m.state.Ahead > 0 {
		syncParts = append(syncParts, m.styles.pushBadge.Render(fmt.Sprintf("↑%d push (P)", m.state.Ahead)))
	}
	if m.state.Behind > 0 {
		syncParts = append(syncParts, m.styles.pullBadge.Render(fmt.Sprintf("↓%d pull (p)", m.state.Behind)))
	}
	syncStr := ""
	if len(syncParts) > 0 {
		syncStr = " " + strings.Join(syncParts, " ")
	}

	var statusBadge string
	switch {
	case m.state.ConflictCount > 0:
		label := fmt.Sprintf("%d conflitos", m.state.ConflictCount)
		if m.state.ConflictCount == 1 {
			label = "1 conflito"
		}
		statusBadge = m.styles.statusConflict.Render(label)
	case m.state.ChangedCount > 0:
		label := fmt.Sprintf("%d alteracoes", m.state.ChangedCount)
		if m.state.ChangedCount == 1 {
			label = "1 alteracao"
		}
		statusBadge = m.styles.statusDirty.Render(label)
	default:
		statusBadge = m.styles.statusClean.Render("Limpo")
	}

	opBadge := ""
	if m.state.ActiveOp != git.OpNone {
		opBadge = "  " + m.styles.statusConflict.Render(m.state.ActiveOp.Label()+" (C:cont A:abort)")
	}

	left := m.styles.repoName.Render("go-vit :: "+truncateText(repo, 24)) + "  " + branchBadge + upstreamStr + syncStr + "  " + statusBadge + opBadge
	if m.loading {
		left += "  " + m.spinner.View()
	}

	boxW := max(24, m.width)
	innerW := max(20, boxW-4)
	return m.styles.headerBox.Width(boxW).Height(3).Render(clampSingleLine(left, innerW))
}

func (m Model) renderTabsRow(innerW int) string {
	changesLabelFull := "[2] Alteracoes"
	changesLabelShort := "[2] Change"
	changesLabelCompact := "[2]Change"
	if m.state.ConflictCount > 0 {
		changesLabelFull = fmt.Sprintf("[2] Alteracoes (%d!)", m.state.ConflictCount)
		changesLabelShort = fmt.Sprintf("[2] Change(%d!)", m.state.ConflictCount)
		changesLabelCompact = fmt.Sprintf("[2]Change(%d!)", m.state.ConflictCount)
	} else if len(m.files) > 0 {
		changesLabelFull = fmt.Sprintf("[2] Alteracoes (%d)", len(m.files))
		changesLabelShort = fmt.Sprintf("[2] Change(%d)", len(m.files))
		changesLabelCompact = fmt.Sprintf("[2]Change(%d)", len(m.files))
	}

	stashesLabelFull := "[4] Stashes"
	stashesLabelShort := "[4] Stash"
	if len(m.stashes) > 0 {
		stashesLabelFull = fmt.Sprintf("[4] Stashes (%d)", len(m.stashes))
		stashesLabelShort = fmt.Sprintf("[4] Stash(%d)", len(m.stashes))
	}

	buildRow := func(l1, l2, l3, l4 string, compact bool) string {
		active := m.styles.tabActive
		inactive := m.styles.tabInactive
		if compact {
			active = active.Padding(0, 0)
			inactive = inactive.Padding(0, 0)
		}
		t1, t2, t3, t4 := inactive.Render(l1), inactive.Render(l2), inactive.Render(l3), inactive.Render(l4)
		switch m.activeTab {
		case tabBranches:
			t1 = active.Render(l1)
		case tabChanges:
			t2 = active.Render(l2)
		case tabCommits:
			t3 = active.Render(l3)
		case tabStashes:
			t4 = active.Render(l4)
		}
		return t1 + " " + t2 + " " + t3 + " " + t4
	}

	row := buildRow("[1] Branches", changesLabelFull, "[3] Historico", stashesLabelFull, false)
	if lipgloss.Width(row) <= innerW {
		return row
	}
	row = buildRow("[1] Branch", changesLabelShort, "[3] Hist", stashesLabelShort, false)
	if lipgloss.Width(row) <= innerW {
		return row
	}
	row = buildRow("[1]Br", changesLabelShort, "[3]Hist", "[4]Stash", true)
	if lipgloss.Width(row) <= innerW {
		return row
	}
	row = buildRow("[1]Br", changesLabelCompact, "[3]Hist", "[4]St", true)
	return clampSingleLine(row, innerW)
}

func (m Model) renderLeftPane(w, h int) string {
	innerW := max(16, w-4)
	innerH := max(3, h-2)

	tabsRow := m.renderTabsRow(innerW)
	headerLines := []string{
		tabsRow,
		m.styles.itemMuted.Render(strings.Repeat("-", innerW)),
	}

	if m.filtering {
		filterLine := m.styles.filterBadge.Render("/") + " " + m.filterInput.View()
		headerLines = append(headerLines, clampSingleLine(filterLine, innerW))
	} else if m.filterQuery != "" {
		filterLine := m.styles.filterBadge.Render("FILTRO") + " " + m.styles.itemNormal.Render(truncateText(m.filterQuery, max(4, innerW-18))) + " " + m.styles.itemMuted.Render("(Esc)")
		headerLines = append(headerLines, clampSingleLine(filterLine, innerW))
	}

	listH := max(1, innerH-len(headerLines))

	var body string
	switch m.activeTab {
	case tabBranches:
		body = m.renderBranchList(innerW, listH)
	case tabChanges:
		body = m.renderFileList(innerW, listH)
	case tabCommits:
		body = m.renderCommitList(innerW, listH)
	case tabStashes:
		body = m.renderStashList(innerW, listH)
	}

	content := strings.Join(append(headerLines, body), "\n")
	clamped := clampBoxContent(content, innerW, innerH)
	return m.styles.paneActive.Width(w).Height(h).Render(clamped)
}

func (m Model) renderBranchList(w, h int) string {
	branches := m.visibleBranches()
	if len(branches) == 0 {
		if m.filterQuery != "" && len(m.branches) > 0 {
			return clampBoxContent(m.styles.emptyState.Width(w).Render("Nenhuma branch corresponde ao filtro.\nPressione Esc para limpar."), w, h)
		}
		return clampBoxContent(m.styles.emptyState.Width(w).Render("Nenhuma branch encontrada.\nPressione 'n' para criar uma."), w, h)
	}

	showHint := h > 1
	visible := h
	if showHint {
		visible = h - 1
	}
	start, end := visibleWindow(m.branchIdx, len(branches), visible)

	var lines []string
	for i := start; i < end; i++ {
		b := branches[i]
		prefix := "  "
		if b.IsCurrent {
			prefix = "* "
		}
		if i == m.branchIdx {
			prefix = "> "
			if b.IsCurrent {
				prefix = ">*"
			}
		}

		label := b.Name
		var suffix string
		if b.IsRemote {
			suffix += " [remota]"
		}
		if b.Ahead > 0 || b.Behind > 0 {
			suffix += fmt.Sprintf(" ↑%d ↓%d", b.Ahead, b.Behind)
		}

		row := truncateText(prefix+label+suffix, w)
		var rendered string
		if i == m.branchIdx {
			rendered = m.styles.itemSelected.Width(w).Render(row)
		} else if b.IsCurrent {
			rendered = m.styles.badgeCurrent.Render(row)
		} else if b.IsRemote {
			rendered = m.styles.badgeRemote.Render(row)
		} else {
			rendered = m.styles.itemNormal.Render(row)
		}
		lines = append(lines, clampSingleLine(rendered, w))
	}

	for len(lines) < visible {
		lines = append(lines, "")
	}

	if showHint {
		remoteState := "off"
		if m.includeRemote {
			remoteState = "on"
		}
		hint := fmt.Sprintf("Enter:trocar  n:nova  M:renomear  d:excluir  m:merge  B:rebase  r:rem(%s)", remoteState)
		lines = append(lines, clampSingleLine(m.styles.itemMuted.Render(truncateText(hint, w)), w))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderFileList(w, h int) string {
	files := m.visibleFiles()
	if len(files) == 0 {
		if m.filterQuery != "" && len(m.files) > 0 {
			return clampBoxContent(m.styles.emptyState.Width(w).Render("Nenhum arquivo corresponde ao filtro.\nPressione Esc para limpar."), w, h)
		}
		return clampBoxContent(m.styles.emptyState.Width(w).Render("Nenhuma alteracao no repositorio.\nPressione '4' para ver stashes."), w, h)
	}

	showHint := h > 1
	visible := h
	if showHint {
		visible = h - 1
	}
	start, end := visibleWindow(m.fileIdx, len(files), visible)

	var lines []string
	for i := start; i < end; i++ {
		f := files[i]
		badge, badgeStyle := m.fileBadge(f)

		cursor := "  "
		if i == m.fileIdx {
			cursor = "> "
		}

		name := f.Path
		if f.OldPath != "" {
			name = f.OldPath + " -> " + f.Path
		}

		maxNameW := max(1, w-lipgloss.Width(cursor)-lipgloss.Width(badge)-1)
		name = truncateText(name, maxNameW)

		var rendered string
		if i == m.fileIdx {
			row := fmt.Sprintf("%s%s %s", cursor, badge, name)
			rendered = m.styles.itemSelected.Width(w).Render(truncateText(row, w))
		} else {
			rendered = cursor + badgeStyle.Render(badge) + " " + m.styles.itemNormal.Render(name)
		}
		lines = append(lines, clampSingleLine(rendered, w))
	}

	for len(lines) < visible {
		lines = append(lines, "")
	}

	if showHint {
		hint := "Espaco/s:stage  Enter/d:diff  c:commit  z:stash  x:descartar"
		lines = append(lines, clampSingleLine(m.styles.itemMuted.Render(truncateText(hint, w)), w))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderCommitList(w, h int) string {
	commits := m.visibleCommits()
	if len(commits) == 0 {
		if m.filterQuery != "" && len(m.commits) > 0 {
			return clampBoxContent(m.styles.emptyState.Width(w).Render("Nenhum commit corresponde ao filtro.\nPressione Esc para limpar."), w, h)
		}
		return clampBoxContent(m.styles.emptyState.Width(w).Render("Nenhum commit encontrado nesta branch."), w, h)
	}

	showHint := h > 1
	visible := h
	if showHint {
		visible = h - 1
	}
	start, end := visibleWindow(m.commitIdx, len(commits), visible)

	var lines []string
	for i := start; i < end; i++ {
		c := commits[i]
		cursor := "  "
		if i == m.commitIdx {
			cursor = "> "
		}

		hash := sanitizeSingleLine(strings.TrimSpace(c.Hash))
		meta := sanitizeSingleLine(c.RelativeDate)
		dec := sanitizeSingleLine(strings.TrimSpace(c.Decorations))
		decBadge := ""
		if dec != "" && w >= lipgloss.Width(cursor+hash)+lipgloss.Width(" ["+dec+"]")+12 {
			decBadge = " [" + truncateText(dec, max(6, w/3)) + "]"
		}

		prefixText := cursor + hash + decBadge + " "
		metaSuffix := " (" + meta + ")"
		avail := w - lipgloss.Width(prefixText) - lipgloss.Width(metaSuffix)

		var rendered string
		if avail >= 4 {
			subj := truncateText(c.Subject, avail)
			if i == m.commitIdx {
				row := fmt.Sprintf("%s%s%s %s%s", cursor, hash, decBadge, subj, metaSuffix)
				rendered = m.styles.itemSelected.Width(w).Render(truncateText(row, w))
			} else {
				decPart := ""
				if decBadge != "" {
					decPart = m.styles.commitRef.Render(decBadge)
				}
				rendered = cursor + m.styles.commitHash.Render(hash) + decPart + " " +
					m.styles.commitSubject.Render(subj) +
					m.styles.commitMeta.Render(metaSuffix)
			}
		} else {
			subj := truncateText(c.Subject, max(1, w-lipgloss.Width(cursor+hash+" ")))
			if i == m.commitIdx {
				row := fmt.Sprintf("%s%s %s", cursor, hash, subj)
				rendered = m.styles.itemSelected.Width(w).Render(truncateText(row, w))
			} else {
				rendered = cursor + m.styles.commitHash.Render(hash) + " " +
					m.styles.commitSubject.Render(subj)
			}
		}
		lines = append(lines, clampSingleLine(rendered, w))
	}

	for len(lines) < visible {
		lines = append(lines, "")
	}

	if showHint {
		refLabel := "HEAD"
		if m.historyCustomRef != "" {
			refLabel = m.historyCustomRef
		}
		moreSuffix := ""
		if m.loadingMoreCommits {
			moreSuffix = "  (carregando...)"
		} else if m.commitsHasMore && m.filterQuery == "" {
			moreSuffix = "  (j/G:+)"
		}
		hint := fmt.Sprintf("Enter/d:diff  ←/→:arq  v:revert  y:cherry-pick  b:%s%s", refLabel, moreSuffix)
		lines = append(lines, clampSingleLine(m.styles.itemMuted.Render(truncateText(hint, w)), w))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderStashList(w, h int) string {
	stashes := m.visibleStashes()
	if len(stashes) == 0 {
		if m.filterQuery != "" && len(m.stashes) > 0 {
			return clampBoxContent(m.styles.emptyState.Width(w).Render("Nenhum stash corresponde ao filtro.\nPressione Esc para limpar."), w, h)
		}
		return clampBoxContent(m.styles.emptyState.Width(w).Render("Nenhum stash salvo.\nUse 'z' na aba Alteracoes para criar um."), w, h)
	}

	showHint := h > 1
	visible := h
	if showHint {
		visible = h - 1
	}
	start, end := visibleWindow(m.stashIdx, len(stashes), visible)

	var lines []string
	for i := start; i < end; i++ {
		st := stashes[i]
		cursor := "  "
		if i == m.stashIdx {
			cursor = "> "
		}

		ref := sanitizeSingleLine(st.Ref)
		meta := sanitizeSingleLine(st.RelativeDate)
		prefixText := cursor + ref + " "
		metaSuffix := " (" + meta + ")"
		avail := w - lipgloss.Width(prefixText) - lipgloss.Width(metaSuffix)

		var rendered string
		if avail >= 4 {
			msgText := truncateText(st.Message, avail)
			if i == m.stashIdx {
				row := fmt.Sprintf("%s%s %s%s", cursor, ref, msgText, metaSuffix)
				rendered = m.styles.itemSelected.Width(w).Render(truncateText(row, w))
			} else {
				rendered = cursor + m.styles.commitHash.Render(ref) + " " +
					m.styles.commitSubject.Render(msgText) +
					m.styles.commitMeta.Render(metaSuffix)
			}
		} else {
			msgText := truncateText(st.Message, max(1, w-lipgloss.Width(prefixText)))
			if i == m.stashIdx {
				row := fmt.Sprintf("%s%s %s", cursor, ref, msgText)
				rendered = m.styles.itemSelected.Width(w).Render(truncateText(row, w))
			} else {
				rendered = cursor + m.styles.commitHash.Render(ref) + " " +
					m.styles.commitSubject.Render(msgText)
			}
		}
		lines = append(lines, clampSingleLine(rendered, w))
	}

	for len(lines) < visible {
		lines = append(lines, "")
	}

	if showHint {
		hint := "Enter/d:diff  a:apply  p:pop  x:drop  n:novo stash  /:filtro"
		lines = append(lines, clampSingleLine(m.styles.itemMuted.Render(truncateText(hint, w)), w))
	}
	return strings.Join(lines, "\n")
}

func (m Model) fileBadge(f git.FileStatus) (string, lipgloss.Style) {
	switch {
	case f.IsConflicted:
		return "[!]", m.styles.badgeConflict
	case f.IsUntracked:
		return "[?]", m.styles.badgeUntracked
	case f.IsStaged && f.IsUnstaged:
		return "[+M]", m.styles.badgeUnstaged
	case f.IsStaged:
		return "[+]", m.styles.badgeStaged
	default:
		return "[M]", m.styles.badgeUnstaged
	}
}

func (m Model) renderCommitFileBar(width int) string {
	if len(m.commitFiles) == 0 {
		return ""
	}
	var label string
	if m.commitFileIdx <= 0 || m.commitFileIdx > len(m.commitFiles) {
		label = fmt.Sprintf("[Todos (%d arq)]  (←/→: inspecionar arquivo)", len(m.commitFiles))
	} else {
		cf := m.commitFiles[m.commitFileIdx-1]
		name := cf.Path
		if cf.OldPath != "" {
			name = cf.OldPath + " -> " + cf.Path
		}
		label = fmt.Sprintf("[Arq %d/%d: %s %s]  (←/→: trocar arquivo)", m.commitFileIdx, len(m.commitFiles), cf.Status, name)
	}
	return clampSingleLine(m.styles.conflictBar.Width(width).Render(truncateText(label, max(1, width-2))), width)
}

func (m Model) renderRightPane(w, h int) string {
	innerW := max(16, w-4)
	innerH := max(3, h-2)

	var title string
	var subBar string

	switch m.activeTab {
	case tabBranches:
		branches := m.visibleBranches()
		if len(branches) > 0 {
			title = fmt.Sprintf("Commits em %s", branches[m.branchIdx].Name)
		} else {
			title = "Commits"
		}

	case tabChanges:
		files := m.visibleFiles()
		if len(files) > 0 {
			f := files[m.fileIdx]
			if f.IsConflicted {
				title = "Conflito: " + f.Path
				barText := truncateText("[1] Manter nossos (ours)   [2] Aceitar deles (theirs)   [e] Abrir no $EDITOR", max(1, innerW-2))
				subBar = clampSingleLine(m.styles.conflictBar.Width(innerW).Render(barText), innerW)
			} else {
				title = "Diff: " + f.Path + "  (Enter/d para tela cheia estilo VS Code)"
			}
		} else {
			title = "Visualizacao"
		}

	case tabCommits:
		commits := m.visibleCommits()
		if len(commits) > 0 {
			c := commits[m.commitIdx]
			title = fmt.Sprintf("Commit %s: %s  (Enter/d para tela cheia)", strings.TrimSpace(c.Hash), c.Subject)
			if len(m.commitFiles) > 0 {
				subBar = m.renderCommitFileBar(innerW)
			}
		} else {
			title = "Detalhes do Commit"
		}

	case tabStashes:
		stashes := m.visibleStashes()
		if len(stashes) > 0 {
			st := stashes[m.stashIdx]
			title = fmt.Sprintf("Stash %s: %s  (Enter/d para tela cheia)", st.Ref, st.Message)
		} else {
			title = "Detalhes do Stash"
		}
	}

	header := clampSingleLine(m.styles.repoName.Render(truncateText(title, innerW)), innerW)
	parts := []string{header}
	if subBar != "" {
		parts = append(parts, subBar)
	}
	parts = append(parts, m.viewport.View())

	content := strings.Join(parts, "\n")
	clamped := clampBoxContent(content, innerW, innerH)
	return m.styles.paneInactive.Width(w).Height(h).Render(clamped)
}

func (m Model) renderCommitLog(width int) string {
	width = max(16, width)
	if len(m.commits) == 0 {
		return clampSingleLine(m.styles.emptyState.Render("Nenhum commit nesta branch ainda."), width)
	}

	var lines []string
	for _, c := range m.commits {
		hash := sanitizeSingleLine(strings.TrimSpace(c.Hash))
		author := sanitizeSingleLine(c.Author)
		relDate := sanitizeSingleLine(c.RelativeDate)
		metaText := fmt.Sprintf("(%s, %s)", author, relDate)
		hashW := lipgloss.Width(hash)
		metaW := lipgloss.Width(metaText)

		avail := width - hashW - 1 - 1 - metaW
		var line string
		if avail >= 6 {
			subj := truncateText(c.Subject, avail)
			line = fmt.Sprintf("%s %s %s",
				m.styles.commitHash.Render(hash),
				m.styles.commitSubject.Render(subj),
				m.styles.commitMeta.Render(metaText),
			)
		} else {
			subj := truncateText(c.Subject, max(1, width-hashW-1))
			line = fmt.Sprintf("%s %s",
				m.styles.commitHash.Render(hash),
				m.styles.commitSubject.Render(subj),
			)
		}
		lines = append(lines, clampSingleLine(line, width))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderConflictDiff(raw string, width int) string {
	width = max(16, width)
	if strings.TrimSpace(raw) == "" {
		return clampSingleLine(m.styles.emptyState.Render("Arquivo em conflito vazio ou binario."), width)
	}

	const (
		secNormal = iota
		secOurs
		secTheirs
	)

	section := secNormal
	rawLines := strings.Split(raw, "\n")
	out := make([]string, 0, len(rawLines))
	for _, rawLine := range rawLines {
		line := sanitizeDiffText(rawLine)
		var rendered string
		switch {
		case strings.HasPrefix(line, "<<<<<<<"):
			section = secOurs
			rendered = m.styles.conflictOurs.Render(fitRunesWidth(line+"  [NOSSOS / OURS]", width))
		case strings.HasPrefix(line, "======="):
			section = secTheirs
			rendered = m.styles.conflictSep.Render(fitRunesWidth(line, width))
		case strings.HasPrefix(line, ">>>>>>>"):
			section = secNormal
			rendered = m.styles.conflictTheirs.Render(fitRunesWidth(line+"  [DELES / THEIRS]", width))
		case section == secOurs:
			rendered = m.styles.diffAddLine.Render(fitRunesWidth("| "+line, width))
		case section == secTheirs:
			rendered = m.styles.conflictTheirs.Render(fitRunesWidth("| "+line, width))
		default:
			rendered = m.styles.itemNormal.Render(fitRunesWidth(line, width))
		}
		out = append(out, clampSingleLine(rendered, width))
	}
	return strings.Join(out, "\n")
}

func (m Model) renderFooter() string {
	maxW := max(20, m.width)

	var top string
	if m.errMsg != "" {
		top = m.styles.bannerError.Render(truncateText("Erro: "+m.errMsg, maxW))
	} else if m.status != "" {
		top = m.styles.bannerInfo.Render(truncateText(m.status, maxW))
	} else {
		top = m.styles.footer.Render(truncateText("f:fetch  p:pull  P:push  /:filtrar  Enter/d:diff  Tab/1-4:abas  ?:ajuda  q:sair", maxW))
	}

	cmdStr := m.lastGitCmd
	if cmdStr == "" {
		cmdStr = "git status --porcelain=v1 --branch"
	}
	avail := max(8, maxW-17)
	bottom := m.styles.cmdBadge.Render("CMD") + " " + m.styles.cmdText.Render("Executado: "+truncateText(cmdStr, avail))

	return clampSingleLine(top, maxW) + "\n" + clampSingleLine(bottom, maxW)
}

func (m Model) renderDiffModal() string {
	modalW := max(40, m.width-4)
	modalH := max(10, m.height-2)
	innerW := max(32, modalW-4)
	innerH := max(4, modalH-2)

	var title string
	var hunkInfo string
	var commitFileBar string
	var controls string

	switch m.activeTab {
	case tabChanges:
		files := m.visibleFiles()
		if len(files) > 0 {
			f := files[m.fileIdx]
			badge, _ := m.fileBadge(f)
			title = fmt.Sprintf("Visualizador de Diff (VS Code) :: %s %s (%d/%d)", badge, f.Path, m.fileIdx+1, len(files))
			if !f.IsConflicted && !f.IsUntracked {
				parsed := parseUnifiedDiff(m.diffText)
				if len(parsed.hunks) > 0 {
					idx := min(max(0, m.activeHunkIdx), len(parsed.hunks)-1)
					hunkInfo = fmt.Sprintf("  Hunk %d/%d", idx+1, len(parsed.hunks))
				}
			}
		} else {
			title = "Visualizador de Diff"
		}
		controls = "l:lado/unif  n/N:hunk  Espaco/s:stage hunk  S:stage arq  x:desc hunk  [/]:arq  e:$EDITOR  Esc/q:fechar"

	case tabCommits:
		commits := m.visibleCommits()
		if len(commits) > 0 {
			c := commits[m.commitIdx]
			title = fmt.Sprintf("Visualizador de Commit :: %s - %s (%d/%d)", c.Hash, c.Subject, m.commitIdx+1, len(commits))
			if len(m.commitFiles) > 0 {
				commitFileBar = m.renderCommitFileBar(innerW)
			}
		} else {
			title = "Visualizador de Commit"
		}
		controls = "l:lado/unif  f/F:arq commit  j/k:rolar  [/]:commit ant/prox  Esc/q:fechar"

	case tabStashes:
		stashes := m.visibleStashes()
		if len(stashes) > 0 {
			st := stashes[m.stashIdx]
			title = fmt.Sprintf("Visualizador de Stash :: %s - %s (%d/%d)", st.Ref, st.Message, m.stashIdx+1, len(stashes))
		} else {
			title = "Visualizador de Stash"
		}
		controls = "l:lado/unif  j/k:rolar  [/]:stash ant/prox  Esc/q:fechar"
	}

	modeText := "[Unificado]"
	if m.sideBySideDiff {
		modeText = "[Lado a Lado]"
	}
	pct := int(m.diffModalViewport.ScrollPercent() * 100)
	statusTag := fmt.Sprintf("%s%s  %d%%", modeText, hunkInfo, pct)

	headerLine := clampSingleLine(
		m.styles.modalTitle.Render(truncateText(title, max(12, innerW-lipgloss.Width(statusTag)-2)))+"  "+m.styles.aheadBehind.Render(statusTag),
		innerW,
	)

	parts := []string{
		headerLine,
		m.styles.itemMuted.Render(strings.Repeat("-", innerW)),
	}
	if commitFileBar != "" {
		parts = append(parts, commitFileBar)
	}
	parts = append(parts,
		m.diffModalViewport.View(),
		m.styles.itemMuted.Render(strings.Repeat("-", innerW)),
		clampSingleLine(m.styles.itemMuted.Render(truncateText(controls, innerW)), innerW),
	)

	content := strings.Join(parts, "\n")
	clamped := clampBoxContent(content, innerW, innerH)
	box := m.styles.paneActive.Width(modalW).Height(modalH).Render(clamped)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) renderModal() string {
	var body string
	switch m.modal {
	case modalHelp:
		body = m.styles.modalTitle.Render("Atalhos do go-vit") + "\n\n" +
			m.help.FullHelpView(m.keys.FullHelp()) + "\n\n" +
			m.styles.itemMuted.Render("Pressione Esc, ? ou Enter para fechar")

	case modalCreateBranch:
		body = m.styles.modalTitle.Render("Criar nova branch") + "\n\n" +
			"Nome da branch:\n" +
			m.input.View() + "\n\n" +
			m.styles.itemMuted.Render("Enter: criar e trocar   Esc: cancelar")

	case modalRenameBranch:
		body = m.styles.modalTitle.Render(fmt.Sprintf("Renomear branch '%s'", m.modalTarget)) + "\n\n" +
			"Novo nome:\n" +
			m.input.View() + "\n\n" +
			m.styles.itemMuted.Render("Enter: renomear   Esc: cancelar")

	case modalCreateStash:
		body = m.styles.modalTitle.Render("Criar Stash") + "\n\n" +
			"Mensagem do stash (opcional):\n" +
			m.input.View() + "\n\n" +
			m.styles.itemMuted.Render("Enter: guardar alteracoes   Esc: cancelar")

	case modalCommit:
		statusLine := m.styles.itemMuted.Render("Enter: commitar   Ctrl+G: gerar com IA   Esc: cancelar")
		if m.loading {
			statusLine = m.styles.modalTitle.Render(m.spinner.View() + " Gerando mensagem com IA...")
		} else if m.errMsg != "" {
			statusLine = m.styles.modalWarn.Render("Erro: " + m.errMsg)
		}
		body = m.styles.modalTitle.Render(fmt.Sprintf("Fazer commit (%d arquivo(s) preparado(s))", m.state.StagedCount)) + "\n\n" +
			"Mensagem:\n" +
			m.input.View() + "\n\n" +
			statusLine

	case modalAPIKey:
		body = m.styles.modalTitle.Render("Configurar chave de API (IA)") + "\n\n" +
			"Cole sua chave de API (sera salva com permissao 0600 em ~/.config/go-vit/config.json):\n" +
			m.input.View() + "\n\n" +
			m.styles.itemMuted.Render("Enter: salvar e gerar   Esc: voltar")

	case modalDeleteBranch:
		body = m.styles.modalTitle.Render("Excluir branch") + "\n\n" +
			fmt.Sprintf("Deseja excluir a branch local '%s'?", m.modalTarget) + "\n\n" +
			m.styles.itemMuted.Render("Enter/s: excluir   Esc/n: cancelar")

	case modalForceDelete:
		body = m.styles.modalTitle.Render("Branch nao mesclada") + "\n\n" +
			m.styles.modalWarn.Render(fmt.Sprintf("A branch '%s' possui commits nao mesclados.", m.modalTarget)) + "\n" +
			"Deseja forcar a exclusao (-D)?\n\n" +
			m.styles.itemMuted.Render("Enter/f/s: forcar exclusao   Esc/n: cancelar")

	case modalDiscard:
		body = m.styles.modalTitle.Render("Descartar alteracoes") + "\n\n" +
			m.styles.modalWarn.Render(fmt.Sprintf("Descartar todas as alteracoes em '%s'?", m.modalTarget)) + "\n" +
			"Esta acao nao pode ser desfeita.\n\n" +
			m.styles.itemMuted.Render("Enter/s: descartar   Esc/n: cancelar")

	case modalDiscardHunk:
		body = m.styles.modalTitle.Render("Descartar hunk") + "\n\n" +
			m.styles.modalWarn.Render(fmt.Sprintf("Descartar este hunk de '%s'?", m.modalTarget)) + "\n" +
			"Esta acao nao pode ser desfeita.\n\n" +
			m.styles.itemMuted.Render("Enter/s: descartar hunk   Esc/n: cancelar")

	case modalDropStash:
		body = m.styles.modalTitle.Render("Descartar stash") + "\n\n" +
			m.styles.modalWarn.Render(fmt.Sprintf("Excluir permanentemente o stash '%s'?", m.modalTarget)) + "\n" +
			"Esta acao nao pode ser desfeita.\n\n" +
			m.styles.itemMuted.Render("Enter/s: excluir stash   Esc/n: cancelar")

	case modalAbortOp:
		body = m.styles.modalTitle.Render("Abortar operacao") + "\n\n" +
			m.styles.modalWarn.Render(fmt.Sprintf("Deseja abortar %s?", m.modalTarget)) + "\n" +
			"O repositorio voltara ao estado anterior ao inicio da operacao.\n\n" +
			m.styles.itemMuted.Render("Enter/s: abortar   Esc/n: cancelar")

	case modalRevertCommit:
		body = m.styles.modalTitle.Render("Reverter commit") + "\n\n" +
			fmt.Sprintf("Criar um novo commit revertendo '%s'?", m.modalTarget) + "\n\n" +
			m.styles.itemMuted.Render("Enter/s: reverter   Esc/n: cancelar")

	case modalCherryPick:
		body = m.styles.modalTitle.Render("Cherry-pick de commit") + "\n\n" +
			fmt.Sprintf("Aplicar o commit '%s' na branch atual?", m.modalTarget) + "\n\n" +
			m.styles.itemMuted.Render("Enter/s: aplicar   Esc/n: cancelar")

	case modalStashSwitch:
		opts := []string{
			"[1] Fazer stash das alteracoes e trocar (recomendado)",
			"[2] Trocar mesmo assim",
			"[3] Cancelar",
		}
		var renderedOpts []string
		for i, o := range opts {
			if i == m.modalChoice {
				renderedOpts = append(renderedOpts, m.styles.modalOptionSel.Render(o))
			} else {
				renderedOpts = append(renderedOpts, m.styles.modalOption.Render(o))
			}
		}
		body = m.styles.modalTitle.Render("Diretorio com alteracoes pendentes") + "\n\n" +
			fmt.Sprintf("Ao trocar para '%s', o que deseja fazer?", m.modalTarget) + "\n\n" +
			strings.Join(renderedOpts, "\n") + "\n\n" +
			m.styles.itemMuted.Render("j/k: navegar   Enter ou 1/2/3: escolher   Esc: cancelar")
	}

	box := m.styles.modalBox.MaxWidth(max(40, m.width-6)).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) renderNotARepo() string {
	msg := m.styles.modalTitle.Render("Nenhum repositorio Git encontrado") + "\n\n" +
		fmt.Sprintf("O diretorio '%s' nao pertence a um repositorio Git.", m.client.Dir()) + "\n" +
		"Execute 'git init' ou abra o go-vit dentro de um projeto Git.\n\n" +
		m.styles.itemMuted.Render("R: verificar novamente   q: sair")
	box := m.styles.modalBox.Render(msg)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func visibleWindow(cursor, total, height int) (int, int) {
	if total <= height {
		return 0, total
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}

func clampBoxContent(content string, maxW, maxH int) string {
	if maxW <= 0 || maxH <= 0 {
		return ""
	}
	return lipgloss.NewStyle().MaxWidth(maxW).MaxHeight(maxH).Render(content)
}

func clampSingleLine(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	return lipgloss.NewStyle().MaxWidth(maxW).MaxHeight(1).Render(s)
}

func sanitizeSingleLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\t", "    ")
}

func truncateText(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	s = sanitizeSingleLine(s)
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return fitRunesWidth(s, maxWidth)
	}
	return fitRunesWidth(s, maxWidth-3) + "..."
}
