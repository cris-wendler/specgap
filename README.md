# specgap

[![ci](https://github.com/cris-wendler/specgap/actions/workflows/ci.yml/badge.svg)](https://github.com/cris-wendler/specgap/actions/workflows/ci.yml)
[![Go 1.17+](https://img.shields.io/badge/go-1.17%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![No dependencies](https://img.shields.io/badge/dependencies-none-success)](go.mod)
[![License MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

Give a coding agent a task, hide the tests that decide the grade, and
measure the distance between what it passed and what it could not see.

Three tasks, in Go and Python. Nine agent runs so far, and every one
scored full marks on both suites. The tasks do catch an implementation
written feature by feature, at 100 visible and 60 hidden. They do not
catch the agent, because the agent does not write that implementation.

Two of the six runs say nothing, because of mistakes in the task rather
than anything the agent did. [docs/results.md](docs/results.md) has what
happened and what it suggests about writing these.

![The agent is given a specification, a stub and eight visible tests, and works until those pass. Five hidden tests it never saw then grade it. They add no features; each asks what happens where two features meet, such as an expired entry still holding a slot, or a failed read counting as a use. The visible score is 100 percent, the hidden score 60, a gap of 40 points.](docs/img/how-it-works.svg)

## Try it

    go install github.com/cris-wendler/specgap/cmd/specgap@latest
    specgap run cache --agent "claude -p 'Read SPEC.md and make the tests pass'"

The tasks travel inside the executable, so that works from anywhere. A
checkout takes priority over them, so editing a task shows up without a
rebuild.

`run` prepares a workspace, starts the agent in it, grades what it left
behind, and throws the workspace away:

    SPECGAP
    task     cache
    visible  100%  8 of 8   the tests the agent could see
    hidden    60%  3 of 5   the tests it could not
    gap       40 points  in 0s

    failed on what it never saw:
      TestLenDoesNotCountExpiredEntries
      TestReadingAnExpiredEntryIsNotAUse

That output comes from `testdata/naive.go.txt`, an implementation written
feature by feature. A current model does better on this task, which is
covered below.

To drive the steps yourself instead:

    specgap prepare cache ./work    # the agent works in ./work
    specgap grade cache ./work

Go 1.17 or newer, no dependencies. A Python task also needs `pytest`.
Release archives for macOS, Linux and Windows are built by
`scripts/build-release.sh` and attached to each release with their
checksums.

## The idea

Test suites usually test one feature at a time, and an agent works until
they pass. So it can finish with everything green and still be wrong
wherever two features meet, because nothing asked.

The hidden tests add no features. They only ask about the seams. In the
cache task an entry that expired is gone, so it must not hold a slot a
live entry needs, must not be counted, and must not win a recency
comparison. Overwriting a key adds nothing, so it must not evict
anything. None of that is in the visible tests.

Read the failure list rather than the score. The two suites are different
sizes, so the percentage is dominated by how many tests each holds: one
failure out of fourteen shows as seven points and can mean a feature is
unreachable. The number is worth something for comparing runs of the same
task and nothing else.

## The tasks

`cache` is written from scratch in one file and takes a minute to read.
It no longer catches anything: a current model scores 100 on both suites,
because a cache with expiry and a size limit is in every textbook. The
result is here rather than quietly dropped, since it is the honest
finding about small invented tasks.

`rate-limiter-python` is a per key limiter with a window and a burst
allowance, written from scratch in Python and graded through pytest.
Eight visible tests, five hidden. An implementation that resets a fixed
window and refills the whole burst with it answers all eight and gets two
seams wrong: what `remaining` should count, and how much comes back when
one window passes.

It is here because agent evaluation mostly happens in Python, and a
harness that could only pose Go problems would say more about the tool
than about agents. A task names the runner it needs, so another language
means another runner rather than rewriting anything.

`zeroturn-threshold` is a real repository at a fixed commit, about 300
tests, with its own conventions and contributing guide. The job is to add
one configuration setting. Two of the repository's test files are removed
from the agent's copy and restored to grade, and they were not written
for this exercise: they exist in that project because changes of exactly
this kind shipped broken.

Three levels of work, written by hand to check that the task
discriminates. All pass every visible test:

| The work | hidden | what the failures say |
| --- | --- | --- |
| nothing done | 82% | the setting does not exist |
| the type, the default, the gate | 94% | nobody can see or change the setting |
| the reference solution | 100% | |

The middle row is why the task exists. It is a complete, working,
reviewable change that passes all 313 tests its author could run, and the
feature is unreachable from the command line.

A real agent scored 100 on both suites. It found the registry without
being told it existed. [docs/results.md](docs/results.md) has the three
runs so far, two of which were spoiled by the task rather than the
agent, and what that suggests about writing them.

That task needs the other repository beside this one:

```sh
git clone https://github.com/cris-wendler/zeroturn
git clone https://github.com/cris-wendler/specgap
cd specgap
```

`SPECGAP_REPO` points at it if you keep them elsewhere. Without it the
task says so and its tests skip.

## Commands

| | |
| --- | --- |
| `specgap tasks` | list the tasks |
| `specgap run <task> --agent <command>` | prepare, run the agent, grade, and report |
| `specgap prepare <task> <dir>` | write a workspace holding only what the agent may see |
| `specgap grade <task> <dir>` | run both suites and report both scores, `--json` for a script |

`run` takes `--runs <n>` for repeated attempts, each in a workspace
nothing has seen, and reports the lowest, median and highest hidden score
with how often each hidden test failed. One attempt is an anecdote: a
task an agent fails once in five is a different finding from one it fails
every time. `--results <file>` appends each attempt as a line of JSON so
scores accumulate across models and dates, and `--keep <dir>` leaves the
workspaces behind to look at.

Hidden tests are planted to grade and removed afterwards, so a workspace
an agent can read never contains them. `SPECGAP_TASKS` sets where the
tasks live if you run the command from somewhere else.

## Adding a task

A task is a directory under `tasks/` holding a `task.json`, a `SPEC.md`
and the tests. Set `runner` to `pytest` for a Python task; leaving it out
means Go. `tasks/cache` is the short form and
`tasks/zeroturn-threshold` is one cut from a repository.

Set `runner` to `pytest` for a Python task; leaving it out means Go.

Write the hidden suite to ask only about seams between things the visible
suite already covers. A hidden test that introduces a requirement the
specification never stated is a broken task, which is why every task also
ships the change that solves it: the suite applies that change and
requires full marks, so a task nobody can solve fails here rather than
showing up as a bad score for an agent that did nothing wrong. An
untouched workspace is graded too, and has to fail.
