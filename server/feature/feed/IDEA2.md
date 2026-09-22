# Convert web pages into RSS feeds

An overview of the workflow would looks like this:

1. users make a request to GET /feeds/search with a URL of the page they want to
   observe.
2. the server samples chunks of elements that look like feed posts from the
   page, and return them to the users as a part of feed candidate.
3. users select which chunks should be handled as posts, which image is the
   thumbnail, which one is the title, etc...
4. users subscribe to the page via PUT /feeds with that URL and post selectors
   that is constructed from choices they made in the previous step.
5. In a feed polling, the server fetches the page, uses the selector to extract
   posts. The subsequent process is the same as RSS/Atom feeds.

## What a group is

A group is a list of post candidates, that is, a list of DOM nodes, whose
attributes such as the title and the link can all be extracted by the same
selector, plus rules where a selector alone is not enough. "Rules" is left open
on purpose: it covers the extra scraping steps that make extraction robust, for
example reading `datetime` instead of the text of a `<time>` element, or
matching the date pattern inside a text node that also holds other text.

Two consequences follow from this definition.

- **A group is defined by extraction, not by markup.** Two nodes belong to the
  same group when one selector set reaches the wanted value in both of them.
  Sharing a tag name or a parent is not the point; it is only evidence.
- **The same node may belong to several groups.** A node that can be extracted
  in two different ways produces two candidates, and the user decides which one
  is correct. Nothing forces a node into exactly one group.

The signature grouping described in "How the groups are produced" is an
approximation of this definition, not the definition itself. It is cheap and it
is usually right, but it is not exact in either direction: items with the same
signature can still need different selectors, which is what the measurement in
"One selector does not reach every item of a group" shows, and items with
different signatures can still share one selector. Where the approximation
fails, the cost is paid by the user, who unchecks a group or picks a different
row, so it does not break the recall requirement.

## What this changes in the detector

Today `DetectPostLists` decides. It picks one group out of many, picks one link
out of the item, picks one image out of several, and guesses the title. Every
open defect in `DISCUSSION.md` is a case where the guess is wrong.

Under this design the server stops deciding and starts enumerating. The user
makes the choice the server used to make. The consequence for the two metrics is
not symmetric, which is why this design is worth the extra screen:

- **Recall must be 1.00.** A post the server never reports as a candidate can
  never be recovered by the user. This is the only hard requirement.
- **Precision may be low.** A group that is not a post list costs the user one
  unchecked checkbox. A wrong title candidate costs one tap on a different row.

Four rules in `unstructured_feed.go` exist only to raise precision, and each one
of them costs recall. They have to change:

| rule today                          | under this design                                 |
| ----------------------------------- | ------------------------------------------------- |
| `minMembers = 3`                    | becomes 1. A hero post with no sibling is a group |
| `unionFraction = 0.25` score cut    | removed. No group is dropped for scoring low      |
| nesting rejection, menu rejection   | removed                                           |
| `titleAndURL`, `image`, `timestamp` | kept, as the default selection, not the only one  |

The scoring goes with the cut. A score exists to decide what to keep, and this
design keeps everything, so there is nothing left for it to decide. No cutting
algorithm can guarantee recall 1.00, so no cutting algorithm can be used here;
that work is delegated to the user instead.

Ordering by score would also be wrong. The user reads this screen next to the
page they just looked at, so the groups are listed in document order, which is
the order the page itself shows them.

## Measurements

Measured on the 25 saved pages in `testdata/`, against the hand written
fixtures, with `minMembers = 1`, no cut, no nesting or menu rejection, no `*`
fallback group, and the groups in document order.

**Candidate recall is 1.00 on every page.** All 1070 fixture posts appear in at
least one enumerated group. The requirement above is reachable; nothing in the
grouping stage has to be invented for it.

The cost is the number of groups. Raw enumeration produces up to 968 groups on
one page, which no user can review. Two reductions remove most of them, and
neither can lose a URL, because a removed group's URLs are all still held by the
group that replaced it:

1. Drop a group whose URL set is identical to another group's. Keep the one that
   comes first in the document.
2. Drop a group whose URL set is a strict subset of another group's.

| page                                  | posts | raw groups | after 1 | after 2 | groups to check | last group |
| ------------------------------------- | ----- | ---------- | ------- | ------- | --------------- | ---------- |
| developer.apple.com-news              | 108   | 968        | 295     | 95      | 1               | 1          |
| deepmind.google-research-publications | 30    | 41         | 33      | 2       | 1               | 1          |
| diggersfactory.com-vinyl-shop-new-ins | 9     | 18         | 10      | 1       | 1               | 1          |
| go.dev-blog                           | 10    | 25         | 13      | 3       | 1               | 2          |
| aws.amazon.com-jp-blogs-news          | 10    | 148        | 31      | 13      | 2               | 2          |
| daily.bandcamp.com-album-of-the-day   | 30    | 155        | 47      | 11      | 1               | 3          |
| daily.bandcamp.com-features           | 30    | 157        | 48      | 12      | 1               | 3          |
| flutter.dev-blog                      | 284   | 12         | 4       | 3       | 2               | 3          |
| paulgraham.com-articles               | 235   | 729        | 241     | 3       | 2               | 3          |
| deepmind.google-blog                  | 25    | 180        | 29      | 4       | 3               | 3          |
| blog.google                           | 8     | 50         | 10      | 4       | 2               | 4          |
| claude.com-blog                       | 24    | 250        | 35      | 5       | 3               | 4          |
| ycombinator.com-blog                  | 10    | 114        | 23      | 11      | 3               | 5          |
| anthropic.com-news                    | 13    | 37         | 17      | 5       | 4               | 5          |
| github.blog-ai-and-ml                 | 19    | 209        | 43      | 21      | 3               | 9          |
| ycombinator.com-blog-tag-essay        | 10    | 110        | 30      | 15      | 3               | 9          |
| qiita.com                             | 30    | 569        | 206     | 76      | 2               | 10         |
| cursor.com-blog                       | 28    | 201        | 43      | 13      | 7               | 12         |
| github.blog                           | 25    | 372        | 76      | 37      | 6               | 32         |
| bbc.com                               | 103   | 507        | 165     | 40      | 27              | 39         |
| developers.openai.com-blog            | 29    | 888        | 577     | 135     | 2               | 135        |

"groups to check" is how many groups the user has to tick before every fixture
post is included. "last group" is the position of the last of them in the
document order list, which is how far the user has to scroll. The second number
is the one that decides whether the screen is usable.

On 14 of the 21 pages that hold posts the user is finished within the first five
groups. There are three exceptions, and they are not the same kind of problem:

- **bbc.com, 27 groups over the first 39.** It is a news portal that renders
  about 26 separate sections, so a large number is correct there.
- **github.blog, 6 groups over the first 32.** The page puts several small
  repeated blocks above its post list.
- **developers.openai.com, the last needed group at position 135 of 135.** This
  one is a defect in the cleaning step, not in the ordering. The page carries
  `<div id="drawer" class="... lg:hidden">`, the navigation drawer shown only on
  a narrow screen, and that drawer holds 1490 elements while the real content
  holds 242. It comes first in the document, so it produces almost every group
  ahead of the post list. `cleanup` cannot see it today: it carries no `hidden`
  attribute, no `aria-hidden`, no `role`, no `nav` tag, and no class named
  exactly `nav` or `footer`. See `hidden` in `unstructured_feed.go`.

The junk the user has to leave unticked is small. Counting the URLs held by the
checked groups that the fixture does not hold: 0 on 15 pages, 1 on 4 pages, 3 on
developers.openai.com and 4 on bbc.com.

### The attributes exist as elements

For every fixture post, taking the item node from a group that holds the post,
and searching the item's own subtree for an element whose text equals the
fixture value exactly:

| page                            | title   | image   | timestamp |
| ------------------------------- | ------- | ------- | --------- |
| bbc.com                         | 102/103 | 61/61   | 15/15     |
| cursor.com-blog                 | 28/28   | 4/6     | 24/24     |
| flutter.dev-blog                | 284/284 | 273/273 | 282/284   |
| daily.bandcamp.com (both pages) | 30/30   | 30/30   | 0/30      |
| the 17 other pages              | all     | all     | all       |

So the design is sound at the item level: the value the user wants is almost
always a single element that can be addressed by a selector. The three gaps have
separate causes:

- cursor.com, 2 images: the `<img>` carries `srcset` and no `src`. This is the
  second half of defect 3 in `DISCUSSION.md` and is unchanged by this design.
- daily.bandcamp.com, all 30 timestamps: the date is a bare text node inside a
  div that also holds the category link, so no selector can isolate it. See "How
  to parse unstructured timestamps?" below.
- bbc.com, 1 title, and flutter.dev, 2 timestamps: individual pages, not
  investigated.

### One selector does not reach every item of a group

For each group the user has to check, the share of its posts whose wanted value
sits at the single relative selector with the widest coverage in that group. The
selectors are built from tag names and `:nth-of-type` only, as the Notes below
require:

| page                 | link      | title     |
| -------------------- | --------- | --------- |
| bbc.com              | 103/103   | 89/103    |
| blog.google          | 7/8       | 5/5       |
| claude.com-blog      | 23/24     | 23/24     |
| cursor.com-blog      | 28/28     | 26/28     |
| github.blog          | 23/25     | 23/25     |
| ycombinator.com-blog | 9/10      | 9/10      |
| the 15 other pages   | all       | all       |
| **total**            | 1065/1070 | 1047/1067 |

The title total is 1067 rather than 1070 because three fixture posts carry no
title. So one selector per group costs 5 links and 20 titles over the whole
corpus, and 14 of those 20 titles are on bbc.com alone. Two causes were
investigated:

1. **An optional element above the target shifts the index.** On blog.google
   `div.uni-hero-carousel__authors` is present on 4 of the 5 slides, which moves
   the block holding the link from `div:nth-of-type(3)` to `div:nth-of-type(2)`
   on the fifth. That is the 1 lost link there.
2. **The same role uses a different tag.** On github.blog the featured card
   writes its title in `<h2>` while the other four use `<h3>`.

The remaining losses, on bbc.com, claude.com, cursor.com and ycombinator.com,
were not investigated one by one.

Removing the `*` group helps here, which is why the numbers are better than they
were when it was still in the design. flutter.dev used to lose 11 titles,
because its list cards have no image block and its featured cards do, so the
title sits at `div > h3` in one variant and at `div:nth-of-type(2) > h3` in the
other. The two variants have different signatures, so once `*` no longer merges
them they become two groups, each with a selector that reaches all of its own
items.

The link is the attribute where this matters, because a post without a URL is
not a post: those 5 links are 5 posts that exist on the screen at subscription
time and are missing at polling time. A missing title only leaves a field empty.
See the open question "What is saved: a selector or a rule?".

## Step 1

Users use the same flow as today's feed subscription: tap the + button in the
feed list page, and enter the URL.

## Step 2

Given the url, the server fetch the resource. If it's a RSS/Atom feed, it falls
into the existing flow. If it's an HTML, the server searches for a link to the
feed in the document header:

```
    <link
      rel="alternate"
      type="application/rss+xml"
      title="RSS"
      href="feed://developer.apple.com/news/rss/news.rss"
    />
```

If it finds it, this case also falls into the existing flow.

Otherwise, the server extracts chunks of post-like DOM nodes with candidate
attributes for the link, title, description, timestamp, and the thumbnail. The
results are returned as part of the response of GET /feeds/search, which would
look something like this:

```json
{
  "feeds": [
    {
      "url": "https://claude.com/blog",
      "site_url": "https://claude.com/blog",
      "title": "Blog | Claude by Anthropic",
      "candidates": [
        // new field. an empty list if the RSS/Atom xml is found.
        {
          "id": 0, // an integer that is unique among the same site
          "selector": "", // selector to the parent group of post nodes
          "count": 12, // number of posts in the group
          "posts": [
            {
              "texts": [
                // text elements in the chunk
                {
                  "selector": "", // selector to this element, relative to the group
                  "value": "Claude Cowork and chat are now one Claude"
                },
                {
                  "selector": "...",
                  "value": "Read more"
                },
                {
                  "selector": "...",
                  "value": "September 16, 2026"
                }
              ],
              "images": [
                // image elements in the chunk
                {
                  "selector": "...",
                  "src": "https://...",
                  "alt": "..." // if any
                }
              ],
              "links": [
                // link elements in the chunk
                {
                  "selector": "...",
                  "value": "https://" // the href value
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

The candidates is a list of groups as defined in "What a group is" above, each
holding one or more post-like chunks. Each group has candidates for the post's
attributes, for example, the texts it a list of possible texts that may be the
title, description, or the timestamp. Once the ambiguity is resolved, the server
can query posts by constructed selectors.

### How the groups are produced

The existing grouping stage is reused as the approximation described in "What a
group is": walk every element and group its element children by subtree
signature at `sigDepth = 2`. Four things change.

1. `minMembers` becomes 1. A group of one item is a candidate.
2. The score cut, the nesting rejection and the menu rejection are removed, and
   so is the score itself.
3. The `*` fallback group is removed. Today, a container whose children split
   into several signatures also produces one group holding all of them, so that
   the detector can still return a single list. That is no longer needed: the
   container produces one group per signature, and the user ticks the ones that
   are post lists. A node does not have to belong to exactly one group.
4. The result is reduced by the two lossless rules above (identical URL set,
   subset URL set) and returned in document order.

### How many groups are returned

All of them. This step enumerates and reduces, and then stops. It does not cap
the list, does not truncate it and does not reorder it, even though the reduced
count still reaches 135 on developers.openai.com and 95 on developer.apple.com.

Cutting the list is the job of the consumers that come after this step: the API
handler that pages the response, and the client that decides how many groups to
show at once. Only they know how many rows fit on the screen the user is looking
at, and only they can let the user ask for more. A cut made here would be
invisible to both, and on a page such as bbc.com, where the needed groups are
spread over the first 39, it would silently break the recall requirement.

Nothing in the result is preselected either. Which group is right is the user's
answer to give, and this step must not encode a guess that the user then has to
undo.

## Step 3

On the client side, users need to resolve the ambiguity before making a
subscription. The subscribe button navigates the users to a new screen, where
candidate chunks and attributes are shown and let users select correct ones
step-by-step.

In the first step, users select a listed group, which displays one or two
previews of the candidate posts, including all possible images and texts:

```
Group 1         [ ]

[img1], [igm2]
Claude Cowork and chat are now one Claude
Read more
September 16, 2026
-------------------
[img1], [igm2]
Claude in Chrome is generally available
Read more
Aug 10, 2026

--------------------
Group 2          [x]

...

[Subscribe]
```

Users can mark correct groups (x). Tapping a group navigate the user to the
second step, in which they are asked to select a correct title:

```
Which group contains the titles?

[x]
Claude Cowork and chat are now one Claude
Claude in Chrome is generally available
---
[ ]
Read more
Read more
---
[ ]
September 16, 2026
Aug 10, 2026
---

[None of them]
[Next]
```

They can claim there's no correct candidate in the list (except for the links).

The same flow follows. Candidate groups for an attribute are displayed, users
select one, go to next step, and so on. Selected candidates are filtered out in
subsequent steps so that users don't select the same element for multiple
attributes (claim the same text for both the title and description).

When everything finishes, the app brings the users back to the first step page,
where the user's selections are reflected on the group preview card (display
only selected ones:

```
Group 1         [ ]

[img2]
Claude Cowork and chat are now one Claude
September 16, 2026
-------------------
[img2]
Claude in Chrome is generally available
Aug 10, 2026

--------------------
Group 2          [x]
...

[Subscribe]
```

Toggling a group checkbox doesn't clear the candidate selections; which groups
to include and which element to include are two seprate things. Tapping a group
navigates the user to the second step page again, in which previous selections
are remembered.

### What a row in the attribute screen is

A row is one relative selector, not one value. The rows are built by grouping
the `texts` of every post in the group by their `selector`, so the row shows the
value that selector produces in each of the first few posts. That is what makes
the screen readable: an item with 7 links produces at most 7 rows for the whole
group, not 7 rows per post.

Rows where the selector reaches only part of the group's items still have to be
shown, because that is the case on bbc.com, github.blog and four other pages.
The row should say so, for example "reaches 89 of 103 posts", which is the real
bbc.com figure, since that number is the user's warning that the choice is not
reliable.

## Step 4

the users then hit the subscribe button to subscribe to the page via PUT /feeds,
with group selectors constructed from the candidate selections:

```json
{
  "url": "https://claude.com/blog",
  "selectors": [
    // new field. omit when posting an rss feed link
    {
      "root": "...", // selector to the group's root
      "title": "...", // relative selector to the post title, if any
      "description": "...", // relative selector to the post description, if any
      "image": "...", // image selector, if any
      "link": "..." // post link selector (required)
    }
  ]
}
```

The server fetches the url and determine if it's an RSS/Atom feed in the same
way as step 2, so the two requests should end up in the same result. Save the
feed and selectors in the DB, then wait for the next polling.

### What the server validates before saving

The request is accepted only if the selectors still produce posts on the page
the server just fetched. The server runs the extraction once and returns the
resulting posts in the response, so the client can show what was subscribed to.
A `root` that matches nothing, or a `link` that matches nothing in any item, is
rejected: saving it would create a feed that is silently empty forever.

The count is not compared against what the client saw. The page may have changed
between the two requests, and that is normal.

## Step 5

When polling the page, fetch the HTML and extract posts using the saved
selectors. The posts are then processed just like usual items from RSS/Atom
feed: being saved as entries, and converted to stories if needed.

### Extraction rules

For each saved selector set, in order:

1. Match `root` against the sanitized document. Each match is one item.
2. For each item, apply `link` relative to it. An item with no link is skipped,
   not reported as a post without a URL.
3. Apply `title`, `description`, `image` and `timestamp` relative to the item.
   Each is optional: an item where the selector matches nothing keeps the field
   empty rather than being dropped.
4. Resolve every URL against the page URL, and drop a post whose URL points back
   at the page itself.
5. Deduplicate by URL across all the selector sets of one feed, keeping the
   first. A page that renders the same post as a grid and as a list would
   otherwise produce it twice.

The post identity is the URL, as it is for RSS. A post whose title or image
changes between polls is the same post.

### When the page changes

A site rebuild changes the DOM and the saved selectors stop matching. The
polling job should record, per feed, the number of items the last poll produced.
Two cases need different handling:

- `root` matches nothing, or produces no post with a link: the feed is broken.
  The user has to be told, and sent back through step 3 for the same URL.
- `root` still matches but an optional attribute selector stopped matching: the
  feed keeps working with that field empty. This is not worth interrupting the
  user for.

---

## Notes

- The selectors are for sanitized HTML, not for raw HTML.
- A group with only one post is valid. Think of a hero post that is displayed at
  the top of the blog site with a large thumbnail, having no sibling.
- The selectors must not use class names. The saved pages identify their cards
  by hashed CSS module names or by Tailwind utility classes, and both change
  whenever the site is rebuilt. This is the same reason `PostList.Selector` is
  built from tag names and positions only today.
- Four of the open defects in `DISCUSSION.md` stop being defects under this
  design, because the user resolves them: defect 3 (which of several images is
  the post image), defect 6 (the title of an item that has no heading), defect 7
  (a post that stands alone, fixed by `minMembers = 1`), and defect 8 (a row of
  off-site link cards, which the user simply leaves unchecked). The half of
  defect 3 that is about `srcset` is not resolved: if `image()` reads no source
  at all, the image never appears as a candidate and the user cannot select it.

## Open questions

### What is saved: a selector or a rule?

This is the question the design depends on. A positional relative selector does
not reach every item of a group, and the link is the attribute where that turns
into lost posts: 5 of the 1070 posts, on blog.google, claude.com, github.blog
and ycombinator.com. Today's heuristic reaches all of them, because it does not
use a position at all.

Three options:

1. **Save the selector only.** Simple, and it is what steps 2 to 4 above
   describe. It loses those 5 posts and 20 titles at polling time.
2. **Save the selector with a fallback.** If the selector matches nothing in an
   item, fall back to today's heuristic for that attribute: the first link for
   the URL, the heading for the title, and so on. Keeps recall, and the user's
   choice still decides the common case.
3. **Save a rule name instead of a selector**, for example `first-link`,
   `heading`, `largest-image`, with the selector used only when the user picks a
   row that no rule describes. The most robust across site rebuilds, and the
   largest change to the API shape above.

Option 2 looks like the smallest change that keeps the recall requirement, but
none of the three has been measured end to end.

### How to parse unstructured timestamps?

TBD. The publish date is optional, but having it is better than missing if
visual timestamp exists.

Two things are now measured and constrain the answer:

- The date is not always its own element. On both daily.bandcamp.com pages it is
  a bare text node next to a category link, so all 30 timestamps are unreachable
  by a selector. The date regex in `timestamp()` reads them correctly today.
- Therefore the timestamp candidate cannot be "an element" alone. It needs an
  extraction mode: take the whole text of the selected element, or take the part
  of it that the date pattern matches. The step 3 screen can offer the extracted
  date as its own row, with the containing element as the selector.

Parsing the extracted text into a real date is a separate problem and is still
open. `Post.Timestamp` keeps the text as the page writes it for this reason.

### Is one selector set per group enough?

The API above saves one attribute selector per group. The measurements say one
link selector reaches every post on 17 of the 21 pages, and one title selector
on 16 of them. On the rest a minority of the items put the title or the link
somewhere else. The answer depends on the question above: with option 2 or 3 one
set is enough, with option 1 the client would have to save several.

### What does the accuracy test measure now?

`metrics_test.go` measures a detector that decides. Under this design the
fixtures should be measured in two separate ways, and the acceptance table has
to be rewritten:

- **Enumeration recall**, which must be 1.00 on every page. This is a hard
  failure.
- **Review cost**, which is how much of the returned list the user has to read.
  It is a cost in taps and scrolling, not a failure, so the acceptance table
  becomes a record of how much work each page costs rather than a pass or fail
  line.
- **Attribute accuracy**, which is today's title, image and timestamp precision.
  It no longer decides what the user sees, but it still says how good the
  extraction is once a selector has been chosen.

The first step is already taken: `TopListPrecision` is gone from
`metrics_test.go`, because it measured the ranking and there is no ranking any
more, and `UsefulGroups` replaces it. `UsefulGroups` is the share of the
returned lists that hold at least one fixture post, so it is the review cost
metric: the user reads every returned list, and this says how many of those
reads lead somewhere. It sits at 1.00 on 19 of the 21 pages today, 0.94 on
bbc.com and 0.86 on github.blog, because the score cut is still in place and
removes the useless lists before they are returned. Removing the cut is what
will make this number drop and make it worth watching.

The existing fixtures are usable for all of this without being rewritten.
