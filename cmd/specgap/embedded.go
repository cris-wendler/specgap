package main

import (
	"io/fs"
	"os"
	"path/filepath"

	specgap "github.com/cris-wendler/specgap"
)

// taskFiles reads the task directory. Files on disk win, so a checkout
// can change a task and see the change without rebuilding, and an
// installed executable falls back to what it was built with.
type taskFiles struct {
	dir string // empty when reading from the executable
}

func openTasks() taskFiles {
	if dir, err := diskTasks(); err == nil {
		return taskFiles{dir: dir}
	}
	return taskFiles{}
}

// diskTasks finds a tasks directory beside the command, by walking up
// from where it was started. SPECGAP_TASKS says where they are for
// anyone running from somewhere else.
func diskTasks() (string, error) {
	if env := os.Getenv("SPECGAP_TASKS"); env != "" {
		if _, err := os.Stat(filepath.Join(env, "tasks")); err != nil {
			return "", err
		}
		return env, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "tasks", "cache", "task.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func (t taskFiles) read(parts ...string) ([]byte, error) {
	if t.dir != "" {
		return os.ReadFile(filepath.Join(append([]string{t.dir}, parts...)...))
	}
	return fs.ReadFile(specgap.Tasks, path(parts...))
}

func (t taskFiles) names() ([]string, error) {
	var out []string
	if t.dir != "" {
		entries, err := os.ReadDir(filepath.Join(t.dir, "tasks"))
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				out = append(out, e.Name())
			}
		}
		return out, nil
	}
	entries, err := fs.ReadDir(specgap.Tasks, "tasks")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// where says which copy of the tasks is being used, for a message that
// has to explain what was not found.
func (t taskFiles) where() string {
	if t.dir != "" {
		return t.dir
	}
	return "the tasks built into this executable"
}

// path joins with forward slashes, which is what an embedded file
// system uses on every operating system.
func path(parts ...string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "/"
		}
		out += p
	}
	return out
}
