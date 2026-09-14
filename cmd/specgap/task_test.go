package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// repoRoot points the command at this repository from wherever the test
// binary runs.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadTaskReadsTheManifest(t *testing.T) {
	os.Setenv("SPECGAP_TASKS", repoRoot(t))
	defer os.Unsetenv("SPECGAP_TASKS")

	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}
	if task.Name != "cache" || task.Kind != "files" {
		t.Fatalf("name %q kind %q", task.Name, task.Kind)
	}
	if task.Summary == "" {
		t.Error("a task with no summary cannot be listed usefully")
	}
	if len(task.Visible) == 0 || len(task.Hidden) == 0 {
		t.Errorf("visible %d hidden %d", len(task.Visible), len(task.Hidden))
	}
}

func TestLoadTaskSaysWhichNameItCouldNotFind(t *testing.T) {
	os.Setenv("SPECGAP_TASKS", repoRoot(t))
	defer os.Unsetenv("SPECGAP_TASKS")

	_, err := loadTask("no-such-task")
	if err == nil {
		t.Fatal("a task that does not exist loaded")
	}
	if !strings.Contains(err.Error(), "no-such-task") {
		t.Errorf("the error does not name it: %v", err)
	}
}

func TestLoadTaskRefusesAManifestItCannotRead(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "tasks", "broken")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "task.json"), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("SPECGAP_TASKS", root)
	defer os.Unsetenv("SPECGAP_TASKS")

	if _, err := loadTask("broken"); err == nil {
		t.Fatal("a manifest that is not JSON loaded")
	}
}

// A broken task must not stop the others being listed, because the list
// is how somebody finds out what is there.
func TestTasksSkipsOneItCannotRead(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "tasks", "good")
	bad := filepath.Join(root, "tasks", "bad")
	for _, d := range []string{good, bad} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := ioutil.WriteFile(filepath.Join(good, "task.json"), []byte(`{"name":"good","kind":"files"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(bad, "task.json"), []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("SPECGAP_TASKS", root)
	defer os.Unsetenv("SPECGAP_TASKS")

	all, err := tasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Name != "good" {
		t.Fatalf("listed %v", all)
	}
}

func TestEveryTaskInThisRepositoryLoads(t *testing.T) {
	os.Setenv("SPECGAP_TASKS", repoRoot(t))
	defer os.Unsetenv("SPECGAP_TASKS")

	all, err := tasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 2 {
		t.Fatalf("only %d tasks loaded", len(all))
	}
	for _, task := range all {
		if task.Name == "" || task.Kind == "" || task.Summary == "" {
			t.Errorf("%+v is missing a name, a kind or a summary", task)
		}
		if len(task.Hidden) == 0 {
			t.Errorf("%s has no hidden tests, so it grades nothing", task.Name)
		}
	}
}

func TestAFilesTaskWritesOnlyWhatTheAgentMaySee(t *testing.T) {
	os.Setenv("SPECGAP_TASKS", repoRoot(t))
	defer os.Unsetenv("SPECGAP_TASKS")

	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := task.write(dir); err != nil {
		t.Fatal(err)
	}
	var written []string
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		written = append(written, e.Name())
	}
	sort.Strings(written)
	want := []string{"SPEC.md", "cache.go", "cache_test.go", "go.mod"}
	if strings.Join(written, ",") != strings.Join(want, ",") {
		t.Fatalf("workspace holds %v, want %v", written, want)
	}
	// The thing that must never be there.
	for _, to := range task.Hidden {
		if _, err := os.Stat(filepath.Join(dir, to)); err == nil {
			t.Errorf("a hidden test was written into the workspace: %s", to)
		}
	}
}

func TestAnUnknownKindIsRefused(t *testing.T) {
	task := Task{Name: "odd", Kind: "something-else"}
	if err := task.write(t.TempDir()); err == nil {
		t.Fatal("a task with a kind nobody implemented was written")
	}
}

func TestRepoPathPrefersTheEnvironment(t *testing.T) {
	elsewhere := t.TempDir()
	os.Setenv("SPECGAP_REPO", elsewhere)
	defer os.Unsetenv("SPECGAP_REPO")

	got, err := Task{Repo: "../does-not-exist"}.repoPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != elsewhere {
		t.Fatalf("got %q, want %q", got, elsewhere)
	}
}

func TestRepoPathSaysWhatIsMissingAndHowToFixIt(t *testing.T) {
	os.Setenv("SPECGAP_TASKS", repoRoot(t))
	defer os.Unsetenv("SPECGAP_TASKS")

	_, err := Task{Repo: "../definitely-not-here"}.repoPath()
	if err == nil {
		t.Fatal("a repository that is not there resolved")
	}
	for _, want := range []string{"../definitely-not-here", "SPECGAP_REPO"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}

// TestMain is a package's entry point, not a test. Grading on it
// measures whether the harness started.
func TestPlantDoesNotGradeTestMain(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "tasks", "t")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "package x\n\nfunc TestMain(m *testing.M) {}\nfunc TestReal(t *testing.T) {}\n"
	if err := ioutil.WriteFile(filepath.Join(dir, "h.txt"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "task.json"),
		[]byte(`{"name":"t","kind":"files","hidden":{"h.txt":"zz_test.go"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("SPECGAP_TASKS", root)
	defer os.Unsetenv("SPECGAP_TASKS")

	task, err := loadTask("t")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	names, remove, err := task.plant(work)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(names, ",") != "TestReal" {
		t.Fatalf("graded %v", names)
	}
	if _, err := os.Stat(filepath.Join(work, "zz_test.go")); err != nil {
		t.Fatalf("the hidden test was not planted: %v", err)
	}
	remove()
	if _, err := os.Stat(filepath.Join(work, "zz_test.go")); !os.IsNotExist(err) {
		t.Fatal("the hidden test was left in the workspace, where an agent could read it")
	}
}
