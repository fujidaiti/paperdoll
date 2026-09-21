# Open defects in unstructured feed detection

Record of the defects found in `unstructured_feed.go` that are still open. It
states what was measured, what was decided, and what was rejected, so the
decisions do not have to be made again from scratch.

The finished work is in `DISCUSSION_ARCHIVE.md`: defect 1 (the URL is taken from
the wrong link), defect 2 (a page section is detected as a post list), defect 4
(a group of site navigation links is detected as a post list), defect 5 (the
nesting rejection splits a card into its parts), the accepted changes with the
measured effect, and the rejected proposals: pairing titles and URLs by slug
matching, and rejecting a list by the host of its links.

All measurements were taken on the 25 saved pages in `testdata/`, using the
fixtures as the current result. Counts refer to the 1104 posts the fixtures
hold, or to the 1073 of them that could be mapped back to their DOM node.

## Defect 3: the item holds several images and the first one is taken

`image()` returns the first `<img>` of the item in document order, reading `src`
and then `data-src`. A card often holds more than one `<img>`, and the first one
is not always the post image.

github.blog ships one image per breakpoint and hides the other with CSS classes:

```html
<img
  width="200"
  height="200"
  src=".../generic-mona-copilot-logo-1-e176...png?resize=200%2C200"
  class="d-none d-sm-block cover-image tease-thumbnail__img"
  alt=""
/>
<img
  width="400"
  height="212"
  src=".../generic-mona-copilot-logo.png?resize=400%2C212"
  class="d-block d-sm-none cover-image tease-thumbnail__img wp-post-image"
  alt="Decorative background featuring two Copilot figures..."
/>
```

`d-none d-sm-block` means hidden below the `sm` breakpoint and shown above it,
and `d-block d-sm-none` is the opposite, so a desktop browser shows the 200x200
square crop and a phone shows the 400x212 wide crop. The card changes from a
side by side layout to a stacked one at that breakpoint, which is why the two
crops differ in shape. `hidden()` cannot see this, because hiding through a
class name requires the site's stylesheet, and the saved page does not carry it.

bbc.com fails on the same rule for a different reason. Its first `<img>` is a
grey placeholder that the site shows while the real image loads:

```html
<img
  src=".../grey-placeholder.png"
  class="... hide-when-no-script"
  aria-label="image unavailable"
/>
```

It carries no `alt`, and the post image follows it in the same card.

### Measurement of the ranking rules

Measured over the 599 posts that are matched at level one and whose fixture
holds an image, after the two archived changes were implemented. A page is
counted right only when the two URLs are exactly equal.

| page                               | posts   | first   | last    | alt     | width   | area    | alt+area |
| ---------------------------------- | ------- | ------- | ------- | ------- | ------- | ------- | -------- |
| bbc.com                            | 48      | 0       | 48      | 48      | 0       | 0       | 48       |
| github.blog                        | 21      | 5       | 19      | 9       | 20      | 21      | 21       |
| deepmind.google-blog               | 23      | 23      | 13      | 23      | 23      | 23      | 23       |
| flutter.dev-blog                   | 273     | 273     | 1       | 273     | 273     | 273     | 273      |
| ycombinator.com-blog-tag-essay     | 3       | 3       | 0       | 3       | 3       | 3       | 3        |
| cursor.com-blog                    | 2       | 0       | 0       | 0       | 0       | 0       | 0        |
| ycombinator.com-blog               | 3       | 0       | 0       | 0       | 0       | 0       | 0        |
| the 9 other pages, all rules equal | 226     | 226     | 226     | 226     | 226     | 226     | 226      |
| **total**                          | **599** | **530** | **307** | **582** | **545** | **546** | **594**  |

The rules are:

- `first`: the first `<img>` in document order, which is what the method does
  today.
- `last`: the last one.
- `alt`: the first `<img>` with a non-empty `alt`, falling back to the first.
- `width`: the largest declared `width` attribute.
- `area`: the largest declared `width` times `height`.
- `alt+area`: the largest declared area among the images with a non-empty `alt`,
  falling back to every image when none carries one.

`alt+area` is the only rule that is never worse than `first` on any page. The
two signals cover different pages: `alt` alone fixes bbc.com and `area` alone
fixes github.blog, because github.blog gives both crops an empty and a filled
`alt` the other way round on some cards. A decorative or placeholder image
carries `alt=""`, which is the accessibility convention, so the `alt` test is
not specific to these two sites.

### The 5 posts no ranking rule reaches

These are not ranking problems and need their own fix:

- cursor.com-blog, 2 posts: the `<img>` carries `srcSet` and no `src` at all, so
  `image()` finds nothing. Reading the largest descriptor of `srcset` when `src`
  and `data-src` are both empty would return the value the fixture holds.
- ycombinator.com-blog, 3 posts: fixed. The image sat outside the item node, in
  the part of the card that the nesting rejection removed from the detected
  list. Defect 5 restored the card, and all 3 images are now correct. The table
  above was measured before that fix, so its ycombinator row no longer holds.

## Defect 6: the title of an item that has no heading

diggersfactory.com is the clearest case. Every one of its 9 posts is matched at
level one, and every title is wrong, because the card has no heading and the
title falls back to the text of the link, which is the whole card:

```
got  "Electronic Hoe Weekend Only Fire ¥5,690"
want "Hoe Weekend"
```

The genre, the artist and the price are separate elements inside the same link.
Eight pages miss the title precision they accept:

| page                                  | title precision | posts |
| ------------------------------------- | --------------- | ----- |
| daily.bandcamp.com-features           | 0.00            | 30    |
| daily.bandcamp.com-album-of-the-day   | 0.00            | 30    |
| deepmind.google-research-publications | 0.00            | 30    |
| developers.openai.com-blog            | 0.00            | 29    |
| cursor.com-blog                       | 0.00            | 24    |
| anthropic.com-news                    | 0.17            | 12    |
| diggersfactory.com-vinyl-shop-new-ins | 0.00            | 9     |
| ycombinator.com-blog-tag-essay        | 0.70            | 10    |

This is the largest remaining loss across the 25 pages. The shape is not the
same on every page. deepmind.google puts the date in front of the title,
`"1 September 2026 Designing Proactive Thought Partners for Writing"`, while
diggersfactory appends the other card fields after it. ycombinator.com-blog
shows a third shape, the first paragraph of the post appended to its heading. It
has not been investigated yet.

## Defect 7: a post that stands alone is not detected

`minMembers` is 3, so a group of fewer than three items is never a list. A page
that shows its newest post on its own, above the list, therefore loses it.

ycombinator.com-blog is the only page this still costs a post:
`/blog/chris-golda-and-grey-baker-general-partners` is the featured post at the
top of the page, written as one section with a heading link, a "Read More" link
and an image link, and nothing repeats it. Url recall stays at 0.90 because of
this one post.

Lowering `minMembers` to 2 or to 1 has not been measured. It would raise the
number of candidate groups on every page, so it has to be measured across all 25
pages before it is considered.

## Defect 8: a row of off-site link cards is detected as a post list

github.blog holds a row of four cards under the class
`featured-external-links-pattern`, each with an image and a title, each linking
to a YouTube playlist:

```
https://www.youtube.com/playlist?list=PL0lo9MOBetEFKNlPHNouEmVeYeyoyGTXC  Explore GitHub Universe 2025
https://www.youtube.com/playlist?list=PL0lo9MOBetEHEHi9h0k_lPn0XZdEeYZDS  Learn about GitHub Copilot
https://www.youtube.com/playlist?list=PL0lo9MOBetEE0goMLEl97vO7slruNVj43  Stay informed with The Download
https://www.youtube.com/playlist?list=PL0lo9MOBetEHvO-spzKBAITkkTqv4RvNl  Explore GitHub Copilot CLI for Beginners
```

It scores 4.00 against a cut of 3.00, so it is returned, and it is the whole
remaining url precision gap on the page: 0.86 against a target of 0.90.

Structurally it is a post list: four repeated cards, each with an image, a title
and one link. Two signals were measured and both were rejected:

- **The link host differs from the page host.** Rejected: 25 posts across three
  pages are legitimately off-site. cursor.com links to press coverage on
  thenewstack.io, techcrunch.com and bloomberg.com, deepmind.google/blog links
  14 of its 25 posts to blog.google, and paulgraham.com holds 2 posts on a CDN.
- **The link carries `target="_blank"`.** Measured over all 25 pages: it would
  remove 3 junk entries and lose 18 real posts, the same cursor.com and
  deepmind.google posts. Rejected.

The class name `featured-external-links-pattern` is a WordPress pattern name and
carries no general meaning, so the class test in `cleanup` does not reach it
either. No further idea has been measured.

## Plan

1. Investigate defect 6, the title of an item that has no heading. It is the
   largest remaining loss: eight pages report a title precision below the value
   they accept, six of them at 0.00, which is about 174 posts.
2. Rank the images of an item with `alt+area` instead of taking the first one,
   and read `srcset` when `src` and `data-src` are empty. This is defect 3, and
   it reaches about 106 posts over four pages.
3. Measure `minMembers` at 2 and at 1 across the 25 pages. This is defect 7.
4. Defect 8 has no measured idea yet.

Level one is otherwise done. Of the 25 pages, only two still miss a level one
target: ycombinator.com-blog at 0.90 url recall, for the one post of defect 7,
and github.blog at 0.86 url precision, for the four links of defect 8. Every
other page reports 1.00 url precision and 1.00 url recall.

If a future page needs slug matching, add it as a check on the chosen pair,
using the normalization and the exact segment rule written down in
`DISCUSSION_ARCHIVE.md`.

The fixtures are written by hand and are not regenerated. `TestAccuracy` in
`metrics_test.go` measures the method against them and against the acceptance
table, so the effect of a change is read from the metrics it prints.
