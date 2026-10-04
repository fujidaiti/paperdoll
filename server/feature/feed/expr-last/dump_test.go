package structures

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// treeDir is where the structures are written, one file per page.
const treeDir = "tree"

// maxSamples is how many values are shown for one field, and maxValue,
// maxBlob and maxPath how long a value, the text of a blob and a path are
// allowed to be. All four are there to keep the file readable on pages whose
// class names are generated.
const (
	maxSamples = 3
	maxValue   = 110
	maxBlob    = 220
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
		out := filepath.Join(treeDir, p.name+".tree")
		if err := os.WriteFile(out, []byte(dumpPage(p, Build(p.doc, p.base))), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// dumpPage writes the structures of one page as the text of its tree file.
func dumpPage(p page, ss []*Structure) string {
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
	sb.WriteString("Under it are up to 3 of the values it read. A blob line points at the\n")
	sb.WriteString("appendix at the end of the file instead, because what a blob holds is\n")
	sb.WriteString("the text of a whole run of nodes rather than a level of the page.\n")
	fmt.Fprintf(&sb, "A path longer than %d characters is cut.\n", maxPath)

	app := &appendix{at: map[*Field]int{}, runs: map[*Field][][]*html.Node{}}
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
				if read.Field.Blob != nil {
					app.runs[read.Field] = append(
						app.runs[read.Field], read.Field.Blob(pi.Node))
					continue
				}
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
		dumpPart(&sb, s.Root, 1, parts, reads, values, app)
	}

	if len(app.list) > 0 {
		sb.WriteString("\n\n== appendix: what the blobs hold ==\n")
		sb.WriteString("A blob reads a run of sibling nodes as one piece, so what it holds is\n")
		sb.WriteString("the text of that whole run and not one value per node. It is written\n")
		sb.WriteString("here rather than in the tree because it is one long piece of text\n")
		sb.WriteString("rather than a level of the page.\n")
		for i, f := range app.list {
			runs := app.runs[f]
			fmt.Fprintf(&sb, "\nblob %d  %s  ×%d subtrees\n",
				i+1, cut(f.Name, maxPath), len(runs))
			for _, run := range runs[:min(len(runs), maxSamples)] {
				fmt.Fprintf(&sb, "  run of %d nodes\n", len(run))
				fmt.Fprintf(&sb, "  %q\n", cut(runText(run), maxBlob))
			}
			if len(runs) > maxSamples {
				fmt.Fprintf(&sb, "  (%d runs)\n", len(runs))
			}
		}
	}
	return sb.String()
}

// An appendix collects the blobs met while the tree is written, so that what
// each of them holds is written below the tree and numbered rather than in the
// middle of it.
type appendix struct {
	at   map[*Field]int
	list []*Field
	runs map[*Field][][]*html.Node
}

func (a *appendix) mark(f *Field) int {
	if n, ok := a.at[f]; ok {
		return n
	}
	a.list = append(a.list, f)
	a.at[f] = len(a.list)
	return len(a.list)
}

// dumpPart writes one part, its fields and everything below it.
func dumpPart(sb *strings.Builder, p *Part, depth int,
	parts map[*Part]int, reads map[*Field]int, values map[*Field][]string,
	app *appendix,
) {
	ind := strings.Repeat("  ", depth)
	fmt.Fprintf(sb, "%spart   %s  ×%d\n", ind, cut(p.Name, maxPath), parts[p])
	for _, f := range p.Fields {
		if f.Blob != nil {
			fmt.Fprintf(sb, "%s  %-6s %s  ×%d  -> blob %d\n",
				ind, "blob", cut(f.Name, maxPath), reads[f], app.mark(f))
			continue
		}
		fmt.Fprintf(sb, "%s  %-6s %s  ×%d\n",
			ind, "field", cut(f.Name, maxPath), reads[f])
		writeSamples(sb, ind+"    ", values[f])
	}
	for _, c := range p.Children {
		dumpPart(sb, c, depth+1, parts, reads, values, app)
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

// runText is every text below the nodes of one run, run together, which is how
// a blob is read: one piece rather than one value per node.
func runText(run []*html.Node) string {
	var parts []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if !observable(n) {
			return
		}
		if n.Type == html.TextNode {
			if s := flat(n.Data); s != "" {
				parts = append(parts, s)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range run {
		walk(n)
	}
	return strings.Join(parts, " ")
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// A blob is a leaf of the output, so the tree names it and the text it holds
// is written once in the appendix.
func TestDumpWritesABlobInTheAppendix(t *testing.T) {
	p := testPage(t, blobBody, blobPost())
	ss := []*Structure{{Root: &Part{
		Name:  "div.card",
		Match: byClass("card"),
		Fields: []*Field{
			{Name: "a.t", Match: firstClass("t")},
			{Name: "div.body > p", Blob: childrenOf("body")},
		},
	}}}

	out := dumpPage(p, ss)
	for _, want := range []string{
		"blob   div.body > p  ×1  -> blob 1",
		"== appendix: what the blobs hold ==",
		"blob 1  div.body > p  ×1 subtrees",
		"run of 2 nodes",
		`"First para Second para inner"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the tree file does not hold %q:\n%s", want, out)
		}
	}
}

// A page with no blob has no appendix.
func TestDumpWritesNoAppendixWithoutABlob(t *testing.T) {
	p := testPage(t, cardsBody, cardsPosts()...)
	if out := dumpPage(p, Build(p.doc, p.base)); strings.Contains(out, "== appendix") {
		t.Errorf("the tree file holds an appendix:\n%s", out)
	}
}
