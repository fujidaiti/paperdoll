# Test pages

Web pages saved with `curl` (without running JavaScript) for the page watch
feature. Each directory has the same pages under the same file names.

- `source/`: the pages as first fetched, on which post matchers are built.
- `robustness/`: the same pages fetched again on 2026-10-11, after their content
  had changed. They show whether matchers built on `source/` still find the
  posts of a later version of a page. has an empty file or an empty list.
- `fixture/`: the posts that a correct matcher finds on each page in `source/`,
  as a JSON list of objects with the fields `title`, `body`, `link`, `thumbnail`
  and `pubDate`. A post leaves out the fields it does not have. Most were
  extracted with matchers built by hand and corrected by hand where the matchers
  were wrong. `go.dev-blog` was written by hand from the page.
- `tree/`: the semantic tree of each page in `source/`.
- `matcher/`: the matchers of each page with a fixture, as the database stores
  them. They are built on `tree/` in the same way as a user of the app builds
  them, and find exactly the posts in `fixture/` (see perfect_matcher_test.go).

`tree/` and `matcher/` are golden files. After changing the tree builder, the
matchers or a fixture, update them with:

```console
go test ./feature/feed -run Golden -update
```
