# Copies the recorded fixtures over the ones the tests read. It refuses to run
# while any fixture still fails verification, and it lists what it would do
# unless --write is given, because the two test readers must be rewritten for
# the new schema in the same change.
import glob, json, os, shutil, subprocess, sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEST = os.path.abspath(os.path.join(HERE, "..", "..", "testdata"))
BACKUP = os.path.join(HERE, "replaced")
names = sorted(os.path.basename(f)[:-5] for f in glob.glob(os.path.join(HERE, "new", "*.json"))
               if ".part" not in f)
subprocess.run([sys.executable, "repair.py"] + names, cwd=HERE, stdout=subprocess.DEVNULL)
out = subprocess.run([sys.executable, "verify.py"] + names, cwd=HERE,
                     capture_output=True, text=True).stdout
bad = [l for l in out.splitlines() if l.startswith("  ERROR")]
if bad:
    print(f"{len(bad)} errors outstanding; nothing copied")
    for l in bad[:10]:
        print(l)
    sys.exit(1)
write = "--write" in sys.argv
total = 0
for n in names:
    src = os.path.join(HERE, "new", n + ".json")
    dst = os.path.join(DEST, n + ".fixture.json")
    posts = len(json.load(open(src, encoding="utf-8"))["posts"])
    total += posts
    print(f"{'copy' if write else 'would copy'} {n:<40} {posts:>4} posts")
    if write:
        # Most of these files are untracked, so the values they hold would be
        # lost for good without a copy.
        # The copy is written once and never replaced, so running this twice
        # cannot overwrite the values it was meant to preserve.
        keep = os.path.join(BACKUP, n + ".fixture.json")
        if os.path.exists(dst) and not os.path.exists(keep):
            os.makedirs(BACKUP, exist_ok=True)
            shutil.copyfile(dst, keep)
        shutil.copyfile(src, dst)
print(f"\n{len(names)} pages, {total} posts" + ("" if write else "  (pass --write to apply)"))
