# Joins the chunked recordings of one page into a single fixture, in the order
# of the page's post list, and reports anything the chunks missed or repeated.
import glob, json, os, sys
HERE = os.path.dirname(os.path.abspath(__file__))
for name in sys.argv[1:]:
    parts = sorted(glob.glob(os.path.join(HERE, "new", name + ".part*.json")))
    if not parts:
        print(f"{name}: no parts"); continue
    want = json.load(open(os.path.join(HERE, "lists", name + ".json"), encoding="utf-8"))["links"]
    order = {u: i for i, u in enumerate(want)}
    posts, seen, stray = [], set(), []
    for f in parts:
        for p in json.load(open(f, encoding="utf-8"))["posts"]:
            u = p["link"]
            if u in seen:
                continue
            seen.add(u)
            if u not in order:
                stray.append(u)
            posts.append(p)
    posts.sort(key=lambda p: order.get(p["link"], 10**6))
    missing = [u for u in want if u not in seen]
    out = {"url": json.load(open(parts[0], encoding="utf-8"))["url"], "posts": posts}
    json.dump(out, open(os.path.join(HERE, "new", name + ".json"), "w", encoding="utf-8"),
              ensure_ascii=False, indent=2)
    print(f"{name}: {len(parts)} parts -> {len(posts)} posts, {len(missing)} missing, {len(stray)} not in the list")
    for u in missing[:5]:
        print("   missing:", u[:95])
    for u in stray[:5]:
        print("   stray  :", u[:95])
