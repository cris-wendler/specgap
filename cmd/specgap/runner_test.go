package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A test tool that is not installed ran nothing, and reading that as a
// suite where every test failed reports the work as wrong when it was
// never graded. Both shapes are covered: the command is not there at
// all, and python3 is there without pytest.
func TestAMissingTestToolIsReportedRatherThanScored(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub interpreter is a shell script")
	}
	dir := t.TempDir()
	stub := filepath.Join(dir, "python3")
	script := "#!/bin/sh\necho 'python3: No module named pytest' >&2\nexit 1\n"
	if err := ioutil.WriteFile(stub, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := runSuite(dir, "pytest", nil); err == nil {
		t.Fatal("a missing pytest was graded instead of reported")
	} else if !strings.Contains(err.Error(), "pytest is not installed") {
		t.Errorf("the error does not name the missing tool: %v", err)
	}

	if err := os.Remove(stub); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if _, err := runSuite(dir, "go", nil); err == nil {
		t.Fatal("a missing go was graded instead of reported")
	} else if !strings.Contains(err.Error(), "go is not installed") {
		t.Errorf("the error does not name the missing tool: %v", err)
	}
}
