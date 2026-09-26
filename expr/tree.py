"""Load the rendered semantic trees and read groups of posts out of them.

The trees are written by TestSemanticTree into
server/feature/feed/testdata/semantic/, and the fixtures beside them say which
URLs are posts.

collect_links.py works one link at a time. This module works on the groups of
siblings, which is what a row of the selection screen is: the user ticks a
group, not a link.
"""

import glob
import json
import os
import re
from urllib.parse import urlsplit

ROOT = os.path.dirname(os.path.abspath(__file__))
TESTDATA = os.path.join(ROOT, "..", "server", "feature", "feed", "testdata")

HEADINGS = {"h1", "h2", "h3", "h4", "h5", "h6"}
DATE = re.compile(
    r"\b(\d{4}-\d{2}-\d{2}|\d{4}/\d{1,2}/\d{1,2}|"
    r"(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?\s+\d{1,2}|"
    r"\d{1,2}\s+(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)|"
    r"\d+\s+(minute|hour|day|week|month|year)s?\s+ago|\d{4}年\d{1,2}月)",
    re.I,
)
# The tags that carry no meaning of their own. A post title written in a div
# and the same title written in a span are the same value.
NEUTRAL = {"p", "span", "div", "li", "em", "strong", "b", "i", "small", "a"}

NUMBER = re.compile(r"^\d+$")
HASH = re.compile(r"^[0-9a-f]{7,}$", re.I)


def normalize(url):
    """The form used to compare a link with a fixture URL. The query and the
    fragment are kept, because developer.apple.com identifies a post by ?id=
    and some pages by #anchor."""
    p = urlsplit(url)
    out = p.netloc.lower() + p.path.rstrip("/")
    if p.query:
        out += "?" + p.query
    if p.fragment:
        out += "#" + p.fragment
    return out


def segments(url):
    p = urlsplit(url)
    return p.netloc.lower(), [s for s in p.path.strip("/").split("/") if s]


def template(url):
    """The shape of a URL: the host, one token per path segment and the names
    of the query parameters. Two links of the same list normally share it."""
    host, segs = segments(url)
    toks = []
    for s in segs:
        if NUMBER.match(s):
            toks.append("{num}")
        elif HASH.match(s) and any(c.isdigit() for c in s):
            toks.append("{hash}")
        else:
            toks.append("{slug}")
    query = urlsplit(url).query
    tail = ""
    if query:
        tail = "?" + ",".join(sorted(p.split("=")[0] for p in query.split("&")))
    return host + "/" + "/".join(toks) + tail


class Node:
    """One node of a tree, with the parent and the depth the JSON does not
    store."""

    def __init__(self, raw, parent, depth, order):
        self.raw = raw
        self.parent = parent
        self.depth = depth
        self.order = order
        self.link = raw.get("link", "")
        self.texts = list(raw.get("texts") or [])
        self.images = list(raw.get("images") or [])
        self.children = []

    @property
    def signature(self):
        """What the node looks like, without its values. Two cards of the same
        list have the same signature even when one of them has no category.

        The tags that carry no meaning of their own are reported as one name,
        so that a card naming its author in a span and a card without an
        author still look alike. The number of children and the presence of an
        image are left out, because a card that is still split into several
        nodes would otherwise never match one that is not."""
        tags = {t["tag"] for t in self.texts}
        return (
            bool(self.link),
            tuple(sorted({"text" if t in NEUTRAL else t for t in tags})),
        )

    def walk(self):
        yield self
        for c in self.children:
            yield from c.walk()

    @property
    def links(self):
        return [n.link for n in self.walk() if n.link]

    @property
    def size(self):
        return sum(1 for _ in self.walk())

    def all_texts(self):
        return [t for n in self.walk() for t in n.texts]

    def all_images(self):
        return [i for n in self.walk() for i in n.images]

    def has_date(self):
        return any(
            t.get("datetime") or DATE.search(t["value"]) for t in self.all_texts()
        )


def build(raw, parent=None, depth=0, counter=None):
    if counter is None:
        counter = [0]
    node = Node(raw, parent, depth, counter[0])
    counter[0] += 1
    for c in raw.get("children") or []:
        node.children.append(build(c, node, depth + 1, counter))
    return node


class Page:
    def __init__(self, name, fixture, root):
        self.name = name
        self.url = fixture["url"]
        self.fixture = fixture
        self.root = root
        self.posts = {normalize(p["url"]) for p in fixture["posts"]}

    def is_post(self, url):
        return bool(url) and normalize(url) in self.posts

    def groups(self):
        """A group is a node with two or more children. Its members are those
        children, which is what one row of the selection screen would be."""
        return [n for n in self.root.walk() if len(n.children) >= 2]


def load():
    pages = []
    for fixture_path in sorted(glob.glob(os.path.join(TESTDATA, "*.fixture.json"))):
        name = os.path.basename(fixture_path)[: -len(".fixture.json")]
        tree_path = os.path.join(TESTDATA, "semantic", name + ".json")
        if not os.path.exists(tree_path):
            continue
        fixture = json.load(open(fixture_path))
        raw = json.load(open(tree_path))
        if not raw or not fixture.get("posts"):
            continue
        page = Page(name, fixture, build(raw))
        if page.root.links:
            pages.append(page)
    return pages


def candidate_links(node, siblings=()):
    """The links of a subtree that could be the post the subtree is about.

    A card carries more than the post it shows, and each kind of extra link is
    told apart by how it sits next to the others rather than by its own text:

    - a link that another link of the same card extends is the author page,
      the category or the section the post belongs to;
    - a shape that occurs more than once in one card, such as five links to
      /tags/<name>, cannot be the single post of that card;
    - a link that several cards of the same list carry is a banner or a menu
      repeated on every card.

    The last test needs the other members of the group, so a card is read in
    the context of its list rather than on its own.
    """
    links = {n.link for n in node.walk() if n.link}
    if len(links) <= 1:
        return links

    paths = {}
    shapes = {}
    for link in links:
        host, segs = segments(link)
        paths.setdefault(host, set()).add(tuple(segs))
        shapes[template(link)] = shapes.get(template(link), 0) + 1

    shared = set()
    if siblings:
        seen = {}
        for s in siblings:
            if s is node:
                continue
            for link in {n.link for n in s.walk() if n.link}:
                seen[link] = seen.get(link, 0) + 1
        shared = {l for l, n in seen.items() if n >= 2}

    out = set()
    for link in links:
        host, segs = segments(link)
        t = tuple(segs)
        if any(o[: len(t)] == t and len(o) > len(t) for o in paths[host]):
            continue
        if shapes[template(link)] > 1:
            continue
        if link in shared:
            continue
        out.add(link)
    return out or links


def representative(node, siblings=(), shape=None):
    """The link a member of a group stands for. Among the candidate links, the
    post is the one whose shape the rest of the group shares, then the one the
    card repeats, then the one the longest text sits on."""
    found = [n for n in node.walk() if n.link]
    if not found:
        return ""
    allowed = candidate_links(node, siblings)
    if shape is not None:
        same = {l for l in allowed if template(l) == shape}
        allowed = same or allowed
    count = {}
    longest = {}
    for n in found:
        if n.link not in allowed:
            continue
        count[n.link] = count.get(n.link, 0) + 1
        value = max((len(t["value"]) for t in n.texts), default=0)
        longest[n.link] = max(longest.get(n.link, 0), value)
    if not count:
        return found[0].link
    return max(count, key=lambda l: (count[l], longest[l]))


def group_links(members):
    """One link per member, read in two passes. The first pass reads every
    member on its own; the second one reads it again knowing the shape most
    members agreed on, which is what an item written as running text needs:
    on developer.apple.com the body of an item links to the documentation and
    to other pages, and only the shape says which link is the item itself."""
    first = [representative(m, members) for m in members]
    counts = {}
    for l in first:
        if l:
            counts[template(l)] = counts.get(template(l), 0) + 1
    if not counts:
        return first
    shape, agreed = max(counts.items(), key=lambda kv: kv[1])
    if agreed < 2:
        return first
    return [representative(m, members, shape) for m in members]


def group_features(members):
    """What a group of siblings looks like, read without the fixtures. Every
    value is read from the members as whole subtrees, because a card is often
    still split into several nodes at this point."""
    n = len(members)
    links = group_links(members)
    linked = [l for l in links if l]
    shapes = {}
    for l in linked:
        shapes[template(l)] = shapes.get(template(l), 0) + 1
    shape = max(shapes, key=lambda k: shapes[k]) if shapes else None
    # How many links of the shape the group agreed on a member holds. One is
    # what a row of the selection screen needs. A page container holds a whole
    # list per member, and a member of a list of authors or of tags holds none
    # of the shape the posts share.
    counts = sorted(
        sum(1 for l in candidate_links(m, members) if template(l) == shape)
        for m in members
    )
    sizes = sorted(m.size for m in members)
    texts = sorted(max((len(t["value"]) for t in m.all_texts()), default=0) for m in members)
    totals = sorted(sum(len(t["value"]) for t in m.all_texts()) for m in members)

    def modal(values):
        seen = {}
        for v in values:
            seen[v] = seen.get(v, 0) + 1
        return max(seen.values()) / n if seen else 0.0

    return {
        "n": n,
        "depth": members[0].depth,
        "link_rate": len(linked) / n,
        "uniq_rate": len(set(linked)) / n if linked else 0.0,
        "tpl_rate": modal([template(l) for l in linked]) if linked else 0.0,
        "sig_rate": modal([m.signature for m in members]),
        "one_link_rate": sum(1 for c in counts if c == 1) / n,
        "med_links": counts[n // 2],
        "med_text": texts[n // 2],
        "med_total_text": totals[n // 2],
        "med_size": sizes[n // 2],
        "head_rate": sum(
            1 for m in members if any(t["tag"] in HEADINGS for t in m.all_texts())
        ) / n,
        "img_rate": sum(1 for m in members if m.all_images()) / n,
        "date_rate": sum(1 for m in members if m.has_date()) / n,
    }


def score(f):
    """How much a group of siblings looks like a list of posts. No single
    value decides it: a tag list inside a card has one link per member and one
    shape, like a post list, and is told apart by what the members hold."""
    content = (
        0.4 * min(f["med_text"] / 40.0, 1.0)
        + 0.3 * f["date_rate"]
        + 0.2 * f["img_rate"]
        + 0.1 * f["head_rate"]
    )
    shape = 0.5 * f["tpl_rate"] + 0.5 * f["sig_rate"]
    size = min(f["n"] / 8.0, 1.0)
    return (
        0.5 * content
        + 0.2 * shape
        + 0.2 * f["one_link_rate"]
        + 0.1 * size * f["link_rate"] * f["uniq_rate"]
    )


def ancestors(node):
    out = set()
    while node is not None:
        out.add(id(node))
        node = node.parent
    return out


def select(page, score_fn=score, alpha=0.8, nesting="split",
           features_fn=group_features):
    """The groups offered to the user, each as the list of its members.

    A group is taken when it reaches a share of the best score of the same
    page. The threshold is relative because the scores of two pages are not
    comparable: a page whose cards carry no date scores lower everywhere.

    A featured card at the top of a list is a group of its own and the list
    sits inside it, so two groups that are taken often overlap. nesting says
    what to do with them:

    - "split" takes out of a group the members that hold another taken group,
      so that the featured card and the list beside it are offered as two
      groups instead of one replacing the other;
    - "none" offers both as they are, which shows the posts of the list twice;
    - "first" keeps the one that scores higher, which is the cheapest for the
      user and loses a list to a featured pair that scores better;
    - "larger" keeps the one with more members.

    features_fn is there so that a search over many settings can read the
    features of a group from a cache instead of reading the tree again.
    """
    scored = sorted(
        ((score_fn(features_fn(g.children)), g) for g in page.groups()),
        key=lambda r: -r[0],
    )
    if not scored:
        return []
    top = scored[0][0]
    taken = [g for value, g in scored if value >= alpha * top]

    if nesting == "split":
        inside = {id(g) for g in taken}
        out = []
        for g in taken:
            kept = [
                m for m in g.children
                if not any(id(n) in inside for n in m.walk())
            ]
            if kept:
                out.append(kept)
        # A group that lost members is scored again, because what is left of
        # the container of a featured card is not what was scored before.
        best = max((score_fn(features_fn(k)) for k in out if len(k) > 1),
                   default=0.0)
        return [
            k for k in out
            if len(k) == 1 or not best or score_fn(features_fn(k)) >= alpha * best
        ]

    if nesting == "none":
        return [g.children for g in taken]

    out = []
    for g in taken:
        clash = [c for c in out if id(g) in ancestors(c) or id(c) in ancestors(g)]
        if not clash:
            out.append(g)
        elif nesting == "larger" and len(g.children) > max(
            len(c.children) for c in clash
        ):
            out = [c for c in out if c not in clash] + [g]
    return [g.children for g in out]


def predicted_posts(groups):
    """The links offered by the groups select returned."""
    out = set()
    for members in groups:
        for link in group_links(members):
            if link:
                out.add(normalize(link))
    return out


def unwrap_texts(node):
    """Merge neighbouring texts that sit in tags with no meaning into one text.

    A list where some cards name the author in a span and others do not gives
    two different shapes to the same list, so the cards stop looking alike.
    Merging the runs removes the difference, because what is compared after it
    is the text of the card rather than the tags it happens to be written in.
    The tag of a merged run is reported as "text".
    """
    for n in node.walk():
        out = []
        for t in n.texts:
            if (
                out
                and t["tag"] in NEUTRAL
                and out[-1]["tag"] in ("text",) + tuple(NEUTRAL)
                and not t.get("datetime")
                and not out[-1].get("datetime")
            ):
                out[-1] = {
                    "tag": "text",
                    "value": out[-1]["value"] + " " + t["value"],
                }
                continue
            out.append(dict(t))
        n.texts = out
    return node
