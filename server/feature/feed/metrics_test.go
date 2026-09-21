package feed

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// fixture is the ideal result for one saved page, written by hand. It holds
// the posts a perfect detector would find, so the distance between the two
// is what the metrics measure.
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

// ratio is one precision or recall value, kept as a fraction so that an empty
// denominator stays visible. A page that dates no post, as paulgraham.com does,
// has no timestamp recall to report, and reporting 0 there would be wrong.
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

// acceptance is the lowest value each metric may take on one page. The values
// are written by hand, per page, and they are targets rather than a record of
// what the method reaches today: several pages are expected to fail until the
// defects in DISCUSSION.md are fixed.
//
// The two levels are measured as follows.
//
// Level one matches the detected posts against the fixture posts by URL, over
// every list the page returns, after the method has removed the URLs that
// appear in more than one list. Precision is the share of detected posts that
// the fixture holds, recall the share of fixture posts that were detected.
//
// Level two measures the three other fields over the posts matched at level
// one. For one field, precision is the share of detected values that equal the
// fixture value, and recall is the share of fixture values that were detected
// and equal. Values are compared exactly, because the fixtures hold the text as
// the page writes it, so any normalization inside the method shows up here.
//
// TopList is the precision of the highest scoring list alone. The other metrics
// take every list together, so a page whose best list is wrong can still reach
// a high precision on the union. This value is what tells the two apart.
type acceptance struct {
	URLPrecision, URLRecall     float64
	TitlePrecision, TitleRecall float64
	ImagePrecision, ImageRecall float64
	TimePrecision, TimeRecall   float64
	TopListPrecision            float64
}

// pageReport is the measured result of one page.
type pageReport struct {
	// got is the number of distinct posts the method returned, want the number
	// the fixture holds. They are kept because a page whose fixture holds no
	// post is checked on these counts alone.
	got, want int
	// Level one: the posts themselves, matched by URL.
	urlP, urlR ratio
	// Level two: the fields of the posts that level one matched.
	titleP, titleR ratio
	imageP, imageR ratio
	timeP, timeR   ratio
	// topP is the precision of the highest scoring list alone.
	topP ratio
	// notes holds the examples of the differences, printed when the page
	// fails.
	notes []string
}

// TestAccuracy measures how well the method reproduces the hand written
// fixtures, page by page, and fails a page whose metrics fall below the values
// in the acceptance table.
func TestAccuracy(t *testing.T) {
	names := make([]string, 0, len(accepted))
	for name := range accepted {
		names = append(names, name)
	}

	for _, name := range names {
		a := accepted[name]
		t.Run(name, func(t *testing.T) {
			want := readFixture(t, filepath.Join("testdata", name+".fixture.json"))
			lists, doc := detectFile(t, name+".html", want.URL)
			checkStructure(t, lists, doc)
			r := measure(lists, want)

			if r.want == 0 && r.got > 0 {
				t.Errorf("the fixture holds no post, but %d were detected", r.got)
			}

			check(t, "url precision", r.urlP, a.URLPrecision)
			check(t, "url recall", r.urlR, a.URLRecall)
			check(t, "title precision", r.titleP, a.TitlePrecision)
			check(t, "title recall", r.titleR, a.TitleRecall)
			check(t, "image precision", r.imageP, a.ImagePrecision)
			check(t, "image recall", r.imageR, a.ImageRecall)
			check(t, "timestamp precision", r.timeP, a.TimePrecision)
			check(t, "timestamp recall", r.timeR, a.TimeRecall)
			check(t, "top list precision", r.topP, a.TopListPrecision)
			if t.Failed() {
				for _, n := range r.notes {
					t.Log(n)
				}
			}
		})
	}
}

func detectFile(t *testing.T, file, page string) ([]PostList, *html.Node) {
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
	// The document is parsed twice: DetectPostLists cleans up the tree it
	// parses, and the selectors are checked against that same tree, which the
	// returned nodes belong to. Reading it back from the posts is enough.
	lists, err := DetectPostLists(f, *u)
	if err != nil {
		t.Fatal(err)
	}
	var root *html.Node
	if len(lists) > 0 {
		for n := lists[0].Posts[0].Node; n != nil; n = n.Parent {
			root = n
		}
	}
	return lists, root
}

// check fails the page when a metric falls below the value the table requires.
// A metric with an empty denominator is skipped, and so is a metric the table
// marks with na.
func check(t *testing.T, name string, got ratio, min float64) {
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

// measure computes every metric of one page. It works in two levels. Level one
// matches the detected posts against the fixture posts by URL. Level two
// compares the other fields of the posts that level one matched.
func measure(lists []PostList, want fixture) pageReport {
	// Index the fixture by URL, which is the key the two levels match on.
	wantByURL := map[string]fixturePost{}
	for _, p := range want.Posts {
		wantByURL[p.URL] = p
	}

	var r pageReport
	// Only the first few of each are kept. A page can differ on hundreds of
	// posts, and reading hundreds of lines does not explain more than reading
	// five: the differences on one page almost always share one cause, and the
	// metrics themselves say how widespread it is. The field limit is higher
	// because three fields are compared on every post.
	const (
		maxURLExamples   = 5
		maxFieldExamples = 15
	)
	var (
		// URLs the method returned that the fixture does not hold.
		extra []string
		// URLs the fixture holds that the method did not return.
		missing []string
		// The fields whose two values differ.
		wrong []string
	)

	// Level one, precision: the share of the detected posts that the fixture
	// holds. A low value means the method returns posts that are not posts.
	//
	// The posts of every list are counted together. The lists divide one page
	// into groups, but the metrics are about the page as a whole. A URL that
	// appears in two lists is counted once.
	gotByURL := map[string]Post{}
	for _, l := range lists {
		for _, p := range l.Posts {
			u := p.URL.String()
			if _, seen := gotByURL[u]; seen {
				continue
			}
			gotByURL[u] = p
			r.urlP.total++
			if _, ok := wantByURL[u]; ok {
				r.urlP.hit++
			} else if len(extra) < maxURLExamples {
				extra = append(extra, u)
			}
		}
	}
	r.got, r.want = len(gotByURL), len(wantByURL)

	// Level one, recall: the share of the fixture posts that were detected. A
	// low value means the method misses real posts.
	for _, p := range want.Posts {
		r.urlR.total++
		if _, ok := gotByURL[p.URL]; ok {
			r.urlR.hit++
		} else if len(missing) < maxURLExamples {
			missing = append(missing, p.URL)
		}
	}

	// The precision of the highest scoring list alone. It separates the quality
	// of the ranking from the quality of the extraction: a page can reach a
	// high precision over all its lists while the list the score picked first
	// is junk.
	if len(lists) > 0 {
		for _, p := range lists[0].Posts {
			r.topP.total++
			if _, ok := wantByURL[p.URL.String()]; ok {
				r.topP.hit++
			}
		}
	}

	// Level two runs over the matched posts only. A post that level one did not
	// match has no counterpart to compare its fields with.
	//
	// For each field, the value is counted in precision when the method
	// reported something, and in recall when the fixture holds something. A
	// field is a hit only when the two strings are exactly equal, because the
	// fixtures record the text as the page writes it. So precision answers
	// "when the method fills this field, how often is it right", and recall
	// answers "of the values the page offers, how many does the method find".
	// A field that neither side carries is counted in neither, which is why a
	// metric with an empty denominator is skipped instead of failing as 0.00.
	for _, w := range want.Posts {
		g, ok := gotByURL[w.URL]
		if !ok {
			continue
		}
		image := ""
		if g.ImageURL != nil {
			image = g.ImageURL.String()
		}
		for _, f := range []struct {
			name         string
			got, want    string
			prec, recall *ratio
		}{
			{"title", g.Title, w.Title, &r.titleP, &r.titleR},
			{"image", image, w.ImageURL, &r.imageP, &r.imageR},
			{"timestamp", g.Timestamp, w.Timestamp, &r.timeP, &r.timeR},
		} {
			equal := f.got == f.want
			if f.got != "" {
				f.prec.total++
				if equal {
					f.prec.hit++
				}
			}
			if f.want != "" {
				f.recall.total++
				if equal {
					f.recall.hit++
				}
			}
			if !equal && len(wrong) < maxFieldExamples {
				wrong = append(
					wrong,
					fmt.Sprintf("  %s of %s:\n    got  %q\n    want %q", f.name, w.URL, f.got, f.want),
				)
			}
		}
	}

	if len(missing) > 0 {
		r.notes = append(r.notes, "posts that were not detected:\n  "+strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		r.notes = append(r.notes, "detected posts that the fixture does not hold:\n  "+strings.Join(extra, "\n  "))
	}
	if len(wrong) > 0 {
		r.notes = append(r.notes, "fields that differ:\n"+strings.Join(wrong, "\n"))
	}
	return r
}

// checkStructure verifies what holds for every page, whatever its fixture says:
// the lists are numbered and ordered, every post carries an absolute URL and
// the node it was read from, no URL is reported twice, and the selector of a
// list finds its posts again in the document. These are properties of the
// method, so they are checked separately from the metrics and a page fails on
// them whatever its acceptance values are.
func checkStructure(t *testing.T, lists []PostList, doc *html.Node) {
	t.Helper()
	seen := map[string]bool{}
	for i, l := range lists {
		if l.ID != i+1 {
			t.Errorf("list %d has ID %d, want %d", i, l.ID, i+1)
		}
		if i > 0 && l.Score > lists[i-1].Score {
			t.Errorf("list %d scores %.1f, above the list before it (%.1f)", i, l.Score, lists[i-1].Score)
		}
		if len(l.Posts) < 1 {
			t.Errorf("list %d holds no post", l.ID)
		}
		for _, p := range l.Posts {
			if !p.URL.IsAbs() {
				t.Errorf("post URL %q is not absolute", p.URL.String())
			}
			// The same post can be rendered twice on one page, as a grid and
			// as a list. It must be reported once.
			if seen[p.URL.String()] {
				t.Errorf("post URL %q is reported twice", p.URL.String())
			}
			seen[p.URL.String()] = true
			if p.Node == nil {
				t.Errorf("post %q carries no node", p.URL.String())
			}
		}
		matched := map[*html.Node]bool{}
		goquery.NewDocumentFromNode(doc).Find(l.Selector).Each(func(_ int, s *goquery.Selection) {
			matched[s.Nodes[0]] = true
		})
		for _, p := range l.Posts {
			if !matched[p.Node] {
				t.Errorf("selector %q does not match the post %q", l.Selector, p.URL.String())
				break
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
	// merged, because the method reports each URL once as well.
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

// accepted holds the acceptance values of every saved page. na means the
// metric is not required on that page.
var accepted = map[string]acceptance{
	// A news front page with 19 sections, mixing articles, live pages and
	// section links. It is the hardest page of the set and is accepted low.
	// Its images are lazy loaded, so the src attribute holds a placeholder.
	"bbc.com": {
		URLPrecision:     0.80,
		URLRecall:        0.85,
		TitlePrecision:   0.95,
		TitleRecall:      0.95,
		ImagePrecision:   0.80,
		ImageRecall:      0.80,
		TimePrecision:    na,
		TimeRecall:       0.50,
		TopListPrecision: 0.20,
	},

	// A shop page. Every item is a record, and the page surrounds them with
	// footer and banner groups that score high enough to be returned.
	"diggersfactory.com-vinyl-shop-new-ins": {
		URLPrecision:     0.80,
		URLRecall:        1.00,
		TitlePrecision:   0.80,
		TitleRecall:      0.80,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    na,
		TimeRecall:       na,
		TopListPrecision: 1.00,
	},

	// A feed page that mixes the post list with campaign banners and event
	// widgets. Part of the feed is loaded by JavaScript, so recall is capped.
	"qiita.com": {
		URLPrecision:     0.90,
		URLRecall:        0.60,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   0.80,
		ImageRecall:      na,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},

	// The "AI for Society" cards are custom elements that keep their title and
	// their image in attributes, so only their links can be read.
	"blog.google": {
		URLPrecision:     0.70,
		URLRecall:        0.60,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    na,
		TimeRecall:       na,
		TopListPrecision: 1.00,
	},

	// Ordinary blog index pages. They are required to be exact.
	"anthropic.com-news": {
		URLPrecision:     1.00,
		URLRecall:        0.90,
		TitlePrecision:   0.90,
		TitleRecall:      0.90,
		ImagePrecision:   na,
		ImageRecall:      na,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"aws.amazon.com-jp-blogs-news": {
		URLPrecision:     0.90,
		URLRecall:        1.00,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"claude.com-blog": {
		URLPrecision:     1.00,
		URLRecall:        0.95,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    0.90,
		TimeRecall:       0.90,
		TopListPrecision: 1.00,
	},
	"cursor.com-blog": {
		URLPrecision:     1.00,
		URLRecall:        0.85,
		TitlePrecision:   0.90,
		TitleRecall:      0.90,
		ImagePrecision:   na,
		ImageRecall:      0.50,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"daily.bandcamp.com-album-of-the-day": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   0.90,
		TitleRecall:      0.90,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"daily.bandcamp.com-features": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   0.90,
		TitleRecall:      0.90,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"deepmind.google-blog": {
		URLPrecision:     1.00,
		URLRecall:        0.95,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"deepmind.google-research-publications": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   0.90,
		TitleRecall:      0.90,
		ImagePrecision:   na,
		ImageRecall:      na,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"developer.apple.com-news": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    0.95,
		TimeRecall:       0.95,
		TopListPrecision: 1.00,
	},
	"developers.openai.com-blog": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   0.90,
		TitleRecall:      0.90,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"flutter.dev-blog": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   0.95,
		ImageRecall:      1.00,
		TimePrecision:    0.95,
		TimeRecall:       0.95,
		TopListPrecision: 1.00,
	},
	"github.blog": {
		URLPrecision:     0.90,
		URLRecall:        0.90,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   0.80,
		ImageRecall:      0.80,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 0.90,
	},
	"github.blog-ai-and-ml": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   0.90,
		ImageRecall:      1.00,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"go.dev-blog": {
		URLPrecision:     0.90,
		URLRecall:        1.00,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   na,
		ImageRecall:      na,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"paulgraham.com-articles": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   1.00,
		TitleRecall:      1.00,
		ImagePrecision:   0.90,
		ImageRecall:      na,
		TimePrecision:    0.90,
		TimeRecall:       na,
		TopListPrecision: 1.00,
	},
	"ycombinator.com-blog": {
		URLPrecision:     0.80,
		URLRecall:        1.00,
		TitlePrecision:   0.90,
		TitleRecall:      0.90,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},
	"ycombinator.com-blog-tag-essay": {
		URLPrecision:     1.00,
		URLRecall:        1.00,
		TitlePrecision:   0.90,
		TitleRecall:      0.90,
		ImagePrecision:   1.00,
		ImageRecall:      1.00,
		TimePrecision:    1.00,
		TimeRecall:       1.00,
		TopListPrecision: 1.00,
	},

	// Pages whose content is rendered by JavaScript. A plain HTTP client
	// receives no post, so the method must return none. The fixtures of these
	// pages hold no post either, which the test checks separately.
	"apple.com-newsroom":      {na, na, na, na, na, na, na, na, na},
	"blog.google-feed":        {na, na, na, na, na, na, na, na, na},
	"reddit.com-r-golang":     {na, na, na, na, na, na, na, na, na},
	"security.apple.com-blog": {na, na, na, na, na, na, na, na, na},
}
