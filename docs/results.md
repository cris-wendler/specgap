# What the runs found

Three agent runs, on 2026-09-14, against the two Go tasks that existed then, with Claude Code
2.1.270 running Opus 5. Every one scored full marks on both suites. The
gap this environment exists to measure did not appear.

| Run | Task | Visible | Hidden | Why it says little |
| --- | --- | --- | --- | --- |
| 1 | `cache` | 100% | 100% | a cache with expiry and a size limit is in every textbook |
| 2 | `zeroturn-threshold` | 100% | 100% | the specification listed the hidden tests |
| 3 | `zeroturn-threshold`, specification rewritten | 100% | 100% | nothing wrong with it |

Only the third is worth anything, and it is a clean negative result.

## The two runs that were spoiled, and how

The first task was written from scratch: a cache with three features that
interact in four places. An implementation written feature by feature
scores 60 on the hidden suite, which is how the task was validated. A
current model scored 100. The problem is a common exercise and the model
had seen the seams before.

The second task was a real repository, which fixes that. It was spoiled a
different way. The specification had a section headed what good looks
like, with four bullets:

- a person can read the value with `policy show` and change it with
  `policy set`
- an invalid value is refused the way other thresholds refuse one
- a configuration file containing the new field loads
- the gate denies in Strict mode when the threshold is crossed

Those are the hidden tests. The first names the key registry, the third
names the schema conformance check, the fourth names the acceptance
tests. Written while trying to produce a clear ticket, and it handed over
the answer key.

## The third run

The specification was rewritten to say what the threshold is and what
crossing it means, and nothing about where settings are registered, which
files mention them, or what finished looks like.

The agent scored 100 on both suites in 304 seconds. It found the key
registry without being told it existed, placed the entry where the
reference solution places it, and added the property to the published
JSON schema with a description of its own. It also added six tests.

## What this says, and what it does not

It says that on a small, well structured repository with its conventions
visible, this model finds the parallel places that a change has to touch.
The premise this environment was built on, that an agent works to the
tests it can see and misses the seams, did not hold here.

It does not refute the benchmark that prompted it. Different tasks, a
different scale of repository, a different model, and four months later.
A result on one task with one model says something narrow.

What it does say to anyone building these environments is that the
environment is the easy part. Two runs of three were spoiled by the task
rather than by the agent, in two different ways that both look like good
practice while writing them: choose a problem clear enough to specify,
and state plainly what done means. The first makes the problem one the
model has seen. The second tells it what will be checked.

A specification has to be clear enough that grading it is fair, and quiet
enough that it does not enumerate the grader. That line is the work.
