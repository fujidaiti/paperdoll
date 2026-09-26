# Reduces a saved page to the elements and attributes a post value can come
# from. Deterministic: it never decides what belongs to a post, it only removes
# what can hold no value at all, and resolves every URL against the page.
import json, glob, os, re, sys
from html.parser import HTMLParser
from urllib.parse import urljoin

DATA = "/Users/fujidaiti/Dev/paperdoll/server/feature/feed/testdata"
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "reduced")
DROP = {"script", "style", "noscript", "svg", "path", "head", "template"}
VOID = {"img", "br", "hr", "input", "source", "meta", "link"}


class Reducer(HTMLParser):
    def __init__(self, page):
        super().__init__(convert_charrefs=True)
        self.page, self.out, self.skip = page, [], 0

    def handle_starttag(self, tag, attrs):
        if tag in DROP:
            self.skip += 1
            return
        if self.skip:
            return
        a = dict(attrs)
        keep = []
        if a.get("href"):
            keep.append(("href", urljoin(self.page, a["href"])))
        src = a.get("src", "")
        if not src or src.lower().startswith("data:"):
            src = a.get("data-src", "") or a.get("data-lazy-src", "")
        if src and not src.lower().startswith("data:"):
            keep.append(("src", urljoin(self.page, src)))
        if a.get("alt"):
            keep.append(("alt", " ".join(a["alt"].split())))
        if a.get("datetime"):
            keep.append(("datetime", a["datetime"].strip()))
        s = "".join(f' {k}="{v}"' for k, v in keep)
        self.out.append(f"<{tag}{s}>")
        if tag in VOID:
            self.out.append(f"</{tag}>")

    def handle_endtag(self, tag):
        if tag in DROP:
            self.skip = max(0, self.skip - 1)
            return
        if self.skip or tag in VOID:
            return
        self.out.append(f"</{tag}>")

    def handle_data(self, d):
        if self.skip:
            return
        d = " ".join(d.split())
        if d:
            self.out.append(d)


os.makedirs(OUT, exist_ok=True)
rows = []
for f in sorted(glob.glob(os.path.join(DATA, "*.html"))):
    name = os.path.basename(f)[:-5]
    fx = os.path.join(DATA, name + ".fixture.json")
    if not os.path.exists(fx):
        continue
    page = json.load(open(fx, encoding="utf-8")).get("url", "")
    raw = open(f, encoding="utf-8", errors="replace").read()
    r = Reducer(page)
    r.feed(raw)
    s = "".join(r.out)
    while True:
        s2 = re.sub(r"<([a-z0-9]+)>\s*</\1>", "", s)
        if s2 == s:
            break
        s = s2
    dst = os.path.join(OUT, name + ".html")
    open(dst, "w", encoding="utf-8").write(s)
    rows.append((name, page, len(raw), len(s)))

rows.sort(key=lambda r: -r[3])
print(f"{'page':<40} {'raw':>9} {'reduced':>9} {'~tok':>7}")
for n, u, a, b in rows:
    print(f"{n:<40} {a:>9} {b:>9} {b//4:>7}")
print(f"\n{len(rows)} pages, {sum(r[3] for r in rows)} bytes reduced, "
      f"~{sum(r[3] for r in rows)//4} tokens total")
