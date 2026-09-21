# Open defects in unstructured feed detection

Record of the defects found in `unstructured_feed.go` that are still open. It
states what was measured, what was decided, and what was rejected, so the
decisions do not have to be made again from scratch.

The finished work is in `DISCUSSION_ARCHIVE.md`: defect 1 (the URL is taken from
the wrong link), defect 2 (a page section is detected as a post list), defect 4
(a group of site navigation links is detected as a post list), the three
accepted changes with the measured effect, and the rejected proposal to pair
titles and URLs by slug matching.

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
- ycombinator.com-blog, 3 posts: the image sits outside the item node, in the
  part of the card that the nesting rejection removed from the detected list.

## Defect 5: the nesting rejection splits a card into its parts

ycombinator.com-blog renders its three newest posts as one card each, and each
card holds a group of its own inside it. The nesting rejection of defect 2 drops
the list of the three cards and keeps the smaller groups found inside them,
which costs the page:

- 1 real post, `/blog/chris-golda-and-grey-baker-general-partners`, so url
  recall stays at 0.90;
- the image and the timestamp of the other three, because the inner group does
  not contain them, so image recall is 0.00 and timestamp recall is 0.67;
- 2 junk entries, `/blog/tag/yc-news` and `/blog/author/garry`, which are items
  of the inner group. They hold 93 bytes of text on average, so the menu
  rejection does not reach them.

github.blog loses one post to the same rule, the hero item that sits alone in
its section. The rule is still right on the page it was written for. What is
missing is a way to prefer the outer list when the outer list is the card and
the inner group is part of the card.

## Defect 6: the title of an item that has no heading

diggersfactory.com is the clearest case. Every one of its 9 posts is matched at
level one, and every title is wrong, because the card has no heading and the
title falls back to the text of the link, which is the whole card:

```
got  "Electronic Hoe Weekend Only Fire ¥5,690"
want "Hoe Weekend"
```

The genre, the artist and the price are separate elements inside the same link.
The same fallback is what daily.bandcamp.com, cursor.com, developers.openai.com
and deepmind.google research publications fail on, all at 0.00 title precision,
so this is the largest remaining loss across the 25 pages. It has not been
investigated yet.

## Plan

1. Rank the images of an item with `alt+area` instead of taking the first one,
   and read `srcset` when `src` and `data-src` are empty. This is defect 3.
2. Investigate defect 6, the title of an item that has no heading. It is the
   largest remaining loss: five pages report 0.00 title precision.
3. Investigate defect 5, the nesting rejection splitting a card into its parts.
   It costs two pages one post each, plus the fields of three more.

If a future page needs slug matching, add it as a check on the chosen pair,
using the normalization and the exact segment rule written down in
`DISCUSSION_ARCHIVE.md`.

The fixtures are written by hand and are not regenerated. `TestAccuracy` in
`metrics_test.go` measures the method against them and against the acceptance
table, so the effect of a change is read from the metrics it prints.
