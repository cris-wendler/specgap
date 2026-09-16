package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Work is what an attempt did to the workspace, as opposed to whether it
// passed.
//
// A score says an agent solved the task. It cannot say whether it solved
// it by changing four files or forty, which is the difference between
// two attempts that a preference between them would be drawn from. The
// environment already runs several attempts from an identical starting
// state and threw all of this away.
//
// Everything here is taken from the workspace itself, before and after,
// so it needs no cooperation from the agent and no access to what the
// model generated.
type Work struct {
	Added   []string `json:"filesAdded,omitempty"`
	Changed []string `json:"filesChanged,omitempty"`
	Removed []string `json:"filesRemoved,omitempty"`
	// Lines counts only files that are text on both sides, because a
	// line count of a compiled binary is not a measure of anything.
	LinesAdded   int `json:"linesAdded"`
	LinesRemoved int `json:"linesRemoved"`
}

// Touched reports how many files the attempt wrote to at all, which is
// the coarsest measure of how widely it worked.
func (w Work) Touched() int {
	return len(w.Added) + len(w.Changed) + len(w.Removed)
}

// Churn reports how many lines moved in either direction. Two attempts
// that both pass and differ by an order of magnitude here did not do the
// same thing.
func (w Work) Churn() int { return w.LinesAdded + w.LinesRemoved }

// fileState is what is remembered about one file between the two
// snapshots. The digest answers whether it changed; the line count
// answers by how much, and is zero for anything that is not text.
type fileState struct {
	digest string
	lines  int
	text   bool
}

// snapshot records every file in a directory. It is taken twice, once
// before the agent runs and once after.
func snapshot(dir string) (map[string]fileState, error) {
	out := map[string]fileState{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// A workspace has no repository of its own, and a directory
			// of build output is not work the agent chose to do.
			switch info.Name() {
			case ".git", "node_modules", "__pycache__", ".pytest_cache":
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, rerr := filepath.Rel(dir, path)
		if rerr != nil {
			return rerr
		}
		st, serr := readState(path)
		if serr != nil {
			// A file that cannot be read is recorded as present and
			// opaque rather than dropped, so it still counts as changed
			// if it appears or disappears.
			out[filepath.ToSlash(rel)] = fileState{digest: "unreadable"}
			return nil
		}
		out[filepath.ToSlash(rel)] = st
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func readState(path string) (fileState, error) {
	f, err := os.Open(path)
	if err != nil {
		return fileState{}, err
	}
	defer f.Close()

	h := sha256.New()
	var head bytes.Buffer
	if _, err := io.Copy(h, io.TeeReader(io.LimitReader(f, 8000), &head)); err != nil {
		return fileState{}, err
	}
	text := !bytes.Contains(head.Bytes(), []byte{0})
	if _, err := io.Copy(h, f); err != nil {
		return fileState{}, err
	}
	st := fileState{digest: hex.EncodeToString(h.Sum(nil)), text: text}
	if text {
		st.lines, err = countLines(path)
		if err != nil {
			return fileState{}, err
		}
	}
	return st, nil
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		n++
	}
	if err := sc.Err(); err != nil {
		// A file with a line too long to scan still counted as a file.
		// Its lines are not counted, which is better than failing the
		// whole attempt over one of them.
		return n, nil
	}
	return n, nil
}

// compare reports what changed between two snapshots.
func compare(before, after map[string]fileState) Work {
	var w Work
	for path, now := range after {
		was, existed := before[path]
		switch {
		case !existed:
			w.Added = append(w.Added, path)
			if now.text {
				w.LinesAdded += now.lines
			}
		case was.digest != now.digest:
			w.Changed = append(w.Changed, path)
			// Only a file that is text on both sides has a line count
			// worth subtracting one from the other.
			if was.text && now.text {
				if d := now.lines - was.lines; d > 0 {
					w.LinesAdded += d
				} else {
					w.LinesRemoved += -d
				}
			}
		}
	}
	for path, was := range before {
		if _, still := after[path]; !still {
			w.Removed = append(w.Removed, path)
			if was.text {
				w.LinesRemoved += was.lines
			}
		}
	}
	sort.Strings(w.Added)
	sort.Strings(w.Changed)
	sort.Strings(w.Removed)
	return w
}

// describe writes the work as one line for the report.
func (w Work) describe() string {
	if w.Touched() == 0 {
		return "changed nothing"
	}
	var parts []string
	if n := len(w.Added); n > 0 {
		parts = append(parts, plural(n, "file")+" added")
	}
	if n := len(w.Changed); n > 0 {
		parts = append(parts, plural(n, "file")+" changed")
	}
	if n := len(w.Removed); n > 0 {
		parts = append(parts, plural(n, "file")+" removed")
	}
	lines := plural(w.LinesAdded, "line") + " in, " + plural(w.LinesRemoved, "line") + " out"
	return strings.Join(parts, ", ") + ", " + lines
}
