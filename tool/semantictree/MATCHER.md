# Post matcher on semantic trees (draft)

This document specifies the post matcher that runs on the semantic trees
produced by `tool/semantictree`. It replaces the selector-based matcher in
`client/lib/features/page_watch/domain/post_matcher.dart`.

## Context

The user annotates a page in the page watch screen. They select one post, label
its fields, and the client generates a matcher that finds the other posts of the
same pattern. The server runs the matchers on each fetch and turns the extracted
posts into feed entries.

The current matcher selects elements by tags, classes and indices. Container
classes are fragile: they include layout, state and hash classes, and the
simplifier merges wrappers, so the same post can have different container
classes in different parts of a page.

## Goal

- A matcher finds the posts of one pattern from a semantic tree and extracts
  their fields.
- A matcher depends on the shape of the tree. Classes are used only to tell
  values apart inside a post.
- A post that a matcher finds but that is not a post, such as a promotion card,
  can be removed by the user with an exclude list.
- The format of the matcher is JSON, stored with the page watch group.

## Approach

- The position of the post node is a path of shapes from the root of the tree.
  The depth of every node is therefore fixed.
- Each field is a path of shapes relative to the post node. The last step, which
  is a value, may also have a leaf condition (tags and classes).
- A node at the post path is a post only when all required fields (title and
  link) are found.
- Each group has an exclude list of link patterns. Posts whose link matches a
  pattern are removed from the group's output.
- Posts from all groups are merged by link.

## Terms

| Term      | Meaning                                                                               |
| --------- | ------------------------------------------------------------------------------------- |
| value     | A node of the tree without children (`{"sig","kind"?,"val"?,"link"?}`).               |
| container | A node of the tree with children (`{"sig","link"?,"ch"}`).                            |
| shape     | The shape of one node, defined below.                                                 |
| post node | The node that holds the values of one post. It can be a container or a value.         |
| field     | One of `title`, `link`, `description`, `body`, `thumbnail`, `pubDate`.                |
| pattern   | The shape of the post nodes that one annotated post represents.                       |
| group     | The posts that the user annotated as one kind, with one matcher and one exclude list. |

### Shape

The shape of a node is a string:

- A container: `card` when it has a link, otherwise `box`.
- A value: its `kind` (`h1`..`h6`, `img`, `time`), or `txt` when it has no kind,
  followed by `+link` when it has a link. Examples: `h2`, `txt+link`,
  `img+link`.

The shape path of a node joins the shapes of the root, its ancestors and the
node itself with `>`, such as `box > box > box > card > box > h2`.

### Tags and classes of a value

The tags and classes of a value are those of all elements the value was made
from, as listed in its `sig`. For the sig `div.TitleWrapper > h2.CardHeadline`,
the tags are `div`, `h2` and the classes are `TitleWrapper`, `CardHeadline`.

Hash classes and utility classes are already removed by the simplifier.

## Matcher format

```json
{
  "post": ["box", "box", "box", "card"],
  "fields": {
    "title": [{ "shape": "box" }, { "shape": "h2", "cls": ["CardHeadline"] }],
    "link": [],
    "thumbnail": [{ "shape": "img+link" }],
    "description": [
      { "shape": "box" },
      { "shape": "txt", "cls": ["Description"], "nth": 0 }
    ]
  },
  "exclude": [
    "*://www.bbc.com/newsletters?*",
    "https://www.bbc.com/pages/download-the-bbc-app"
  ]
}
```

- `post`: the shape path of the post node from the root. The first item is the
  shape of the root.
- `fields`: for each labeled field, the steps from the post node to the value.
  An empty list means the post node itself, such as the link of a `card` post
  node, or the title and link of a post node that is a single `h2+link` value.
- A step:
  - `shape` (required): the shape of the child node.
  - `tags`, `cls` (optional, last step only): the tags and classes that the
    value must have. The value may have more.
  - `nth` (optional): the zero-based index among the children that match
    `shape`, `tags` and `cls`. It is used only when the other conditions cannot
    tell the field from its siblings (see "Building a matcher").
- `exclude`: the exclude list of the group (see "Exclude list").

`title` and `link` are required in every matcher. The other fields are optional.

## Extraction

For each group:

1. Find the candidate post nodes: start from the root, and at each step keep the
   children whose shape equals the next item of `post`. The root must match the
   first item.
2. For each candidate, resolve each field: start from the post node and, at each
   step, keep the children that match the step. A step with `nth` keeps only the
   n-th match. If several nodes remain at the end, the first one in document
   order is the value.
3. Keep only the candidates where `title` and `link` both resolve. A candidate
   where an optional field does not resolve is still a post, without that field.
4. Read the field values:
   - `title`, `description`: `val` of the value.
   - `link`: `link` of the node.
   - `thumbnail`: `link` of the `img` value.
   - `pubDate`: `val` of a `time` value, which is an RFC 3339 timestamp, or else
     the text `val`.
   - `body`: the values inside the node (format to be defined with the body
     markup).
5. Remove the posts whose link matches a pattern in `exclude`.

Then, for all groups together:

6. Merge the posts that have the same link, in document order. The first post
   provides each field, and later posts fill only the fields that the earlier
   ones do not have. For example, BBC splits some posts into two sibling cards
   with the same link: one with the image and one with the title.

## Building a matcher

The client builds the matcher when the user labels the fields of a post.

1. `post` is the shape path of the selected post node.
2. For each labeled field, the steps are the shapes from the post node to the
   labeled node, without leaf conditions.
3. The candidate posts are the nodes found by extraction steps 1–3 with these
   steps (shapes only).
4. Leaf condition of each field: the tags and classes of the labeled value that
   the corresponding value of every candidate post also has. The corresponding
   value of a candidate is the one at the same position among the children with
   the same shape as in the labeled post. When the candidates share no class,
   the step has no `cls`.
5. If the labeled value still has siblings that match its last step, such as the
   title, author and reading time of a cursor card (all `card > txt`), the last
   step gets `nth`: the index of the labeled value among those matches. This is
   the only case where an index is used.

One matcher covers one pattern. A post of the same kind whose shape path
differs, such as a deepmind post without an image (the simplifier merges its
single-child wrapper, so the title is one level higher), is a separate pattern.
The user annotates one post of each pattern.

## Exclude list

The matcher of a group can find nodes that have the same shape path and leaf
conditions as the posts but are not posts. For example, the "Discover more from
the BBC" promotion cards have the same shape and the same title class
(`h2.IndexCardHeading-styles__CardHeadlineStyled`) as the news cards. The
exclude list removes such items.

### User flow

1. The user annotates a post and labels its fields. The client builds the
   matcher of the group and shows the posts it finds.
2. The user taps "Exclude" (next to "Edit fields") on a post that is not a post.
   Its link is added to the exclude list, and the post is removed from the
   group's output.
3. The user excludes another post of the same group. The client tries to merge
   the new link into an existing pattern (see "Merging links").
4. The user repeats step 3 until they confirm that the group is done.

The user can also remove a pattern from the list. The posts it removed then come
back.

### Patterns

A pattern is either a link or a glob. A link is compared after the same
normalization that identifies feed entries.

A glob is made of the parts of a URL:

- scheme
- host
- each path segment
- the query, as one part

The fragment is not part of a pattern.

A part that is `*` matches any value of that part, including an empty query.
Every other part matches only the same text. A glob matches a link when the two
have the same number of path segments and every part matches. Example:
`*://www.bbc.com/newsletters?*`.

### Merging links

When the user excludes a link `L` in a group:

1. For each pattern `P` in the list, in order, build the merge of `P` and `L`:
   - The merge fails if the hosts differ, or if the numbers of path segments
     differ.
   - Otherwise, each part that is equal in `P` and `L` is kept, and each part
     that differs becomes `*`. A part that is already `*` in `P` stays `*`.
2. Accept the first merge that matches no link of the group's posts. These are
   all posts the group's matcher finds in the current page, except the excluded
   ones, not only the labeled posts. The merge replaces `P`.
3. If no merge is accepted, add `L` to the list as a link.

Example with the BBC promotions:

| Excluded link                                                       | Exclude list after the merge                                                      |
| ------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `http://www.bbc.com/newsletters?RoyalWatch&at_medium=onsite`        | `http://www.bbc.com/newsletters?RoyalWatch&at_medium=onsite`                      |
| `https://www.bbc.com/newsletters?howtosleepbetter&at_medium=onsite` | `*://www.bbc.com/newsletters?*`                                                   |
| `https://www.bbc.com/pages/download-the-bbc-app`                    | `*://www.bbc.com/newsletters?*`, `https://www.bbc.com/pages/download-the-bbc-app` |

The third link cannot merge into the glob, because its path has two segments and
the glob's path has one. If the numbers were equal, the merge would still be
rejected when it matched a post link, such as a news article.

## Known limitations

- An item that is not a post can share the shape path and leaf conditions of
  real posts. The exclude list removes it after the user sees it. Until then, it
  appears as one feed entry, since entries are identified by their link.
- A glob is checked only against the posts of the page when the user creates it.
  A later post can match it and be removed. The user can fix this by editing the
  exclude list.
- If a site changes the depth of its posts, for example by adding a wrapper, the
  matcher finds no posts until the user annotates the page again.

## Open questions

- The tree format: whether values get separate `tags` and `cls` instead of
  `sig`, and whether containers keep their tags (such as `section`).
- Whether a group can hold several patterns, so that the user can add a second
  pattern of the same kind (such as the deepmind post without an image) to an
  existing group, or whether each pattern is a separate group.
- Dates in plain text, such as "September 2026" or "3 hrs ago", are `txt`
  values. The format of `pubDate` taken from such a text is not defined.
