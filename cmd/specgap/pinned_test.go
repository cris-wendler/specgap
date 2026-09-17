package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// pinnedRepo builds a repository whose working tree has moved on from
// the commit it returns. That is the only shape in which a task pinned
// to a commit can disagree with the repository it was cut from, and it
// is the shape every real one is in: the repository keeps being worked
// on after the task is written.
func pinnedRepo(t *testing.T) (dir, commit string) {
	t.Helper()
	dir = t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=t@e.invalid",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=t@e.invalid",
			"GIT_CONFIG_GLOBAL="+filepath.Join(dir, "gitconfig"), "GIT_CONFIG_NOSYSTEM=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "--quiet", "--initial-branch=main", ".")
	write := func(name, body string) {
		if err := ioutil.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module pinned\n\ngo 1.17\n")
	write("app.go", "package pinned\n\nfunc Answer() int { return 1 }\n")
	// The hidden test as it was when the task was written.
	write("hidden_test.go", "package pinned\n\nimport \"testing\"\n\nfunc TestAtThePin(t *testing.T) {\n\tif Answer() != 1 {\n\t\tt.Fatal(\"no\")\n\t}\n}\n")
	git("add", "-A")
	git("commit", "--quiet", "--message", "the commit the task is pinned to")
	commit = git("rev-parse", "HEAD")

	// The repository moves on. The hidden test now asks for something the
	// pinned commit knows nothing about.
	write("app.go", "package pinned\n\nfunc Answer() int { return 1 }\n\nfunc Later() int { return 2 }\n")
	write("hidden_test.go", "package pinned\n\nimport \"testing\"\n\nfunc TestAfterThePin(t *testing.T) {\n\tif Later() != 2 {\n\t\tt.Fatal(\"no\")\n\t}\n}\n")
	git("add", "-A")
	git("commit", "--quiet", "--message", "work that came after")
	return dir, commit
}

// gitIn runs git in a fixture repository built by pinnedRepo.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=t@e.invalid",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=t@e.invalid",
		"GIT_CONFIG_GLOBAL="+filepath.Join(dir, "gitconfig"), "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func commitAll(t *testing.T, dir, message string) {
	t.Helper()
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "--quiet", "--message", message)
}

func headOf(t *testing.T, dir string) string {
	t.Helper()
	return gitIn(t, dir, "rev-parse", "HEAD")
}

func pinnedTask(dir, commit string) Task {
	return Task{
		Name: "pinned", Kind: "repo", Repo: dir, Commit: commit,
		Hidden: map[string]string{"hidden_test.go": "hidden_test.go"},
		files:  openTasks(), name: "pinned",
	}
}

// A task that names a commit is a promise that it grades one fixed pair
// of code and tests. The hidden tests used to be read from the working
// tree while the workspace was cut from the commit, so the two were
// pinned differently and drifted apart the moment anybody edited one of
// those files. The task then graded code from one point in history
// against tests from another, and the reference solution failed on a
// test for a feature its own commit predates.
func TestHiddenTestsComeFromTheCommitTheWorkspaceWasCutFrom(t *testing.T) {
	dir, commit := pinnedRepo(t)
	task := pinnedTask(dir, commit)

	work := t.TempDir()
	if err := task.write(work); err != nil {
		t.Fatal(err)
	}
	names, remove, err := task.plant(work)
	if err != nil {
		t.Fatal(err)
	}
	defer remove()

	planted, err := ioutil.ReadFile(filepath.Join(work, "hidden_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(planted), "TestAfterThePin") {
		t.Error("the hidden test came from the working tree, so the task grades " +
			"code from the commit against tests from today")
	}
	if !strings.Contains(string(planted), "TestAtThePin") {
		t.Fatal("the hidden test is not the one the pinned commit had")
	}
	if len(names) != 1 || names[0] != "TestAtThePin" {
		t.Fatalf("graded on %v, want only the test the pin has", names)
	}
}

// The whole point of pinning is that the grade does not move when the
// repository does. This runs the suite, which is what a real attempt
// does, and requires it to pass on untouched pinned code.
func TestAPinnedTaskStillGradesAfterTheRepositoryMovesOn(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is needed to run the planted suite")
	}
	dir, commit := pinnedRepo(t)
	task := pinnedTask(dir, commit)

	work := t.TempDir()
	if err := task.write(work); err != nil {
		t.Fatal(err)
	}
	names, remove, err := task.plant(work)
	if err != nil {
		t.Fatal(err)
	}
	defer remove()

	res, err := runSuite(work, task.Runner, names)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Failed) > 0 {
		t.Fatalf("the pinned code failed its own pinned tests: %v", res.Failed)
	}
	if len(res.Passed) != 1 {
		t.Fatalf("passed %v, want the one test the pin has", res.Passed)
	}
}

// A hidden file the task ships itself is not in the repository at all,
// and must still be found. This is the other half of the lookup, and
// reading from a commit must not have broken it.
func TestAHiddenFileTheTaskShipsIsStillFound(t *testing.T) {
	dir, commit := pinnedRepo(t)
	task := pinnedTask(dir, commit)
	task.Hidden = map[string]string{"acceptance_test.go.txt": "extra_test.go"}
	task.name = "zeroturn-threshold"

	work := t.TempDir()
	if err := task.write(work); err != nil {
		t.Fatal(err)
	}
	_, remove, err := task.plant(work)
	if err != nil {
		t.Fatalf("a hidden file shipped by the task was not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "extra_test.go")); err != nil {
		t.Fatal("the shipped hidden file was not written into the workspace")
	}
	// Planted tests are taken away again after grading, so the workspace
	// is left as the agent had it.
	remove()
	if _, err := os.Stat(filepath.Join(work, "extra_test.go")); !os.IsNotExist(err) {
		t.Error("the planted file was left behind after grading")
	}
}

// A hidden path that is in neither place has to say so, naming the
// commit, because the usual cause is a file that was renamed after the
// task was written.
func TestAMissingHiddenPathNamesTheCommit(t *testing.T) {
	dir, commit := pinnedRepo(t)
	task := pinnedTask(dir, commit)
	task.Hidden = map[string]string{"not_there_test.go": "not_there_test.go"}

	work := t.TempDir()
	if err := task.write(work); err != nil {
		t.Fatal(err)
	}
	_, _, err := task.plant(work)
	if err == nil {
		t.Fatal("a hidden path that does not exist was accepted")
	}
	for _, want := range []string{"not_there_test.go", commit} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not name %q: %v", want, err)
		}
	}
}
