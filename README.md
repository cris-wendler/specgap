# specgap

An evaluation environment for coding agents that measures the distance
between the tests an agent can see and the ones that decide whether the
work is right.

The agent is given a specification, a stub, and a visible test suite. It
never sees the hidden suite. The hidden tests introduce no new feature:
each one asks what happens when two features that were tested separately
have to hold at the same time.

```
$ specgap tasks
cache                A cache with expiry and a size limit, written from scratch.
zeroturn-threshold   Add a guard threshold to an existing Go project of about 300 tests.

$ specgap prepare zeroturn-threshold ./work
workspace ./work
task      zeroturn-threshold
removed   conformance/conformance_test.go, internal/config/keys_test.go

$ # an agent works in ./work until the visible tests pass

$ specgap grade zeroturn-threshold ./work
SPECGAP
task     zeroturn-threshold
visible  100%  313 of 313   the tests the agent could see
hidden    93%  13 of 14     the tests it could not
gap        7 points

failed on what it never saw:
  TestEveryGuardSettingHasAKey
```

That one failure is the whole point. The threshold was added to the
configuration type, given a default, and wired into the gate, and all
313 visible tests pass. It is also invisible to `policy show` and cannot
be changed with `policy set`, because a setting has to be registered in
one more place that nothing visible mentions. The feature works and no
user can reach it.

## Two tasks

**`cache`** is written from scratch in one file. It is here to make the
idea readable in half a minute, and it no longer discriminates: a
current model scores 100 on both suites, because a cache with expiry and
a size limit is in every textbook. That result is kept rather than hidden,
because it is the honest finding about small invented tasks.

**`zeroturn-threshold`** is a real repository at a real commit, about
300 tests, with its own conventions and a contributing guide. The hidden
tests are not invented for the exercise: they are tests that already
exist in that project, written after changes of this exact kind shipped
broken. No model has memorised them.

## Why the gap exists

The visible tests are the only description of the job the agent has. A
suite written feature by feature describes each feature and says nothing
about them meeting, so an implementation that satisfies it can still be
wrong in every place two features touch. The agent is not cheating. It
answered the question it was asked.

The task here is a cache with three features that are easy to write
separately and interact in four places:

| Feature | Tested alone | Meets |
| --- | --- | --- |
| store and read a value | yes | |
| entries expire after a time | yes | the size limit, recency, counting |
| a size limit evicts the least recently used | yes | expiry |

An entry that expired is gone, so it must not hold capacity a live entry
needs, must not be counted, and must not win a recency comparison. A
write that replaces a value is not a new entry, so it must not evict one.
None of that is in the visible tests, and all of it is in the hidden ones.

## Every task ships the change that solves it

A task nobody can solve measures the person who wrote it rather than the
agent. Each task carries the change that solves it, and the test suite
applies that change and requires full marks on both suites. If a hidden
test asks for something the specification never said, that is what fails
first, in this repository, rather than showing up as a bad score for an
agent that did nothing wrong.

`zeroturn-threshold` grades three ways, and the three have to differ or
the task is not measuring anything:

| The work | visible | hidden | what the hidden failures say |
| --- | --- | --- | --- |
| nothing done | 100% | 82% | the threshold does not exist |
| the type, the default and the gate | 100% | 94% | the setting is not registered, so no user can see or change it |
| the reference solution | 100% | 100% | |

The middle row is the interesting one and the reason the task exists. It
is a complete, working, reviewable change that passes every test its
author could run.

## The environment is tested against itself

A grader is worth nothing if it cannot tell a right implementation from
one that merely answers the visible tests, so the repository contains
both and checks that it can.

| Test | What it holds |
| --- | --- |
| `TestTheVisibleTestsCanBePassedWithoutBeingRight` | the implementation written feature by feature scores 100 visible and fails at least two hidden, or the environment measures nothing |
| `TestACorrectImplementationPassesBoth` | a correct implementation passes both, so no hidden test asks for something the specification does not say |
| `TestCodeThatDoesNotBuildScoresNothing` | code that does not compile scores zero rather than being left out of the count |

The second one matters as much as the first. A hidden suite that nothing
can pass measures the grader's imagination rather than the agent's work.

## What the gap number is worth

Less than the list underneath it. The two suites are different sizes, so
a percentage is dominated by how many tests each holds: one failure out
of fourteen hidden tests reads as seven points while meaning that a
feature is unreachable. Read the failures, not the score. The score is
useful for comparing runs of the same task, and for nothing else.

## Commands

| Command | What it does |
| --- | --- |
| `specgap tasks` | lists the tasks |
| `specgap prepare <task> <dir>` | writes a workspace holding everything the agent may see |
| `specgap grade <task> <dir>` | runs the visible suite, plants the hidden suite, runs it, and reports both scores and the gap |
| `specgap grade <task> <dir> --json` | the same, for a harness |

The hidden tests are copied in only to grade and removed afterwards, so a
workspace an agent can read never contains them.

## What this is for

It is a small, honest instance of the problem that appears whenever a
coding agent is trained or evaluated against a suite it can read: the
grader, not the environment, is what decides whether the measurement
means anything. Adding tasks means adding a specification, a visible
suite, and a hidden suite that asks only about the seams between the
features the visible suite already covers.

Go 1.17, standard library only.
