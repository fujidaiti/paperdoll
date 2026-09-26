# expr-scored-tree

An experiment that asks which **node** of the semantic tree is a post, instead
of which **group** of siblings is a list of posts. Nothing in this directory is
used by the feed package. The trees are read from the JSON the feed package
already writes under `testdata/semantic/`.

## Why

The selection in `semantic_groups.go` scores a group of siblings and offers the
groups whose score reaches a share of the best score of the page. The score of a
group is not comparable between pages, so the threshold has to be relative, and
a group that is partly right cannot be corrected.

Labelling nodes separates the two questions. Deciding whether one node is a post
is the hard part. Once the labels are right, grouping is mostly reading which
labelled nodes are siblings, because a post never contains another post.

## What the three functions do

`FoldCards(root)` folds every card into a single node before anything is scored.
`BuildSemanticTree` keeps a subtree split as soon as it carries two distinct
links, so a card that also links to its category, to its author or to a "read
more" page stays a subtree. The card is recognised from what its siblings agree
on: most children of one node carry a link of the same URL template, so a child
that carries exactly one link of that template stands for one post, and every
other link it carries is taken out. A link that carries a text of 40 characters
or more is kept, because such a link is a second post rather than a category.

`ScoreNodes(root, threshold)` writes `Score` and `Post` into every node, in
place. The score is half what the node holds, which is a long text, a date, an
image and a heading, and half how well the node agrees with its siblings, which
is the URL shape they share, the look they share and whether the node carries
exactly one link of that shape. The second half is what makes the number
comparable between pages.

`Groups(root)` reads the rows of the selection screen out of a scored tree. The
marked nodes under one parent are one list, and the lists of different parents
are merged when their links share a URL shape. The merge is what a page that
writes each section as a separate container needs: `developer.apple.com` holds
its posts under 5 parents and `bbc.com` under 24.

## Results on the 30 saved pages

Node labelling, at the threshold 0.68:

| measure   | value |
| --------- | ----- |
| precision | 0.880 |
| recall    | 0.906 |

The threshold is an absolute number, not a share of the best score of the page.
`TestThresholdSweep` prints the separation from 0.40 to 0.90; 0.68 is where the
f1 score peaks, at 0.893.

The selection itself is measured against the fixtures by five numbers, all
produced by `TestMetrics`:

| measure          | value | what it says                                                          |
| ---------------- | ----- | --------------------------------------------------------------------- |
| found(any)       | 0.989 | share of posts a ticked group reaches, by a link anywhere in a member |
| found(primary)   | 0.686 | the same, but the member's own link has to be the post's link         |
| recall(required) | 0.976 | share of posts one member holds every required value of               |
| recall(all)      | 0.927 | the same, for every recorded value, required or not                   |
| extra per member | 2.83  | values a member holds that the fixture records nowhere for its posts  |

The fixture of a page records every value the page writes beside a post, each
with the kind of value it is and whether the post needs it, so a value the
selection offers is either one of them or noise. A value matches when the two
normalized forms are equal, and the values of a member are counted once each, so
a card that repeats an avatar offers one value rather than six.

The two `found` numbers say different things. found(any) asks whether the post
was reached at all; found(primary) asks whether the member that reached it is a
card, because a member whose own link is the post's link stands for that one
post while a member that merely contains the link is a part of the page. The gap
between 0.989 and 0.686 is the cards `FoldCards` still fails to fold.

recall(all) is reported rather than tuned against, because it depends on how
thoroughly the optional values of a page were recorded, and 30 pages recorded
separately differ in that.

`extra per member` never reaches zero. An image that is neither a thumbnail nor
an author photo, and a link other than the post's own, have no place in the
schema, so a card that carries a badge or a category link is charged for it.

Three pages carry most of the noise: `technologyreview.com` at 121 values per
member, `developer.apple.com-news` at 82 and `edition.cnn.com-us` at 28. On each
of them a ticked group holds a member that covers a whole section rather than
one card. `go.dev-blog` is the one page whose recall(required) is 0.000 against
a found of 1.000: every description is moved into the list container by
`mergeLinkless`, so no member can hold one. That is a loss in the tree rather
than in the selection.

## What folding the cards changed

Before `FoldCards`, a member of a group was a subtree: over the groups the user
ticks, 1406 members carried 950 levels of depth between them. Folding leaves
1326 members with 184 levels.

That change was measured with an earlier set of metrics, against the earlier
fixtures, so the numbers it moved cannot be compared with the five above. What
still holds is the reason folding helps: a card that also links to its category
or its author was never folded, so the member had depth the user had to read
through and offered those extra links and their texts as values of the post.
found(primary) is now the number that says how often a member is a card, and at
0.686 it shows the same problem is not solved.

Two other ideas were measured under the earlier metrics and are not used.
Keeping only the innermost marked node of a chain made the members cheaper to
read but lost the date and the image, because the innermost node holds only the
title. Dropping a marked node that holds several posts, so that its own cards
replace it, made the offered values worse, because such a node holds mostly
attributes while the fragments left behind hold mostly noise.

One idea is left open. Much of what a member offers is the words a list repeats
on every card: "Read more", the separator between the date and the author, and
on `paulgraham.com` a 1x1 spacer image. Removing a value that at least half of
the members of a group carry word for word lowered the noise and cost a little
recall when a few posts shared a value the filter took away. It needs measuring
again against `extra per member` before it can be judged, and whether the trade
is worth making is a decision about the screen rather than about the tree.

## Running

```sh
go test ./server/feature/feed/expr-scored-tree/ -v
```

| test                   | what it does                                              |
| ---------------------- | --------------------------------------------------------- |
| `TestNodeScore`        | precision and recall of the labels, per page and in total |
| `TestThresholdSweep`   | the separation at every threshold                         |
| `TestGroups`           | rows, ticks and URL recall of the grouping                |
| `TestMetrics`          | the five measures of the selection, per page and in total |
| `TestWriteScoredTrees` | writes the scored trees into `testdata/scored/`           |

The scored trees are the trees of `testdata/semantic/` with `score` and `post`
on every node, so a change to the score can be read in a diff.
