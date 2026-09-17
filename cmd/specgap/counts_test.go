package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The numbers in these documents were kept by hand beside the table they
// describe, and they drifted. The README said nine agent runs and the
// results document said twelve, of the same runs. Both said two of the
// six when the table held five. The README said the pinned repository
// has about 300 tests and the task file said about 380.
//
// Nobody reads both documents at once except a person deciding whether
// to believe either. So the numbers are read out of the table now, and
// out of the task file, rather than written down twice.

func resultsDoc(t *testing.T) string {
	t.Helper()
	b, err := ioutil.ReadFile(filepath.Join(repoRoot(t), "docs", "results.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// flat joins wrapped prose into one line, so a sentence broken across a
// line break still reads as a sentence.
func flat(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// runsAndConfigurations reads the one table whose first column is Runs.
// Every row carries a run count, and the number of rows is the number of
// configurations tried. Scoping it to that table matters: the document
// holds other tables whose rows also begin with a number.
func runsAndConfigurations(t *testing.T) (runs, configurations int) {
	t.Helper()
	lines := strings.Split(resultsDoc(t), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "| Runs |") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatal("docs/results.md has no table of runs, so this test checks nothing")
	}
	for _, l := range lines[start+1:] {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "|") {
			break
		}
		cells := strings.Split(strings.Trim(l, "|"), "|")
		n, err := strconv.Atoi(strings.TrimSpace(cells[0]))
		if err != nil {
			continue // the --- separator row
		}
		runs += n
		configurations++
	}
	if configurations == 0 {
		t.Fatal("the table of runs has no rows, so this test checks nothing")
	}
	return runs, configurations
}

var words = map[int]string{
	1: "one", 2: "two", 3: "three", 4: "four", 5: "five",
	6: "six", 7: "seven", 8: "eight", 9: "nine", 10: "ten",
	11: "eleven", 12: "twelve",
}

// Both documents open by saying how many runs there have been, and it is
// the first number a reader checks against the table underneath.
func TestBothDocumentsAgreeOnHowManyRuns(t *testing.T) {
	runs, _ := runsAndConfigurations(t)

	// The word immediately before "agent runs", and nothing else on the
	// line. Searching the whole line for the digit matched the 9 inside
	// the date 2026-09-14 and passed while the sentence said twelve.
	count := regexp.MustCompile(`(?i)(\w+) agent runs`)

	for _, d := range []struct{ name, text string }{
		{"docs/results.md", resultsDoc(t)},
		{"README.md", readme(t)},
	} {
		m := count.FindStringSubmatch(flat(d.text))
		if m == nil {
			t.Errorf("%s does not say how many agent runs there have been", d.name)
			continue
		}
		got := strings.ToLower(m[1])
		if got != words[runs] && got != strconv.Itoa(runs) {
			t.Errorf("%s says %q and the table in docs/results.md adds up to %d runs",
				d.name, m[0], runs)
		}
	}
}

// Both documents say how many configurations were spoiled by the task
// rather than by the agent. The count is the number of rows in the
// table. Only these two sentences are checked, because "two of" appears
// in this prose for unrelated reasons.
func TestBothDocumentsAgreeOnHowManyConfigurations(t *testing.T) {
	_, configurations := runsAndConfigurations(t)
	sentences := []*regexp.Regexp{
		regexp.MustCompile(`(?i)two of (?:the )?(\w+) say nothing`),
		regexp.MustCompile(`(?i)two of (?:the )?(\w+) configurations were wasted`),
	}

	found := 0
	for _, d := range []struct{ name, text string }{
		{"docs/results.md", flat(resultsDoc(t))},
		{"README.md", flat(readme(t))},
	} {
		for _, re := range sentences {
			for _, m := range re.FindAllStringSubmatch(d.text, -1) {
				found++
				got := strings.ToLower(m[1])
				if got != words[configurations] && got != strconv.Itoa(configurations) {
					t.Errorf("%s says %q and the table holds %d configurations",
						d.name, m[0], configurations)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("neither document says how many configurations were spoiled, so this test checks nothing")
	}
}

// The pinned repository's size is stated in the task file and again in
// the README, and the two disagreed by eighty tests. The task file is
// the one the command reads, so the README is compared with it.
func TestTheReadmeAgreesWithTheTaskOnTheSizeOfThePinnedRepository(t *testing.T) {
	b, err := ioutil.ReadFile(filepath.Join(repoRoot(t), "tasks", "zeroturn-threshold", "task.json"))
	if err != nil {
		t.Skip("the zeroturn-threshold task is not present")
	}
	var task struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(b, &task); err != nil {
		t.Fatal(err)
	}

	size := regexp.MustCompile(`about (\d+) tests`)
	m := size.FindStringSubmatch(flat(task.Summary))
	if m == nil {
		t.Skip("the task summary does not state a test count")
	}
	r := size.FindStringSubmatch(flat(readme(t)))
	if r == nil {
		t.Fatal("the README does not say how big the pinned repository is, and the task does")
	}
	if r[1] != m[1] {
		t.Errorf("the README says about %s tests and the task says about %s", r[1], m[1])
	}
}
