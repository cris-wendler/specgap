# specgap

Give a coding agent a task. Hide some of the tests. See what it got wrong
in the part it could not see.

## Try it

```sh
go build -o specgap ./cmd/specgap
./specgap tasks
./specgap prepare cache ./work
```

`./work` now holds a spec, a stub, and eight tests. Point an agent at it.
When the agent is done:

```sh
./specgap grade cache ./work
```

```
SPECGAP
task     cache
visible  100%  8 of 8   the tests the agent could see
hidden    60%  3 of 5   the tests it could not
gap       40 points

failed on what it never saw:
  TestLenDoesNotCountExpiredEntries
  TestReadingAnExpiredEntryIsNotAUse
```

That output comes from `testdata/naive.go.txt`, an implementation written
feature by feature. A current model does better on this task, which is
covered below.

Requirements: Go 1.17 or newer. No dependencies.

## The idea

Test suites usually test one feature at a time. An agent works until
those tests pass, so it can finish with every test green and still be
wrong wherever two features meet, because nothing asked.

The hidden tests add no features. They only ask about the seams.

Take the `cache` task. Three features, each easy alone:

- store and read a value
- entries expire after a time
- a size limit evicts the least recently used

Now the seams. An expired entry is gone, so it must not hold a slot a
live entry needs, must not be counted by `Len`, and must not win a
recency comparison. Overwriting a key adds nothing, so it must not evict
anything. None of that is in the visible tests. All of it is hidden.

## The tasks

### `cache`

Written from scratch, one file, about a minute to read.

**It no longer catches anything.** A current model scores 100 on both
suites, because a cache with expiry and a size limit is in every
textbook. That result is kept here rather than quietly dropped, because
it is the honest finding about small invented tasks.

### `zeroturn-threshold`

A real repository at a fixed commit. About 300 tests, its own
conventions, its own contributing guide. The job is to add one
configuration setting.

Two of the repository's test files are removed from the agent's copy and
restored to grade. They were not written for this exercise. They exist in
that project because changes of exactly this kind shipped broken.

Three levels of work, all passing every visible test:

| The work | hidden | what the failures say |
| --- | --- | --- |
| nothing done | 82% | the setting does not exist |
| the type, the default, the gate | **94%** | nobody can see or change the setting |
| the reference solution | 100% | |

The middle row is why the task exists. That is a complete, working,
reviewable change that passes all 313 tests its author could run, and the
feature is unreachable from the command line.

This task needs the other repository checked out beside this one, or
`SPECGAP_REPO` pointing at it. Without it the task says so and its tests
skip.

## Commands

| Command | What it does |
| --- | --- |
| `specgap tasks` | list the tasks |
| `specgap prepare <task> <dir>` | write a workspace holding only what the agent may see |
| `specgap grade <task> <dir>` | run both suites, report both scores |
| `specgap grade <task> <dir> --json` | the same, for a script |

The hidden tests are planted only to grade, then removed. A workspace an
agent can read never contains them.

`SPECGAP_TASKS` sets where the tasks live, for running the command from
another directory.

## Reading the result

**Read the failure list, not the score.** The two suites are different
sizes, so the percentage is dominated by how many tests each holds. One
failure out of fourteen shows as 7 points and means a feature is
unreachable.

The score is useful for comparing runs of the same task. Nothing else.

## Two things the environment checks about itself

A grader that cannot tell good work from work that merely passes is worse
than no grader. So:

**Every task ships the change that solves it.** The suite applies it and
requires full marks. If a hidden test asks for something the spec never
said, that fails here, not as a bad score for an agent that did nothing
wrong.

**Doing nothing has to score badly.** An untouched workspace is graded in
the suite, and must fail.

There is also a check that code which does not compile scores zero rather
than being skipped, and one that the feature-by-feature cache
implementation scores 100 visible and fails at least two hidden.

## Adding a task

A task is a directory under `tasks/` with a `task.json`, a `SPEC.md`, and
the tests. Hidden tests come from the repository the task is cut from, or
ship with the task. See `tasks/cache/task.json` for the short form and
`tasks/zeroturn-threshold/task.json` for one cut from a repository.

Write the hidden suite to ask only about seams between things the visible
suite already covers. A hidden test that introduces a requirement the
spec never stated is a broken task, and the reference solution is what
catches that.
