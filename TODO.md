# TODO

## Page watch

### Fields at different positions in posts of one kind

A field step is a shape and an index (see `tool/semantictree/MATCHER.md`), so
one matcher has only one path to each field. When posts of one kind have a field
at different positions, the user must annotate them as several groups.

The simplifier no longer merges a wrapper around a single node, which moved the
values of a post one level up when the post lacked an optional value, such as
the category of an anthropic.com/news card. It still replaces a container
without text and links with its children (`unwrapTextless`), so a wrapper of an
image is kept in a post that has a caption with the image and removed in a post
that does not.

Possible ways to let one group cover both:

- Allow several paths for one field, tried in order. The client adds a path when
  the user labels the field again on a post where it is not found.
- Make the simplifier keep the same structure in such cases. This is harder,
  because the simplifier does not know the posts.

### Bodies without a container

A link that covers only part of a text splits the text into values, so a
paragraph with a link in it is a container and a paragraph without one is a
value. A body labeled on the container of its paragraphs keeps its path in every
post. Some pages have no such container, such as chsmc.org, whose posts hold the
header and the paragraphs side by side. There, a body can only be labeled on one
paragraph. In posts whose paragraph has a different shape, the matcher takes
another paragraph as the body, or finds none and drops the post, since the body
is a required field. This is accepted as a limitation of matchers.
