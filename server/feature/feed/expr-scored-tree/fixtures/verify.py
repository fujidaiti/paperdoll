# Checks a recorded fixture against the reduced page it was read from. Nothing
# here judges whether a value belongs to a post; it only proves that every
# recorded value is really written in the page, and that the schema holds.
import json, os, sys, re
from html.parser import HTMLParser

HERE = os.path.dirname(os.path.abspath(__file__))
OUTDIR = os.environ.get("OUTDIR", "new")
TEXT_KINDS = {"title", "description", "pub-date", "update-date", "author",
              "category", "reading-time", "body"}
IMAGE_KINDS = {"thumbnail", "avatar"}
REQUIRED_KINDS = {"title", "description", "pub-date", "thumbnail"}
# A value may arrive as several runs, for example a title written around a
# <code> element, so no kind is limited to one value.
AT_MOST_ONCE = set()


class Page(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.runs, self.urls, self.datetimes, self.alts = set(), set(), set(), set()
        self.flat = []

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        for k in ("href", "src"):
            if a.get(k):
                self.urls.add(a[k])
        if a.get("datetime"):
            self.datetimes.add(a["datetime"].strip())
        if a.get("alt"):
            self.alts.add(" ".join(a["alt"].split()))

    def handle_data(self, d):
        d = " ".join(d.split())
        if d:
            self.runs.add(d)
            self.flat.append(d)


def norm(s):
    return " ".join(str(s).split())


def check(name):
    page = Page()
    page.feed(open(os.path.join(HERE, "reduced", name + ".html"), encoding="utf-8").read())
    flat = " ".join(page.flat)
    fx = json.load(open(os.path.join(HERE, OUTDIR, name + ".json"), encoding="utf-8"))
    errs, warns = [], []
    seen = {}
    for i, p in enumerate(fx.get("posts", [])):
        at = f"post {i}"
        link = norm(p.get("link", ""))
        if not link:
            errs.append(f"{at}: no link")
        elif link in seen:
            errs.append(f"{at}: link already recorded by post {seen[link]}: {link}")
        else:
            seen[link] = i
        if link and link not in page.urls:
            errs.append(f"{at}: link is not an href in the page: {link}")
        counts = {}
        for v in p.get("texts", []):
            kind, val = v.get("kind"), norm(v.get("value", ""))
            counts[kind] = counts.get(kind, 0) + 1
            if kind is not None and kind not in TEXT_KINDS:
                errs.append(f"{at}: text kind not allowed: {kind!r}")
            want = kind in REQUIRED_KINDS
            if bool(v.get("required")) != want:
                errs.append(f"{at}: required should be {want} for kind {kind!r}")
            if not val:
                errs.append(f"{at}: empty text value")
            elif val in page.runs or val in page.datetimes or val in page.alts:
                pass
            elif val in flat:
                errs.append(f"{at}: text spans several elements, so it is not one run "
                            f"of the page: {val[:60]!r}")
            else:
                errs.append(f"{at}: text is not written in the page: {val[:70]!r}")
        for v in p.get("images", []):
            kind, val = v.get("kind"), norm(v.get("value", ""))
            counts[kind] = counts.get(kind, 0) + 1
            if kind is not None and kind not in IMAGE_KINDS:
                errs.append(f"{at}: image kind not allowed: {kind!r}")
            want = kind in REQUIRED_KINDS
            if bool(v.get("required")) != want:
                errs.append(f"{at}: required should be {want} for kind {kind!r}")
            if val not in page.urls:
                errs.append(f"{at}: image is not a src in the page: {val[:70]!r}")
        for k in AT_MOST_ONCE:
            if counts.get(k, 0) > 1:
                errs.append(f"{at}: {counts[k]} values of kind {k!r}, at most one allowed")
        if "links" in p:
            errs.append(f"{at}: a 'links' list is not part of the schema")
        if counts.get("title", 0) == 0:
            warns.append(f"{at}: no title")
    print(f"{name}: {len(fx.get('posts', []))} posts, {len(errs)} errors, {len(warns)} warnings")
    for e in errs[:40]:
        print("  ERROR  " + e)
    for w in warns[:20]:
        print("  warn   " + w)
    return len(errs)


sys.exit(min(1, sum(check(n) for n in sys.argv[1:])))
