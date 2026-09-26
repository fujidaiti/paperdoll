package scoredtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// threshold is the score a node has to reach to be called a post. It is an
// absolute number rather than a share of the best score of the page, which the
// group score needs, because the agreement terms make the score comparable
// between pages.
const threshold = 0.68

// scoredDir is where the scored trees are written, one JSON file per page.
const scoredDir = "testdata/scored"

type fixture struct {
	URL   string `json:"url"`
	Posts []struct {
		Link string `json:"link"`
	} `json:"posts"`
}

type page struct {
	name  string
	root  *Node
	posts map[string]bool
}

func load(t *testing.T) []page {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "testdata", "*.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []page
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".fixture.json")
		tree := filepath.Join("..", "testdata", "semantic", name+".json")
		if _, err := os.Stat(tree); err != nil {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var f fixture
		if err := json.Unmarshal(raw, &f); err != nil {
			t.Fatal(err)
		}
		if len(f.Posts) == 0 {
			continue
		}
		root, err := Load(tree)
		if err != nil {
			t.Fatal(err)
		}
		if len(root.Links()) == 0 {
			continue
		}
		FoldCards(root)
		posts := map[string]bool{}
		for _, p := range f.Posts {
			posts[Normalize(p.Link)] = true
		}
		out = append(out, page{name, root, posts})
	}
	if len(out) == 0 {
		t.Fatal("no page found")
	}
	return out
}

// truePosts returns the nodes that are a post: the largest node whose subtree
// covers exactly one URL the fixture lists. The largest one is taken because
// every node between the card and the link covers that same post, and the card
// is the one the selection screen would show.
func (p page) truePosts() map[*Node]bool {
	covers := func(n *Node) int {
		seen := map[string]bool{}
		for _, l := range n.Links() {
			if p.posts[Normalize(l)] {
				seen[Normalize(l)] = true
			}
		}
		return len(seen)
	}
	out := map[*Node]bool{}
	for _, n := range p.root.Walk() {
		if covers(n) != 1 {
			continue
		}
		if n.Parent != nil && covers(n.Parent) == 1 {
			continue
		}
		out[n] = true
	}
	return out
}

// candidates are the nodes a label could be given to. A node whose subtree
// holds no link can never be a post, so it is left out of the counts.
func candidates(root *Node) []*Node {
	var out []*Node
	for _, n := range root.Walk() {
		if len(n.Links()) > 0 {
			out = append(out, n)
		}
	}
	return out
}

// TestNodeScore measures how well the score separates the post nodes from
// every other node, page by page and over the corpus.
func TestNodeScore(t *testing.T) {
	pages := load(t)
	var tp, fp, fn int
	fmt.Printf("%-40s %6s %6s %6s %6s %9s %9s\n",
		"page", "nodes", "posts", "marked", "hit", "precision", "recall")
	for _, p := range pages {
		ScoreNodes(p.root, threshold)
		want := p.truePosts()
		var pTP, pFP, pFN int
		for _, n := range candidates(p.root) {
			switch {
			case n.Post && want[n]:
				pTP++
			case n.Post:
				pFP++
			case want[n]:
				pFN++
			}
		}
		tp, fp, fn = tp+pTP, fp+pFP, fn+pFN
		fmt.Printf("%-40s %6d %6d %6d %6d %9s %9s\n",
			p.name, len(candidates(p.root)), pTP+pFN, pTP+pFP, pTP,
			rate(pTP, pTP+pFP), rate(pTP, pTP+pFN))
	}
	fmt.Printf("\ntotal: precision %s, recall %s, marked %d, posts %d\n",
		rate(tp, tp+fp), rate(tp, tp+fn), tp+fp, tp+fn)
}

// TestThresholdSweep prints the separation at every threshold, so that the one
// the package uses can be read off rather than guessed.
func TestThresholdSweep(t *testing.T) {
	pages := load(t)
	type sample struct {
		score float64
		post  bool
	}
	var all []sample
	for _, p := range pages {
		ScoreNodes(p.root, threshold)
		want := p.truePosts()
		for _, n := range candidates(p.root) {
			all = append(all, sample{n.Score, want[n]})
		}
	}
	fmt.Printf("%9s %9s %9s %6s\n", "threshold", "precision", "recall", "f1")
	for x := 40; x <= 90; x += 2 {
		cut := float64(x) / 100
		var tp, fp, fn int
		for _, s := range all {
			switch {
			case s.score >= cut && s.post:
				tp++
			case s.score >= cut:
				fp++
			case s.post:
				fn++
			}
		}
		f1 := 0.0
		if tp > 0 {
			f1 = 2 * float64(tp) / float64(2*tp+fp+fn)
		}
		fmt.Printf("%9.2f %9s %9s %6.3f\n", cut, rate(tp, tp+fp), rate(tp, tp+fn), f1)
	}
}

// TestGroups measures the review cost of the rows the grouping produces: how
// many rows the page shows, how many of them the user has to tick to reach
// every post, and how many of the fixture posts the ticked rows return.
func TestGroups(t *testing.T) {
	pages := load(t)
	var rows, ticks, found, want int
	fmt.Printf("%-40s %6s %6s %8s %8s %6s\n",
		"page", "rows", "ticks", "found", "posts", "recall")
	for _, p := range pages {
		ScoreNodes(p.root, threshold)
		groups := Groups(p.root)

		// The rows the user ticks, chosen greedily: the row that adds the most
		// posts, then the next one, until nothing is added. This is the same
		// cover the feed package measures its groups with.
		left := map[string]bool{}
		for u := range p.posts {
			left[u] = true
		}
		pTicks, pFound := 0, 0
		for {
			best, gain := -1, 0
			for i, g := range groups {
				adds := map[string]bool{}
				for _, l := range g.Links {
					if l != "" && left[Normalize(l)] {
						adds[Normalize(l)] = true
					}
				}
				if len(adds) > gain {
					best, gain = i, len(adds)
				}
			}
			if best < 0 {
				break
			}
			for _, l := range groups[best].Links {
				delete(left, Normalize(l))
			}
			pTicks++
			pFound += gain
		}
		rows += len(groups)
		ticks += pTicks
		found += pFound
		want += len(p.posts)
		fmt.Printf("%-40s %6d %6d %8d %8d %6s\n",
			p.name, len(groups), pTicks, pFound, len(p.posts),
			rate(pFound, len(p.posts)))
	}
	fmt.Printf("\ntotal: rows %d, ticks %d, rows per tick %.2f, URL recall %s\n",
		rows, ticks, float64(rows)/float64(ticks), rate(found, want))
}

// TestWriteScoredTrees writes the scored tree of every page, so that a change
// to the score can be read in a diff the way the semantic trees already are.
func TestWriteScoredTrees(t *testing.T) {
	pages := load(t)
	if err := os.MkdirAll(scoredDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range pages {
		ScoreNodes(p.root, threshold)
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetIndent("", "  ")
		// A text of the page may hold "<", ">" or "&", and an encoder writes
		// them as escape sequences unless this is turned off.
		enc.SetEscapeHTML(false)
		if err := enc.Encode(p.root); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(scoredDir, p.name+".json")
		if err := os.WriteFile(out, b.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	names := make([]string, 0, len(pages))
	for _, p := range pages {
		names = append(names, p.name)
	}
	sort.Strings(names)
	fmt.Printf("wrote %d scored trees into %s\n", len(names), scoredDir)
}

func rate(a, b int) string {
	if b == 0 {
		return "-"
	}
	return fmt.Sprintf("%.3f", float64(a)/float64(b))
}
