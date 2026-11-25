package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/ai"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
)

const autoRefreshInterval = 2 * time.Second

type autoRefreshTickMsg struct{}

func autoRefreshTickCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(time.Time) tea.Msg {
		return autoRefreshTickMsg{}
	})
}

type repoLoadedMsg struct {
	state    git.RepoState
	branches []git.Branch
	status   []git.FileStatus
	stashes  []git.StashEntry
	lastCmd  string
	silent   bool
	err      error
}

type commitsLoadedMsg struct {
	ref     string
	commits []git.Commit
	silent  bool
	err     error
}

type moreCommitsLoadedMsg struct {
	ref     string
	skip    int
	commits []git.Commit
	err     error
}

type diffLoadedMsg struct {
	path   string
	diff   string
	silent bool
	err    error
}

type commitDiffLoadedMsg struct {
	hash    string
	fileIdx int
	files   []git.CommitFile
	diff    string
	silent  bool
	err     error
}

type stashDiffLoadedMsg struct {
	ref    string
	diff   string
	silent bool
	err    error
}

type actionDoneMsg struct {
	message string
	lastCmd string
	err     error
}

type mergeDoneMsg struct {
	branch   string
	isRebase bool
	res      git.MergeResult
	lastCmd  string
	err      error
}

type deleteBranchErrMsg struct {
	branch  string
	force   bool
	lastCmd string
	err     error
}

type editorFinishedMsg struct {
	err error
}

func loadRepoCmd(client *git.Client, includeRemote bool) tea.Cmd {
	return loadRepoWithSilentCmd(client, includeRemote, false)
}

func loadRepoSilentCmd(client *git.Client, includeRemote bool) tea.Cmd {
	return loadRepoWithSilentCmd(client, includeRemote, true)
}

func loadRepoWithSilentCmd(client *git.Client, includeRemote, silent bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		state, err := client.InspectRepo(ctx)
		if err != nil {
			return repoLoadedMsg{lastCmd: client.LastCommand(), silent: silent, err: err}
		}
		if state.Kind == git.KindNotARepo {
			return repoLoadedMsg{state: state, lastCmd: client.LastCommand(), silent: silent}
		}
		branches, err := client.ListBranches(ctx, includeRemote)
		if err != nil {
			return repoLoadedMsg{state: state, lastCmd: client.LastCommand(), silent: silent, err: err}
		}
		status, err := client.ListStatus(ctx)
		if err != nil {
			return repoLoadedMsg{state: state, branches: branches, lastCmd: client.LastCommand(), silent: silent, err: err}
		}
		stashes, _ := client.ListStashes(ctx)
		return repoLoadedMsg{
			state:    state,
			branches: branches,
			status:   status,
			stashes:  stashes,
			lastCmd:  client.LastCommand(),
			silent:   silent,
		}
	}
}

func loadCommitsCmd(client *git.Client, ref string) tea.Cmd {
	return loadCommitsWithSilentCmd(client, ref, false)
}

func loadCommitsWithSilentCmd(client *git.Client, ref string, silent bool) tea.Cmd {
	return func() tea.Msg {
		commits, err := client.ListCommits(context.Background(), ref, 50)
		return commitsLoadedMsg{ref: ref, commits: commits, silent: silent, err: err}
	}
}

func loadMoreCommitsCmd(client *git.Client, ref string, skip, limit int) tea.Cmd {
	return func() tea.Msg {
		commits, err := client.ListCommitsPage(context.Background(), ref, limit, skip)
		return moreCommitsLoadedMsg{ref: ref, skip: skip, commits: commits, err: err}
	}
}

func loadDiffCmd(client *git.Client, file git.FileStatus) tea.Cmd {
	return loadDiffWithSilentCmd(client, file, false)
}

func loadDiffWithSilentCmd(client *git.Client, file git.FileStatus, silent bool) tea.Cmd {
	return func() tea.Msg {
		diff, err := client.DiffFile(context.Background(), file)
		return diffLoadedMsg{path: file.Path, diff: diff, silent: silent, err: err}
	}
}

func loadCommitDiffCmd(client *git.Client, hash string) tea.Cmd {
	return loadCommitDiffWithSilentCmd(client, hash, false)
}

func loadCommitDiffWithSilentCmd(client *git.Client, hash string, silent bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		diff, err := client.ShowCommit(ctx, hash)
		var files []git.CommitFile
		if err == nil {
			files, _ = client.ListCommitFiles(ctx, hash)
		}
		return commitDiffLoadedMsg{hash: hash, fileIdx: 0, files: files, diff: diff, silent: silent, err: err}
	}
}

func loadCommitFileDiffCmd(client *git.Client, hash string, fileIdx int, files []git.CommitFile) tea.Cmd {
	return loadCommitFileDiffWithSilentCmd(client, hash, fileIdx, files, false)
}

func loadCommitFileDiffWithSilentCmd(client *git.Client, hash string, fileIdx int, files []git.CommitFile, silent bool) tea.Cmd {
	return func() tea.Msg {
		if fileIdx <= 0 || fileIdx > len(files) {
			diff, err := client.ShowCommit(context.Background(), hash)
			return commitDiffLoadedMsg{hash: hash, fileIdx: 0, files: files, diff: diff, silent: silent, err: err}
		}
		diff, err := client.DiffCommitFile(context.Background(), hash, files[fileIdx-1].Path)
		return commitDiffLoadedMsg{hash: hash, fileIdx: fileIdx, files: files, diff: diff, silent: silent, err: err}
	}
}

func loadStashDiffCmd(client *git.Client, ref string) tea.Cmd {
	return loadStashDiffWithSilentCmd(client, ref, false)
}

func loadStashDiffWithSilentCmd(client *git.Client, ref string, silent bool) tea.Cmd {
	return func() tea.Msg {
		diff, err := client.ShowStash(context.Background(), ref)
		return stashDiffLoadedMsg{ref: ref, diff: diff, silent: silent, err: err}
	}
}

func switchBranchCmd(client *git.Client, name string) tea.Cmd {
	return func() tea.Msg {
		if err := client.SwitchBranch(context.Background(), name); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Branch alterada para %s", name), lastCmd: client.LastCommand()}
	}
}

func stashAndSwitchCmd(client *git.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := client.StashPush(ctx, "go-vit: stash automatico antes de trocar para "+name); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		if err := client.SwitchBranch(ctx, name); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Stash salvo e branch alterada para %s", name), lastCmd: client.LastCommand()}
	}
}

func createBranchCmd(client *git.Client, name string) tea.Cmd {
	return func() tea.Msg {
		if err := client.CreateBranch(context.Background(), name); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Branch %s criada e selecionada", name), lastCmd: client.LastCommand()}
	}
}

func renameBranchCmd(client *git.Client, oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		if err := client.RenameBranch(context.Background(), oldName, newName); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Branch %s renomeada para %s", oldName, newName), lastCmd: client.LastCommand()}
	}
}

func deleteBranchCmd(client *git.Client, name string, force bool) tea.Cmd {
	return func() tea.Msg {
		if err := client.DeleteBranch(context.Background(), name, force); err != nil {
			return deleteBranchErrMsg{branch: name, force: force, lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Branch %s excluida", name), lastCmd: client.LastCommand()}
	}
}

func mergeBranchCmd(client *git.Client, name string) tea.Cmd {
	return func() tea.Msg {
		res, err := client.MergeBranch(context.Background(), name)
		return mergeDoneMsg{branch: name, res: res, lastCmd: client.LastCommand(), err: err}
	}
}

func rebaseBranchCmd(client *git.Client, onto string) tea.Cmd {
	return func() tea.Msg {
		res, err := client.RebaseBranch(context.Background(), onto)
		return mergeDoneMsg{branch: onto, isRebase: true, res: res, lastCmd: client.LastCommand(), err: err}
	}
}

func continueOpCmd(client *git.Client, op git.ActiveOpKind) tea.Cmd {
	return func() tea.Msg {
		if err := client.ContinueActiveOp(context.Background(), op); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("%s continuado com sucesso", op.Label()), lastCmd: client.LastCommand()}
	}
}

func abortOpCmd(client *git.Client, op git.ActiveOpKind) tea.Cmd {
	return func() tea.Msg {
		if err := client.AbortActiveOp(context.Background(), op); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("%s abortado", op.Label()), lastCmd: client.LastCommand()}
	}
}

func toggleStageCmd(client *git.Client, file git.FileStatus) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if file.IsStaged && !file.IsUnstaged && !file.IsConflicted {
			if err := client.UnstageFile(ctx, file.Path); err != nil {
				return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
			}
			return actionDoneMsg{message: fmt.Sprintf("Removido do stage: %s", file.Path), lastCmd: client.LastCommand()}
		}
		if err := client.StageFile(ctx, file.Path); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Preparado (staged): %s", file.Path), lastCmd: client.LastCommand()}
	}
}

func toggleStageAllCmd(client *git.Client, allStaged bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if allStaged {
			if err := client.UnstageAll(ctx); err != nil {
				return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
			}
			return actionDoneMsg{message: "Todos os arquivos removidos do stage", lastCmd: client.LastCommand()}
		}
		if err := client.StageAll(ctx); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: "Todos os arquivos preparados (staged)", lastCmd: client.LastCommand()}
	}
}

func applyHunkCmd(client *git.Client, path, patch string, reverse bool) tea.Cmd {
	return func() tea.Msg {
		if err := client.ApplyHunkCached(context.Background(), patch, reverse); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		verb := "Hunk preparado (staged)"
		if reverse {
			verb = "Hunk removido do stage"
		}
		return actionDoneMsg{message: fmt.Sprintf("%s: %s", verb, path), lastCmd: client.LastCommand()}
	}
}

func discardHunkCmd(client *git.Client, path, patch string, staged bool) tea.Cmd {
	return func() tea.Msg {
		if err := client.DiscardHunk(context.Background(), patch, staged); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Hunk descartado: %s", path), lastCmd: client.LastCommand()}
	}
}

func commitCmd(client *git.Client, message string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Commit(context.Background(), message); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: "Commit realizado com sucesso", lastCmd: client.LastCommand()}
	}
}

func stashPushMsgCmd(client *git.Client, message string) tea.Cmd {
	return func() tea.Msg {
		if err := client.StashPush(context.Background(), message); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: "Alteracoes guardadas no stash", lastCmd: client.LastCommand()}
	}
}

func stashApplyCmd(client *git.Client, ref string) tea.Cmd {
	return func() tea.Msg {
		if err := client.StashApply(context.Background(), ref); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Stash aplicado: %s", ref), lastCmd: client.LastCommand()}
	}
}

func stashPopRefCmd(client *git.Client, ref string) tea.Cmd {
	return func() tea.Msg {
		if err := client.StashPopRef(context.Background(), ref); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Stash restaurado (pop): %s", ref), lastCmd: client.LastCommand()}
	}
}

func stashDropCmd(client *git.Client, ref string) tea.Cmd {
	return func() tea.Msg {
		if err := client.StashDrop(context.Background(), ref); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Stash descartado: %s", ref), lastCmd: client.LastCommand()}
	}
}

func discardFileCmd(client *git.Client, file git.FileStatus) tea.Cmd {
	return func() tea.Msg {
		if err := client.DiscardFile(context.Background(), file); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Alteracoes descartadas: %s", file.Path), lastCmd: client.LastCommand()}
	}
}

func resolveConflictCmd(client *git.Client, path string, side git.ConflictSide) tea.Cmd {
	return func() tea.Msg {
		if err := client.ResolveConflict(context.Background(), path, side); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		label := "nossos (ours)"
		if side == git.SideTheirs {
			label = "deles (theirs)"
		}
		return actionDoneMsg{message: fmt.Sprintf("Conflito resolvido (%s): %s", label, path), lastCmd: client.LastCommand()}
	}
}

func fetchCmd(client *git.Client) tea.Cmd {
	return func() tea.Msg {
		if err := client.Fetch(context.Background()); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: "Fetch concluido", lastCmd: client.LastCommand()}
	}
}

func pullCmd(client *git.Client) tea.Cmd {
	return func() tea.Msg {
		if err := client.Pull(context.Background()); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: "Pull concluido", lastCmd: client.LastCommand()}
	}
}

func pushCmd(client *git.Client) tea.Cmd {
	return func() tea.Msg {
		msg, err := client.Push(context.Background())
		if err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: msg, lastCmd: client.LastCommand()}
	}
}

func revertCommitCmd(client *git.Client, hash string) tea.Cmd {
	return func() tea.Msg {
		if err := client.RevertCommit(context.Background(), hash); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Commit %s revertido", hash), lastCmd: client.LastCommand()}
	}
}

func cherryPickCmd(client *git.Client, hash string) tea.Cmd {
	return func() tea.Msg {
		if err := client.CherryPick(context.Background(), hash); err != nil {
			return actionDoneMsg{lastCmd: client.LastCommand(), err: err}
		}
		return actionDoneMsg{message: fmt.Sprintf("Cherry-pick de %s aplicado", hash), lastCmd: client.LastCommand()}
	}
}

func openEditorCmd(rootDir, relPath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	fullPath := filepath.Join(rootDir, filepath.FromSlash(relPath))
	c := exec.Command(editor, fullPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

type commitMsgGeneratedMsg struct {
	message string
	err     error
}

func generateCommitMsgCmd(client *git.Client, apiKey, model string) tea.Cmd {
	return func() tea.Msg {
		diff, err := client.DiffStaged(context.Background())
		if err != nil {
			return commitMsgGeneratedMsg{err: err}
		}
		aiClient := ai.NewClient(apiKey, model)
		msg, err := aiClient.GenerateCommitMessage(context.Background(), diff)
		return commitMsgGeneratedMsg{message: msg, err: err}
	}
}
