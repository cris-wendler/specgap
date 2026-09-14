# specgap

An evaluation environment for coding agents that measures the distance
between the tests an agent can see and the ones that decide whether the
work is right.

The agent is given a specification, a stub, and a visible test suite. It
never sees the hidden suite. The hidden tests introduce no new feature:
each one asks what happens when two features that were tested separately
have to hold at the same time.

```
$ specgap prepare ./work
workspace ./work
the agent gets SPEC.md, cache.go and cache_test.go, and nothing else

$ # an agent works in ./work until the visible tests pass

$ specgap grade ./work
SPECGAP
visible  100%  8 of 8   the tests the agent could see
hidden    60%  3 of 5   the tests it could not
gap       40 points

failed on what it never saw:
  TestLenDoesNotCountExpiredEntries
  TestReadingAnExpiredEntryIsNotAUse
```

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

## Commands

| Command | What it does |
| --- | --- |
| `specgap prepare <dir>` | writes the specification, the stub, the visible tests and a module file |
| `specgap grade <dir>` | runs the visible suite, plants the hidden suite, runs it, and reports both scores and the gap |
| `specgap grade <dir> --json` | the same, for a harness |

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
