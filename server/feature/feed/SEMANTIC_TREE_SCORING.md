# Reading posts out of the semantic tree

## Context

The semantic tree in `semantic_tree.go` folds a page into nodes that each hold
at most one link, plus the texts and images around it. `SEMANTIC_TREE.md`
describes that step and ends with an open question: which node is a post list.

This document answers that question with a measured rule. It has two parts,
because the selection screen asks two things at once:

1. which group of the tree is a list of posts, a group being a node with two or
   more children;
2. which link of a group member is the post, since a member usually holds more
   than one link.

Everything below was measured over the rendered trees in `testdata/semantic/`
against the hand written fixtures in `testdata/`. The rule was written and
measured in Python first, and the notebook shows every step with its charts.

- The tree:
  file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/SEMANTIC_TREE.md
- The code:
  file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/semantic_tree.go
- The measurements:
  file:///Users/fujidaiti/Dev/paperdoll/expr/group_scoring.ipynb
- The Python version of the rule:
  file:///Users/fujidaiti/Dev/paperdoll/expr/tree.py

## Goal

A rule that returns the groups of a page that are lists of posts, together with
the post link of every member, so that the tree can feed the selection screen
and the acceptance table in `metrics_test.go` can be run against it.

The rule has to work on every saved page, not on the average of them, because
one page returning nothing is a feed that never appears for the user.

## Approach

Score a group of siblings from several values at once rather than filtering on
any single one, and compare that score with the best group of the same page
instead of with a fixed number. Choose the link of a member in the context of
its group, because what separates the post link from the author link is how the
other members of the list are written, not the link itself.

## What the corpus holds

21 of the saved pages have a fixture with posts. They hold 2530 nodes carrying a
link, 2049 distinct links of which 1070 are posts, and 720 candidate groups.

A rule that accepts every link is right 52% of the time, and a rule that accepts
every group is right for 90 of the 720. Those are the two numbers to beat.

The links that are not posts are of four kinds: navigation such as "Home" and
"Quickstart"; tag and category links; author links; and links written inside the
body text of a post.

## Part 1: which group is a list of posts

### What is possible at best

Taking the fixtures in hand and picking the group that covers the most posts,
then the next one, until the page is covered, every post of every page is
reachable. The folding loses nothing, so the whole problem is selection.

### Three problems, and what each needs

**A card holds more links than the post.** A `qiita.com` card holds the post
link twice, the author page, an event banner and five tag links. Three tests
tell the extra links apart, and none of them reads the text of the link:

- a link whose path another link of the same card extends is the page _about_
  the post, such as the author page or the category;
- a URL shape that occurs more than once inside one card, such as five links to
  `/tags/<name>`, cannot be the single post of that card;
- a link that several members of the same group carry is a banner or a menu
  repeated on every card.

Over the target group of every page this brings the average number of links per
member from 1.73 to 1.26, and 88% of members are left with exactly one
candidate.

**A junk group looks exactly like a list of posts.** A list of tags inside a
card has one link per member, one URL shape and one signature, the same as a
list of posts. Only what the members hold separates them: the text of a tag is a
few characters and a tag carries no date and no image. This is why the rule is a
score rather than a test.

**The scores of two pages are not comparable.** The correct group scores between
0.57 and 0.70 depending on the page, while junk groups on other pages score
higher than that. A fixed threshold therefore cannot work, and the threshold is
a share of the best score of the same page.

### The score

```
score = 0.5 * content + 0.2 * shape + 0.2 * one_link_rate + 0.1 * size
```

- **content** is what the members hold: the median longest text, capped at 40
  characters, plus the share of members carrying a date, an image or a heading.
- **shape** is how much the members look alike: the share agreeing on the URL
  template, and the share agreeing on the signature.
- **one_link_rate** is the share of members holding exactly one candidate link
  of the shape the group agreed on.
- **size** grows with the number of members and is capped at eight.

A group is offered when its score reaches `alpha` times the best score of the
page. `alpha = 0.8` is the setting measured below.

### Two groups, one inside the other

A featured card at the top of a list is a group of its own, and the list sits
inside it, so two groups that are both offered often overlap. Four ways of
resolving that, at `alpha = 0.8`:

| what is done with the pair                        | groups | precision | recall | shown twice |
| ------------------------------------------------- | ------ | --------- | ------ | ----------- |
| take the members holding another chosen group out | 72     | 0.971     | 0.969  | 0.058       |
| offer both as they are                            | 81     | 0.959     | 0.972  | 0.064       |
| keep the one that scores higher                   | 53     | 0.966     | 0.924  | 0.031       |
| keep the one with more members                    | 61     | 0.965     | 0.948  | 0.053       |

The first line wins on precision, on recall and on the number of groups at the
same time, so there is nothing to trade. It also matches what the page means:
the featured post and the list beside it are two things to offer, not one
instead of the other. What is left of a group after it loses members is scored
again, because it is no longer the group that was scored.

Keeping the one that scores higher is the rule that loses the whole list of
`cursor.com` to a two-member featured pair, which is where this was found.

### Result

`alpha = 0.8`, 72 groups over 21 pages, precision 0.971, recall 0.969.

| page                                  | groups | posts | tp  | fp  | fn  |
| ------------------------------------- | ------ | ----- | --- | --- | --- |
| anthropic.com-news                    | 3      | 13    | 13  | 0   | 0   |
| aws.amazon.com-jp-blogs-news          | 1      | 10    | 10  | 1   | 0   |
| bbc.com                               | 19     | 103   | 91  | 0   | 12  |
| blog.google                           | 1      | 8     | 5   | 0   | 3   |
| claude.com-blog                       | 4      | 24    | 23  | 0   | 1   |
| cursor.com-blog                       | 10     | 28    | 24  | 0   | 4   |
| daily.bandcamp.com-album-of-the-day   | 2      | 30    | 30  | 7   | 0   |
| daily.bandcamp.com-features           | 2      | 30    | 30  | 8   | 0   |
| deepmind.google-blog                  | 3      | 25    | 25  | 0   | 0   |
| deepmind.google-research-publications | 1      | 30    | 30  | 0   | 0   |
| developer.apple.com-news              | 1      | 108   | 100 | 7   | 8   |
| developers.openai.com-blog            | 1      | 29    | 29  | 0   | 0   |
| diggersfactory.com-vinyl-shop-new-ins | 1      | 9     | 9   | 0   | 0   |
| flutter.dev-blog                      | 1      | 284   | 284 | 0   | 0   |
| github.blog-ai-and-ml                 | 2      | 19    | 19  | 0   | 0   |
| github.blog                           | 5      | 25    | 21  | 0   | 4   |
| go.dev-blog                           | 1      | 10    | 10  | 1   | 0   |
| paulgraham.com-articles               | 2      | 235   | 235 | 1   | 0   |
| qiita.com                             | 4      | 30    | 30  | 6   | 0   |
| ycombinator.com-blog-tag-essay        | 4      | 10    | 10  | 0   | 0   |
| ycombinator.com-blog                  | 4      | 10    | 9   | 0   | 1   |

`bbc.com` is offered as 19 groups, and that is correct rather than a failure.
Every one of them holds posts only, none repeats a post of another, and they
read as the sections of a homepage: sport, world news, health, travel, food,
science. A homepage is not a blog index, and the user selecting six sections out
of nineteen is the page working as intended. The 12 posts it does not reach sit
in sections of two to four members that fall just under the threshold.

### Are the weights fitted to these pages

Partly. A grid search over the four weights and the threshold shows a wide
plateau rather than a single peak, so the result does not depend on one exact
number. The honest figure is leave-one-page-out, where the weights are chosen on
twenty pages and measured on the twenty-first: **precision 0.963, recall
0.977**. It matches the figure above, which is the sign that the rule is not
fitted to the corpus.

## Part 2: which link of a member is the post

Three ways of choosing, measured over the 910 members of the target group of
every page:

| how the link is chosen                                   | right |
| -------------------------------------------------------- | ----- |
| the first link in document order                         | 887   |
| ranked by repetition inside the card and by text length  | 871   |
| the same ranking, then the URL shape the group agreed on | 897   |

The middle line is worse than the first one overall, and it is still needed.
Document order fails where the card puts something else first: 3 of 15 on
`bbc.com` and 21 of 30 on `qiita.com`. The ranking repairs both and breaks
`developer.apple.com`, where an item is running text and a link in the body is
repeated more often than the item's own link, so 108 right becomes 74.

Reading the group twice repairs that in turn. The members of one list agree on
the shape of their links, and keeping, for every member, the candidate with that
shape brings `developer.apple.com` back to 100 of 108 while keeping the two
pages the ranking repaired.

This is the general result of the measurement: a value read from one node is
never enough, and what a member holds has to be read against what the other
members of the same group hold.

## Signals that were measured and set aside

Measured per distinct link, where a signal counts as present when it is present
on any node holding that link:

| signal                             | cover | precision | recall |
| ---------------------------------- | ----- | --------- | ------ |
| a heading tag                      | 609   | 0.91      | 0.52   |
| the path starts with the page path | 690   | 0.90      | 0.58   |
| an image                           | 860   | 0.87      | 0.70   |
| a text of 25+ characters           | 1061  | 0.81      | 0.80   |
| a text of 15+ characters           | 1529  | 0.67      | 0.96   |
| a deeper path than the page        | 1562  | 0.45      | 0.66   |
| a datetime attribute               | 100   | 0.53      | 0.05   |
| the same link twice in one card    | 57    | 0.77      | 0.04   |

Taking the first three together and requiring a text of 15 characters or more
gives precision 0.89 and recall 0.92, and returns nothing at all on
`github.blog` and `ycombinator.com-blog`, whose cards hold the image on the card
rather than on the post link. That is what the group score replaces.

Two signals are weak enough to be worth naming so that they are not tried again.
A `datetime` attribute says almost nothing, because a date is written as a plain
string far more often than as a `<time>` element. Sibling shape repetition,
which is what the enumeration in `unstructured_feed.go` relies on, gives
precision 0.58 and recall 0.78 here, because the shape of a node after folding
is short and many unrelated links share it.

### Text unwrapping

Neighbouring texts written in tags that carry no meaning of their own, such as
`p`, `span` and `div`, can be read as one text, so that a list where some cards
name the author in a span and others do not stops producing two shapes for one
list. Measured on the target group of every page, the share of members agreeing
on one signature rises from 0.915 to 0.982 and no page gets worse. Most of the
gain comes from leaving the number of children and the presence of an image out
of the signature, which a card still split into several nodes would never match.

It does not change the final numbers on this corpus, because the shape part
carries 0.2 of the score and the pages whose signatures improve were already
found through their content. It would matter on a page whose cards carry no
date, no image and short titles. The simplified signature is kept because it is
never worse and is simpler. Merging the text values themselves, rather than only
the signature, costs about 0.005 of F1, because a merged text is longer and a
list of tags then also looks more like a list of posts.

## In Go

The rule is implemented in `semantic_groups.go` as `SelectPostGroups`, and
`TestSemanticGroups` measures it against the fixtures with an acceptance table
of its own. It reproduces the Python result exactly, page by page: 72 groups,
precision 0.971, recall 0.969.

- The code:
  file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/semantic_groups.go
- The measurement:
  file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/semantic_groups_test.go

### Against the enumeration

Over the same 21 pages, `EnumeratePostGroups` returns 523 groups, of which 67
have to be ticked to reach every post. Reading the posts out of the tree returns
72 groups instead, and almost every one of them is worth ticking.

| page                                  | enumeration | to tick | tree   | posts found      | other links |
| ------------------------------------- | ----------- | ------- | ------ | ---------------- | ----------- |
| anthropic.com-news                    | 5           | 2       | 3      | 13               | 0           |
| aws.amazon.com-jp-blogs-news          | 13          | 2       | 1      | 10               | 1           |
| bbc.com                               | 40          | 23      | 19     | 91               | 0           |
| blog.google                           | 4           | 2       | 1      | 5                | 0           |
| claude.com-blog                       | 5           | 3       | 4      | 23               | 0           |
| cursor.com-blog                       | 13          | 2       | 10     | 24               | 0           |
| daily.bandcamp.com-album-of-the-day   | 11          | 1       | 2      | 30               | 7           |
| daily.bandcamp.com-features           | 12          | 1       | 2      | 30               | 8           |
| deepmind.google-blog                  | 4           | 3       | 3      | 25               | 0           |
| deepmind.google-research-publications | 2           | 1       | 1      | 30               | 0           |
| developer.apple.com-news              | 95          | 1       | 1      | 100              | 7           |
| developers.openai.com-blog            | 135         | 1       | 1      | 29               | 0           |
| diggersfactory.com-vinyl-shop-new-ins | 1           | 1       | 1      | 9                | 0           |
| flutter.dev-blog                      | 3           | 2       | 1      | 284              | 0           |
| github.blog                           | 43          | 10      | 5      | 21               | 0           |
| github.blog-ai-and-ml                 | 21          | 3       | 2      | 19               | 0           |
| go.dev-blog                           | 3           | 1       | 1      | 10               | 1           |
| paulgraham.com-articles               | 3           | 1       | 2      | 235              | 1           |
| qiita.com                             | 84          | 2       | 4      | 30               | 6           |
| ycombinator.com-blog                  | 11          | 3       | 4      | 9                | 0           |
| ycombinator.com-blog-tag-essay        | 15          | 2       | 4      | 10               | 0           |
| **total**                             | **523**     | **67**  | **72** | **1037 of 1070** | **31**      |

The number of rows the user has to read falls from 523 to 72. The price is the
one thing the enumeration does perfectly: its URL recall is 1.00 on every page,
while the tree loses 33 posts, mostly the 12 of `bbc.com` that sit in sections
just under the threshold and the 8 of `developer.apple.com` whose link in the
body is the one the group agrees with.

### Which of the two to offer the user

The tree, at a lower threshold than 0.8, with the enumeration kept as the answer
to a page where the tree finds nothing.

The recall of the enumeration looks like the stronger argument, and the reading
cost looks like the weaker one, but the table above says otherwise. Only 67 of
the 523 groups the enumeration returns are worth ticking. The user reaches the
same posts either way; what differs is that one path asks them to look at 523
rows to find 67, and the other asks them to look at 72 rows that are almost all
correct. The enumeration does not offer more, it offers the same result with
eight times the reading.

The 33 posts the tree loses are also not posts it failed to see. They are posts
it saw and declined to offer, because their group scored just under 0.8 times
the best score of the page. That moves with the threshold:

| alpha | groups | posts found | other links | precision | recall |
| ----- | ------ | ----------- | ----------- | --------- | ------ |
| 0.90  | 42     | 959         | 16          | 0.984     | 0.896  |
| 0.85  | 56     | 982         | 32          | 0.968     | 0.918  |
| 0.80  | 72     | 1037        | 31          | 0.971     | 0.969  |
| 0.75  | 97     | 1047        | 53          | 0.952     | 0.979  |
| 0.70  | 119    | 1057        | 91          | 0.921     | 0.988  |
| 0.65  | 143    | 1055        | 122         | 0.896     | 0.986  |
| 0.60  | 230    | 1054        | 253         | 0.806     | 0.985  |

0.75 recovers 10 of the 33 for 25 more groups, and 0.70 recovers 20 of them for
47 more groups. Both still ask the user to read far fewer rows than the 523 of
the enumeration. Which of the two to take depends on how the selection screen is
used, and that is the one part of this decision the measurements cannot settle:
if a site is set up once, the extra rows are paid once and 0.70 is right; if the
screen is opened often, 0.75 keeps the reading shorter.

Below 0.65 the trade stops working. Recall does not improve, precision falls
quickly, and at 0.60 the page root and the tag lists come in, which is the
symptom described in the open points below.

Note that no threshold reaches the recall of the enumeration. Recall stops near
0.99 and never reaches 1.00, because about 11 posts are not lost to the
threshold at all: `blog.google` has 3 whose titles exist only in an attribute,
and `cursor.com` has 4 that point at other hosts. Those need the fixes in the
open points, not a different threshold. This is the reason to keep
`EnumeratePostGroups` in the code rather than delete it: it is the answer when a
user says the list they want is not on the screen.

The asymmetry behind all of this is that a group the user does not need costs
them one glance and they can ignore it, while a post that was never offered
cannot be recovered, because the user does not know it existed.

### Cost

Measured on the two largest saved pages, parsing included, on an Apple M1:

| page                       | enumeration | tree  |
| -------------------------- | ----------- | ----- |
| developers.openai.com-blog | 24 ms       | 37 ms |
| developer.apple.com-news   | 33 ms       | 53 ms |

The first version was three times slower than this, because the test that drops
a banner repeated on every card read every sibling again for each member.
Counting once per group how many members carry each link makes that test linear,
which is where the difference went.

## Open points

- **A title written in an attribute is not in the tree.** `blog.google` loses
  three posts because their titles are not text anywhere. The page is built from
  custom elements and the title sits in an attribute, as in
  `<uni-simple-article-card headline="AI for everyone in every language">`.
  Reading the text-like attributes of an element that has no text of its own
  would fix those three, and `semantic_tree.go` has to carry them first.
- **A tag list can reach 80% of the best score of its page.** That is where the
  extra links on `qiita.com` and `ycombinator.com` come from. Comparing a group
  with the other groups of the page, rather than only with the best one, is the
  next thing to try.
- **A post on another host has no shape to agree with.** `cursor.com` lists
  press coverage on `techcrunch.com` and `bloomberg.com`, so those links share
  no template with the rest of the list.
- **Length is measured in characters.** `qiita.com` is written in Japanese,
  where a title of the same meaning is shorter. It was not the cause of any
  failure measured here, but it is the first thing to check when a page fails.
