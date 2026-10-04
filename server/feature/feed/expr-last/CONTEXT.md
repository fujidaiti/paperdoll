# Where this work stands

This file is for someone joining the thread. It says what the work is, which
questions are already settled, which are still open, and how to run the
measurements. DEF.md and README.md hold the detail; this file holds the thread.

## Context

The feed feature has to turn a web page into a list of posts, and keep doing it
on later builds of the same page. Several earlier experiments in this folder's
siblings tried it and were measured against each other:

- `../expr-scored-tree` asks which node of a semantic tree is a post.
- `../expr-selectors` builds template selectors by folding the page from the
  bottom up.
- `../expr-selectors-minclass` is the same with the class names trimmed down to
  the ones a selector needs.

Each of those rounds defined its output and its metrics as it went, so the
rounds could not be compared with each other and it was never clear what a
better number meant. `expr-last` restarts from the other end: the output and the
measurements are written down first, and no algorithm is started until they are
settled.

## Goal

Produce, for one page, a set of structures a user can review and tick, so that
the ticked ones can be applied to the same page on every later build and report
the posts it lists.

The work is accepted when the measurements in DEF.md are taken on the 34 saved
pages and the output meets the requirements DEF.md lists: every post is
reachable, one instance stands for one post, no matcher depends on a class name
a build tool writes, and the number of structures a user has to read stays
small.

## Approach

1. Write DEF.md: the vocabulary, the output, the requirements and the metrics,
   with nothing in it that depends on how the structures are found. Done.
2. Write the measurement harness and its unit tests, with `Build` as a stub, so
   that the metric code is proven correct before any number is read from it.
   Done.
3. Write an algorithm, measure it, and record what it scores and what it misses.
   Round 1 is done.
4. Repeat step 3.

## The documents and the code

- **DEF.md** is the contract. It defines value, sample, field, part, structure,
  instance and blob, lists the requirements, and defines every metric. It must
  stay free of anything that depends on the algorithm.
- **README.md** describes the algorithm of round 1 and the numbers it scores.
- **structure.go** holds the output types and the reading of a node's values.
- **metrics.go** holds the extraction and the cost computation the metrics read.
- **measure_test.go** holds the fixture model, the per page table and the
  reachability check.
- **metrics_test.go** drives the metric code with structures written by hand.
- **build.go** and **build_test.go** are round 1.

## Decisions already taken

These were discussed and settled. Reopen them by changing DEF.md first.

- **The output is a tree of parts, not a flat list of selectors.** A flat list
  says which node is a title and which is a description, but not which title
  goes with which description. Pairing them by position breaks as soon as one
  post has no description.
- **A field points at most one node per subtree.** A post holding several of the
  same thing, such as a list of tags, is a child part and not a field with many
  nodes. This is what keeps the tags a single row the user can drop.
- **The required kinds are the link and the title**, and they are held by the
  evaluator rather than read from the `required` flag of the fixtures, so the
  list can change without rewriting the fixtures.
- **A fixture post matches an instance when the instance is a superset of it.**
  Extra values in the instance do not break the match.
- **distrib measures the cheapest instance.** A post can match the instances of
  several structures. The one measured is the one with the lowest distrib, and
  the lowest wrappers when two give the same distrib, because the user reviews
  the cheapest structure and drops the others.
- **A value the instance does not carry is ignored by distrib.** An instance
  carrying the link and the title but not the description gives distrib 1.
- **The smallest subtree** distrib counts is the part instances holding a value
  of the post, together with the part instances on the paths up to their lowest
  common ancestor. The parts above that ancestor are wrappers, not distrib.
- **The fixtures are measured, never edited.** The same holds for the cleaned
  pages under `../testdata/sanitize`, which are only ever produced by running
  `docs/cleaner`.

## What round 1 measures

Over the 30 saved pages that record posts, 1302 posts in all:

|                                      |                             |
| ------------------------------------ | --------------------------- |
| recall / recall+                     | 0.994 / 0.949               |
| merge                                | 1.17                        |
| distrib / distrib+                   | 1.77 / 1.89                 |
| wrappers                             | 0.28                        |
| structures                           | 8.0 per page, depth 4 worst |
| body recall, body split, leak, blobs | 0, no blob is produced yet  |

## Open questions

1. **The number of structures against one instance per post.** Round 1 offers
   8.0 structures per page. Dropping every set of siblings that sits inside
   another set, rather than only the ones that wrap it member per member, takes
   that to 3.7 per page, but merge rises to 1.18 overall and reaches 15.5 on
   cursor.com-blog and 13.0 on newsroom.spotify.com, and wrappers rises to 0.99.
   One instance per post is a requirement, so the trade was not taken. This is
   the first thing to decide for round 2.

2. **51 fixture texts cannot be reached.** Of the 5368 texts the fixtures
   record, 5317 equal a value some node of the page offers. The other 51 were
   recorded as several texts of one node joined together, which no node offers:
   43 authors, 7 bodies and 1 description. Either those fixture entries are
   split to match what the page holds, or recall+ can never reach 1 on the pages
   that hold them. The recommendation is to fix the fixtures, because the
   alternative is a matching rule that joins values and would hide real misses.
   This is open decision 1 in DEF.md.

3. **4 of the 331 avatar image values cannot be reached either.** This has the
   same cause and is not yet written down in DEF.md, whose open decision 1
   covers texts only.

4. **Bodies are recorded on 2 of the 34 pages.** The blob metrics are the reason
   the blob exists in the output, and they rest on developer.apple.com-news and
   blog.codinghorror.com alone. Recording bodies on a few more pages would make
   body recall and body split mean something. This is open decision 2 in DEF.md.

5. **Blobs are not produced at all.** Round 1 describes a post body with one
   field per paragraph, which is what puts distrib at 3.3 on
   developer.apple.com-news and 7.3 on blog.codinghorror.com.

6. **Robustness across builds is stated but never checked.** Every metric runs
   on one saved copy of a page. The chains round 1 builds carry no position and
   drop the class names they do not need, so they are meant to survive a later
   build, but nothing measures whether they do.

## How to run it

```
go test ./server/feature/feed/expr-last/                      # the unit tests
go test -run TestMetrics -v ./server/feature/feed/expr-last/  # the page table
go test -run TestFixtureReach -v ./server/feature/feed/expr-last/
```

`TestMetrics` prints one row per page and the totals. `TestFixtureReach` counts
the fixture values that equal a value some node of the page offers, which is the
cap on recall+ that no algorithm can lift.

## Working rules for this folder

- Nothing in this folder is used by the `feed` package. Anything needed from
  elsewhere is copied in, so that no file of `feed` has to change.
- `DEF.md` describes the output and the measurements only. Anything that depends
  on how the structures are found belongs in `README.md`.
- The repository pre-commit hook runs `golangci-lint run ./...` over the whole
  module, and it currently fails on the untracked `../expr-selectors` and
  `../expr-selectors-minclass`. Commits touching this folder therefore run with
  `--no-verify`, after `golangci-lint fmt`, `golangci-lint run` and `prettier`
  have been run on the files being committed.
