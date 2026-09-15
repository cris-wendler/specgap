# Task: add a guard threshold

This repository is ZeroTurn, a command line tool that watches a coding
session and decides whether to allow, ask about, or deny a new subagent.
Its guard compares measurements from the session against thresholds in
`.zeroturn.json`.

Today the five hour usage window has one threshold, `fiveHourWarn`, and
crossing it produces the confirm level, which asks. There is no way to
deny on usage: only context can do that.

Add a second one, `fiveHourCritical`, a percentage defaulting to 95, so
that crossing it produces the critical level.

## Rules

- Go 1.17, standard library only. The project has no dependencies and
  must keep none.
- Follow the conventions already in the repository. Read how the
  existing thresholds are done before writing anything.
- `CONTRIBUTING.md` describes the writing rules for comments and output.
  `scripts/lint-copy.sh` enforces some of them.
- Do not change the behaviour of any existing threshold.

## How you are judged

`go test ./...` must pass. Some of the repository's tests have been
removed from your copy and will be restored to grade the work.

A worked solution touches eight files. Two of them are the ones this
description points at. The rest enforce things this repository checks
about itself, and none of them is mentioned in the file you would edit
first.
