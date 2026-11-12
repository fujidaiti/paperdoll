# Test pages

Web pages saved with `curl` (without running JavaScript) for finding the posts
of a page. Each directory has the same pages under the same file names.

- `source/`: the pages as first fetched, on which post selectors are built.
- `robustness/`: the same pages fetched again on 2026-10-11, after their content
  had changed. They show whether selectors built on `source/` still find the
  posts of a later version of a page. has an empty file or an empty list.
- `fixture/`: the posts that a correct selector finds on each page in `source/`,
  as a JSON list of objects with the fields `title`, `body`, `link`, `thumbnail`
  and `pubDate`. A post leaves out the fields it does not have. Most were
  extracted with selectors built by hand and corrected by hand where the
  selectors were wrong. `go.dev-blog` was written by hand from the page.
- `simplified/`: the simplified page of each page in `source/` (see
  `SimplifyPage`).
- `selector/`: the extraction rules of each page with a fixture, as the database
  stores them. They are built on `simplified/` in the same way as a user of the
  app builds them, and find exactly the posts in `fixture/` (see
  perfect_selector_test.go).
- `extract/`: small pages written by hand, one for each case of
  `TestExtractPosts`, with the posts that the selectors of the case find in
  `<case>.posts.json`.
- `body-text.html`: a small page written by hand, with one node for each case of
  `TestBodyText`.

`simplified/` and `selector/` are golden files. After changing the simplifier,
the selectors or a fixture, update them with:

```console
go test ./feature/feed -run Golden -update
```
