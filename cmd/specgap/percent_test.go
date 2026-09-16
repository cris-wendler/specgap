package main

import "testing"

// A report that says 100% when one test in four hundred failed is the
// overstatement this environment exists to catch, and it was in the
// environment. The first run of the repository task reported
// "visible 100% 402 of 403".
func TestAnImperfectScoreIsNeverShownAsPerfect(t *testing.T) {
	if got := showPercent(percent(402, 403)); got == "100" {
		t.Errorf("402 of 403 renders as %q, which reads as everything passed", got)
	}
	if got := showPercent(percent(402, 403)); got != "<100" {
		t.Errorf("402 of 403 renders as %q, want <100", got)
	}
	// And a run where nothing passed must not read as nothing attempted.
	if got := showPercent(percent(1, 1000)); got == "  0" {
		t.Errorf("1 of 1000 renders as %q, which reads as nothing passed", got)
	}
}

// The whole numbers keep the column they have always had, because the
// report is read as a table.
func TestAWholeScoreKeepsItsColumn(t *testing.T) {
	for _, c := range []struct {
		part, whole int
		want        string
	}{
		{8, 8, "100"},
		{0, 8, "  0"},
		{4, 8, " 50"},
		{2, 8, " 25"},
	} {
		if got := showPercent(percent(c.part, c.whole)); got != c.want {
			t.Errorf("%d of %d renders as %q, want %q", c.part, c.whole, got, c.want)
		}
	}
}
