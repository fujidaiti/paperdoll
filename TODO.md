# TODO

## Page watch

### Fields at different positions in posts of one kind

The simplifier merges a wrapper that has a single child into that child. When a
post lacks an optional value, a wrapper can lose all its other children and be
merged, so the values that remain move one level up. For example, the
anthropic.com/news cards wrap the category and the date in one box. In a card
without a category, the date becomes a direct child of the card:

```
card                     card
  time                     box
  h4                         txt   (category)
  txt                        time
                           h4
                           txt
```

A field step is a shape and an index (see `tool/semantictree/MATCHER.md`), so
one matcher has only one path to the date. Both kinds of cards have the same
post path, but the user must annotate them as two groups. Leaving out the date
would make them one group, at the cost of the dates of all posts.

Possible ways to let one group cover both:

- Allow several paths for one field, tried in order. The client adds a path when
  the user labels the field again on a post where it is not found.
- Keep wrappers in the simplifier when other posts of the same kind keep theirs.
  This is harder, because the simplifier does not know the posts.
