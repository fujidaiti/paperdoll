# expr

Experiments about the feed feature that are easier to run in Python than in Go.
Nothing here is used by the server.

## Contents

| file                        | what it is                                                   |
| --------------------------- | ------------------------------------------------------------ |
| `collect_links.py`          | exports every link of the rendered semantic trees into a CSV |
| `data/links.csv`            | the export, one row per link, labelled with the fixtures     |
| `link_classification.ipynb` | telling a post link from the other links of a page           |
| `tree.py`                   | loads the trees and reads groups of posts out of them        |
| `group_scoring.ipynb`       | telling a list of posts from the other groups of a page      |
| `requirements.txt`          | the packages the notebooks need                              |

`link_classification.ipynb` asks which links of a page are posts.
`group_scoring.ipynb` asks which groups of the tree are lists of posts, which is
the question the selection screen asks. The second one reads the trees directly
through `tree.py` and needs no CSV.

## Setup

```sh
cd expr
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

## Running

The data is already in `data/links.csv`, so the notebook can be opened right
away:

```sh
.venv/bin/jupyter lab link_classification.ipynb
.venv/bin/jupyter lab group_scoring.ipynb
```

Export the data again after the Go test rewrites the trees, which it does on
every run of `go test ./server/feature/feed`:

```sh
.venv/bin/python collect_links.py
```

The notebooks are stored with their output, so the charts are visible without
running them. Run all the cells after changing a rule.

## What `data/links.csv` holds

One row per (page, link). A page often repeats a link on several nodes, for
example on an empty element covering a whole card and again on the title, so the
values of those nodes are merged into one row.

| column                          | meaning                                                             |
| ------------------------------- | ------------------------------------------------------------------- |
| `page`                          | the saved page, named after the file in `testdata/`                 |
| `page_url`                      | the URL the page was saved from                                     |
| `url`                           | the link, already resolved against the page                         |
| `is_post`                       | whether the fixture lists this URL as a post                        |
| `nodes`                         | how many nodes of the tree carry the link                           |
| `heading`, `images`, `texts`    | what those nodes hold                                               |
| `longest_text`, `title`         | the length and the value of the longest text of those nodes         |
| `datetime`                      | whether one of the texts came from an element with a date attribute |
| `min_depth`, `order`            | where the shallowest node sits in the tree and in the document      |
| `siblings`                      | how many other links the card holding the link holds                |
| `under_page_path`, `path_depth` | what the URL path says about the link                               |
