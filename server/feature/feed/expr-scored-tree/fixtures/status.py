# Runs the repair and the verifier over every fixture recorded so far, and
# prints one line per page with the gap probe's per-post figure beside it.
import glob, json, os, subprocess, sys

HERE = os.path.dirname(os.path.abspath(__file__))
names = sorted(os.path.basename(f)[:-5] for f in glob.glob(os.path.join(HERE, "new", "*.json")))
if not names:
    sys.exit("nothing recorded yet")
subprocess.run([sys.executable, "repair.py"] + names, cwd=HERE, stdout=subprocess.DEVNULL)
out = subprocess.run([sys.executable, "verify.py"] + names, cwd=HERE,
                     capture_output=True, text=True).stdout
bad = {}
for line in out.splitlines():
    if line.startswith("  ERROR"):
        bad[cur] = bad.get(cur, 0) + 1
    elif ":" in line and " posts," in line:
        cur = line.split(":")[0]
print(f"{'page':<40} {'posts':>6} {'errors':>7} {'kinds'}")
tot = 0
for n in names:
    d = json.load(open(os.path.join(HERE, "new", n + ".json"), encoding="utf-8"))
    ks = {}
    for p in d.get("posts", []):
        for v in p.get("texts", []) + p.get("images", []):
            ks[v.get("kind")] = ks.get(v.get("kind"), 0) + 1
    tot += len(d.get("posts", []))
    order = sorted(ks.items(), key=lambda x: -x[1])
    print(f"{n:<40} {len(d.get('posts', [])):>6} {bad.get(n, 0):>7} "
          + " ".join(f"{k}:{v}" for k, v in order))
print(f"\n{len(names)} pages recorded, {tot} posts, {sum(bad.values())} errors")
