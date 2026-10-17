# Test pages

Web pages saved with `curl` (without running JavaScript) for the page watch
feature. Each directory has the same pages under the same file names.

- `source/`: the pages as first fetched, on which post matchers are built.
- `robustness/`: the same pages fetched again on 2026-10-11, after their content
  had changed. They show whether matchers built on `source/` still find the
  posts of a later version of a page.
- `matcher/`: post matchers built by hand in the page annotation screen, on the
  semantic trees of the pages in `source/`. A page that has no usable matcher
  has an empty file or an empty list.
- `fixture/`: the posts that a correct matcher finds on each page in `source/`,
  as a JSON list of objects with the fields `title`, `body`, `link`, `thumbnail`
  and `pubDate`. A post leaves out the fields it does not have. Most were
  extracted with the matchers in `matcher/` and corrected by hand where the
  matchers are wrong. Pages without a matcher have no fixture, except
  `go.dev-blog`, whose fixture was written by hand from the page.
