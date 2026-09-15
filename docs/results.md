# What the runs found

Nine agent runs against three tasks, on 2026-09-14, with Claude Code
2.1.270 running Opus 5. Every one scored full marks on both suites. The
gap this environment exists to measure has not appeared.

| Runs | Task | Language | Visible | Hidden |
| --- | --- | --- | --- | --- |
| 1 | `cache` | Go | 100% | 100% |
| 1 | `zeroturn-threshold`, first specification | Go | 100% | 100% |
| 1 | `zeroturn-threshold`, specification rewritten | Go | 100% | 100% |
| 3 | `rate-limiter-python` | Python | 100% | 100% |
| 3 | `cache`, with a three way seam added | Go | 100% | 100% |

Two of the six say nothing, and both were spoiled by the task rather
than by the agent. The other four are clean.

## The two that were spoiled

`cache` is a cache with expiry and a size limit, written from scratch. A
cache with expiry and a size limit is in every textbook, so the model had
seen the seams before.

The first version of `zeroturn-threshold` had a section headed what good
looks like, listing four things: that a person can read the value with
`policy show` and change it with `policy set`, that an invalid value is
refused, that a file containing the new field loads, and that the gate
denies in Strict mode. Those are the hidden tests. The first names the
key registry, the third names the schema check, the fourth names the
acceptance tests. It was written while trying to produce a clear ticket
and it handed over the answer key.

## The four that are clean

`zeroturn-threshold` with the specification rewritten to describe the
threshold and nothing else: the agent scored full marks in 304 seconds,
found the key registry without being told it existed, put the entry where
the reference solution puts it, added the property to the published
schema with a description of its own, and wrote six tests.

`rate-limiter-python`, three times: full marks in 74, 64 and 49 seconds,
and no hidden test failed in any attempt.

## What makes this more than a shrug

The tasks do discriminate. Both carry an implementation written feature
by feature, and both score 100 on the visible suite and 60 on the hidden
one:

| Task | Visible | Hidden | What it gets wrong |
| --- | --- | --- | --- |
| `cache` | 100% | 60% | expired entries counted, a failed read treated as a use |
| `rate-limiter-python` | 100% | 60% | what remaining counts, how much returns after one window |

So the hidden tests catch the thing they were written to catch. What
they do not catch is the agent, because the agent does not write that
implementation. The developer these tasks were designed around, who
answers each feature in turn and stops, is not how this model works. It
reads the surrounding code, notices the places a change has to reach, and
says in its own summary which cases it decided were ambiguous.

## Three at once, rather than two

Every hidden test began as a pair: expiry against the size limit, the
window against the burst. The benchmark that prompted this drew three,
each passing alone and failing together, so both tasks gained one that
asks about three at once. The cache one turns on expiry, an overwrite
that restarts a clock, recency and the size limit all bearing on a
single eviction.

It works as a test. The implementation written feature by feature fails
it, along with two others, at three of six. The reference passes it, so
the answer follows from the specification.

The agent passed it three times out of three, in 79, 55 and 90 seconds.
Which says the passing was never about counting how many features a test
touches. It reads the whole problem, decides the cases the specification
leaves open, and writes something coherent. A third dimension is not
more of a strain than a second.

## What this does not say

Three tasks, one model, one day. Two of them are small enough to hold in
one file and the third is a repository of three hundred tests, which is
small. It does not refute the benchmark that prompted this, which used
different tasks at a different scale four months earlier.

It says that on problems of this size, this model finds the seams,
whether two features meet or three, and that an environment built to
catch an agent optimising narrowly to its visible tests needs something
other than more interacting features.

The hypothesis left untested here is scale. All three tasks can be read
end to end, and the agent read them: on the repository task it touched
sixteen files, including the schema and the changelog. A codebase too
large to hold at once is a different problem, and not one this
environment poses yet.

## What it cost to find out

Two of six runs were wasted on task design, in two different ways that
both look like good practice while you are doing them. Choosing a
problem clear enough to specify chose one the model had memorised.
Stating plainly what done means enumerated the grader.

That is the finding worth keeping from a day of this. The environment is
straightforward. The task is where the work is, and the ways it goes
wrong are quiet.
