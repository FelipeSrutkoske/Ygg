package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up                 key.Binding
	Down               key.Binding
	Top                key.Binding
	Bottom             key.Binding
	PageUp             key.Binding
	PageDown           key.Binding
	NextTab            key.Binding
	Tab1               key.Binding
	Tab2               key.Binding
	Tab3               key.Binding
	Tab4               key.Binding
	ToggleRemotes      key.Binding
	Switch             key.Binding
	NewBranch          key.Binding
	RenameBranch       key.Binding
	DeleteBranch       key.Binding
	MergeBranch        key.Binding
	RebaseBranch       key.Binding
	Stage              key.Binding
	StageAll           key.Binding
	Commit             key.Binding
	Stash              key.Binding
	ApplyStash         key.Binding
	Discard            key.Binding
	ViewDiff           key.Binding
	ToggleDiffMode     key.Binding
	NextHunk           key.Binding
	PrevHunk           key.Binding
	RevertCommit       key.Binding
	CherryPick         key.Binding
	ToggleCommitBranch key.Binding
	CommitPrevFile     key.Binding
	CommitNextFile     key.Binding
	ContinueOp         key.Binding
	AbortOp            key.Binding
	KeepOurs           key.Binding
	AcceptTheirs       key.Binding
	OpenEditor         key.Binding
	Filter             key.Binding
	Fetch              key.Binding
	Pull               key.Binding
	Push               key.Binding
	Refresh            key.Binding
	ConfigureAPIKey    key.Binding
	Help               key.Binding
	Quit               key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("k/cima", "subir"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("j/baixo", "descer"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "início"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "fim"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u", "K"),
			key.WithHelp("K/PgUp", "rolar painel dir. acima"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d", "J"),
			key.WithHelp("J/PgDn", "rolar painel dir. abaixo"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "trocar aba"),
		),
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "aba Branches"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "aba Alterações"),
		),
		Tab3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "aba Histórico"),
		),
		Tab4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "aba Stashes"),
		),
		ToggleRemotes: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "mostrar/ocultar remotas"),
		),
		Switch: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "trocar branch"),
		),
		NewBranch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "nova branch"),
		),
		RenameBranch: key.NewBinding(
			key.WithKeys("M", "f2"),
			key.WithHelp("M/F2", "renomear branch"),
		),
		DeleteBranch: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "excluir branch"),
		),
		MergeBranch: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mesclar na atual"),
		),
		RebaseBranch: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "rebase na branch"),
		),
		Stage: key.NewBinding(
			key.WithKeys("space", "s"),
			key.WithHelp("Espaço/s", "preparar/remover (arq ou hunk)"),
		),
		StageAll: key.NewBinding(
			key.WithKeys("shift+space", "S"),
			key.WithHelp("Shift+Espaço/S", "preparar/remover tudo"),
		),
		Commit: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "fazer commit"),
		),
		Stash: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "criar stash / abrir stashes"),
		),
		ApplyStash: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "aplicar stash"),
		),
		Discard: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "descartar alterações / dropar stash"),
		),
		ViewDiff: key.NewBinding(
			key.WithKeys("enter", "d"),
			key.WithHelp("Enter/d", "abrir diff VS Code"),
		),
		ToggleDiffMode: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "lado a lado / unificado"),
		),
		NextHunk: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "próximo hunk"),
		),
		PrevHunk: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "hunk anterior"),
		),
		RevertCommit: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "reverter commit"),
		),
		CherryPick: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "cherry-pick commit"),
		),
		ToggleCommitBranch: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "alternar HEAD / branch"),
		),
		CommitPrevFile: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "arquivo ant. do commit"),
		),
		CommitNextFile: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "próx. arquivo do commit"),
		),
		ContinueOp: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "continuar merge/rebase"),
		),
		AbortOp: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "abortar merge/rebase"),
		),
		KeepOurs: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "manter nossos (ours)"),
		),
		AcceptTheirs: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "aceitar deles (theirs)"),
		),
		OpenEditor: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "abrir no $EDITOR"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filtrar itens"),
		),
		Fetch: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fetch"),
		),
		Pull: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pull / pop stash"),
		),
		Push: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "push"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R", "f5", "ctrl+r"),
			key.WithHelp("R/F5", "atualizar"),
		),
		ConfigureAPIKey: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("Ctrl+K", "chave de API"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "ajuda"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "sair"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextTab, k.Filter, k.ViewDiff, k.Fetch, k.Pull, k.Push, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.PageUp, k.PageDown, k.NextTab, k.Filter},
		{k.Switch, k.ToggleRemotes, k.NewBranch, k.RenameBranch, k.DeleteBranch, k.MergeBranch, k.RebaseBranch},
		{k.Stage, k.StageAll, k.Commit, k.Stash, k.ApplyStash, k.Discard, k.ViewDiff, k.ToggleDiffMode, k.NextHunk, k.PrevHunk, k.OpenEditor},
		{k.RevertCommit, k.CherryPick, k.ToggleCommitBranch, k.CommitPrevFile, k.CommitNextFile, k.ContinueOp, k.AbortOp, k.KeepOurs, k.AcceptTheirs},
		{k.Fetch, k.Pull, k.Push, k.Refresh, k.ConfigureAPIKey, k.Help, k.Quit},
	}
}
