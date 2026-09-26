"""Collect every link of the rendered semantic trees into one CSV.

The trees are written by TestSemanticTree into
server/feature/feed/testdata/semantic/, and the fixtures beside them say which
URLs are posts. One row per (page, link): the values of the nodes carrying that
link are merged, because a page often repeats a link on several nodes.

Run it again after the Go test rewrites the trees:

    python3 collect_links.py
"""

import csv
import glob
import json
import os
from urllib.parse import urlsplit

ROOT = os.path.dirname(os.path.abspath(__file__))
TESTDATA = os.path.join(ROOT, "..", "server", "feature", "feed", "testdata")
OUT = os.path.join(ROOT, "data", "links.csv")

HEADINGS = {"h1", "h2", "h3", "h4", "h5", "h6"}


def normalize(u):
    """The form used to compare a link with a fixture URL. The query and the
    fragment are kept, because developer.apple.com identifies a post by
    ?id= and some pages by #anchor."""
    p = urlsplit(u)
    out = p.netloc.lower() + p.path.rstrip("/")
    if p.query:
        out += "?" + p.query
    if p.fragment:
        out += "#" + p.fragment
    return out


def walk(node, depth, order, out):
    """Collect (node, depth, document order, number of siblings) of every node
    that carries a link."""
    if node is None:
        return order
    if node.get("link"):
        out.append((node, depth, order))
        order += 1
    for child in node.get("children") or []:
        order = walk(child, depth + 1, order, out)
    return order


def cards(node, out):
    """A card is a node with two or more children. For each one, report the set
    of links of every child, which is what a rule comparing siblings reads."""
    children = node.get("children") or []
    if len(children) >= 2:
        per_child = []
        for child in children:
            found = []
            walk(child, 0, 0, found)
            per_child.append({n["link"] for n, _, _ in found})
        out.append(per_child)
    for child in children:
        cards(child, out)


def main():
    rows = []
    for fixture_path in sorted(glob.glob(os.path.join(TESTDATA, "*.fixture.json"))):
        page = os.path.basename(fixture_path)[: -len(".fixture.json")]
        tree_path = os.path.join(TESTDATA, "semantic", page + ".json")
        if not os.path.exists(tree_path):
            continue
        fixture = json.load(open(fixture_path))
        tree = json.load(open(tree_path))
        posts = {normalize(p["url"]) for p in fixture["posts"]}
        page_path = urlsplit(fixture["url"]).path.rstrip("/")

        found = []
        walk(tree, 0, 0, found)
        if not found or not posts:
            continue

        # The children of a card that hold a link, per link, so that a rule
        # comparing siblings can be measured in the notebook.
        card_list = []
        cards(tree, card_list)
        siblings = {}
        for per_child in card_list:
            for link_set in per_child:
                for link in link_set:
                    # How many links the card holds beside this one.
                    others = len({l for s in per_child for l in s}) - 1
                    siblings[link] = max(siblings.get(link, 0), others)

        merged = {}
        for node, depth, order in found:
            link = node["link"]
            texts = node.get("texts") or []
            images = node.get("images") or []
            row = merged.setdefault(
                link,
                {
                    "page": page,
                    "page_url": fixture["url"],
                    "url": link,
                    "is_post": normalize(link) in posts,
                    "nodes": 0,
                    "heading": False,
                    "images": 0,
                    "texts": 0,
                    "longest_text": 0,
                    "datetime": False,
                    "min_depth": depth,
                    "order": order,
                    "title": "",
                    "siblings": siblings.get(link, 0),
                },
            )
            row["nodes"] += 1
            row["heading"] = row["heading"] or any(t["tag"] in HEADINGS for t in texts)
            row["images"] += len(images)
            row["texts"] += len(texts)
            row["datetime"] = row["datetime"] or any(t.get("datetime") for t in texts)
            row["min_depth"] = min(row["min_depth"], depth)
            row["order"] = min(row["order"], order)
            for t in texts:
                if len(t["value"]) > row["longest_text"]:
                    row["longest_text"] = len(t["value"])
                    row["title"] = t["value"][:120]

        link_path = lambda u: urlsplit(u).path.rstrip("/")
        for row in merged.values():
            path = link_path(row["url"])
            row["under_page_path"] = bool(page_path) and path.startswith(page_path + "/")
            row["path_depth"] = len([s for s in path.split("/") if s])
            rows.append(row)

    fields = [
        "page", "page_url", "url", "is_post", "nodes", "heading", "images",
        "texts", "longest_text", "datetime", "min_depth", "order", "siblings",
        "under_page_path", "path_depth", "title",
    ]
    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    with open(OUT, "w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=fields)
        w.writeheader()
        for row in rows:
            w.writerow({k: row[k] for k in fields})
    pages = len({r["page"] for r in rows})
    print(f"{len(rows)} links over {pages} pages -> {OUT}")


if __name__ == "__main__":
    main()
