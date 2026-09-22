# Server plan: subscribing to a web page that has no feed

## Context

`IDEA2.md` describes the design and the measurements behind it: the server stops
deciding which elements of a page are posts and starts enumerating them, and the
user resolves the ambiguity in the app. The API contract for that flow is
already merged, and the client plan is written against it, so the shape of the
request and the response is fixed and is not a decision this plan makes.

- Design and measurements:
  file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/IDEA2.md
- What this plan leaves out:
  file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/FUTURE_WORK.md
- Open defects of today's detector:
  file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/DISCUSSION.md
- API paths: file:///Users/fujidaiti/Dev/paperdoll/api/paths/feeds.yaml
- API schemas: file:///Users/fujidaiti/Dev/paperdoll/api/components/schemas.yaml
- Client plan:
  file:///Users/fujidaiti/Dev/paperdoll/client/docs/web-page-subscription.md

What exists today:

- `DetectPostLists` in `unstructured_feed.go` returns one ranked and cut list of
  post lists, with the title, the image and the timestamp already chosen. It is
  called from the metrics test only. Nothing in the API uses it.
- `Service.SearchFeeds` parses the fetched body with `gofeed` and fails when the
  URL is a plain HTML page. There is no feed discovery from
  `<link rel="alternate">`.
- `Service.Subscribe` ignores selectors, stores no posts, and saves nothing but
  the feed row and the subscription row.
- `Job.Do` in `polling_job.go` polls with `gofeed` only.

## Goal

A user can subscribe to an HTML page that publishes no feed, and the page then
behaves like any other feed.

Acceptance criteria:

1. `GET /feeds/search` with an RSS/Atom URL answers exactly as it does today,
   with an empty `post_groups`.
2. `GET /feeds/search` with an HTML page that links to a feed in its head
   answers with that feed, and with an empty `post_groups`.
3. `GET /feeds/search` with a plain HTML page answers with `post_groups` holding
   every enumerated group in document order, uncapped.
4. Enumeration recall is 1.00 on all 25 saved pages: every fixture post appears
   in at least one returned group. This is the one hard requirement.
5. `PUT /feeds` with `selectors` validates them against the page it fetches,
   answers 400 when a `root` matches nothing or a `link` matches nothing in any
   item, and otherwise saves the feed, the selectors and the extracted posts.
6. `GET /feeds/{id}/timeline` answers with those posts before the first poll
   runs.
7. A poll of a page feed reads the saved selectors, extracts the posts and
   stores the new ones, using the same insert and story path as an RSS feed.

## Approach

The work splits into three layers that can be built in parallel.

- **An enumeration layer.** `DetectPostLists` is replaced by
  `EnumeratePostGroups`, which returns every group and, for every post of a
  group, the candidate links, texts and images it carries. Nothing is scored,
  cut, rejected or preselected. This is the shape step 2 of `IDEA2.md`
  describes, and it is also the shape the extraction needs.
- **A selector layer.** A selector is a key, not a CSS selector. The enumerator
  writes it, and the only operation ever performed on it is string equality
  against the keys of a later enumeration run. Nothing compiles it and nothing
  matches it against a DOM tree, which removes two problems that a CSS engine
  would bring:
  - `cascadia.Compile(":scope > div")` fails with "unknown pseudoclass or
    pseudoelement :scope", and `:scope` is part of the agreed API.
  - `goquery`'s `Find` matches descendants at any depth, so `div > h3` applied
    to an item returns both the card heading and a heading nested deeper in the
    card. Measured on a two-heading item: 2 matches where the path means 1.
- **A storage and polling layer.** The selectors are rows attached to the feed,
  and a feed that has rows is polled by enumerating the page again and reading
  the attributes the saved keys name.

The two API handlers are thin: they call the layers above and translate to JSON.
The per-selector shape the API returns is produced in that translation, not by
the enumerator.

## What a selector is

A selector names one attribute of one post inside one group. It is written by
the enumerator and read back by looking it up in the output of another
enumeration run of the same page.

- `root` is the path from the document root to the items of the group, for
  example `html > body > main > section:nth-of-type(2) > div > article`.
- `link`, `title`, `description`, `image` and `timestamp` are paths from one
  item down to an element inside it, for example `div > h3`. The literal
  `:scope` means the item element itself, which happens when the whole card is a
  link.
- A step is a tag name, optionally followed by `:nth-of-type(n)`. The position
  is written only when the parent holds more than one element of that tag, which
  is the rule `selector()` already follows.
- One container can hold two shapes of the same tag, for example a featured card
  and a plain one, and the path above is built from tag names only, so the two
  groups would write the same key. The second and later of them therefore carry
  `:nth-shape(n)`, which is not a CSS pseudo-class. It exists so that a saved
  key names exactly one group; without it the extraction would answer with the
  items of whichever group came first. This happens on 7 of the 25 saved pages,
  12 times on developers.openai.com.
- Class names are never used, for the reason given in `IDEA2.md`: the saved
  pages name their cards with hashed CSS module names or Tailwind utilities, and
  both change on every rebuild.

The format reads as a CSS selector on purpose, because the client shows it and
because it stays readable in the database, but the server never treats it as
one. What the server needs instead is that two runs over the same page write the
same string for the same element. Two things follow:

- One function writes every key, and the same `sanitize` runs at search time, at
  subscribe time and at poll time.
- A saved key has no meaning outside a run of the enumerator. When the page is
  rebuilt and the key is no longer written, the attribute is simply absent, in
  the same way a CSS selector that stopped matching would be.

## Work breakdown

Tasks 1, 3 and 5 have no dependency on each other and can start at the same
time. Tasks 1 and 2 both change `unstructured_feed.go`, so one person should
take them together; task 1 is listed on its own because it is what every other
task depends on.

**1. Selector keys, in `unstructured_feed.go`.**

```go
// itemSelector writes the key of one element inside a post item: the path of
// tag steps from the item down to it, joined by " > ". The item itself is
// ":scope".
func itemSelector(item, el *html.Node) string

// rootSelector writes the key of a group: the path from the document root to
// its items. This is today's selector(), renamed.
func rootSelector(container *html.Node, sig string) string
```

A step carries `:nth-of-type(n)` only when the parent holds more than one
element of that tag, which is what makes two items of the same shape produce the
same key. There is no parser and no matcher, because nothing ever reads a key
back into a structure: reading an attribute is a string comparison in task 6.

**2. Enumeration, replacing the decision code in `unstructured_feed.go`.**

```go
// Group is one list of post-like items in a page.
type Group struct {
	Selector string // key of the group, from rootSelector
	Posts    []PostCandidate
}

// PostCandidate is one item of a group with everything that can be read out of
// it. The attributes are enumerated per post, not per group: a group is a list
// of posts, and two posts of one group may carry different attributes.
type PostCandidate struct {
	Node   *html.Node
	Links  []Attribute
	Texts  []Attribute
	Images []Attribute
}

// Attribute is one value found in a post, with the key that names it.
type Attribute struct {
	Selector string
	Value    string
	Alt      string // images only
}

func EnumeratePostGroups(r io.Reader, page url.URL) ([]Group, error)
```

Grouping keeps today's signature walk at `sigDepth = 2` and changes five things,
all of them listed in "What this changes in the detector" in `IDEA2.md`:
`minMembers` becomes 1, the `unionFraction` cut and the score go away, the
nesting and menu rejections go away, the `*` fallback group goes away, and the
result is reduced by the two lossless rules (drop a group whose URL set is
identical to another's, drop a group whose URL set is a strict subset of
another's) and returned in document order.

The attributes of one post, in document order inside the item:

- **Links**: every element carrying an accepted `href`, using today's `linkOf`,
  so self links and non-http targets stay out. `Value` is the URL resolved
  against the page URL.
- **Texts**: every element whose own text is not empty and which has no element
  child carrying text. Taking only these leaf blocks stops one heading from
  producing an entry for the heading, for its link and for the card around it.
  `Value` is the element text collapsed to single spaces, except for a `<time>`
  element carrying `datetime`, where it is the attribute value. That exception
  is what makes a timestamp key usable at poll time without a second extraction
  mode.
- **Images**: every `<img>`, reading `src` and then `data-src` as `image()` does
  today. `Value` is the resolved URL and `Alt` is the `alt` attribute.

`DetectPostLists`, `PostList` and the score constants are deleted with this
task. `titleAndURL`, `timestamp`, `image` and `heading` stay, because the
metrics still use them.

**3. Feed discovery, in `feed.go`.**

Split `fetchFeed` into a fetch and a parse step, and add the branch the flow
needs:

1. Fetch the URL once.
2. If `gofeed` parses the body, it is a feed. This is today's path.
3. Otherwise parse it as HTML and look for
   `<link rel="alternate" type="application/rss+xml">` or `atom+xml` in the
   head. Rewrite a `feed://` href to `https://`, resolve it against the page
   URL, fetch it and go back to step 2.
4. Otherwise it is a plain page.

The same three-step decision runs at subscribe time, so put it in one function
that both handlers call.

**4. `GET /feeds/search`, in `feed.go` and `api/server.go`.**

`SearchFeeds` returns the feed attributes plus the groups. On a plain page the
feed attributes come from the page itself: `title` from `<title>`, `site_url`
from the requested URL, and no description.

The API answers per selector, while the enumerator answers per post, so the
handler does the transform. For one group:

1. Choose the sample. Start with the first `sampleSize` posts, then walk the
   remaining posts in order and add any post that carries a key none of the
   already chosen posts carries. The sample keeps the document order of the
   posts, and `sampled` is its size.
2. For each attribute kind, collect the keys in the order they first appear over
   all the posts, not only the sampled ones. This is the row order the client
   renders.
3. `values` holds one entry per sampled post, in sample order: the value that
   post carries under the key, or an empty entry when it carries none.
4. `count` is the number of posts and `id` is the position of the group in the
   list.

`sampleSize` is a constant, 3.

Step 1 is what makes every row usable. A group holds posts that carry a
description and posts that do not, because the signature grouping compares two
levels of the subtree only. If the sample were the first three posts alone, a
key that only the eighth post carries would produce a row of empty values, and
the user could not tell what it is or whether to pick it. The rule costs a
larger sample on a group whose posts differ a lot, which is the second thing to
measure in the size measurement under "Open points".

Put the transform in `api/server.go` next to the other schema translation, and
add `post_groups` to `feedAttrsSchema`. The list is returned whole, in document
order.

**5. Storage, in a new migration.**

```sql
CREATE TABLE feed_post_selectors (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    feed_id bigint NOT NULL REFERENCES feeds (id),
    root text NOT NULL,
    link text NOT NULL,
    title text,
    description text,
    image text,
    published_at text
);
```

The rows are inserted in the order the request lists them and read back with
`ORDER BY id`, which is the only ordering the extraction needs: it decides which
set wins when two of them produce the same post URL.

`published_at` rather than `timestamp`, because `timestamp` is a type name in
Postgres and reads badly as a column. A feed that has rows here is a page feed;
no column on `feeds` is needed to say so.

**6. `PUT /feeds` with selectors, in `feed.go` and `api/server.go`.**

```go
// Selectors is one saved set of keys for reading posts out of a page.
type Selectors struct {
	Root, Link, Title, Description, Image, Timestamp string
}

// ExtractPosts enumerates the page and reads the posts the key sets name.
func ExtractPosts(r io.Reader, page url.URL, sets []Selectors) ([]Post, error)
```

`ExtractPosts` runs `EnumeratePostGroups` on the page and then looks the keys up
in its result. It never touches the DOM tree itself. For each set, in the order
the sets are given:

1. Find the group whose `Selector` equals `Root`. When there is none, the set
   produces no post.
2. For each post of that group, find the entry of `Links` whose `Selector`
   equals `Link`. A post that carries no such entry is skipped, because a post
   without a URL is not a post.
3. Read `Title`, `Description` and `Timestamp` from `Texts` and `Image` from
   `Images` the same way. Each is optional, and a post that carries no entry
   under that key keeps the field empty.
4. An empty key means the user chose nothing for that attribute, which is not
   the same as a key that is absent from the post.
5. Deduplicate by URL across all the sets, first wins.

Nothing here resolves a URL or drops a self link: the enumerator already
resolved every link against the page URL, and `linkOf` already dropped self
links and non-http targets. Nothing here reads a `<time>` element either,
because the enumerator already wrote the `datetime` value as the text value.

`Subscribe` takes the selectors as a new argument. When the list is empty it
behaves as it does today. When it is not:

1. Run the discovery of task 3. A URL that turns out to be a feed is subscribed
   to as a feed and the selectors are ignored.
2. Run `ExtractPosts`. A set whose `Root` names no group, and a set whose `Link`
   names nothing in any post of its group, is a 400 naming that set. There is no
   separate parse step and no malformed-selector case: a key the enumerator did
   not write is simply a key nothing carries.
3. Insert the feed row and the subscription row as today, insert the selector
   rows, and insert the extracted posts as feed entries with the same statement
   the polling job uses, so `GET /feeds/{id}/timeline` answers right away.
4. Answer with `Feed`, unchanged.

**7. Polling, in `polling_job.go`.**

`Job.Do` loads the selector rows of its feed first. With no rows it runs today's
`gofeed` path. With rows it fetches the page, calls `ExtractPosts`, which
enumerates the page again, and builds `entryRecord`s:

- `dedupKey` is the post URL, which is the convention `normalizeEntry` already
  falls back to.
- `title` is the extracted title, empty when the user chose none.
- `publishedAt` is the extracted timestamp parsed with `dateparse`, which is
  already an indirect dependency. When it cannot be parsed, use the time of the
  poll, so that a page feed still produces stories. An entry is inserted once,
  so this date is the moment the post was first seen and does not move on later
  polls.

Everything from the insert statement onwards is unchanged, so lift the part of
`Do` that starts at the insert into a function taking `[]entryRecord` and call
it from both branches.

`IDEA2.md` also asks for a broken-feed signal when the saved selectors stop
matching after a site rebuild. That needs a column on `feeds`, an API field and
a screen, and none of the three exists. It is out of scope here and written up
in `FUTURE_WORK.md`.

**8. Metrics, in `metrics_test.go`.**

Rewrite the acceptance table around the three measurements named in "What does
the accuracy test measure now?" in `IDEA2.md`:

- **Enumeration recall**, which must be 1.00 on every page and fails the test
  otherwise.
- **Review cost**, asserted per page like every other metric: the number of
  groups returned, the number the user has to tick to cover every fixture post,
  and the position of the last of them. A lower number is better here, so the
  hand-written value is a ceiling rather than a floor, and a page fails when it
  goes above it.

  Set every ceiling to what the method produces once the task is finished, so
  the table passes in this change. The numbers in the `IDEA2.md` measurement
  table say what to expect: 135 groups on developers.openai.com and 39 on
  bbc.com are the two bad ones. They are accepted as they are for now. Making
  them smaller is future work, and the table is what will show it happening,
  because the ceilings can then be lowered one page at a time.

- **Attribute accuracy**, which is today's title, image and timestamp measure,
  applied to the key with the widest coverage in each group.

Add one measurement that the design depends on and that nothing checks today:
for every group a user would tick, take the link key with the widest coverage
and count the posts of the group that carry it. Those are the posts that survive
to polling time. `IDEA2.md` predicts 1065 of 1070. That number is what decides
the open question "What is saved: a selector or a rule?", so it has to be
visible in the table.

## Test plan

**Unit tests, in `unstructured_feed_test.go`.** Selector keys over a small hand
written document: `:scope` for the item itself, a key with no position where the
parent holds one element of that tag, a key with `:nth-of-type(n)` where it
holds several, and the same key written for two items of one group.

**Transform tests, next to the handler.** One hand built `[]Group` turned into
the API shape: the row order follows first appearance over all the posts,
`values` holds exactly `sampled` entries, and a post that carries no value under
a key produces an empty entry rather than a shifted array. The case the sampling
rule exists for gets its own test: a group of ten posts where only the eighth
carries a description key is sampled so that the eighth post is in the sample
and the row shows its value.

**Corpus tests, in `metrics_test.go`.** The measurements of task 8, over the 25
saved pages. Enumeration recall is the only hard failure.

**Extraction tests, in `unstructured_feed_test.go`.** `ExtractPosts` against two
or three saved pages, with the key sets taken from an enumeration run of the
same page: a post that carries no link key is skipped, an optional key that no
post carries leaves the field empty, a `root` that names no group produces
nothing, and two sets covering the same post produce it once.

**Integration tests, in `itest/feed_test.go`.** These need the scraper to answer
from a fixture rather than from the network; check how `itest` serves pages
today before starting, and serve one of the saved pages from a local test
server.

1. Searching a saved page answers with groups in document order and with feed
   attributes taken from the page title.
2. Searching an RSS URL answers with no groups. This protects criterion 1.
3. Subscribing with a valid selector set writes the feed row, the subscription
   row, the selector rows and the entries, and the timeline answers with them.
4. Subscribing with a `root` that names no group answers 400 and writes nothing.
5. Subscribing with a `link` that no post carries answers 400 and writes
   nothing.
6. Polling a page feed inserts only the posts that are new since the
   subscription.

## Decisions taken, so nobody has to ask

- **The selectors belong to the feed row, not to the subscription.** Entries are
  stored per feed, so two users who pick different selectors for one URL would
  otherwise write conflicting entries under the same feed. The first subscriber
  writes the selector rows and a later subscriber's set is validated, answered
  with 200 and then dropped. The later user still gets the feed, and the feed
  still works.
- **A selector is a key into an enumeration result, not a CSS selector.** The
  server writes it and the server reads it back, so nothing has to compile it.
  Subscribing and polling then read the page through exactly the same code as
  the screen the user answered on, which removes a whole class of difference
  between what the user saw and what the feed stores.
- **Option 1 of "What is saved: a selector or a rule?"**: the selector alone,
  with no heuristic fallback. It is what the API describes, and the measurement
  added in task 8 shows what it costs before anyone builds option 2.
- **An unparsable date becomes the time of the first poll**, rather than staying
  empty. An entry with no `published_at` never becomes a story, so leaving it
  empty would mean a page feed produces nothing for the newspaper.
- **`post_groups` is not paged.** The list is returned whole, as
  `GET /feeds/search` already documents, and the client renders it lazily.
- **A group is not preselected and not ordered by quality.** Document order
  only.
- **`sampleSize` is 3.**

## What changed while building it

Three things turned out differently from the plan above. They are described in
place, and listed here so that the difference is easy to find.

- **The shape number in a group key**, described in "What a selector is". The
  plan assumed a group key is unique by construction, and it is not.
- **The reduction compares the first link of each item**, not every link of it.
  Taking every link makes the group of `<body>`, whose single item holds every
  link of the page, a superset of every other group, and the subset rule then
  reduces the whole page to that one group. Measured: 1 group per page instead
  of the 3 to 135 the table in `IDEA2.md` expects.
- **The metrics measure what the user can obtain, not what exists.** A fixture
  post counts as reached only when one link key of one group produces it, and a
  title only when one text key of the ticked group produces it. This is stricter
  than "the value is somewhere in the group" and it is what task 8 was asked
  for. Url recall is 1.00 on all 25 pages under that rule.

## Open points

- **Enumeration runs on every poll, and it is cheap enough.** `ExtractPosts`
  reads the page through `EnumeratePostGroups`, so a poll does the work of a
  search request. Measured on the largest saved pages, enumerating and building
  the whole response takes 22 to 55 ms and allocates 18 to 47 MB. A poll is a
  background job with a 30 second timeout, so this is not a problem. If a page
  ever makes it one, the fix is to enumerate only the group whose key the set
  names, which the reduction rules make less simple than it sounds.
- **The response of a large page is a few hundred kilobytes.** Measured as the
  JSON of `post_groups` alone: 704 KB on flutter.dev, 592 KB on
  developers.openai.com, 444 KB on qiita.com, 372 KB on bbc.com and 260 KB on
  developer.apple.com. Two different causes: developers.openai.com pays for its
  135 groups, and flutter.dev pays for its 284 items in 3 groups, because every
  distinct key over all the items is a row. Nothing is done about it in this
  change. The first lever is `sampleSize`, the second is the text candidate
  rule, and the third is paging the groups.
- **The sample grows to 48 items on developer.apple.com.** The rule that the
  sample covers every candidate costs little on a group whose items are alike,
  and a lot on one that mixes shapes: 3 items on every other page measured, 48
  there. It is the rule the user asked for, so it stays, but it is the reason
  that page's rows are long.
- **The drawer on developers.openai.com.** `cleanup` cannot see it, so it
  produces almost every group ahead of the post list and pushes the needed group
  to position 135. It does not break recall, only review cost. See the
  measurement table in `IDEA2.md`.
- **Timestamps that are not their own element**, on daily.bandcamp.com. The
  agreed `PostSelectors.timestamp` is a plain selector, so those 30 dates stay
  unreachable until the API carries an extraction mode.
