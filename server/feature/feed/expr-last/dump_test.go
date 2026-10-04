package structures

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// treeDir is where the structures are written, one file per page.
const treeDir = "tree"

// maxSamples is how many values are shown for one field, and maxValue and
// maxPath how long a value and a path are allowed to be. All three are there
// to keep the file readable on pages whose class names are generated.
const (
	maxSamples = 3
	maxValue   = 110
	maxPath    = 160
)

// TestWriteTrees writes the structures of each page, one file per page, as an
// indented tree with the values each field reads, so that the output can be
// read rather than only counted.
func TestWriteTrees(t *testing.T) {
	if err := os.MkdirAll(treeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range load(t) {
		ss := Build(p.doc, p.base)
		r := measure(p, ss)

		var sb strings.Builder
		fmt.Fprintf(&sb, "# %s\n", p.name)
		fmt.Fprintf(&sb, "url               %s\n", p.base)
		fmt.Fprintf(&sb, "posts in fixture  %d\n", len(p.posts))
		fmt.Fprintf(&sb, "structures        %d\n", r.structures)
		fmt.Fprintf(&sb, "recall            %s, recall+ %s\n",
			share(r.match, r.posts), share(r.matchExact, r.posts))
		fmt.Fprintf(&sb, "merge             %.2f\n", mean(r.merge))
		fmt.Fprintf(&sb, "distrib           %.2f, wrappers %.2f\n\n",
			mean(r.distrib), mean(r.wrappers))
		sb.WriteString("A part line is one part: the path its matcher reads, as direct child\n")
		sb.WriteString("steps from the page for a root part and from the part above it for any\n")
		sb.WriteString("other, and how many subtrees it matched in the page. A field line is\n")
		sb.WriteString("one field: the path its matcher reads inside the subtree, self when it\n")
		sb.WriteString("reads that subtree itself, and in how many subtrees it found a node.\n")
		sb.WriteString("Under it are up to 3 of the values it read.\n")
		fmt.Fprintf(&sb, "A path longer than %d characters is cut.\n", maxPath)

		for i, s := range ss {
			instances := Extract(s, p.doc, p.base)

			// What each part and each field of this structure produced in the
			// page, so that the tree shows what the matchers reach rather than
			// what they were read from.
			parts := map[*Part]int{}
			reads := map[*Field]int{}
			values := map[*Field][]string{}
			var walk func(*PartInstance)
			walk = func(pi *PartInstance) {
				parts[pi.Part]++
				for _, read := range pi.Read {
					reads[read.Field]++
					values[read.Field] = append(values[read.Field], read.Values...)
				}
				for _, c := range pi.Children {
					walk(c)
				}
			}
			for _, in := range instances {
				walk(in.Root)
			}

			fmt.Fprintf(&sb, "\nstructure %d  ×%d instances\n", i+1, len(instances))
			dumpPart(&sb, s.Root, 1, parts, reads, values)
		}

		out := filepath.Join(treeDir, p.name+".tree")
		if err := os.WriteFile(out, []byte(sb.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// dumpPart writes one part, its fields and everything below it.
func dumpPart(sb *strings.Builder, p *Part, depth int,
	parts map[*Part]int, reads map[*Field]int, values map[*Field][]string,
) {
	ind := strings.Repeat("  ", depth)
	fmt.Fprintf(sb, "%spart   %s  ×%d\n", ind, cut(p.Name, maxPath), parts[p])
	for _, f := range p.Fields {
		kind := "field"
		if f.Blob != nil {
			kind = "blob"
		}
		fmt.Fprintf(sb, "%s  %-6s %s  ×%d\n",
			ind, kind, cut(f.Name, maxPath), reads[f])
		writeSamples(sb, ind+"    ", values[f])
	}
	for _, c := range p.Children {
		dumpPart(sb, c, depth+1, parts, reads, values)
	}
}

// writeSamples prints up to maxSamples distinct values of one field.
func writeSamples(sb *strings.Builder, ind string, vals []string) {
	seen := map[string]bool{}
	var keep []string
	for _, v := range vals {
		if seen[v] {
			continue
		}
		seen[v] = true
		if len(keep) < maxSamples {
			keep = append(keep, v)
		}
	}
	for _, v := range keep {
		fmt.Fprintf(sb, "%s%q\n", ind, cut(v, maxValue))
	}
	if len(seen) > len(keep) {
		fmt.Fprintf(sb, "%s(%d values)\n", ind, len(vals))
	}
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
