package structures

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// requiredKinds are the kinds of value a post is not worth reporting without.
// DEF.md says the evaluator holds this list rather than reading the required
// flag of the fixture files, so that it can be changed without rewriting them.
var requiredKinds = map[string]bool{"link": true, "title": true}

// A fixture is the ideal result for one saved page, written by hand. It holds
// every value the page shows beside each post, so the distance between it and
// what Build produces is what the metrics measure.
type fixture struct {
	URL   string        `json:"url"`
	Posts []fixturePost `json:"posts"`
}

type fixturePost struct {
	Link   string         `json:"link"`
	Texts  []fixtureValue `json:"texts"`
	Images []fixtureValue `json:"images"`
}

type fixtureValue struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// A postValue is one recorded value of a post, normalized the way the values
// of a node are, so that the two can be compared by equality.
type postValue struct {
	kind  string
	value string
}

func (p fixturePost) values(base *url.URL) []postValue {
	out := []postValue{{kind: "link", value: resolve(base, p.Link)}}
	for _, t := range p.Texts {
		if v := flat(t.Value); v != "" {
			out = append(out, postValue{kind: t.Kind, value: v})
		}
	}
	for _, i := range p.Images {
		if v := resolve(base, i.Value); v != "" {
			out = append(out, postValue{kind: i.Kind, value: v})
		}
	}
	return out
}

type page struct {
	name  string
	base  *url.URL
	doc   *html.Node
	posts []fixturePost
}

// load reads the saved pages that record at least one post. The four pages
// that record none carry no metric.
func load(t *testing.T) []page {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "testdata", "*.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []page
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".fixture.json")
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var f fixture
		if err := json.Unmarshal(raw, &f); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(f.Posts) == 0 {
			continue
		}
		base, err := url.Parse(f.URL)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		saved, err := os.Open(filepath.Join("..", "testdata", name+".html"))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := html.Parse(saved)
		_ = saved.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out = append(out, page{name: name, base: base, doc: doc, posts: f.Posts})
	}
	return out
}

// A result holds one page measured. The means are kept as the numbers they are
// taken over, so that the totals of several pages can be summed before the
// means are taken.
type result struct {
	name  string
	posts int

	match      int // posts that match an instance
	matchExact int

	merge       []int // per instance that matches a post, how many it stands for
	distrib     []int // per matched post, the parts it is spread over
	distribPlus []int
	wrappers    []int

	structures int
	depth      int

	bodyValues int // fixture values of kind body
	bodyInBlob int
	bodySplit  []int // per post with a body, the blobs it is spread over
	nonBody    int   // fixture values of any other kind
	leak       int
	blobs      int
}

// measure takes the structures rather than calling Build, so that the metrics
// can be checked against structures written by hand.
func measure(p page, ss []*Structure) result {
	r := result{name: p.name, posts: len(p.posts)}

	r.structures = len(ss)
	var instances []*Instance
	for _, s := range ss {
		if d := s.Depth(); d > r.depth {
			r.depth = d
		}
		instances = append(instances, Extract(s, p.doc, p.base)...)
	}
	held := make([]map[string]bool, len(instances))
	for i, in := range instances {
		held[i] = in.valueSet()
	}

	byBlob := blobReads(instances)
	r.blobs = len(blobFields(ss))

	stands := make([]int, len(instances))
	for _, post := range p.posts {
		recorded := post.values(p.base)

		// The values of the post, split into the ones it must carry to be
		// matched at all and the whole set exact matching asks for.
		required, every := map[string]bool{}, map[string]bool{}
		for _, v := range recorded {
			every[v.value] = true
			if requiredKinds[v.kind] {
				required[v.value] = true
			}
		}
		if len(required) == 0 {
			continue
		}

		// The cheapest instance is the one taken, because a post the output
		// reaches twice costs the user only the cheaper of the two reviews.
		best, bestWrap, bestPlus := -1, 0, -1
		for i, in := range instances {
			if !covers(held[i], required) {
				continue
			}
			stands[i]++

			// Every value of the post the instance carries has to be reached,
			// not only the required ones.
			want := map[string]bool{}
			for v := range every {
				if held[i][v] {
					want[v] = true
				}
			}
			distrib, wrappers := in.cost(want)
			if best < 0 || distrib < best || (distrib == best && wrappers < bestWrap) {
				best, bestWrap = distrib, wrappers
			}
			if covers(held[i], every) && (bestPlus < 0 || distrib < bestPlus) {
				bestPlus = distrib
			}
		}
		if best >= 0 {
			r.match++
			r.distrib = append(r.distrib, best)
			r.wrappers = append(r.wrappers, bestWrap)
		}
		if bestPlus >= 0 {
			r.matchExact++
			r.distribPlus = append(r.distribPlus, bestPlus)
		}

		blobsOfPost := map[*Field]bool{}
		for _, v := range recorded {
			if v.kind == "body" {
				r.bodyValues++
				if len(byBlob[v.value]) > 0 {
					r.bodyInBlob++
					for f := range byBlob[v.value] {
						blobsOfPost[f] = true
					}
				}
				continue
			}
			r.nonBody++
			if len(byBlob[v.value]) > 0 {
				r.leak++
			}
		}
		if len(blobsOfPost) > 0 {
			r.bodySplit = append(r.bodySplit, len(blobsOfPost))
		}
	}
	for _, n := range stands {
		if n > 0 {
			r.merge = append(r.merge, n)
		}
	}
	return r
}

// covers reports whether the instance carries every value of want, which is
// what matching a fixture post means: the instance is a superset of the post.
func covers(held, want map[string]bool) bool {
	for v := range want {
		if !held[v] {
			return false
		}
	}
	return true
}

func mean(ns []int) float64 {
	if len(ns) == 0 {
		return 0
	}
	sum := 0
	for _, n := range ns {
		sum += n
	}
	return float64(sum) / float64(len(ns))
}

func share(n, d int) string {
	if d == 0 {
		return "na"
	}
	return fmt.Sprintf("%.3f", float64(n)/float64(d))
}

// TestMetrics measures every saved page and prints one row per page. Build is
// a stub, so every number is the number an output of no structures scores.
func TestMetrics(t *testing.T) {
	pages := load(t)
	var tot result
	fmt.Printf("%-42s %6s %7s %7s %6s %7s %7s %8s %6s %5s\n", "page",
		"posts", "recall", "recall+", "merge", "distrib", "distrib+",
		"wrappers", "struct", "depth")
	for _, p := range pages {
		r := measure(p, Build(p.doc, p.base))
		fmt.Printf("%-42s %6d %7s %7s %6.2f %7.2f %8.2f %8.2f %6d %5d\n", r.name,
			r.posts, share(r.match, r.posts), share(r.matchExact, r.posts),
			mean(r.merge), mean(r.distrib), mean(r.distribPlus),
			mean(r.wrappers), r.structures, r.depth)

		tot.posts += r.posts
		tot.match += r.match
		tot.matchExact += r.matchExact
		tot.merge = append(tot.merge, r.merge...)
		tot.distrib = append(tot.distrib, r.distrib...)
		tot.distribPlus = append(tot.distribPlus, r.distribPlus...)
		tot.wrappers = append(tot.wrappers, r.wrappers...)
		tot.structures += r.structures
		if r.depth > tot.depth {
			tot.depth = r.depth
		}
		tot.bodyValues += r.bodyValues
		tot.bodyInBlob += r.bodyInBlob
		tot.bodySplit = append(tot.bodySplit, r.bodySplit...)
		tot.nonBody += r.nonBody
		tot.leak += r.leak
		tot.blobs += r.blobs
	}
	fmt.Printf("\n%d pages, %d posts\n", len(pages), tot.posts)
	fmt.Printf("recall %s, recall+ %s\n",
		share(tot.match, tot.posts), share(tot.matchExact, tot.posts))
	fmt.Printf("merge %.2f, distrib %.2f, distrib+ %.2f, wrappers %.2f\n",
		mean(tot.merge), mean(tot.distrib), mean(tot.distribPlus), mean(tot.wrappers))
	fmt.Printf("structures %.1f per page, depth %d worst\n",
		float64(tot.structures)/float64(len(pages)), tot.depth)
	fmt.Printf("body recall %s, body split %.2f, leak %d (%s), blobs %d\n",
		share(tot.bodyInBlob, tot.bodyValues), mean(tot.bodySplit),
		tot.leak, share(tot.leak, tot.nonBody), tot.blobs)
}

// TestFixtureReach counts the fixture values that equal a value some node of
// the page offers. A value no node offers can never be matched, so it is a cap
// on recall+ that no algorithm can lift. It also checks that the values a node
// offers are normalized the same way the fixtures are read.
func TestFixtureReach(t *testing.T) {
	byKind := map[string][2]int{}
	var texts, textsReached int
	for _, p := range load(t) {
		offered := map[string]bool{}
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if !observable(n) {
				return
			}
			for _, v := range values(n, p.base) {
				offered[v] = true
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(p.doc)
		for _, post := range p.posts {
			for _, v := range post.values(p.base) {
				c := byKind[v.kind]
				c[1]++
				if offered[v.value] {
					c[0]++
				}
				byKind[v.kind] = c
			}
			for _, text := range post.Texts {
				texts++
				if offered[flat(text.Value)] {
					textsReached++
				}
			}
		}
	}
	kinds := make([]string, 0, len(byKind))
	for k := range byKind {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		c := byKind[k]
		name := k
		if name == "" {
			name = "(none)"
		}
		fmt.Printf("%-14s %5d of %5d reached, %5d missing\n",
			name, c[0], c[1], c[1]-c[0])
	}
	fmt.Printf("\ntexts %d of %d reached, %d missing\n",
		textsReached, texts, texts-textsReached)
}
