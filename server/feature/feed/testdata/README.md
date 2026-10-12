# Test pages

Web pages saved with `curl` (without running JavaScript) for the page watch
feature. Each directory has the same pages under the same file names.

- `source/`: the pages as first fetched, on which post matchers are built.
- `robustness/`: the same pages fetched again on 2026-10-11, after their content
  had changed. They show whether matchers built on `source/` still find the
  posts of a later version of a page.
