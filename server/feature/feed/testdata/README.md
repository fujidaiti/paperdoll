# Test data of the post selectors

## Unit cases

Each behaviour of the page simplifier, the rule generator, the extraction and
the selector editor has its own case: a txtar file whose comment documents the
behaviour and whose name becomes the name of the test. To add a case, add a
file. The expected output of a case is its specification, so it is written by
hand, and `-update` never touches it.

| Directory    | Sections                                    | Test                |
| ------------ | ------------------------------------------- | ------------------- |
| `simplify/`  | `input.html`, `want.html`                   | `TestSimplifyPage`  |
| `generate/`  | `page.html`, `marks.json`, `rules.json`     | `TestGenerateRules` |
| `marks-for/` | `page.html`, `rules.json`, `marks.json`     | `TestMarksFor`      |
| `extract/`   | `page.html`, `rules.json`, `posts.json`     | `TestExtractPosts`  |
| `body/`      | `page.html`, `want.txt`                     | `TestBodyText`      |
| `editor/`    | `page.html`, `marks.json`, and `state.json` | `TestEditorState`   |
| `editor/`    | `page.html`, `marks.json`, and `error.txt`  | `TestResolveMarks`  |

A `page.html` is a page that is already simplified, with the node ID of each
element in `data-id`, so a bug in the simplifier does not fail these cases. It
may be indented.

- A `generate/` case can add `marks-reordered.json`, which must give the same
  rules.
- A `marks-for/` case can add `regenerated.json` when the rules generated from
  the marks differ from `rules.json`.

## Web pages

`webpage/<name>/` holds a real page, which checks that the pieces hold up on
messy HTML and when a page changes. Its test URL is `http://` plus the directory
name with `-` read as `/`.

| File                     | Kind    | Holds                                                                                                                              |
| ------------------------ | ------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `source.html`            | fixture | The page as fetched with `curl`, without JavaScript                                                                                |
| `posts.fixture.json`     | fixture | The posts that the rules find, by node ID and value                                                                                |
| `preview.fixture.json`   | fixture | The posts that the rules extract, merged by link                                                                                   |
| `marks.fixture.json`     | fixture | The marks that a person makes to get those posts                                                                                   |
| `simplified.golden.html` | golden  | The simplified page                                                                                                                |
| `rules.golden.json`      | golden  | The rules generated from the marks                                                                                                 |
| `later/`                 | both    | The same page fetched later, with its own `source.html`, `posts.fixture.json`, `preview.fixture.json` and `simplified.golden.html` |

A post in `posts.fixture.json` takes each field from the first selector that
finds its node, so `preview.fixture.json` holds the fields that the other
selectors add.

`viewer/` serves the pages with the posts of their fixtures highlighted, to
check a fixture by eye:

```console
go run ./testdata/viewer
```

## Updating

Golden files are outputs. After a change to the simplifier or the generator,
rewrite them and review the diff:

```console
go test ./feature/feed -run Golden -update
```

Fixtures are the source of truth, and no test rewrites them. When a change moves
the node IDs of a page, the web page tests fail even with `-update`, and a
person fixes the fixtures by hand.
