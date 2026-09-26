package scoredtree

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The five measurements of the selection, all read against the fixtures. The
// fixture of a page records every value the page writes beside a post, each
// with the kind of value it is and whether the post needs it, so a value the
// selection offers is either one of them or noise.
//
//   - found(any) is the share of fixture posts a ticked group reaches, where a
//     member reaches a post by carrying its link anywhere in its subtree.
//   - found(primary) is the same share, but the member's own link has to be
//     the post's link. It is lower whenever a card was not folded into one
//     node, which is the difference between offering a card and offering a
//     part of the page that happens to contain one.
//   - recall(required) is the share of posts for which one member of a ticked
//     group holds every value the fixture marks required.
//   - recall(all) is the same, for every recorded value. It depends on how
//     thoroughly the optional values of a page were recorded, so it is
//     reported rather than tuned against.
//   - extra per member is how many values a member holds that the fixture
//     records nowhere for the posts that member reaches. Lower is better. It
//     never reaches zero, because a card can hold a badge or a category link
//     that the schema has no place for.

// attrs is what the fixture records about one post.
type attrs struct {
	Link   string  `json:"link"`
	Texts  []value `json:"texts"`
	Images []value `json:"images"`
}

// value is one recorded value. A value with no kind is a legitimate part of
// the post that none of the kinds fits, such as a video duration or a price.
type value struct {
	Kind     string `json:"kind,omitempty"`
	Value    string `json:"value"`
	Required bool   `json:"required"`
}

// values returns the post's recorded values in the form the member's values
// are compared against. required limits the result to the values without which
// the post is not covered.
func (a attrs) values(required bool) map[string]bool {
	out := map[string]bool{}
	for _, v := range a.Texts {
		if required && !v.Required {
			continue
		}
		out[flat(v.Value)] = true
	}
	for _, v := range a.Images {
		if required && !v.Required {
			continue
		}
		out[unescape(v.Value)] = true
	}
	return out
}

func readAttrs(t *testing.T, name string) []attrs {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "testdata", name+".fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Posts []attrs `json:"posts"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f.Posts
}

func flat(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// unescape is how an image URL is compared. Resolving a URL against the page
// percent encodes every character outside ASCII, so an image whose file name
// holds an em dash or Korean text arrives encoded while the fixture records it
// as it is written in the page.
func unescape(raw string) string {
	if out, err := url.PathUnescape(raw); err == nil {
		return flat(out)
	}
	return flat(raw)
}

// member is what one row of a group offers the user, read once so that the
// measurements do not walk the same subtree again.
type member struct {
	// own is the link of the member node itself, which is what says the member
	// stands for one post rather than for a part of the page.
	own string
	// links is every link of the subtree.
	links map[string]bool
	// vals is every value of the subtree, counted once each. A page that
	// repeats an avatar in one card offers the reader one value, not six.
	vals map[string]bool
}

func read(n *Node) member {
	m := member{own: Normalize(n.Link), links: map[string]bool{}, vals: map[string]bool{}}
	for _, x := range n.Walk() {
		if x.Link != "" {
			m.links[Normalize(x.Link)] = true
		}
		for _, t := range x.Texts {
			m.vals[flat(t.Value)] = true
			if t.Datetime != "" {
				m.vals[flat(t.Datetime)] = true
			}
		}
		for _, i := range x.Images {
			m.vals[unescape(i.URL)] = true
		}
	}
	return m
}

// holds reports whether the member carries every value of want.
func (m member) holds(want map[string]bool) bool {
	for v := range want {
		if !m.vals[v] {
			return false
		}
	}
	return true
}

func TestMetrics(t *testing.T) {
	var posts, any, primary, covReq, covAll int
	var extra, members int

	fmt.Printf("%-40s %6s %7s %8s %8s %7s %7s\n",
		"page", "posts", "any", "primary", "req", "all", "extra")
	for _, p := range load(t) {
		ScoreNodes(p.root, threshold)
		groups := Groups(p.root)
		fixture := readAttrs(t, p.name)

		byLink := map[string]attrs{}
		for _, a := range fixture {
			byLink[Normalize(a.Link)] = a
		}

		// What each group offers, read once. A member is paired with the posts
		// it reaches, and its values are charged against their values together.
		type row struct {
			reaches map[string]bool
			primary map[string]bool
			covReq  map[string]bool
			covAll  map[string]bool
			extra   int
			members int
		}
		rows := make([]row, len(groups))
		for i, g := range groups {
			r := row{reaches: map[string]bool{}, primary: map[string]bool{},
				covReq: map[string]bool{}, covAll: map[string]bool{}}
			for _, node := range g.Members {
				m := read(node)
				r.members++
				// The posts this member reaches, and what they record.
				req, all := map[string]bool{}, map[string]bool{}
				for u := range m.links {
					a, ok := byLink[u]
					if !ok {
						continue
					}
					r.reaches[u] = true
					for v := range a.values(true) {
						req[v] = true
					}
					for v := range a.values(false) {
						all[v] = true
					}
				}
				if _, ok := byLink[m.own]; ok {
					r.primary[m.own] = true
				}
				// Coverage is asked of one post at a time, because a member
				// that reaches two posts holds the values of both.
				for u := range r.reaches {
					a := byLink[u]
					if m.holds(a.values(true)) {
						r.covReq[u] = true
					}
					if m.holds(a.values(false)) {
						r.covAll[u] = true
					}
				}
				for v := range m.vals {
					if !all[v] {
						r.extra++
					}
				}
				// A link that is not the post's own is a value with no place
				// in the fixture, so it is charged as noise.
				for u := range m.links {
					if _, ok := byLink[u]; !ok {
						r.extra++
					}
				}
			}
			rows[i] = r
		}

		// The groups the user ticks, chosen greedily: the group that reaches
		// the most posts not reached yet, then the next one, until nothing is
		// added. Only these are charged, because the others are never opened.
		left := map[string]bool{}
		for _, a := range fixture {
			left[Normalize(a.Link)] = true
		}
		gotAny, gotPrimary := map[string]bool{}, map[string]bool{}
		gotReq, gotAll := map[string]bool{}, map[string]bool{}
		var pExtra, pMembers int
		for {
			best, gain := -1, 0
			for i, r := range rows {
				n := 0
				for u := range r.reaches {
					if left[u] {
						n++
					}
				}
				if n > gain {
					best, gain = i, n
				}
			}
			if best < 0 {
				break
			}
			for u := range rows[best].reaches {
				delete(left, u)
				gotAny[u] = true
			}
			for u := range rows[best].primary {
				gotPrimary[u] = true
			}
			for u := range rows[best].covReq {
				gotReq[u] = true
			}
			for u := range rows[best].covAll {
				gotAll[u] = true
			}
			pExtra += rows[best].extra
			pMembers += rows[best].members
		}

		posts += len(fixture)
		any += len(gotAny)
		primary += len(gotPrimary)
		covReq += len(gotReq)
		covAll += len(gotAll)
		extra += pExtra
		members += pMembers

		perMember := 0.0
		if pMembers > 0 {
			perMember = float64(pExtra) / float64(pMembers)
		}
		fmt.Printf("%-40s %6d %7s %8s %8s %7s %7.1f\n", p.name, len(fixture),
			rate(len(gotAny), len(fixture)), rate(len(gotPrimary), len(fixture)),
			rate(len(gotReq), len(fixture)), rate(len(gotAll), len(fixture)), perMember)
	}
	fmt.Printf("\nfound(any) %s, found(primary) %s, over %d posts\n",
		rate(any, posts), rate(primary, posts), posts)
	fmt.Printf("recall(required) %s, recall(all) %s\n",
		rate(covReq, posts), rate(covAll, posts))
	fmt.Printf("extra per member %.2f (%d values over %d members)\n",
		float64(extra)/float64(members), extra, members)
}
