# Post structure extraction

Note: this file must describe only things that are independent of the post
structure extraction algorithm.

## Goals

Given a web page, it generates a set of _structures_ for posts listed in the
page. The structures can be later used to extract values users want to observe,
from the same page of different builds. They should be robust enough to absorbe
small changes that may be introduced per build, such as random class names,
absent/extra siblings.

Users can review the structures and tick them to claim which values they want to
observe. A good output is well-generalized and organized so that they won't be
overwelmed by a wall of rows to review.

## Input/Output

The algorithm receives a root node of the input HTML, and outputs a list of
structures. A structure is a tree of _parts_, and a part is a set of _fields_.

### Values

Every node of the input tree offers a set of values, which is what a user can
observe:

- one value for each text written directly inside the node, in document order.
  Text inside a child node belongs to that child, not to this one. Runs of
  whitespace are collapsed to one space and the ends are trimmed.
- one value for each attribute that can carry something a user would observe:
  `href`, `src`, `data-src`, `poster`, `srcset`, `alt`, `title`, `datetime`,
  `content` and `value`. A URL is resolved against the page URL, and a `srcset`
  is split into one value per candidate URL.

The texts of one node are kept as separate values rather than one joined string,
so that a value can always be traced back to the place it was written. A
consumer that wants one string joins them in document order itself.

### Sample

A sample is one fragment the algorithm found in the input page, such as a post
title (h3) or a link text (a). It points exactly one node of that page, and the
values it offers are the values of that node. A sample of a link text therefore
offers both the text and the href, because both are needed to make up the link
text.

A sample belongs to the input page only. It is kept in the output so that the
review screen can show the user what was actually found, and it is never applied
to a page of a later build. Extraction on a later build is done by the matchers
described below.

### Structure and part

A structure stands for the shape of one post, representing a boundary for a
chunk of related information. For example, the posts in the Today's headline
news section (see the Field section below) consist of a title and a description,
so the title field and the description field make up a generalized post that
stands for the posts in that section.

It is called a structure and not a candidate because the same object survives
the review: before the review it is one of the choices offered to the user, and
after the review it is what the polling job applies to every later build.

A structure is a **tree of parts**, because the shape of a post is often split
across containers in the markup. A part is one node of that tree. It holds a set
of fields and a **part matcher**:

- input: a DOM subtree. For the root part this is the whole page. For any other
  part it is one subtree its parent part matched.
- output: a list of nodes, each being the root of one subtree inside the input.
  The subtrees do not overlap, and they keep the order of the page.

A structure whose tree is a single part is the common case, and a structure with
only one field is valid.

### Instance

An **instance** is one subtree the root part's matcher found. It stands for one
post. An instance is a tree as well: inside it, the matcher of each child part
runs and finds subtrees, and each subtree found this way is a **part instance**.
The root of the instance tree is the part instance of the root part.

A part instance holds the values that the fields of its part read in that
subtree, and its children are the part instances of the child parts.

One part may produce several part instances inside one instance, so the instance
tree is not always a one-to-one copy of the structure tree. A part that stands
for a tag chip gives one part instance per tag.

Instances are derived rather than stored: on a later build the same matchers
find the instances of that build.

The subtrees one part matches do not have to have exactly the same shape. A part
may match post cards that differ in whether a byline is present, and a field
whose matcher finds nothing in such a subtree simply contributes nothing there.
This is so that an implementation can cover posts with optional fields with one
structure instead of one structure per shape. A part that matches exactly one
shape is equally valid. Where the markup of a region disagrees between subtrees
and no field matcher can cover it, that region is a blob.

### Field

A field stands for one attribute of a post, such as the title. It is a set of
samples that are semantically the same fragment in different instances, e.g.,
the titles of posts in the same section within the page. For example, say the
page has two sections of posts: Weekly trends and Today's headline news:

[Today's headline news]

- News title 1
  - Description 1
- News title 2
  - Description 2
- News title 3

[Weekly trends]

- News title 1
  - Description 1
- News title 4
  - Description 4
- News title 5
- News title 6
  - Description 6

In this case, there should be two title fields, one in each section. One holds
the samples of the post titles in the Today's headline news, and the other holds
the titles in the weekly trends:

- [Field1] = { sample of "News title 1", sample of "News title 2", sample of
  "News title 3" }
- [Field2] = { sample of "News title 1", sample of "News title 4", ... }

Of course there should also be a description field for each section. Users can
tick Field1 if they want to observe the post titles in the Today's headline news
section, or don't tick otherwise.

A field also has a **field matcher**, which covers all the fragments its samples
point to. This is what is used later to extract semantically the same fragment
from the same page of different builds, e.g., the updated page listing headlines
of the next week.

- input: one subtree that the part holding the field matched, not the whole
  page.
- output: at most one node inside that subtree.
- observed: the values of that node.

Applied to the page as a whole a field matcher therefore finds one node per
subtree its part matched, which is many. It is only inside a single subtree that
it finds at most one. That restriction is what makes it useful: a title that
could be anywhere on the page says nothing about which post it belongs to.

Because a field offers at most one node per subtree, a post that holds several
of the same thing, such as a list of tags, is not one field with many nodes. It
is a child part whose part matcher finds one subtree per tag. This keeps the
review step where the user can drop the tags on their own.

A field is a tick unit. The user can tick one field on its own, for example the
titles of a section without its descriptions.

### Blob

A blob is a claim about a subtree: this subtree has a semantic role, such as the
body of a post; the sibling posts hold the same role; and its shape does not
agree with the shape the siblings use for that same role.

A blob is a field, and is close to a field holding post titles. Both stand for a
chunk of marked-up text that the user ticks as one thing. The difference is that
a blob keeps the original structure of that chunk instead of reducing it to one
string, so a consumer can still tell a heading from a paragraph, and can still
reach the links and images inside it.

A blob field has a **blob matcher** in place of a field matcher:

- input: one subtree that the part holding the field matched.
- output: a run of consecutive sibling nodes under one parent inside that
  subtree, and everything below them.
- observed: the values of every node in that run, in document order.

The samples of a blob field are runs of sibling nodes rather than single nodes,
for the same reason.

A blob is a leaf of the output. No field or part describes anything inside it,
and the user ticks it as one row. The user cannot tick part of a body.

### Ticking

A part is a tick unit as well. Ticking a part ticks every field in it and every
part below it, so the user does not have to tick the title field and the
description field separately to observe whole posts.

Ticking a structure means ticking its root part, which takes the whole
structure. Users can't tick only a part below the root: ticking a part
implicitly ticks all its ancestor parts.

### Why a structure is needed

A structure is how the output meets the requirement that it says which values
belong to the same post. The fields alone do not: they say which node is a title
and which is a description, but not which title goes with which description.
Pairing them by position does not work either, because as soon as one post has
no description, the second description lines up with the third title. Note that
in the example above, "News title 3" and "News title 5" have no description, so
the two fields have different sizes.

The root part's matcher removes the question: a title and a description belong
to the same post when they are found in the same instance.

### Nesting

Think of posts that have small flyers at the bottom of the card:

```
[Thumbnail]
News title
#today #tech
#apple
```

Since typical HTML pages lay such flyers out in a container separate from other
elements like the title, and structures are infered from the input HTML, it
would be common that the structure of a post card has a child part, which has a
field that stands for the flyers.

A part matches a subtree inside the subtree its parent matched. Its part matcher
is therefore applied inside one subtree the parent matched, and everything it
finds sits inside that subtree.

Ticking a part means taking that part and everything below it, so the user does
not have to tick the flyer part separately to observe the flyers of a post. The
child part exists so that the user can drop the flyers while keeping the rest,
not so that they have to tick it to get them.

## Requirements

- The algorithm never drops any sample, field, part and structure. Who judges is
  the user. The algorithm just provides choices.

- The output must say which extracted values belong to the same post. Observing
  a page means reporting posts, and one post is a link together with the title,
  date, image and body that belong to that same link. A set of titles and a set
  of links with no correspondence between them cannot be turned into posts, so
  an output that only says "these are the titles" and "these are the links" does
  not meet the goals.

- The fields must cover all values in the input page, except for the elements
  listed below, which never carry a value a user would observe:
  - `script`, `style`, `noscript`, `template`, `head` and everything in them.
  - elements hidden from the page, meaning `hidden`, `aria-hidden="true"`,
    `display:none` or `visibility:hidden` in an inline style attribute.
  - comment nodes.

  Note that `nav`, `header` and `footer` are not on this list. They are removed
  by the user at review time, not by the algorithm, because a page can list
  posts inside any of them.

- All the subtrees one part matches must stand for the same kind of chunk. A
  part may match a few subtrees that are not posts, such as a promotion card
  sitting in the same list, but it must not be a container that happens to hold
  several unrelated things. The number of subtrees may be one: a page that shows
  a single hero post above a list is a real case, so a structure with one
  instance is valid.

- One instance must stand for one post. A part must not be so wide that a single
  subtree holds the values of two posts. This is what the `merge` metric
  measures.

- A field matcher and a part matcher must not depend on values that change per
  build. In particular they must not require a class name that a build tool
  generates, such as `style-1w7apwp` or `storycard__header--474c6553bfa`.

## Metrics

Given an output structure list, the evaluator uses it to extract values from the
fixture pages, gathers them per instance to make up possible posts, and measures
the metrics below. Every metric is computed over all the structures a page
produces. No user behaviour is simulated.

### How values are extracted

Extraction takes a structure and a page, and goes in three steps:

1. apply the root part's matcher to the page, which gives one instance per
   occurrence of the chunk.
2. inside each instance, apply the field matcher of every field of the root
   part, which gives at most one node per field, and read that node's values.
3. for every child part, apply its part matcher inside the instance, and repeat
   step 2 for each subtree it finds. Repeat this for the parts below them, and
   add every value found to the instance.

The values of one instance are what these three steps produce for it, so a
structure yields exactly as many sets of values as the root part's matcher finds
instances. A field whose matcher finds nothing in a given subtree contributes
nothing there, which is how an absent description or a missing thumbnail is
represented.

The same three steps run at review time and at polling time. The only difference
is the page they are applied to.

### How an instance is compared to a fixture post

A fixture post records a link, a list of texts and a list of images. An instance
offers a set of values, and a fixture post is also a set of values once its
link, texts and images are put together. A fixture value matches an instance
when it equals one of the instance's values, after the same normalization the
values were built with.

The `required` flag stored in the fixture files is ignored. The evaluator
instead holds its own list of required kinds, so that the list can be changed
without rewriting the fixtures. It starts as the title and the link.

A fixture post **matches** an instance when every value of a required kind
matches that instance. A fixture post **exactly matches** an instance when every
value matches, whatever its kind. Note that a fixture (exactly) matches an
instance even the instance has extra values that the fixture doesn't have. So,
the question is: "is this instance a superset of this fixture?".

### Post metrics

- recall, the share of the fixture posts from the source page, that match one of
  the instances of any structure.

- recall+, the same share using exact matching.

- merge, the mean number of fixture posts one instance stands for. It is counted
  over the instances that match at least one fixture post, so junk instances do
  not move it. 1 is best and means each instance stands for a single post.

  It is what stops the output from collapsing into one monolithic structure. An
  algorithm can produce a single part that matches the whole page, with one
  field per post title, one per post link and so on. That output has recall 1,
  distrib 1 and structures 1, while being a positional enumeration that cannot
  survive a later build. Its merge is the number of posts in the page.

  This metric assumes no two posts of a page share both a link and a title,
  otherwise one correct instance would count as standing for two posts. That
  holds for all 34 fixtures today: no page records the same link twice.

- distrib, the mean number of **parts** a fixture post needs. For one post that
  matches an instance, take the smallest subtree of that instance holding every
  matching value of the post, and count the distinct parts that the part
  instances in it belong to. The smallest subtree is unique and includes the
  part instances in between that hold no value of the post, because the user
  still passes through them to reach the ones that do.

  A matching value means a value of the post that the instance carries, required
  or not. A value the instance does not carry is ignored. An instance that
  carries the link and the title of a post but not its description therefore
  gives distrib 1, because the description is not a value the user reaches
  through this instance at all.

  A post can match the instances of several structures. The one measured is the
  cheapest: the lowest distrib, and the lowest wrappers when two instances give
  the same distrib. The user reviews the cheapest structure and drops the
  others, so the others are not a review cost for this post.

  1 is best and means one part holds every field of the post, so the user
  reviews it without expanding anything. It goes above 1 when the values of a
  post are spread over several levels of the instance.

  It counts distinct parts and not part instances, because the review screen
  shows one row per part however many subtrees that part matches. A post with
  ten tags therefore scores the same as a post with two tags, which is right:
  the user ticks the tag part once in both cases.

  It counts parts and not fields on purpose. The number of fields grows with the
  number of attributes a post has, which is not a fault of the output, while a
  post whose values are spread over several parts costs the user extra rows to
  expand for the same attributes.

- distrib+, the same mean using exact matching. It is taken over the posts that
  exactly match an instance, and over those instances only. Every value of the
  post is a matching value there, because an exact match carries them all.

- wrappers, the mean number of parts of an instance that sit outside the
  smallest subtree distrib measured, counted the same way: distinct parts, not
  part instances, and read from the same instance distrib measured. 0 is best.
  These are parts the user reads or expands without reaching any value of the
  post, so they are review cost that buys nothing. distrib alone does not report
  them: an instance of four parts whose values sit in the lower two gives the
  same distrib as an instance of exactly those two parts.

### Review size metrics

- structures, the number of structures the page produces. This is the size of
  the wall the goals ask to keep small, and it is the number that matters most,
  because reading the fields of one structure is much cheaper than visiting two
  structures for the same fields.
- depth, the deepest chain of parts inside a structure on the page.

### Blob metrics

A blob is judged on whether it covers the body and nothing else. The fixture
values of kind `body` are the bodies; every other fixture value belongs to
something the user should be able to tick on its own.

- body recall, the share of the fixture `body` values of the page that appear
  among the values of some blob.
- body split, the mean number of blobs one fixture post's `body` values are
  spread across, counting only posts that have a body. 1 is best and means the
  whole body of a post is one blob. Above 1 indicates the blob detection was
  failed in that page.
- leak, the number of fixture values that are **not** of kind `body` and appear
  among the values of some blob. Each one is a value the user can no longer tick
  on its own, because it was swallowed by a body. It is also reported as a share
  of all non-body fixture values of the page.
- blobs, the number of blobs the page produces. Read together with leak: a page
  with no blobs has no leak and no body recall either.

Note that only 2 of the 34 fixture pages record `body` values,
developer.apple.com-news and blog.codinghorror.com, so body recall and body
split rest on those two pages. Leak is measurable on all 34.

## Accepted limitations

- Lack of perfect matchers: it's impossible to create a matcher that can absorb
  any changes that may be introduced per page build. The recall should reach 1
  for the fixtures, but accept it to below 1 in production.
- Junk structures: users can drop junks by hands, so the recall matters more and
  some structures that stand for nothing useful are acceptable, as long as there
  are not so many that the review becomes a wall.
- The best value distrib can reach is not 1 on every page, and the value it can
  reach is unknown. A field matcher points at most one node per subtree, so a
  post that records several values of the same kind coming from different nodes
  needs a child part, which puts distrib above 1 however good the output is.
  This affects 124 of the 1302 fixture posts, on 12 of the 30 pages that have
  posts, the largest being flutter.dev-blog 33, qiita.com 29, bbc.com 18 and
  aws.amazon.com-jp-blogs-news 10. So distrib is read as a trend within one
  page, not compared across pages.

  Note that several values of one kind do not always need a child part. A
  `pub-date` recorded twice is usually the `datetime` attribute and the visible
  text of one node, and one field covers both because a field reads every value
  of its node.

- Robustness across builds is not measured. Every metric runs on one saved copy
  of a page, so the goal of absorbing per-build changes is stated but not
  checked.
- The metrics do not measure whether the screen makes the right structure easy
  to find, which is a question about ordering and about what a collapsed row
  shows.

## Open decisions

1. **51 fixture texts cannot be reached.** Of the 5368 texts the 34 fixtures
   record, 5317 equal a value some node offers. The other 51 were recorded as
   several texts of one node joined together, which no node offers: 43 authors,
   7 bodies and 1 description. Either those fixture entries are split to match
   what the page contains, or recall+ can never reach 1 on the pages that hold
   them. I recommend fixing the fixtures, because the alternative is a matching
   rule that joins values and would hide real misses.
2. **Bodies are recorded on 2 of the 34 fixture pages.** The blob metrics are
   the reason to add the blob, and they rest on those two pages. Recording body
   values on a few more pages would make them mean something.
