package git

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type branchHeader struct {
	Kind     RepoKind
	Branch   string
	Upstream string
	Ahead    int
	Behind   int
}

func parseStatusPorcelainZ(data []byte) (branchHeader, []FileStatus, error) {
	header := branchHeader{Kind: KindNormal}
	if len(data) == 0 {
		return header, nil, nil
	}

	records := bytes.Split(data, []byte{0})
	if len(records) > 0 && len(records[len(records)-1]) == 0 {
		records = records[:len(records)-1]
	}

	idx := 0
	if len(records) > 0 && bytes.HasPrefix(records[0], []byte("## ")) {
		header = parseBranchHeader(string(records[0][3:]))
		idx = 1
	}

	var files []FileStatus
	for i := idx; i < len(records); i++ {
		rec := records[i]
		if len(rec) == 0 {
			continue
		}
		if len(rec) < 4 || rec[2] != ' ' {
			return header, nil, fmt.Errorf("malformed status record: %q", string(rec))
		}

		x := rec[0]
		y := rec[1]
		path := string(rec[3:])
		var oldPath string

		// no modo -z, rename/copy emite o caminho novo primeiro e o original no proximo registro
		if x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			if i+1 >= len(records) {
				return header, nil, fmt.Errorf("missing original path for rename/copy: %q", path)
			}
			i++
			oldPath = string(records[i])
		}

		files = append(files, buildFileStatus(x, y, path, oldPath))
	}

	return header, files, nil
}

func buildFileStatus(x, y byte, path, oldPath string) FileStatus {
	untracked := x == '?' && y == '?'
	conflicted := isConflictedPair(x, y)
	ignored := x == '!' && y == '!'

	staged := !untracked && !conflicted && !ignored && x != ' '
	unstaged := !untracked && !conflicted && !ignored && y != ' '

	return FileStatus{
		Path:         path,
		OldPath:      oldPath,
		IndexStatus:  x,
		WorkStatus:   y,
		IsConflicted: conflicted,
		IsUntracked:  untracked,
		IsStaged:     staged,
		IsUnstaged:   unstaged,
	}
}

func isConflictedPair(x, y byte) bool {
	if x == 'U' || y == 'U' {
		return true
	}
	return (x == 'D' && y == 'D') || (x == 'A' && y == 'A')
}

func parseBranchHeader(line string) branchHeader {
	for _, prefix := range []string{"No commits yet on ", "Initial commit on "} {
		if strings.HasPrefix(line, prefix) {
			return branchHeader{
				Kind:   KindUnborn,
				Branch: strings.TrimPrefix(line, prefix),
			}
		}
	}

	if line == "HEAD (no branch)" {
		return branchHeader{
			Kind: KindDetached,
		}
	}

	info := branchHeader{Kind: KindNormal}

	if local, rest, ok := strings.Cut(line, "..."); ok {
		info.Branch = local
		if upstream, track, hasTrack := strings.Cut(rest, " "); hasTrack {
			info.Upstream = upstream
			info.Ahead, info.Behind = parseAheadBehind(track)
		} else {
			info.Upstream = rest
		}
	} else {
		info.Branch = line
	}

	return info
}

func parseAheadBehind(track string) (ahead, behind int) {
	trimmed := strings.TrimSpace(track)
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	if trimmed == "" {
		return 0, 0
	}

	for _, part := range strings.Split(trimmed, ",") {
		part = strings.TrimSpace(part)
		if val, ok := strings.CutPrefix(part, "ahead "); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				ahead = n
			}
		} else if val, ok := strings.CutPrefix(part, "behind "); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				behind = n
			}
		}
	}
	return ahead, behind
}

func parseBranches(data []byte) []Branch {
	if len(data) == 0 {
		return nil
	}

	records := bytes.Split(data, []byte{0})
	var branches []Branch
	for _, rec := range records {
		if len(rec) == 0 {
			continue
		}
		fields := strings.Split(string(rec), "\x1f")
		if len(fields) < 8 {
			continue
		}

		head := strings.TrimSpace(fields[0])
		fullRef := fields[1]
		shortName := fields[2]
		upstream := fields[3]
		track := fields[4]
		lastHash := fields[5]
		lastSubject := fields[6]
		relDate := fields[7]

		if strings.HasPrefix(fullRef, "refs/remotes/") && strings.HasSuffix(fullRef, "/HEAD") {
			continue
		}

		ahead, behind := parseAheadBehind(track)
		branches = append(branches, Branch{
			Name:         shortName,
			IsCurrent:    head == "*",
			IsRemote:     strings.HasPrefix(fullRef, "refs/remotes/"),
			Upstream:     upstream,
			Ahead:        ahead,
			Behind:       behind,
			LastHash:     lastHash,
			LastSubject:  lastSubject,
			RelativeDate: relDate,
		})
	}
	return branches
}

func parseCommits(data []byte) []Commit {
	if len(data) == 0 {
		return nil
	}

	records := bytes.Split(data, []byte{0})
	var commits []Commit
	for _, rec := range records {
		rec = bytes.Trim(rec, "\r\n")
		if len(rec) == 0 {
			continue
		}
		fields := strings.Split(string(rec), "\x1f")
		if len(fields) < 4 {
			continue
		}
		var decorations string
		if len(fields) >= 5 {
			decorations = strings.TrimSpace(fields[4])
		}
		commits = append(commits, Commit{
			Hash:         strings.TrimSpace(fields[0]),
			Subject:      strings.TrimRight(fields[1], "\r\n"),
			Author:       strings.TrimSpace(fields[2]),
			RelativeDate: strings.TrimSpace(fields[3]),
			Decorations:  decorations,
		})
	}
	return commits
}

func parseCommitFiles(data []byte) []CommitFile {
	if len(data) == 0 {
		return nil
	}
	tokens := bytes.Split(data, []byte{0})
	var files []CommitFile
	for i := 0; i < len(tokens); i++ {
		tok := strings.TrimSpace(string(tokens[i]))
		if tok == "" {
			continue
		}
		if i+1 >= len(tokens) {
			break
		}
		status := tok
		path1 := string(tokens[i+1])
		i++
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if i+1 >= len(tokens) {
				break
			}
			path2 := string(tokens[i+1])
			i++
			files = append(files, CommitFile{
				Status:  status,
				OldPath: path1,
				Path:    path2,
			})
			continue
		}
		files = append(files, CommitFile{
			Status: status,
			Path:   path1,
		})
	}
	return files
}

func parseStashes(data []byte) []StashEntry {
	if len(data) == 0 {
		return nil
	}
	records := bytes.Split(data, []byte{0})
	var stashes []StashEntry
	for _, rec := range records {
		rec = bytes.Trim(rec, "\r\n")
		if len(rec) == 0 {
			continue
		}
		fields := strings.Split(string(rec), "\x1f")
		if len(fields) < 3 {
			continue
		}
		ref := strings.TrimSpace(fields[0])
		msg := strings.TrimRight(fields[1], "\r\n")
		relDate := strings.TrimSpace(fields[2])
		idx := len(stashes)
		if open := strings.Index(ref, "{"); open >= 0 {
			if closeIdx := strings.Index(ref[open+1:], "}"); closeIdx >= 0 {
				if n, err := strconv.Atoi(ref[open+1 : open+1+closeIdx]); err == nil {
					idx = n
				}
			}
		}
		stashes = append(stashes, StashEntry{
			Index:        idx,
			Ref:          ref,
			Message:      msg,
			RelativeDate: relDate,
		})
	}
	return stashes
}
