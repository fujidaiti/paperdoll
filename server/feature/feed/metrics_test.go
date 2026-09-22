package feed

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fixture is the ideal result for one saved page, written by hand. It holds
// the posts a perfect detector would find, so the distance between the two is
// what the metrics measure.
type fixture struct {
	URL   string        `json:"url"`
	Posts []fixturePost `json:"posts"`
}

type fixturePost struct {
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
}

// ratio is one recall value, kept as a fraction so that an empty denominator
// stays visible. A page that dates no post, as paulgraham.com does, has no
// timestamp recall to report, and reporting 0 there would be wrong.
type ratio struct{ hit, total int }

func (r ratio) defined() bool { return r.total > 0 }

func (r ratio) value() float64 {
	if r.total == 0 {
		return 0
	}
	return float64(r.hit) / float64(r.total)
}

// na marks a metric that is not required on a page, either because the page
// carries nothing to measure it against or because the value is accepted as it
// is for now.
const na = -1.0

// acceptance is what one page is required to produce. The three measurements
// are the ones named in "What does the accuracy test measure now?" in
// IDEA2.md, and they are read in two different directions.
//
// URLRecall and the three attribute recalls are floors: the page fails when it
// falls below them. URLRecall is the hard requirement of the whole design and
// is 1.00 everywhere, because a post the server never enumerates can never be
// recovered by the user.
//
// Groups, Check and Last are ceilings: the page fails when it goes above them.
// They measure the review cost of the group screen, where a lower number is
// better, so a ceiling is what protects the number from growing unnoticed.
// They are set to what the method produces today rather than to a target, and
// two pages are accepted as bad: developers.openai.com returns 135 groups, and
// bbc.com needs 27 of its first 39. Lowering them one page at a time is how
// the improvement will be visible; see "Open points" in PLAN.md.
type acceptance struct {
	// Groups is how many groups the page may return.
	Groups int
	// Check is how many groups the user has to tick before every fixture post
	// is included, and Last is the position of the last of them in the
	// returned list, which is how far the user has to scroll.
	Check, Last int
	// URLRecall is the share of the fixture posts that one link key of one
	// returned group reaches. This is the measurement the open question "What
	// is saved: a selector or a rule?" turns on: these are the posts that
	// survive to polling time.
	URLRecall float64
	// The share of the fixture values that one key of the ticked group
	// reaches, for the three optional attributes.
	TitleRecall, ImageRecall, TimeRecall float64
}

// pageReport is the measured result of one page.
type pageReport struct {
	groups      int
	check, last int
	urlR        ratio
	titleR      ratio
	imageR      ratio
	timeR       ratio
	// notes holds examples of the differences, printed when the page fails.
	notes []string
}

// TestAccuracy measures how well the enumeration reproduces the hand written
// fixtures, page by page, and fails a page whose numbers fall outside the
// acceptance table.
//
// Run it with PAPERDOLL_DUMP=1 to print the measured table instead of only
// checking it, which is how the ceilings below are written.
func TestAccuracy(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no fixture found in testdata")
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, strings.TrimSuffix(filepath.Base(f), ".fixture.json"))
	}
	sort.Strings(names)

	dump := os.Getenv("PAPERDOLL_DUMP") != ""
	for _, name := range names {
		a, ok := accepted[name]
		t.Run(name, func(t *testing.T) {
			if !ok && !dump {
				t.Fatal("no acceptance values for this page. Run the test " +
					"with PAPERDOLL_DUMP=1 and add the line it prints to " +
					"the table at the end of metrics_test.go.")
			}
			want := readFixture(t, filepath.Join("testdata", name+".fixture.json"))
			groups := enumerateFile(t, name+".html", want.URL)
			checkStructure(t, groups)
			r := measure(groups, want)

			if dump {
				fmt.Printf("\t%q: {%d, %d, %d, %s, %s, %s, %s},\n",
					name, r.groups, r.check, r.last,
					dumpRatio(r.urlR), dumpRatio(r.titleR),
					dumpRatio(r.imageR), dumpRatio(r.timeR))
				return
			}

			if len(want.Posts) == 0 && r.groups > 0 {
				// A page whose content is rendered by JavaScript carries no
				// post for a plain HTTP client, so it must produce no group.
				t.Errorf("the fixture holds no post, but %d groups were returned", r.groups)
			}
			atMost(t, "groups", r.groups, a.Groups)
			atMost(t, "groups to check", r.check, a.Check)
			atMost(t, "last group to check", r.last, a.Last)
			atLeast(t, "url recall", r.urlR, a.URLRecall)
			atLeast(t, "title recall", r.titleR, a.TitleRecall)
			atLeast(t, "image recall", r.imageR, a.ImageRecall)
			atLeast(t, "timestamp recall", r.timeR, a.TimeRecall)
			if t.Failed() {
				for _, n := range r.notes {
					t.Log(n)
				}
			}
		})
	}
}

func dumpRatio(r ratio) string {
	if !r.defined() {
		return "na"
	}
	// Two decimals, rounded down, so that the printed value is one the page
	// still reaches.
	return fmt.Sprintf("%.2f", float64(int(r.value()*100))/100)
}

func enumerateFile(t *testing.T, file, page string) []Group {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", file))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	u, err := url.Parse(page)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := EnumeratePostGroups(f, *u)
	if err != nil {
		t.Fatal(err)
	}
	return groups
}

// atLeast fails the page when a recall falls below the value the table
// requires. A metric with an empty denominator is skipped, and so is a metric
// the table marks with na.
func atLeast(t *testing.T, name string, got ratio, min float64) {
	t.Helper()
	if min == na {
		return
	}
	if !got.defined() {
		if min > 0 {
			t.Errorf("%s: nothing to measure, but the table requires %.2f", name, min)
		}
		return
	}
	if got.value() < min {
		t.Errorf("%s: %.2f (%d/%d), want at least %.2f", name, got.value(), got.hit, got.total, min)
	}
}

// atMost fails the page when a review cost rises above the value the table
// accepts.
func atMost(t *testing.T, name string, got, max int) {
	t.Helper()
	if got > max {
		t.Errorf("%s: %d, want at most %d", name, got, max)
	}
}

// measure computes every metric of one page.
//
// Everything is measured the way the user reads the screen: a group is ticked
// as a whole, and one row of it is picked for each attribute. So a fixture
// post counts as reached only when a single link key of a single group
// produces its URL, and a title counts as reached only when a single text key
// of that group produces the title of that post. Measuring over all the keys
// of a group instead would report a recall the user cannot obtain.
func measure(groups []Group, want fixture) pageReport {
	r := pageReport{groups: len(groups)}
	wantByURL := map[string]fixturePost{}
	for _, p := range want.Posts {
		wantByURL[p.URL] = p
	}
	r.urlR.total = len(want.Posts)
	if len(want.Posts) == 0 {
		return r
	}

	// What each link key of each group reaches, counted in fixture posts
	// only. A key that reaches nothing from the fixture is not listed.
	type choice struct {
		group int
		key   string
		urls  map[string]bool
	}
	var choices []choice
	for i, g := range groups {
		byKey := map[string]map[string]bool{}
		for _, p := range g.Posts {
			for _, l := range p.Links {
				if _, ok := wantByURL[l.Value]; !ok {
					continue
				}
				if byKey[l.Selector] == nil {
					byKey[l.Selector] = map[string]bool{}
				}
				byKey[l.Selector][l.Value] = true
			}
		}
		keys := make([]string, 0, len(byKey))
		for k := range byKey {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			choices = append(choices, choice{group: i, key: k, urls: byKey[k]})
		}
	}

	// Greedy cover: the user ticks the group that adds the most posts, then
	// the next one, until nothing is left to add. This is the review cost.
	left := map[string]bool{}
	for u := range wantByURL {
		left[u] = true
	}
	picked := map[int]string{}
	for {
		best, bestN := -1, 0
		for i, c := range choices {
			n := 0
			for u := range c.urls {
				if left[u] {
					n++
				}
			}
			if n > bestN {
				best, bestN = i, n
			}
		}
		if best < 0 {
			break
		}
		c := choices[best]
		for u := range c.urls {
			delete(left, u)
		}
		if _, ok := picked[c.group]; !ok {
			picked[c.group] = c.key
			r.check++
			if c.group+1 > r.last {
				r.last = c.group + 1
			}
		}
	}
	r.urlR.hit = len(want.Posts) - len(left)

	const maxExamples = 5
	var missing []string
	for _, p := range want.Posts {
		if left[p.URL] && len(missing) < maxExamples {
			missing = append(missing, p.URL)
		}
	}
	if len(missing) > 0 {
		r.notes = append(r.notes, "posts that no single link key reaches:\n  "+strings.Join(missing, "\n  "))
	}

	// The attributes of the posts that were reached, over the groups the user
	// ticked. For one group and one attribute, the row that matches the most
	// fixture values is the row the user would pick, so it is the one measured.
	for i, linkKey := range picked {
		g := groups[i]
		// The fixture post each item of the group stands for, read through the
		// link key the user picked.
		item := make([]fixturePost, len(g.Posts))
		for j, p := range g.Posts {
			if v, ok := valueOf(p.Links, linkKey); ok {
				item[j] = wantByURL[v]
			}
		}
		texts := func(p PostCandidate) []Attribute { return p.Texts }
		images := func(p PostCandidate) []Attribute { return p.Images }
		for _, f := range []struct {
			name   string
			values func(PostCandidate) []Attribute
			want   func(fixturePost) string
			into   *ratio
		}{
			{"title", texts, func(p fixturePost) string { return p.Title }, &r.titleR},
			{"image", images, func(p fixturePost) string { return p.ImageURL }, &r.imageR},
			{"timestamp", texts, func(p fixturePost) string { return p.Timestamp }, &r.timeR},
		} {
			hits := map[string]int{}
			total := 0
			for j, p := range g.Posts {
				w := f.want(item[j])
				if w == "" {
					continue
				}
				total++
				for _, a := range f.values(p) {
					if a.Value == w {
						hits[a.Selector]++
					}
				}
			}
			best := 0
			for _, n := range hits {
				if n > best {
					best = n
				}
			}
			f.into.hit += best
			f.into.total += total
		}
	}
	return r
}

// checkStructure verifies what holds for every page, whatever its fixture
// says. These are properties of the method, so a page fails on them whatever
// its acceptance values are.
func checkStructure(t *testing.T, groups []Group) {
	t.Helper()
	keys := map[string]bool{}
	for i, g := range groups {
		if len(g.Posts) == 0 {
			t.Errorf("group %d holds no item", i)
		}
		// Two groups that write the same key cannot be told apart when the
		// saved key is read back, so ExtractPosts would answer with the wrong
		// items. See the shape number in rootSelector.
		if keys[g.Selector] {
			t.Errorf("group %d repeats the key %q", i, g.Selector)
		}
		keys[g.Selector] = true
		for j, p := range g.Posts {
			if len(p.Links) == 0 {
				t.Errorf("group %d item %d carries no link", i, j)
				continue
			}
			for _, l := range p.Links {
				if u, err := url.Parse(l.Value); err != nil || !u.IsAbs() {
					t.Errorf("group %d item %d carries the link %q, which is not absolute", i, j, l.Value)
				}
			}
			seen := map[string]bool{}
			for _, as := range [][]Attribute{p.Links, p.Texts, p.Images} {
				for _, a := range as {
					if a.Selector == "" {
						t.Errorf("group %d item %d carries a value with no key", i, j)
					}
					seen[a.Selector] = true
				}
			}
		}
	}
}

func readFixture(t *testing.T, path string) fixture {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f fixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if f.URL == "" {
		t.Fatalf("%s carries no url", path)
	}
	// A page can render the same post twice, as a grid and again as a
	// carousel. The fixture records it once, with the fields of both copies
	// merged.
	seen := map[string]bool{}
	for i, p := range f.Posts {
		if p.URL == "" {
			t.Fatalf("%s: post %d carries no url", path, i)
		}
		if seen[p.URL] {
			t.Fatalf("%s: post %d repeats the url %s. Merge the two entries into one.", path, i, p.URL)
		}
		seen[p.URL] = true
	}
	return f
}

// accepted holds the acceptance values of every saved page, written by hand
// from a PAPERDOLL_DUMP run. The fields are, in order: groups, groups to
// check, last group to check, then the url, title, image and timestamp recall.
var accepted = map[string]acceptance{
	"anthropic.com-news": {5, 2, 5, 1.00, 1.00, na, 1.00},

	"aws.amazon.com-jp-blogs-news": {13, 2, 2, 1.00, 1.00, 1.00, 1.00},

	// A news front page with 19 sections, mixing articles, live pages and
	// section links. It renders about 26 separate lists, so a large number of
	// groups to tick is correct here rather than a defect. Its images are lazy
	// loaded, so the src attribute holds a placeholder.
	"bbc.com": {40, 23, 39, 1.00, 0.86, 0.98, 0.92},

	// The "AI for Society" cards are custom elements that keep their title and
	// their image in attributes, so only their links can be read.
	"blog.google": {4, 2, 4, 1.00, 1.00, 1.00, na},

	"claude.com-blog": {5, 3, 4, 1.00, 1.00, 0.88, 1.00},

	// Two of its images carry srcset and no src, and the cards that hold an
	// image are a minority, so one image key reaches a third of the posts.
	"cursor.com-blog": {13, 2, 4, 1.00, 0.87, 0.33, 0.86},

	// The date is a bare text node next to a category link, so no key reaches
	// it. All 30 timestamps are lost, which is the case that the open question
	// "How to parse unstructured timestamps?" in IDEA2.md is about.
	"daily.bandcamp.com-album-of-the-day": {11, 1, 3, 1.00, 1.00, 1.00, 0.00},

	// Same page shape, same unreachable dates as the page above.
	"daily.bandcamp.com-features": {12, 1, 3, 1.00, 1.00, 1.00, 0.00},

	"deepmind.google-blog": {4, 3, 3, 1.00, 0.96, 0.54, 0.88},

	"deepmind.google-research-publications": {2, 1, 1, 1.00, 1.00, na, 1.00},

	// 95 groups, but the first one holds every post. Half of its dates are
	// written as plain text next to other text, which no key can isolate.
	"developer.apple.com-news": {95, 1, 1, 1.00, 1.00, 0.56, 0.50},

	// The page carries a navigation drawer of 1490 elements that cleanup cannot
	// see, and that drawer comes first in the document, so the one group worth
	// ticking is the last of the 135. See "Open points" in PLAN.md.
	"developers.openai.com-blog": {135, 1, 135, 1.00, 1.00, 1.00, 1.00},

	// A shop page. Every item is a record.
	"diggersfactory.com-vinyl-shop-new-ins": {1, 1, 1, 1.00, 1.00, 1.00, na},

	"flutter.dev-blog": {3, 2, 3, 1.00, 1.00, 1.00, 0.99},

	// The page puts several small repeated blocks above its post list, so the
	// posts are spread over 10 groups.
	"github.blog": {43, 10, 38, 1.00, 1.00, 0.80, 0.84},

	"github.blog-ai-and-ml": {21, 3, 9, 1.00, 1.00, 1.00, 1.00},

	"go.dev-blog": {3, 1, 2, 1.00, 1.00, na, 1.00},

	"paulgraham.com-articles": {3, 1, 3, 1.00, 1.00, na, na},

	// A feed page that mixes the post list with campaign banners and event
	// widgets. Part of the feed is loaded by JavaScript.
	"qiita.com": {84, 2, 9, 1.00, 1.00, na, 1.00},

	"ycombinator.com-blog": {11, 3, 5, 1.00, 1.00, 1.00, 1.00},

	"ycombinator.com-blog-tag-essay": {15, 2, 9, 1.00, 1.00, 1.00, 1.00},

	// Pages whose content is rendered by JavaScript. A plain HTTP client
	// receives no post, so the enumeration must return no group at all.
	"apple.com-newsroom":      {0, 0, 0, na, na, na, na},
	"blog.google-feed":        {0, 0, 0, na, na, na, na},
	"reddit.com-r-golang":     {0, 0, 0, na, na, na, na},
	"security.apple.com-blog": {0, 0, 0, na, na, na, na},
}
