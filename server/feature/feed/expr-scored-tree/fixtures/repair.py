# Repairs one class of recording error deterministically: a value whose only
# difference from the page is a folded character, such as a curly quote written
# as ASCII. The value is replaced by the text the page really holds. Anything
# that does not match a single page text this way is left alone for verify.py
# to reject.
import json, os, sys
from html.parser import HTMLParser

HERE = os.path.dirname(os.path.abspath(__file__))
FOLD = {"‘": "'", "’": "'", "‚": "'", "‛": "'",
        "“": '"', "”": '"', "„": '"', "′": "'", "″": '"',
        "–": "-", "—": "-", "−": "-", " ": " ",
        "…": "...", "­": ""}


def fold(s):
    for a, b in FOLD.items():
        s = s.replace(a, b)
    return " ".join(s.split())


class Runs(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.runs, self.order = set(), []
    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        if a.get("alt"):
            self.runs.add(" ".join(a["alt"].split()))
    def handle_data(self, d):
        d = " ".join(d.split())
        if d:
            self.runs.add(d)
            self.order.append(d)



def split_runs(order, val):
    """The consecutive page runs that together spell val, or None."""
    want = fold(val).replace(" ", "")
    for i in range(len(order)):
        acc = ""
        for j in range(i, min(i + 8, len(order))):
            acc += fold(order[j]).replace(" ", "")
            if acc == want:
                return order[i:j + 1]
            if not want.startswith(acc):
                break
    return None

for name in sys.argv[1:]:
    r = Runs()
    r.feed(open(os.path.join(HERE, "reduced", name + ".html"), encoding="utf-8").read())
    folded = {}
    for run in r.runs:
        folded.setdefault(fold(run), set()).add(run)
    path = os.path.join(HERE, "new", name + ".json")
    fx = json.load(open(path, encoding="utf-8"))
    fixed = split = 0
    for p in fx.get("posts", []):
        for v in p.get("texts", []):
            val = " ".join(str(v.get("value", "")).split())
            if val in r.runs:
                v["value"] = val
                continue
            cands = folded.get(fold(val))
            if cands and len(cands) == 1:
                v["value"] = next(iter(cands))
                fixed += 1
                continue
            # A value the page writes as several runs, because an element such
            # as <code> sits inside it. Split it back into those runs.
            parts = split_runs(r.order, val)
            if parts:
                # A fragment with no letter and no digit is a separator, which
                # the extractor drops before it reaches the tree.
                parts = [x for x in parts if any(c.isalnum() for c in x)]
            if parts:
                v["value"] = parts[0]
                for extra in parts[1:]:
                    p.setdefault("_extra", []).append({"kind": v.get("kind"), "value": extra})
                split += 1
            else:
                v["value"] = val

    # A value that is the beginning or the end of exactly one run, because the
    # rest of that run was recorded as a separate value and already merged
    # away. The whole run is what rule 4 asks for.
    grown = 0
    for p in fx.get("posts", []):
        for v in p.get("texts", []):
            val = " ".join(str(v.get("value", "")).split())
            if not val or val in r.runs or len(val) < 8:
                continue
            f = fold(val)
            hits = [run for run in r.runs
                    if run != val and (fold(run).startswith(f) or fold(run).endswith(f))]
            if len(hits) == 1:
                v["value"] = hits[0]
                grown += 1

    # The inverse of the split above: values the page writes as a single run,
    # which an agent cut apart at a separator. Rule 4 says a value is a whole
    # run, so they are put back together, and the page's exact text is used.
    merged = 0
    for p in fx.get("posts", []):
        texts = p.get("texts", [])
        used = set()
        for i in range(len(texts)):
            if i in used:
                continue
            vi = " ".join(str(texts[i].get("value", "")).split())
            if vi in r.runs:
                continue
            for j in range(i + 1, len(texts)):
                if j in used:
                    continue
                vj = " ".join(str(texts[j].get("value", "")).split())
                if vj in r.runs:
                    continue
                hit = None
                for sep in (" ", ""):
                    cands = folded.get(fold(sep.join([vi, vj])))
                    if cands and len(cands) == 1:
                        hit = next(iter(cands))
                        break
                if hit:
                    texts[i]["value"] = hit
                    used.add(j)
                    merged += 1
                    break
        if used:
            p["texts"] = [t for k, t in enumerate(texts) if k not in used]

    for p in fx.get("posts", []):
        if "_extra" in p:
            p.setdefault("texts", []).extend(p.pop("_extra"))

    # required follows from the kind, so it is set here rather than trusted.
    REQUIRED = {"title", "description", "pub-date", "thumbnail"}
    flags = 0
    for p in fx.get("posts", []):
        for v in p.get("texts", []) + p.get("images", []):
            want = v.get("kind") in REQUIRED
            if bool(v.get("required")) != want:
                v["required"] = want
                flags += 1
    json.dump(fx, open(path, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
    print(f"{name}: repaired {fixed} values, split {split}, merged {merged}, grown {grown}, {flags} required flags")
