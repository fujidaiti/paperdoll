# Same probe, but each card's region is bounded by the neighbouring posts'
# links in document order, so a neighbour's values cannot leak in.
import json, os, sys
from html.parser import HTMLParser

HERE = os.path.dirname(os.path.abspath(__file__))
name = sys.argv[1]
raw = open(os.path.join(HERE, "reduced", name + ".html"), encoding="utf-8").read()
fx = json.load(open(os.path.join(HERE, "new", name + ".json"), encoding="utf-8"))

class Runs(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.items = []
    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        if a.get("src"): self.items.append(("img", a["src"]))
        if a.get("datetime"): self.items.append(("dt", a["datetime"].strip()))
    def handle_data(self, d):
        d = " ".join(d.split())
        if d: self.items.append(("text", d))

# Last occurrence of each post link, in document order.
spots = []
for p in fx["posts"]:
    i = raw.rfind('href="' + p["link"] + '"')
    spots.append((i, p))
spots.sort()
missing = {}
for n, (i, p) in enumerate(spots):
    lo = (spots[n-1][0] + i) // 2 if n > 0 else max(0, i - 1200)
    hi = (spots[n+1][0] + i) // 2 if n + 1 < len(spots) else min(len(raw), i + 1200)
    have = {" ".join(v["value"].split()) for v in p.get("texts", []) + p.get("images", [])}
    r = Runs(); r.feed(raw[lo:hi])
    for k, v in r.items:
        if v in have or v == p["link"]:
            continue
        missing.setdefault(v, [0, k])
        missing[v][0] += 1
print(f"distinct values inside a card region that the fixture does not record: {len(missing)}")
tot = sum(v[0] for v in missing.values())
print(f"occurrences: {tot} over {len(fx['posts'])} posts = {tot/len(fx['posts']):.1f} per post\n")
for v, (n, k) in sorted(missing.items(), key=lambda x: -x[1][0])[:22]:
    print(f"  x{n:<3} [{k}] {v[:90]}")
