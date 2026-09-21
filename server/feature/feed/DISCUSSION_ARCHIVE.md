# Archive: finished work on unstructured feed detection

Defects that are fixed and decisions that are closed, moved out of
`DISCUSSION.md` so that file holds only open work. Nothing here needs an action.
It is kept so the decisions do not have to be made again from scratch.

All measurements were taken on the 25 saved pages in `testdata/`, using the
fixtures as the current result. Counts refer to the 1104 posts the fixtures
hold, or to the 1073 of them that could be mapped back to their DOM node.

## The code before the fixes

`DetectPostLists` picked the two main fields of a post independently:

- `URL` is `links(item)[0]`, the first element with an `href` in document order,
  after filtering (unresolvable links, `<link>`, links back to the page itself
  and bare host links are dropped).
- `Title` is the text of the first `h1`-`h6` in the item. If the item has no
  heading, it is the text of the first element carrying an `href`, found by a
  separate walk inside `title()` that applies none of the filters above. If
  there is no such element either, it is the item text, cut to 120 characters.

Nothing connects the two picks, which is the root of defects 1 and 2.

## Defect 1 (fixed): the URL is taken from the wrong link

github.blog renders a category badge link above the post heading:

```html
<div class="mb-1"><a href="https://github.blog/ai-and-ml/">AI &amp; ML</a></div>
<h2>
  <a href=".../migrating-the-github-copilot-runtime-to-rust-using-copilot/"
    >Migrating the GitHub Copilot runtime to Rust…</a
  >
</h2>
```

The badge is the first link, so it becomes `Post.URL`, while the title comes
from the heading. The post is reported under its category URL.

This is worse than a wrong field. Every card in such a list reports the same
category URL, so `seen`, the set of distinct URLs in the group, collapses to one
or two entries and the check `len(seen) < minMembers` skips the whole group. Two
containers on github.blog are dropped this way:

| container                         | cards | distinct first links | result       |
| --------------------------------- | ----- | -------------------- | ------------ |
| `main > section > div > div > ul` | 4     | 1                    | 4 posts lost |
| `main > section > … > div > ul`   | 3     | 2                    | 3 posts lost |

A third container of 3 cards survives with 3 distinct category URLs. Two of them
were already claimed by another list, so the deduplication left it with a single
entry. That is fixture list 5: one "post" pointing at
`https://github.blog/developer-skills/`.

A scan of all 25 pages for containers where the first-link rule collapses while
a heading-link rule would not found this pattern on github.blog only.

## Defect 2 (fixed): a page section is detected as a post list

Fixture list 1 of github.blog is not a post list. Its items are the top level
sections of the page:

```
1 <div> nlinks=14 first=https://github.blog/ai-and-ml/          head="Should you read the code, is RAG dead…"
2 <div> nlinks=20 first=https://github.blog/latest/             head="Latest"
3 <div> nlinks=6  first=https://github.blog/changelog/          head="Changelog"
4 <div> nlinks=16 first=https://github.blog/engineering/        head="Engineering"
5 <div> nlinks=8  first=https://www.youtube.com/github          head="Spotlight"
6 <div> nlinks=13 first=https://github.blog/news-insights/      head="News & insights"
7 <div> nlinks=4  first=https://docs.github.com/                head="The world's largest developer platform"
```

Each item holds 4 to 20 links, its first link is the section's own "see all"
link, and the real post lists (fixture lists 2 to 6) are all descendants of
these items. The group wins the page with score 14 because all 7 sections carry
a date, which triggers the `×2.0` bonus.

## Accepted and implemented: pair the title and the URL

The URL is taken from the element the title came from.

1. Find the title node as today: the first `h1`-`h6` with non-empty text.
2. If a heading was found, take its link, in this order:
   1. the first accepted link below the heading;
   2. otherwise, the first ancestor carrying an `href`, walking up from the
      heading to the item node inclusive, which covers a card wrapped in one
      link and covers the custom elements on blog.google;
   3. otherwise, `links(item)[0]`, as before.
3. If no heading was found, take the title from the element that produced
   `Post.URL`, instead of the separate unfiltered walk in `title()`. The two
   walks could disagree, because only `links()` drops self links and bare host
   links.

Every candidate passes the same filter as `links()`. If no heading link
qualifies, the item keeps the URL it had before, so no item can lose its URL or
stop being a post.

Implemented as `titleAndURL`, which replaces `title`. `links` now returns the
element together with the resolved target, so the two picks can come from one
element.

Predicted effect on the 1073 posts mapped to their nodes:

| case                                        | posts |
| ------------------------------------------- | ----- |
| no heading in the item, rule does not apply | 458   |
| heading link is the URL already reported    | 566   |
| heading found but it holds no link          | 46    |
| heading link differs, URL changes           | 3     |

Plus the 7 posts of defect 1, which return once the group is no longer skipped.

## Accepted and implemented: reject a group whose items contain another detected group

A post does not contain other posts. On github.blog this removes fixture list 1
exactly and keeps lists 2 to 6, because every other list is a descendant of list
1's items. The rejection runs inside the score cut loop in `DetectPostLists`.

Consequences, measured on the fixture:

- 6 junk entries are removed: 5 category links whose posts are reported
  correctly by the other lists, plus `youtube.com/github` and
  `github.com/customer-stories`.
- 1 real post is removed with them, the hero item "Should you read the code, is
  RAG dead, and did Skills kill MCP?". It sits alone in its section with no
  repeated sibling, so no structural rule can find it. It was reported before
  only by accident, and with the wrong URL.
- Lists 2 to 6 are unchanged. After list 1 is gone the top score is 12, so the
  cut becomes 3 and the lists scoring 12, 10, 8, 6 and 4 all remain. None of
  list 1's URLs appears in another list, so nothing is pushed out or brought
  back by the deduplication.

Rejecting only the items that contain a detected group, rather than the whole
group, gives the same result: it leaves 2 items, below `minMembers = 3`.

## Measured effect of the two changes

Only five pages changed. Every other page is identical.

| page                 | url precision | url recall   | top list precision | posts     |
| -------------------- | ------------- | ------------ | ------------------ | --------- |
| github.blog          | 0.56 -> 0.83  | 0.60 -> 1.00 | 0.00 -> 1.00       | 27 -> 30  |
| qiita.com            | 0.95 -> 1.00  | 0.70 -> 1.00 | 1.00 -> 1.00       | 22 -> 30  |
| bbc.com              | 0.83 -> 0.92  | 0.88 -> 0.85 | 0.21 -> 1.00       | 109 -> 96 |
| blog.google          | 0.71 -> 1.00  | 0.62 -> 0.62 | 1.00 -> 1.00       | 7 -> 5    |
| ycombinator.com-blog | 0.42 -> 0.41  | 1.00 -> 0.90 | 1.00 -> 1.00       | 24 -> 22  |

Title precision and recall reached 1.00 on github.blog and on qiita.com, and
rose from 0.97 to 0.99 on bbc.com. The 8 posts qiita.com gained come from the
title and URL pairing alone and were not predicted above.

Two points that the measurement added to the decisions:

- The nesting rejection must compare only the lists that pass the score cut.
  Comparing against every candidate group destroys the pages whose cards hold a
  repeated group of their own, for example a row of tags: developer.apple.com
  falls to 0.01 url precision, and aws.amazon.com and qiita.com to 0.00.
- Dropping a list lowers the best score, so the cut is recomputed and the step
  repeats until the set stops changing. Letting lists that were below the first
  cut back in afterwards was tested and rejected: on github.blog it admits four
  YouTube playlist cards and one "see all" link, and url precision falls from
  0.83 to 0.58.

Keeping the two changes in separate steps matters: after the title and URL
pairing, the junk removed by the nesting rejection no longer includes a real
post that the pairing has not already fixed.

## Rejected for now: slug matching as the first step

The proposal was to pair a title with a URL by matching the normalized title
against the URL, across all pairs of texts and links in an item, and to fall
back to the structural rule when no pair matches.

The rule itself was refined during the discussion and is worth keeping written
down:

- Normalization: lowercase; whitespace and the separators `-`, `_`, `/`, the en
  dash and the em dash become `-`; every other character that is not `a-z` or
  `0-9` is dropped; non-ASCII is percent-encoded as UTF-8 with uppercase hex.
  Dropping the hyphen instead of treating it as a separator breaks compound
  words and costs 36 matches, for example `multimodel` against the real segment
  `multi-model`, and `costefficient` against `cost-efficient`.
- Matching: one path segment of the URL must be exactly equal to the title slug.
  Substring and prefix matching are not reliable and are not used.

Measured coverage with that rule: 341 of 1104 posts, 31%. It works on
daily.bandcamp.com (60 of 60), flutter.dev (200 of 284), github.blog AI & ML (18
of 19), ycombinator (19 of 34) and claude.com (8 of 23). It never works on
developer.apple.com (108 posts, numeric `?id=`), paulgraham.com (235),
deepmind.google publications (30), developers.openai.com (29), cursor.com (24),
qiita.com (22), anthropic.com (12) and aws.amazon.com jp (11), because those
sites do not build their URLs from the title. The remaining misses on the
matching sites are editorial: the slug is a different phrase from the heading,
for example "Material and Cupertino decoupling are here" against the segment
`decoupling-material-cupertino`.

On the 458 posts whose item has no heading, which is the case the proposal was
aimed at, the rule finds 66 matches. 64 agree with the URL the current algorithm
already reports. The 2 that differ are:

```
ycombinator.com/blog
  title   : "BillionToOne Goes Public — The Startup That Made Genetic Testing Universal"
  current : /blog/billiontoone              (correct)
  matched : /blog/author/jared-friedman     (the byline text matches the author URL)

diggersfactory.com
  title   : "Contact us"
  current : /contact                        (junk item in a footer group)
  matched : /return-and-refund-policy       (a neighbouring link's own text)
```

So on this page set the rule corrects nothing and damages one correct pairing.
The cause is intrinsic: a byline link and a policy link are built from their own
text exactly as a post URL is built from its title, so they match just as
strongly. Restricting the text candidates to leaf text only, rather than the
text of every element, was tested and changes nothing, because both wrong
candidates are leaf texts. Excluding them requires knowing which text is the
title, and once that is known the pairing is already decided.

Therefore slug matching cannot run before the structural rule. It remains useful
as a check on a pair that has already been chosen: if the chosen URL has no
segment equal to the title slug, and exactly one other link in the item does,
prefer that link. With that framing it cannot damage a correct pairing, and it
would have caught defect 1 on its own. This is recorded as a possible later
step, not as work to do.

## Defect 4 (fixed): a group of site navigation links is detected as a post list

ycombinator.com-blog returned 22 posts for a fixture of 10, and
diggersfactory.com-vinyl-shop-new-ins returned 27 for a fixture of 9. The extra
entries came from whole lists that are menus:

```
ycombinator.com-blog
  score 12.0  6 items   6 in fixture   the real post list
  score  3.3  11 items  0 in fixture   /blog/tag/admissions, /blog/tag/advice, …
  score  3.0  3 items   1 in fixture
  score  3.0  1 item    1 in fixture
  score  3.0  1 item    1 in fixture

diggersfactory.com-vinyl-shop-new-ins
  score  9.0  9 items   9 in fixture   the real post list
  score  2.7  9 items   0 in fixture   /vinyl-shop?genre=8, ?genre=11|18, …
  score  2.7  9 items   0 in fixture   /about, /blog, /terms, /privacy-policy, …
```

The nesting rejection of defect 2 does not remove them, because these groups are
siblings of the real list, not ancestors of it. Their scores sit just above the
cut: on diggersfactory the cut is 9.0 × 0.25 = 2.25 and the menus score 2.7, and
on ycombinator the cut is 3.0 and the tag list scores 3.3.

The score already recognises them. Each of these three groups was multiplied by
0.3 because its average item text is below `minItemText`: 10 and 11 bytes on
diggersfactory, 10 bytes on ycombinator, against 72 and 284 bytes for the two
real lists. The penalty alone is not enough, because a menu with many entries
still outscores the cut. Raising `unionFraction` instead was rejected: 0.33
would remove the ycombinator tag list, but it would also remove the three groups
below it that hold real posts.

### Accepted and implemented: a menu is only returned when the page holds nothing better

A group that got the `avgText < minItemText` penalty is dropped while a group
without that penalty is kept beside it. `PostList.menu` records the penalty, and
the rejection runs in the score cut loop next to the nesting rejection.

The condition is needed because a page can write its posts as bare links.
paulgraham.com/articles is exactly that: its only list holds 235 items with an
average item text of 22 bytes, so it carries the penalty and is right. It is the
only page of the 25 whose real list is a menu, and it has no other candidate, so
it is unaffected.

Measured over all 25 pages, only the two pages above changed:

| page                                  | posts    | url precision | url recall   |
| ------------------------------------- | -------- | ------------- | ------------ |
| diggersfactory.com-vinyl-shop-new-ins | 27 -> 9  | 0.33 -> 1.00  | 1.00 -> 1.00 |
| ycombinator.com-blog                  | 22 -> 11 | 0.41 -> 0.82  | 0.90 -> 0.90 |

Both pages now pass level one, which accepts 0.80 url precision on each of them.
No other page lost or gained a post.

The 2 entries ycombinator.com-blog still reports that the fixture does not hold
are `/blog/tag/yc-news` and `/blog/author/garry`. They are not a menu: they sit
in a 3 item group whose average item text is 93 bytes, which is one of the
groups the nesting rejection left behind when it split the hero cards. They are
part of the level two problem recorded in `DISCUSSION.md`, not of this defect.

### Accepted and implemented: a region that names itself in its class attribute is chrome

diggersfactory.com writes its page footer as `<section class="jsx-… footer">`,
and the two menus sit inside it, one of them under `<ul class="links">`.
`cleanup` removed `<nav>`, `role=navigation|contentinfo|banner` and the
`<header>` and `<footer>` tags outside a sectioning element, so none of its
tests reached this region. `named` now compares the class names of an element
against the whole words `footer`, `nav` and `navigation`, and the region is
removed with everything below it before any group is scored.

Measured on all 25 pages: no list that holds real posts has an ancestor carrying
any of the words `footer`, `nav`, `navigation`, `menu`, `sidebar`, `links`,
`tags`, `pagination`, `social` or `breadcrumb`, so removing such a region cannot
cost a post on this page set. With the menu rejection disabled, this test alone
brings diggersfactory.com from 27 detected posts to 9, which is the whole
fixture, and changes no other page.

Only whole words are compared. A class name that contains the word says nothing
about the region: github.blog writes `featured-external-links-pattern__list`
around 4 junk entries, and a post card is free to carry a class such as
`post-links`.

The test cannot replace the menu rejection, because most sites name nothing. Of
the five pages that produced junk lists, only diggersfactory uses words at all:

| page                 | markup around the junk list                       | named |
| -------------------- | ------------------------------------------------- | ----- |
| diggersfactory.com   | `<section class="footer">`, `<ul class="links">`  | yes   |
| ycombinator.com-blog | `<div class="ml-20">`, `<a class="flex">`         | no    |
| github.blog          | `featured-external-links-pattern__list`           | no    |
| bbc.com              | `IndexCard-styles__IndexCardStyled-sc-3a0d8802-0` | no    |
| go.dev-blog          | `blogtitle`                                       | no    |

Tailwind utility classes and hashed CSS module names carry no meaning, and they
are the majority. With the class test alone and the menu rejection disabled,
ycombinator.com-blog returns to 0.41 url precision.
