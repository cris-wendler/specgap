# Task: add a guard threshold

This repository is ZeroTurn, a command line tool that watches a coding
session and decides whether to allow, ask about, or deny a new subagent.
Its guard compares measurements from the session against thresholds in
`.zeroturn.json`.

Add one more threshold.

## What to add

`guard.limits.fiveHourCritical`, a percentage, default `95`.

When five hour usage is at or above it, the condition is **critical**.
The existing `guard.limits.fiveHourWarn` stays as it is and continues to
produce the confirm level.

Critical already means something in this project: in Strict mode a
critical condition denies a new subagent, where a confirm condition only
asks. Adding this threshold therefore gives a way to deny on usage,
which today is only possible on context.

## What good looks like

- A person can read the value with `zeroturn policy show` and change it
  with `zeroturn policy set guard.limits.fiveHourCritical 90`.
- An invalid value is refused the same way other thresholds refuse one.
- A configuration file containing the new field loads, and one written
  by `zeroturn init` contains it.
- The gate denies in Strict mode when the threshold is crossed.

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
