# Post matcher on semantic trees (draft)

This document specifies the post matcher that runs on the semantic trees
produced by `tool/semantictree`. The client implements it in
`client/lib/features/page_watch/domain/post_matcher.dart`.

## Context

The user annotates a page in the page watch screen. They select one post, label
its fields, and the client generates a matcher that finds the other posts of the
same shape. The server runs the matchers on each fetch and turns the extracted
posts into feed entries.

The current matcher selects elements by tags, classes and indices. Classes are
fragile: they include layout, state and hash classes, and the simplifier merges
wrappers, so the same post can have different classes in different parts of a
page.

## Goal

- A matcher finds the posts of one shape from a semantic tree and extracts their
  fields.
- A matcher depends only on the shapes of the nodes and their positions, not on
  tags or classes.
- A post that a matcher finds but that is not a post, such as a promotion card,
  can be removed by the user with an exclude list.
- The format of the matcher is JSON, stored with the page watch group.

## Approach

- The position of the post node is a path of shapes from the root of the tree.
  The depth of every node is therefore fixed.
- Each field is a path of steps from the post node. A step is a shape and an
  index among the siblings of that shape.
- A node at the post path is a post only when all labeled fields are found.
- Each group has an exclude list of link patterns. Posts whose link matches a
  pattern are removed from the group's output.
- Posts from all groups are merged by link.

## Terms

| Term      | Meaning                                                                               |
| --------- | ------------------------------------------------------------------------------------- |
| value     | A node of the tree without children (`{"kind"?,"val"?,"link"?}`).                     |
| container | A node of the tree with children (`{"link"?,"ch"}`).                                  |
| shape     | The shape of one node, defined below.                                                 |
| post node | The node that holds the values of one post. It can be a container or a value.         |
| field     | One of `title`, `link`, `body`, `thumbnail`, `pubDate`.                               |
| group     | The posts found by one matcher, built from one annotated post, with one exclude list. |

There is no description field. A description and a body are both plain text that
summarizes the post, so `body` covers both.

### Shape

The shape of a node is a string:

- A container: `card` when it has a link, otherwise `box`.
- A value: its `kind` (`h1`..`h6`, `img`, `time`), or `txt` when it has no kind,
  followed by `+link` when it has a link. Examples: `h2`, `txt+link`,
  `img+link`.

The shape path of a node joins the shapes of the root, its ancestors and the
node itself with `>`, such as `box > box > box > card > box > h2`.

## Matcher format

```json
{
  "post": ["box", "box", "box", "card"],
  "fields": {
    "title": [
      { "shape": "box", "nth": 0 },
      { "shape": "h2", "nth": 0 }
    ],
    "link": [],
    "thumbnail": [{ "shape": "img+link", "nth": 0 }],
    "body": [
      { "shape": "box", "nth": 0 },
      { "shape": "txt", "nth": 0 }
    ]
  },
  "exclude": ["www.bbc.com/newsletters", "www.bbc.com/*/*"]
}
```

- `post`: the shape path of the post node from the root. The first item is the
  shape of the root.
- `fields`: for each labeled field, the steps from the post node to the value.
  An empty list means the post node itself, such as the link of a `card` post
  node, or the title and link of a post node that is a single `h2+link` value.
- A step:
  - `shape`: the shape of the child node.
  - `nth`: the zero-based index of the child among the children with the same
    shape. For example, the title, author and reading time of a cursor card are
    `txt` values with `nth` 0, 1 and 2.
- `exclude`: the exclude list of the group (see "Exclude list").

The user must label `title` and `link` in every matcher. The other fields are
optional to label.

## Extraction

For each group:

1. Find the candidate post nodes: start from the root, and at each step keep all
   children whose shape equals the next item of `post`. The root must match the
   first item.
2. For each candidate, resolve each field: start from the post node and, at each
   step, take the child at index `nth` among the children with the step's shape.
   If there is no such child, the field does not resolve.
3. Keep only the candidates where every labeled field resolves.
4. Read the field values:
   - `title`: `val` of the value.
   - `link`: `link` of the node.
   - `thumbnail`: `link` of the `img` value.
   - `pubDate`: `val` of a `time` value, which is an RFC 3339 timestamp. Only a
     `time` value can be labeled as `pubDate`. The simplifier makes a `time`
     value only from a date with a year, a month and a day, so a text such as
     "September 2026" or "3 hrs ago" stays `txt` and cannot be a `pubDate`.
   - `body`: the values of the node and inside it, as marked-up text (see
     `client/lib/features/page_watch/domain/body_markup.dart`).
5. Remove the posts whose link matches a pattern in `exclude`.

Then, for all groups together:

6. Merge the posts that have the same link, in document order. The first post
   provides each field, and later posts fill only the fields that the earlier
   ones do not have. For example, BBC splits some posts into two sibling cards
   with the same link: one with the image and one with the title.

## Building a matcher

The client builds the matcher when the user labels the fields of a post.

1. `post` is the shape path of the selected post node.
2. For each labeled field, there is one step for each node from the post node
   down to the labeled node, excluding the post node. The step has the shape of
   the node and its index among its siblings of the same shape.

Since the index counts only siblings of the same shape, a missing value of
another shape does not move a field. For example, a deepmind post without a
category still has its date at `txt` 0 and its "Learn more" link at
`txt+link` 0. A missing value of the same shape does move the values after it.

Each group has one matcher. A post of the same kind whose shape path differs,
such as a deepmind post without any image (the simplifier merges its
single-child wrapper, so the title is one level higher), is not found by the
matcher. The user annotates such a post as another group.

## Possible extensions

If shapes and positions cannot tell posts or fields apart, a step can also
require tags and classes of the node. The semantic tree would then need to carry
the tags and classes of the elements each node was made from, without hash
classes and utility classes.

## Exclude list

The matcher of a group can find nodes that have the same shapes and positions as
the posts but are not posts. For example, the "Discover more from the BBC"
promotion cards have the same shape as the news cards, and even the same title
class (`h2.IndexCardHeading-styles__CardHeadlineStyled`), so classes would not
tell them apart either. The exclude list removes such items.

### User flow

1. The user annotates a post and labels its fields. The client builds the
   matcher of the group and shows the posts it finds.
2. The user taps "Exclude" (next to "Edit fields") on a post that is not a post.
   A pattern for its link is added to the exclude list (see "Excluding a link"),
   and the post is removed from the group's output.
3. The user excludes another post of the same group. The client tries to merge
   the new link into an existing pattern (see "Excluding a link").
4. The user repeats step 3 until they confirm that the group is done.

The user can also remove a pattern from the list. The posts it removed then come
back.

### Patterns

A pattern is a file glob that matches the host and path of a link, such as
`www.bbc.com/newsletters`. The scheme, the query and the fragment of a link are
never matched: the simplifier keeps only http and https links, and query
parameters are always ignored. In a pattern:

- `*` matches any characters except `/`.
- `**` matches any characters, including `/`.
- `?` matches one character except `/`.
- Every other character matches itself.

For example, a pattern `www.bbc.com/newsletters` matches
`http://www.bbc.com/newsletters?RoyalWatch&at_medium=onsite`.

Patterns can be any globs, so that users can write their own in a future
version. The client generates only patterns whose first segment is a host and
whose other segments are either the text of a path segment or `*`, such as
`www.bbc.com/*/docs`. Merging (below) applies only to patterns of this form, and
leaves other patterns unchanged.

### Excluding a link

When the user excludes a link `L` in a group, the client checks each candidate
pattern against the links of the group's posts. These are all posts that the
group's matcher finds in the current page, except the excluded ones, not only
the labeled posts. A candidate is accepted only when it matches none of them.

1. For each generated pattern `P` in the list, in order, build the merge of `P`
   and the host and path of `L`:
   - The merge fails if the hosts differ, or if the numbers of segments differ.
   - Otherwise, each segment that is equal in both is kept, and each segment
     that differs becomes `*`.

   The first accepted merge replaces `P`.

2. If no merge is accepted, add the host and path of `L`, if it is accepted.
3. Otherwise, `L` cannot be excluded, because a post has the same host and path.
   The client tells the user so.

Example with the BBC promotions, where the group's posts link to news articles
such as `https://www.bbc.com/news/articles/c34gdjk1ne8yo`:

| Excluded link                                                | Exclude list after the step                                         |
| ------------------------------------------------------------ | ------------------------------------------------------------------- |
| `http://www.bbc.com/newsletters?RoyalWatch&at_medium=onsite` | `www.bbc.com/newsletters`                                           |
| `https://www.bbc.com/pages/download-the-bbc-app`             | `www.bbc.com/newsletters`, `www.bbc.com/pages/download-the-bbc-app` |
| `https://www.bbc.com/video/docs`                             | `www.bbc.com/newsletters`, `www.bbc.com/*/*`                        |

- The first pattern already removes the other newsletter promotions, such as
  `https://www.bbc.com/newsletters?howtosleepbetter&at_medium=onsite`, so the
  user does not exclude them one by one.
- The second link cannot merge into the first pattern, because its path has two
  segments and the pattern's path has one.
- The third link merges into the second pattern. If a post of the group linked
  to a path with two segments, such as
  `https://www.bbc.com/travel/worlds-table`, the merge would be rejected, and
  `www.bbc.com/video/docs` would be added instead.

## Known limitations

- An item that is not a post can share the shapes and positions of real posts.
  The exclude list removes it after the user sees it. Until then, it appears as
  one feed entry, since entries are identified by their link.
- An item that has the same host and path as a post, and differs only in its
  query, such as `/item?id=123`, cannot be excluded.
- A glob is checked only against the posts of the page when the user creates it.
  A later post can match it and be removed. The user can fix this by editing the
  exclude list.
- If a site changes the depth of its posts, for example by adding a wrapper, the
  matcher finds no posts until the user annotates the page again.
