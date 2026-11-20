package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/config"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
	"github.com/atotto/clipboard"
)

type tabKind int

const (
	tabBranches tabKind = iota
	tabChanges
	tabCommits
)

type modalKind int

const (
	modalNone modalKind = iota
	modalHelp
	modalCreateBranch
	modalDeleteBranch
	modalForceDelete
	modalStashSwitch
	modalCommit
	modalAPIKey
	modalDiscard
	modalDiffView
	modalRevertCommit
	modalCherryPick
)

type Model struct {
	client           *git.Client
	state            git.RepoState
	branches         []git.Branch
	files            []git.FileStatus
	commits          []git.Commit
	commitsRef       string
	diffText         string
	diffPath         string
	commitDiffText   string
	commitDiffHash   string
	historyCustomRef string
	activeTab        tabKind
	branchIdx        int
	fileIdx          int
	commitIdx        int
	includeRemote    bool

	filtering   bool
	filterQuery string
	filterInput textinput.Model

	sideBySideDiff    bool
	diffModeSet       bool
	diffModalViewport viewport.Model

	modal       modalKind
	modalTarget string
	modalFile   git.FileStatus
	modalChoice int
	input       textinput.Model

	viewport viewport.Model
	spinner  spinner.Model
	help     help.Model
	keys     keyMap
	styles   styles

	width      int
	height     int
	loading    bool
	status     string
	errMsg     string
	lastGitCmd string
	quitting   bool
}

func NewModel(client *git.Client) Model {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	vp := viewport.New()
	diffVP := viewport.New()

	ti := textinput.New()
	ti.CharLimit = 120
	ti.SetWidth(44)

	fi := textinput.New()
	fi.CharLimit = 80
	fi.SetWidth(32)
	fi.Placeholder = "digite para filtrar..."

	h := help.New()
	st := defaultStyles()

	return Model{
		client:            client,
		viewport:          vp,
		diffModalViewport: diffVP,
		spinner:           sp,
		input:             ti,
		filterInput:       fi,
		help:              h,
		keys:              defaultKeyMap(),
		styles:            st,
		sideBySideDiff:    true,
		loading:           true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadRepoCmd(m.client, m.includeRemote))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		m.help.SetWidth(max(20, msg.Width-8))
		m.syncRightPane()
		if m.modal == modalDiffView {
			m.syncDiffModal()
		}
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case repoLoadedMsg:
		m.loading = false
		if msg.lastCmd != "" {
			m.lastGitCmd = msg.lastCmd
		}
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.state = msg.state
		m.branches = msg.branches
		m.files = msg.status
		m.clampCursors()
		if m.activeTab == tabCommits && m.state.Kind != git.KindNotARepo {
			return m, loadCommitsCmd(m.client, m.historyRef())
		}
		return m, m.refreshSelectionCmd()

	case commitsLoadedMsg:
		m.commitsRef = msg.ref
		if msg.err != nil {
			m.commits = nil
			m.commitDiffText = ""
			m.commitDiffHash = ""
			m.viewport.SetContent(m.styles.bannerError.Render("Erro ao carregar commits: " + msg.err.Error()))
			return m, nil
		}
		m.commits = msg.commits
		m.clampCursors()
		if m.activeTab == tabCommits {
			vis := m.visibleCommits()
			if len(vis) > 0 {
				return m, loadCommitDiffCmd(m.client, vis[m.commitIdx].Hash)
			}
			m.commitDiffText = ""
			m.commitDiffHash = ""
			m.syncRightPane()
			if m.modal == modalDiffView {
				m.syncDiffModal()
			}
			return m, nil
		}
		m.syncRightPane()
		return m, nil

	case diffLoadedMsg:
		m.diffPath = msg.path
		if msg.err != nil {
			m.diffText = ""
			m.viewport.SetContent(m.styles.bannerError.Render("Erro ao carregar diff: " + msg.err.Error()))
			if m.modal == modalDiffView {
				m.diffModalViewport.SetContent(m.styles.bannerError.Render("Erro ao carregar diff: " + msg.err.Error()))
			}
			return m, nil
		}
		m.diffText = msg.diff
		m.syncRightPane()
		if m.modal == modalDiffView && m.activeTab == tabChanges {
			m.syncDiffModal()
		}
		return m, nil

	case commitDiffLoadedMsg:
		m.commitDiffHash = msg.hash
		if msg.err != nil {
			m.commitDiffText = ""
			m.viewport.SetContent(m.styles.bannerError.Render("Erro ao carregar diff do commit: " + msg.err.Error()))
			if m.modal == modalDiffView {
				m.diffModalViewport.SetContent(m.styles.bannerError.Render("Erro ao carregar diff do commit: " + msg.err.Error()))
			}
			return m, nil
		}
		m.commitDiffText = msg.diff
		m.syncRightPane()
		if m.modal == modalDiffView && m.activeTab == tabCommits {
			m.syncDiffModal()
		}
		return m, nil

	case actionDoneMsg:
		m.loading = false
		if msg.lastCmd != "" {
			m.lastGitCmd = msg.lastCmd
		}
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = ""
			return m, nil
		}
		m.errMsg = ""
		m.status = msg.message
		return m, m.reloadRepo()

	case commitMsgGeneratedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.errMsg = ""
		m.input.SetValue(msg.message)
		m.input.CursorEnd()
		return m, nil

	case mergeDoneMsg:
		m.loading = false
		if msg.lastCmd != "" {
			m.lastGitCmd = msg.lastCmd
		}
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = ""
			return m, nil
		}
		m.errMsg = ""
		if msg.res.Conflicted {
			m.activeTab = tabChanges
			m.status = fmt.Sprintf(
				"Conflitos ao mesclar %s (%d arquivo(s)). Resolva na aba Alteracoes & Conflitos.",
				msg.branch,
				len(msg.res.Conflicts),
			)
		} else {
			m.status = fmt.Sprintf("Branch %s mesclada com sucesso", msg.branch)
		}
		return m, m.reloadRepo()

	case deleteBranchErrMsg:
		m.loading = false
		if msg.lastCmd != "" {
			m.lastGitCmd = msg.lastCmd
		}
		if !msg.force {
			m.modal = modalForceDelete
			m.modalTarget = msg.branch
			m.errMsg = ""
			return m, nil
		}
		m.errMsg = msg.err.Error()
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.errMsg = "Erro ao abrir editor: " + msg.err.Error()
		}
		return m, m.reloadRepo()

	case tea.KeyPressMsg:
		if m.modal != modalNone {
			return m.handleModalKey(msg)
		}
		if m.filtering {
			return m.handleFilterKey(msg)
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) reloadRepo() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Tick, loadRepoCmd(m.client, m.includeRemote))
}

func (m Model) historyRef() string {
	return m.historyCustomRef
}

func (m Model) visibleBranches() []git.Branch {
	q := strings.ToLower(strings.TrimSpace(m.filterQuery))
	if q == "" {
		return m.branches
	}
	var out []git.Branch
	for _, b := range m.branches {
		if strings.Contains(strings.ToLower(b.Name), q) {
			out = append(out, b)
		}
	}
	return out
}

func (m Model) visibleFiles() []git.FileStatus {
	q := strings.ToLower(strings.TrimSpace(m.filterQuery))
	if q == "" {
		return m.files
	}
	var out []git.FileStatus
	for _, f := range m.files {
		if strings.Contains(strings.ToLower(f.Path), q) ||
			strings.Contains(strings.ToLower(f.OldPath), q) {
			out = append(out, f)
		}
	}
	return out
}

func (m Model) visibleCommits() []git.Commit {
	q := strings.ToLower(strings.TrimSpace(m.filterQuery))
	if q == "" {
		return m.commits
	}
	var out []git.Commit
	for _, c := range m.commits {
		if strings.Contains(strings.ToLower(c.Hash), q) ||
			strings.Contains(strings.ToLower(c.Subject), q) ||
			strings.Contains(strings.ToLower(c.Author), q) {
			out = append(out, c)
		}
	}
	return out
}

func (m *Model) clampCursors() {
	branches := m.visibleBranches()
	if len(branches) == 0 {
		m.branchIdx = 0
	} else if m.branchIdx >= len(branches) {
		m.branchIdx = len(branches) - 1
	}

	files := m.visibleFiles()
	if len(files) == 0 {
		m.fileIdx = 0
	} else if m.fileIdx >= len(files) {
		m.fileIdx = len(files) - 1
	}

	commits := m.visibleCommits()
	if len(commits) == 0 {
		m.commitIdx = 0
	} else if m.commitIdx >= len(commits) {
		m.commitIdx = len(commits) - 1
	}
}

func (m *Model) refreshSelectionCmd() tea.Cmd {
	if m.state.Kind == git.KindNotARepo {
		return nil
	}

	switch m.activeTab {
	case tabBranches:
		branches := m.visibleBranches()
		if len(branches) == 0 {
			m.commits = nil
			m.commitsRef = ""
			m.syncRightPane()
			return nil
		}
		return loadCommitsCmd(m.client, branches[m.branchIdx].Name)

	case tabChanges:
		files := m.visibleFiles()
		if len(files) == 0 {
			m.diffText = ""
			m.diffPath = ""
			m.syncRightPane()
			return nil
		}
		return loadDiffCmd(m.client, files[m.fileIdx])

	case tabCommits:
		commits := m.visibleCommits()
		if len(commits) == 0 {
			m.commitDiffText = ""
			m.commitDiffHash = ""
			m.syncRightPane()
			return nil
		}
		return loadCommitDiffCmd(m.client, commits[m.commitIdx].Hash)
	}

	return nil
}

func (m *Model) openDiffModal() {
	if !m.diffModeSet {
		m.sideBySideDiff = m.width >= 100
	}
	m.modal = modalDiffView
	m.syncDiffModal()
	m.diffModalViewport.GotoTop()
}

func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filterQuery = ""
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		m.clampCursors()
		return m, m.refreshSelectionCmd()
	case "enter":
		m.filtering = false
		m.filterInput.Blur()
		return m, nil
	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.filterQuery = strings.TrimSpace(m.filterInput.Value())
		m.clampCursors()
		return m, tea.Batch(cmd, m.refreshSelectionCmd())
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		m.quitting = true
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Help) {
		m.modal = modalHelp
		return m, nil
	}
	if key.Matches(msg, m.keys.Refresh) {
		m.errMsg = ""
		m.status = "Atualizando..."
		return m, m.reloadRepo()
	}
	if msg.String() == "esc" && m.filterQuery != "" {
		m.filterQuery = ""
		m.filterInput.SetValue("")
		m.clampCursors()
		return m, m.refreshSelectionCmd()
	}

	if m.state.Kind == git.KindNotARepo {
		return m, nil
	}

	// 1/2/e tem prioridade quando o arquivo selecionado esta em conflito
	if m.activeTab == tabChanges {
		files := m.visibleFiles()
		if len(files) > 0 && files[m.fileIdx].IsConflicted {
			f := files[m.fileIdx]
			switch {
			case key.Matches(msg, m.keys.KeepOurs):
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, resolveConflictCmd(m.client, f.Path, git.SideOurs))
			case key.Matches(msg, m.keys.AcceptTheirs):
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, resolveConflictCmd(m.client, f.Path, git.SideTheirs))
			case key.Matches(msg, m.keys.OpenEditor):
				return m, openEditorCmd(m.state.RootDir, f.Path)
			}
		}
	}

	switch {
	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.filterInput.SetValue(m.filterQuery)
		m.filterInput.Focus()
		return m, nil
	case key.Matches(msg, m.keys.NextTab):
		switch m.activeTab {
		case tabBranches:
			m.activeTab = tabChanges
			return m, m.refreshSelectionCmd()
		case tabChanges:
			m.activeTab = tabCommits
			return m, loadCommitsCmd(m.client, m.historyRef())
		default:
			m.activeTab = tabBranches
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Tab1):
		if m.activeTab != tabBranches {
			m.activeTab = tabBranches
			return m, m.refreshSelectionCmd()
		}
		return m, nil
	case key.Matches(msg, m.keys.Tab2):
		if m.activeTab != tabChanges {
			m.activeTab = tabChanges
			return m, m.refreshSelectionCmd()
		}
		return m, nil
	case key.Matches(msg, m.keys.Tab3):
		if m.activeTab != tabCommits {
			m.activeTab = tabCommits
			return m, loadCommitsCmd(m.client, m.historyRef())
		}
		return m, nil
	case key.Matches(msg, m.keys.Fetch):
		m.loading = true
		m.errMsg = ""
		m.status = "Executando fetch..."
		return m, tea.Batch(m.spinner.Tick, fetchCmd(m.client))
	case key.Matches(msg, m.keys.Pull):
		m.loading = true
		m.errMsg = ""
		m.status = "Executando pull..."
		return m, tea.Batch(m.spinner.Tick, pullCmd(m.client))
	case key.Matches(msg, m.keys.Push):
		m.loading = true
		m.errMsg = ""
		m.status = "Executando push..."
		return m, tea.Batch(m.spinner.Tick, pushCmd(m.client))
	case key.Matches(msg, m.keys.PageUp):
		m.viewport.HalfPageUp()
		return m, nil
	case key.Matches(msg, m.keys.PageDown):
		m.viewport.HalfPageDown()
		return m, nil
	}

	switch m.activeTab {
	case tabBranches:
		return m.handleBranchKey(msg)
	case tabChanges:
		return m.handleChangesKey(msg)
	case tabCommits:
		return m.handleCommitsKey(msg)
	}
	return m, nil
}

func (m Model) handleBranchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	branches := m.visibleBranches()
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.branchIdx > 0 {
			m.branchIdx--
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Down):
		if m.branchIdx < len(branches)-1 {
			m.branchIdx++
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Top):
		if len(branches) > 0 && m.branchIdx != 0 {
			m.branchIdx = 0
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Bottom):
		if len(branches) > 0 && m.branchIdx != len(branches)-1 {
			m.branchIdx = len(branches) - 1
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.ToggleRemotes):
		m.includeRemote = !m.includeRemote
		return m, m.reloadRepo()
	case key.Matches(msg, m.keys.NewBranch):
		m.modal = modalCreateBranch
		m.input.SetValue("")
		m.input.Placeholder = "nome-da-nova-branch"
		m.input.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Switch):
		if len(branches) == 0 {
			return m, nil
		}
		b := branches[m.branchIdx]
		if b.IsCurrent {
			return m, nil
		}
		if m.state.ChangedCount > 0 || m.state.ConflictCount > 0 {
			m.modal = modalStashSwitch
			m.modalTarget = b.Name
			m.modalChoice = 0
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, switchBranchCmd(m.client, b.Name))
	case key.Matches(msg, m.keys.DeleteBranch):
		if len(branches) == 0 {
			return m, nil
		}
		b := branches[m.branchIdx]
		if b.IsCurrent {
			m.errMsg = "Nao e possivel excluir a branch atual"
			return m, nil
		}
		if b.IsRemote {
			m.errMsg = "Exclusao disponivel apenas para branches locais"
			return m, nil
		}
		m.modal = modalDeleteBranch
		m.modalTarget = b.Name
		return m, nil
	case key.Matches(msg, m.keys.MergeBranch):
		if len(branches) == 0 {
			return m, nil
		}
		b := branches[m.branchIdx]
		if b.IsCurrent {
			return m, nil
		}
		m.loading = true
		m.errMsg = ""
		m.status = fmt.Sprintf("Mesclando %s...", b.Name)
		return m, tea.Batch(m.spinner.Tick, mergeBranchCmd(m.client, b.Name))
	}
	return m, nil
}

func (m Model) handleChangesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	files := m.visibleFiles()
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.fileIdx > 0 {
			m.fileIdx--
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Down):
		if m.fileIdx < len(files)-1 {
			m.fileIdx++
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Top):
		if len(files) > 0 && m.fileIdx != 0 {
			m.fileIdx = 0
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Bottom):
		if len(files) > 0 && m.fileIdx != len(files)-1 {
			m.fileIdx = len(files) - 1
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.ViewDiff):
		if len(files) == 0 {
			return m, nil
		}
		m.openDiffModal()
		return m, nil
	case key.Matches(msg, m.keys.Stage):
		if len(files) == 0 {
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, toggleStageCmd(m.client, files[m.fileIdx]))
	case key.Matches(msg, m.keys.StageAll):
		if len(m.files) == 0 {
			return m, nil
		}
		allStaged := true
		for _, f := range m.files {
			if !f.IsStaged || f.IsUnstaged || f.IsUntracked || f.IsConflicted {
				allStaged = false
				break
			}
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, toggleStageAllCmd(m.client, allStaged))
	case key.Matches(msg, m.keys.Commit):
		if m.state.ConflictCount > 0 {
			m.errMsg = "Resolva os conflitos antes de fazer commit"
			return m, nil
		}
		if m.state.StagedCount == 0 {
			m.errMsg = "Nenhum arquivo preparado (staged). Pressione Espaco ou 's' antes de commitar."
			return m, nil
		}
		m.modal = modalCommit
		m.input.SetValue("")
		m.input.Placeholder = "Mensagem do commit"
		m.input.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Stash):
		m.loading = true
		hasChanges := m.state.ChangedCount > 0
		return m, tea.Batch(m.spinner.Tick, stashToggleCmd(m.client, hasChanges))
	case key.Matches(msg, m.keys.Discard):
		if len(files) == 0 {
			return m, nil
		}
		m.modal = modalDiscard
		m.modalFile = files[m.fileIdx]
		m.modalTarget = m.modalFile.Path
		return m, nil
	}
	return m, nil
}

func (m Model) handleCommitsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	commits := m.visibleCommits()
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.commitIdx > 0 {
			m.commitIdx--
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Down):
		if m.commitIdx < len(commits)-1 {
			m.commitIdx++
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Top):
		if len(commits) > 0 && m.commitIdx != 0 {
			m.commitIdx = 0
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.Bottom):
		if len(commits) > 0 && m.commitIdx != len(commits)-1 {
			m.commitIdx = len(commits) - 1
			return m, m.refreshSelectionCmd()
		}
	case key.Matches(msg, m.keys.ViewDiff):
		if len(commits) == 0 {
			return m, nil
		}
		m.openDiffModal()
		return m, nil
	case key.Matches(msg, m.keys.RevertCommit):
		if len(commits) == 0 {
			return m, nil
		}
		m.modal = modalRevertCommit
		m.modalTarget = commits[m.commitIdx].Hash
		return m, nil
	case key.Matches(msg, m.keys.CherryPick):
		if len(commits) == 0 {
			return m, nil
		}
		m.modal = modalCherryPick
		m.modalTarget = commits[m.commitIdx].Hash
		return m, nil
	case key.Matches(msg, m.keys.ToggleCommitBranch):
		if m.historyCustomRef != "" {
			m.historyCustomRef = ""
			m.status = "Historico: exibindo HEAD atual"
		} else if branches := m.visibleBranches(); len(branches) > 0 {
			idx := min(m.branchIdx, len(branches)-1)
			m.historyCustomRef = branches[idx].Name
			m.status = fmt.Sprintf("Historico: exibindo branch %s", m.historyCustomRef)
		} else if len(m.branches) > 0 {
			m.historyCustomRef = m.branches[0].Name
			m.status = fmt.Sprintf("Historico: exibindo branch %s", m.historyCustomRef)
		}
		m.commitIdx = 0
		return m, loadCommitsCmd(m.client, m.historyRef())
	}
	return m, nil
}

func (m Model) handleModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	switch m.modal {
	case modalHelp:
		if k == "esc" || k == "?" || k == "q" || k == "enter" {
			m.modal = modalNone
		}
		return m, nil

	case modalDiffView:
		switch {
		case k == "esc" || k == "q" || k == "enter":
			m.modal = modalNone
			return m, nil
		case key.Matches(msg, m.keys.ToggleDiffMode):
			m.diffModeSet = true
			m.sideBySideDiff = !m.sideBySideDiff
			m.syncDiffModal()
			return m, nil
		case k == "up" || k == "k":
			m.diffModalViewport.ScrollUp(1)
			return m, nil
		case k == "down" || k == "j":
			m.diffModalViewport.ScrollDown(1)
			return m, nil
		case k == "pgup" || k == "ctrl+u" || k == "K":
			m.diffModalViewport.HalfPageUp()
			return m, nil
		case k == "pgdown" || k == "ctrl+d" || k == "J":
			m.diffModalViewport.HalfPageDown()
			return m, nil
		case k == "g" || k == "home":
			m.diffModalViewport.GotoTop()
			return m, nil
		case k == "G" || k == "end":
			m.diffModalViewport.GotoBottom()
			return m, nil
		case k == "[":
			if m.activeTab == tabChanges {
				if m.fileIdx > 0 {
					m.fileIdx--
					return m, m.refreshSelectionCmd()
				}
			} else if m.activeTab == tabCommits {
				if m.commitIdx > 0 {
					m.commitIdx--
					return m, m.refreshSelectionCmd()
				}
			}
			return m, nil
		case k == "]":
			if m.activeTab == tabChanges {
				files := m.visibleFiles()
				if m.fileIdx < len(files)-1 {
					m.fileIdx++
					return m, m.refreshSelectionCmd()
				}
			} else if m.activeTab == tabCommits {
				commits := m.visibleCommits()
				if m.commitIdx < len(commits)-1 {
					m.commitIdx++
					return m, m.refreshSelectionCmd()
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Stage):
			if m.activeTab == tabChanges {
				files := m.visibleFiles()
				if len(files) > 0 {
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, toggleStageCmd(m.client, files[m.fileIdx]))
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.OpenEditor):
			if m.activeTab == tabChanges {
				files := m.visibleFiles()
				if len(files) > 0 {
					return m, openEditorCmd(m.state.RootDir, files[m.fileIdx].Path)
				}
			}
			return m, nil
		}
		return m, nil

	case modalCreateBranch:
		switch k {
		case "esc":
			m.modal = modalNone
			m.input.Blur()
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			m.modal = modalNone
			m.input.Blur()
			if name == "" {
				return m, nil
			}
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, createBranchCmd(m.client, name))
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case modalCommit:
		switch k {
		case "esc":
			m.modal = modalNone
			m.input.Blur()
			return m, nil
		case "ctrl+g":
			cfg, err := config.Load()
			if err != nil {
				m.errMsg = fmt.Sprintf("Erro ao carregar configuracao: %v", err)
				return m, nil
			}
			if strings.TrimSpace(cfg.AIAPIKey) == "" {
				m.modal = modalAPIKey
				m.input.SetValue("")
				m.input.Placeholder = "cole sua chave de API..."
				m.input.EchoMode = textinput.EchoPassword
				m.input.EchoCharacter = '•'
				m.input.Focus()
				return m, nil
			}
			m.loading = true
			m.errMsg = ""
			return m, tea.Batch(m.spinner.Tick, generateCommitMsgCmd(m.client, cfg.AIAPIKey, cfg.AIModel))
		case "enter":
			message := strings.TrimSpace(m.input.Value())
			m.modal = modalNone
			m.input.Blur()
			if message == "" {
				m.errMsg = "A mensagem do commit nao pode estar vazia"
				return m, nil
			}
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, commitCmd(m.client, message))
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case modalAPIKey:
		switch k {
		case "esc":
			m.input.EchoMode = textinput.EchoNormal
			m.input.Placeholder = "feat: sua mensagem de commit"
			m.input.SetValue("")
			m.modal = modalCommit
			m.input.Focus()
			return m, nil
		case "ctrl+v", "ctrl+shift+v":
			if text, err := clipboard.ReadAll(); err == nil {
				text = strings.TrimSpace(text)
				m.input.SetValue(m.input.Value() + text)
				m.input.CursorEnd()
			}
			return m, nil
		case "enter":
			apiKey := strings.TrimSpace(m.input.Value())
			m.input.EchoMode = textinput.EchoNormal
			m.input.Placeholder = "feat: sua mensagem de commit"
			m.input.SetValue("")
			if apiKey == "" {
				m.errMsg = "A chave de API nao pode estar vazia"
				m.modal = modalCommit
				m.input.Focus()
				return m, nil
			}
			cfg, _ := config.Load()
			cfg.AIAPIKey = apiKey
			if err := config.Save(cfg); err != nil {
				m.errMsg = fmt.Sprintf("Erro ao salvar chave: %v", err)
				m.modal = modalCommit
				m.input.Focus()
				return m, nil
			}
			m.modal = modalCommit
			m.input.Focus()
			m.loading = true
			m.errMsg = ""
			return m, tea.Batch(m.spinner.Tick, generateCommitMsgCmd(m.client, cfg.AIAPIKey, cfg.AIModel))
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case modalDeleteBranch:
		switch k {
		case "esc", "n", "N":
			m.modal = modalNone
			return m, nil
		case "enter", "s", "S", "y", "Y":
			target := m.modalTarget
			m.modal = modalNone
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, deleteBranchCmd(m.client, target, false))
		}
		return m, nil

	case modalForceDelete:
		switch k {
		case "esc", "n", "N":
			m.modal = modalNone
			return m, nil
		case "enter", "f", "F", "s", "S", "y", "Y":
			target := m.modalTarget
			m.modal = modalNone
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, deleteBranchCmd(m.client, target, true))
		}
		return m, nil

	case modalDiscard:
		switch k {
		case "esc", "n", "N":
			m.modal = modalNone
			return m, nil
		case "enter", "s", "S", "y", "Y":
			f := m.modalFile
			m.modal = modalNone
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, discardFileCmd(m.client, f))
		}
		return m, nil

	case modalRevertCommit:
		switch k {
		case "esc", "n", "N":
			m.modal = modalNone
			return m, nil
		case "enter", "s", "S", "y", "Y":
			target := m.modalTarget
			m.modal = modalNone
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, revertCommitCmd(m.client, target))
		}
		return m, nil

	case modalCherryPick:
		switch k {
		case "esc", "n", "N":
			m.modal = modalNone
			return m, nil
		case "enter", "s", "S", "y", "Y":
			target := m.modalTarget
			m.modal = modalNone
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, cherryPickCmd(m.client, target))
		}
		return m, nil

	case modalStashSwitch:
		switch k {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "up", "k":
			if m.modalChoice > 0 {
				m.modalChoice--
			}
			return m, nil
		case "down", "j":
			if m.modalChoice < 2 {
				m.modalChoice++
			}
			return m, nil
		case "1":
			m.modalChoice = 0
			return m.execStashSwitch()
		case "2":
			m.modalChoice = 1
			return m.execStashSwitch()
		case "3":
			m.modal = modalNone
			return m, nil
		case "enter":
			return m.execStashSwitch()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) execStashSwitch() (tea.Model, tea.Cmd) {
	target := m.modalTarget
	choice := m.modalChoice
	m.modal = modalNone
	switch choice {
	case 0:
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, stashAndSwitchCmd(m.client, target))
	case 1:
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, switchBranchCmd(m.client, target))
	default:
		return m, nil
	}
}
