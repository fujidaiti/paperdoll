# The semantic tree

An experiment that stands beside the enumeration in `unstructured_feed.go` and
is not used by it. The code is in `semantic_tree.go`, and the tree of every
saved page is rendered into `testdata/semantic/` by `TestSemanticTree`, which
rewrites the files on every run.

## The problem it attacks

The enumeration groups sibling elements by the shape of their markup two levels
deep, and it requires the shapes to be equal. A list whose items are written
differently therefore falls apart into several groups. On
`https://www.anthropic.com/news` the four side posts of the featured section
produce two groups, because the first of them carries no category label:

```
a(div(time)|h4|p)          <- no category
a(div(span|time)|h4|p)
a(div(span|time)|h4|p)
```

The same rule makes every wrapper element around a list a group of its own, so
one list is offered to the user once per wrapper. Two reductions remove the
copies afterwards, and what is left is still 95 groups on
`developer.apple.com/news` and 135 on `developers.openai.com/blog`.

## The idea

What matters is not how a post is marked up but which values belong to the same
post. A post is made of a link, texts and images, so the tree keeps only those
and drops every element that carries none of them.

A subtree is folded into a single node when it carries at most one distinct
link. The assumption is that a link identifies a post, so everything around a
single link describes that one post, whatever markup holds it:

```
<a href="u"><div><div><p>text</p></div></div></a>
```

becomes one node with the link `u` and the text `text`. A subtree that carries
two or more distinct links describes more than one post, so it keeps one child
per part that holds a post. A wrapper that adds no value of its own and holds a
single child is replaced by that child.

A node therefore holds one link at most, and it is stored as a single string.
Measured over the rendered trees, 2530 nodes hold one distinct link, 991 hold
none, and none holds two. 168 nodes did hold the same URL more than once,
because a card is often covered by an empty element repeating its link, and
those repeats are dropped.

## What was measured

Against the hand written fixtures in `testdata/`, with two definitions:

- a **group node** is a node with at least two children where each child that
  holds a post holds exactly one post, which is the equivalent of a group on the
  selection screen;
- **to cover** is how many group nodes a user has to tick before every fixture
  post is included, taking the largest first. `2 + 1` means two group nodes plus
  one post that no group node covers.

```
page                                    posts  nodes  groups   to cover    today
anthropic.com-news                         13     29       2      2 + 1        5
aws.amazon.com-jp-blogs-news               10    175       1          1       13
bbc.com                                   103    196      21     21 + 5       40
blog.google                                 8     21       3          2        4
claude.com-blog                            24     86       4      2 + 1        5
cursor.com-blog                            28    123      12          5       13
daily.bandcamp.com-album-of-the-day        30     73       2          1       11
daily.bandcamp.com-features                30     76       1          1       12
deepmind.google-blog                       25     40       2      2 + 1        4
deepmind.google-research-publications      30     43       1          1        2
developer.apple.com-news                  108   1003       5          1       95
developers.openai.com-blog                 29    841       2          1      135
diggersfactory.com-vinyl-shop-new-ins       9     15       1          1        1
flutter.dev-blog                          284    289       1          1        3
github.blog                                25    193       6          6       43
github.blog-ai-and-ml                      19    105       2          2       21
go.dev-blog                                10     27       1          1        3
paulgraham.com-articles                   235    500       2          1        3
qiita.com                                  30    490       1          1       84
ycombinator.com-blog                       10     71       2      2 + 1       11
ycombinator.com-blog-tag-essay             10     88       2          2       15
```

`today` is how many groups `EnumeratePostGroups` returns for the same page.

Results:

- No fixture post is lost by the folding. URL recall is 1.00 on every page,
  which is the hard requirement of the design.
- The number of groups a user has to read collapses. On 14 of the 21 pages a
  single group node covers every post.
- The two problems of the anthropic page are gone without a rule aimed at them.
  The hero is one node, the side posts are siblings under another node, and the
  missing category label splits nothing, because no markup shape is compared.
- The reductions of `EnumeratePostGroups` are not needed here. A wrapper cannot
  produce a copy of a group, because a wrapper that adds no value is replaced by
  its child.

The pages that leave a post uncovered (`+ 1`) leave a featured post that stands
alone in its container. That is correct, and it only asks the selection screen
to accept a group of one.

## Merging the nodes that carry no link

A node that carries no link is never a post. Leaving it as a child of a group
offers the user a post that does not exist, so its values are moved into the
nodes around it, on the assumption that a value next to a link describes the
post that link points at, the way a tag, a banner or a date does.

Note that a node that carries no link also has no children, because a node only
gets children when its subtree holds two links or more. The nodes being merged
are always leaves.

### The vertical rule, which is kept

A child that carries no link is removed, and its texts and images are moved up
into the parent:

```json
{ "children": [{ "texts": ["A"] }, { "link": "u", "texts": ["B"] }] }
```

becomes

```json
{ "texts": ["A"], "children": [{ "link": "u", "texts": ["B"] }] }
```

Measured over the corpus: 966 link-less children before, 0 after, and the tree
shrinks from 4487 nodes to 3521. Nothing is lost, because the multiset of all
links, texts and images is identical before and after. Attribution does not get
worse either: the number of fixture titles, dates and images that sit on the
same node as their post link is unchanged on every page, so no value that
belonged to one post was pulled up into a container of many.

The largest single effect is on `developer.apple.com/news`, which lists its
posts as running text rather than as cards: 1003 nodes come down to 637, and an
item now carries its own image beside the child that carries its link.

### The horizontal rule, which was dropped

A run of neighbouring children that carry no link is replaced by one node
holding the values of all of them:

```json
{ "children": [{ "texts": ["A"] }, { "texts": ["B"] }] }
```

becomes

```json
{ "children": [{ "texts": ["A", "B"] }] }
```

It was implemented, measured and removed. The rule can only produce a result
that the vertical rule does not immediately consume when _every_ child of a node
carries no link, and that case occurs zero times over the 25 saved pages. The
rule does merge 62 runs, 46 of them on `developer.apple.com`, but each merged
node is then moved up into the parent anyway, and the values land in the same
order either way, because texts and images are separate lists. On this corpus it
is exactly a no-op.

The shape it was written for is a card whose link sits on an ancestor rather
than inside the card, so that all children of the card carry no link. If a saved
page with that shape is added, the rule is worth writing again.

## What a text carries

Folding a subtree into one node throws its markup away, so a text is stored as
an object rather than as a plain string:

```json
{
  "tag": "time",
  "value": "Aug 14, 2026",
  "datetime": "2026-08-14T12:00:00.000Z"
}
```

`tag` is the element the text was read from. It was chosen after matching every
fixture value back to the element that holds it on the 25 saved pages:

| value           | where it is written                                                       |
| --------------- | ------------------------------------------------------------------------- |
| title (1067)    | `a` 382, `h3` 309, `h2` 222, `span` 49, `p` 47, `div` 46, `h1` 5, other 6 |
| timestamp (724) | `span` 356, `time` 146, `p` 108, `div` 52, 62 not found as one text       |

So the element name is a strong signal for a title: a heading, or the text of
the link itself, covers 921 of 1067 titles. It is a weak signal for a date,
because `time` covers only 146 of 724. A date is better found by reading the
string: a plain date pattern matches 622 of the 724 timestamps and only 8 of the
1067 titles. Being inside a link separates nothing, since 1012 of 1067 titles
and 392 of 724 timestamps are both inside one.

`datetime` is the machine readable date of a `<time>` element. It is kept beside
`value` instead of replacing it, which is what the enumeration does today. The
two say different things, and only `value` can be shown to the reader: on
`cursor.com` the pair is "Aug 14, 2026" and "2026-08-14T12:00:00.000Z". Of the
6295 texts in the corpus, 163 carry one.

Nothing else is stored. A judgment such as "this text is the date" is left to
the scoring step, so that the rendered trees keep showing where a scoring rule
would be wrong. Class names and element ids are not stored either, because most
of the saved pages build them from utility classes or hashes and they name
nothing.

## Open points

- **A card that holds a second link splits.** When a card carries an author, a
  category or a tag link next to the post link, it holds two distinct links, so
  it is not folded. The post link lands in one child and the date in a sibling.
  Measured against the fixtures, the date is in the same node as the post link
  on 12 of 108 items on `developer.apple.com`, and on none of the items of
  `qiita.com`, `aws.amazon.com`, `github.blog` and `ycombinator.com`. Every
  value is still in the tree, but reading a post means walking a subtree that
  also holds the author name and the category. The vertical rule does not reach
  this case, because the node holding the date has children of its own when the
  body text contains links.
- **Which node is a post list is not yet a rule.** The tables above were
  produced with the fixtures in hand. The largest node that covers every post is
  usually not the list: on `anthropic.com` it is the page container, which has
  three children. The candidate rule to test is that a node is a group when at
  least two of its children each hold exactly one link, and the group is the set
  of those children.
- **Whether the tree replaces the enumeration or feeds it** is open. The next
  step is to read posts out of the tree with the rule above and to run the
  acceptance table in `metrics_test.go` against them.
