# expr-last

A first algorithm for the output DEF.md defines. It is an experiment, and
nothing in this directory is used by the feed package.

- DEF.md states the goals, the input, the output and the metrics. It says
  nothing about how the structures are found.
- build.go is the algorithm.
- structure.go and metrics.go hold the output types, the extraction and the cost
  computation the metrics read.
- `go test -run TestMetrics -v ./server/feature/feed/expr-last/` prints the
  table below.
- `tree/` holds the structures of each page, one file per page, written by
  `go test -run TestWriteTrees ./server/feature/feed/expr-last/`. A file shows
  the path each matcher reads, how many subtrees it reached and samples of the
  values it read, so that the output can be read rather than only counted.

## How it works

A post list is a set of sibling elements that repeat. The algorithm looks for
those sets, keeps the ones that stand for posts, and turns each of them into one
structure.

1. **Collect the sets.** For every element of the page, its element children are
   grouped by tag name. A group of two or more is a candidate set when at least
   two of its members look like one post, meaning the member holds an `href`, a
   text of at least 15 characters written in the page, and more than one place
   carrying a value. The last of the three is what tells a post card apart from
   a tag chip, which is a single link carrying a single name.

2. **Drop the sets that merge posts.** A set is dropped when one of its members
   holds a longer set that fills it. Longer, because the repeated blocks inside
   a single card, such as its tags, belong to one card while the list of cards
   runs over the page. Filling it, because a card is mostly made of the post,
   while the tags of a card take a small corner of it.

3. **Drop the sets that wrap another set.** A set that sits inside another one,
   one member per member, describes the same posts with fewer values, so the
   outer one is kept.

4. **Read the path to the members.** The path is a chain of tags and class
   names, read as direct child steps from the page, with no position in it, so
   that a sibling appearing or disappearing does not break it. Every class name
   the chain does not need is then dropped, and the ones that look generated,
   such as `style-1w7apwp`, are tried first. Two sets that the same chain
   reaches, such as the cards of two sections, give one structure.

5. **Describe what the chain found.** The subtrees are walked level by level, so
   that the nodes of every subtree sitting at the same path are seen together. A
   path holding one node per subtree becomes a field. A path holding several
   nodes inside one subtree becomes a child part, because a field matcher points
   at most one node per subtree. The children of one tag name are told apart by
   the class names they carry, and only a class name that appears under more
   than one subtree is used, so that a name written per card is ignored.

## What it scores

```
page                                        posts  recall recall+  merge distrib distrib+ wrappers struct depth
anthropic.com-news                             13   0.923   0.923   1.00    1.00     1.00     0.00      5     3
aws.amazon.com-jp-blogs-news                   10   1.000   1.000   1.00    3.00     3.00     2.00      4     3
bbc.com                                       103   1.000   0.961   2.13    1.11     1.19     0.31     13     4
blog.acolyer.org                               10   1.000   1.000   1.00    1.00     1.00     0.00     11     3
blog.codinghorror.com                           3   1.000   0.667   1.00    7.33     7.00     2.67      2     4
blog.google                                     8   0.625   0.625   1.67    1.00     1.00     0.00      3     4
claude.com-blog                                24   0.958   0.958   1.11    1.00     1.57     0.00     18     4
cursor.com-blog                                28   1.000   0.857   2.00    1.04     1.00     0.00      9     4
daily.bandcamp.com-album-of-the-day            30   1.000   1.000   1.00    1.00     1.00     0.00      3     3
daily.bandcamp.com-features                    30   1.000   1.000   1.00    1.00     1.00     0.00      3     3
deepmind.google-blog                           25   0.960   0.960   1.00    1.00     1.00     0.00      9     4
deepmind.google-research-publications          30   1.000   1.000   1.00    1.00     1.00     0.00      8     4
devblogs.microsoft.com                         39   1.000   0.923   1.02    1.23     1.22     1.00      5     3
developer.apple.com-news                      110   1.000   0.982   1.04    3.32     3.33     1.48     18     4
developers.openai.com-blog                     29   1.000   1.000   1.00    1.00     1.00     0.00      4     2
diggersfactory.com-vinyl-shop-new-ins           9   1.000   1.000   1.00    3.00     3.00     0.00      2     4
edition.cnn.com-us                             42   1.000   1.000   1.00    1.26     2.14     0.17     15     4
engineering.atspotify.com                      13   0.923   0.923   1.00    1.00     1.75     0.00      2     2
engineering.fb.com                             12   1.000   1.000   1.76    1.83     1.83     1.75      8     4
flutter.dev-blog                              284   1.000   1.000   1.00    2.99     2.99     0.00      3     4
github.blog-ai-and-ml                          19   1.000   1.000   1.00    1.89     1.89     0.32      8     4
github.blog                                    25   1.000   0.960   1.67    1.72     2.38     0.12     15     4
go.dev-blog                                    10   1.000   0.000   1.00    1.00     0.00     0.00      4     4
newsroom.spotify.com                           21   1.000   1.000   1.55    1.24     1.29     1.14     13     4
oreilly.com-radar                              19   1.000   0.947   1.00    1.32     1.33     0.00      7     4
paulgraham.com-articles                       235   1.000   1.000   1.00    1.00     1.00     0.00      2     2
qiita.com                                      30   1.000   0.000   1.00    2.00     0.00     1.00      6     4
technologyreview.com                           71   0.986   0.958   1.49    1.06     1.82     0.14     25     4
ycombinator.com-blog-tag-essay                 10   1.000   1.000   1.00    1.00     1.30     0.00      8     3
ycombinator.com-blog                           10   1.000   1.000   1.58    1.10     1.10     0.00      7     4

30 pages, 1302 posts
recall 0.994, recall+ 0.949
merge 1.17, distrib 1.77, distrib+ 1.89, wrappers 0.28
structures 8.0 per page, depth 4 worst
body recall 0.000, body split 0.00, leak 0 (0.000), blobs 0
--- PASS: TestMetrics (0.39s)
PASS
```

## What it does not do yet

- **No blobs.** Nothing in the output is a blob, so body recall, body split and
  leak are all zero. The two pages that record bodies, developer.apple.com-news
  and blog.codinghorror.com, are described with one field per paragraph instead,
  which is what puts distrib at 3.3 and 7.3 there.

- **It covers the post lists only.** DEF.md asks the fields to cover every value
  of the page. The structures cover the sets of siblings that look like post
  lists, and a value written outside one of them is reached by no field.

- **Eight structures per page is still a wall.** It is the metric DEF.md calls
  the one that matters most. Dropping every set that sits inside another set,
  not only the ones wrapping it member per member, takes it to 3.7 per page, but
  the review cost moves to the other metrics: merge rises to 1.18 with single
  pages reaching 15.5, and wrappers rises to 0.99. One instance per post is a
  requirement, so that trade was not taken.

- **blog.google reaches 0.625.** It is the lowest recall of the 30 pages. The
  page writes its cards as custom elements, and five of its eight posts sit in
  sets of siblings the first step does not collect.

- **Robustness is not measured.** Every number above is taken on one saved copy
  of a page, so the chains are built to survive a later build but are never
  tried on one.
