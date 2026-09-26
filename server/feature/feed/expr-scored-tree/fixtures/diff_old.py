# Compares the recorded post links with the old fixture's, as a second opinion.
# The old values are not trusted; this only points at pages worth a second look.
import glob, json, os, sys
HERE = os.path.dirname(os.path.abspath(__file__))
OLD = os.path.join(HERE, "..", "..", "testdata")
def norm(u):
    u = u.split("#")[0].rstrip("/")
    for p in ("https://", "http://", "www."):
        if u.startswith(p): u = u[len(p):]
    return u.lower()
print(f"{'page':<40} {'new':>4} {'old':>4} {'only-new':>9} {'only-old':>9}")
for f in sorted(glob.glob(os.path.join(HERE, "new", "*.json"))):
    n = os.path.basename(f)[:-5]
    new = {norm(p["link"]) for p in json.load(open(f, encoding="utf-8"))["posts"]}
    o = os.path.join(OLD, n + ".fixture.json")
    old = {norm(p["url"]) for p in json.load(open(o, encoding="utf-8"))["posts"]} if os.path.exists(o) else set()
    print(f"{n:<40} {len(new):>4} {len(old):>4} {len(new - old):>9} {len(old - new):>9}")
    if len(old - new) > 2:
        for u in sorted(old - new)[:6]:
            print(f"      only in the old fixture: {u[:90]}")
