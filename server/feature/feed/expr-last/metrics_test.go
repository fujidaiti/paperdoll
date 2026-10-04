package structures

import (
	"net/url"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// The matchers below are written by hand so that the metrics can be measured
// against an output that is known. They are not what the algorithm will
// produce; they only have to meet the contract DEF.md states for a matcher.

func hasClass(n *html.Node, word string) bool {
	if n.Type != html.ElementNode {
		return false
	}
	return slices.Contains(strings.Fields(attr(n, "class")), word)
}

// byClass finds every element carrying the class, without descending into one
// it already found, so that the subtrees it returns do not overlap.
func byClass(word string) PartMatcher {
	return func(root *html.Node) []*html.Node {
		var out []*html.Node
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if n != root && hasClass(n, word) {
				out = append(out, n)
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(root)
		return out
	}
}

// nthClass finds the n-th element carrying the class, counting from 0. A field
// that reaches past the first one is how a single part can be made to hold the
// values of several posts, which is what merge reports.
func nthClass(word string, nth int) FieldMatcher {
	return func(root *html.Node) *html.Node {
		found := byClass(word)(root)
		if nth >= len(found) {
			return nil
		}
		return found[nth]
	}
}

func firstClass(word string) FieldMatcher { return nthClass(word, 0) }

// firstTag finds the first element with the tag name.
func firstTag(name string) FieldMatcher {
	return func(root *html.Node) *html.Node {
		var found *html.Node
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if found != nil {
				return
			}
			if n != root && n.Type == html.ElementNode && n.Data == name {
				found = n
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(root)
		return found
	}
}

// self returns the subtree the part matched, which is how a part that stands
// for one tag chip reads the text of that chip.
func self() FieldMatcher {
	return func(root *html.Node) *html.Node { return root }
}

// allChildren returns the element children of the subtree the part matched,
// which is a run of consecutive siblings under one parent.
func allChildren() BlobMatcher {
	return func(root *html.Node) []*html.Node {
		var out []*html.Node
		for c := root.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				out = append(out, c)
			}
		}
		return out
	}
}

// childrenOf returns the element children of the first element carrying the
// class, which is a run of consecutive siblings under one parent.
func childrenOf(word string) BlobMatcher {
	return func(root *html.Node) []*html.Node {
		holder := firstClass(word)(root)
		if holder == nil {
			return nil
		}
		var out []*html.Node
		for c := holder.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				out = append(out, c)
			}
		}
		return out
	}
}

func testPage(t *testing.T, body string, posts ...fixturePost) page {
	t.Helper()
	doc, err := html.Parse(strings.NewReader("<html><body>" + body + "</body></html>"))
	if err != nil {
		t.Fatal(err)
	}
	base, err := url.Parse("https://e.com/")
	if err != nil {
		t.Fatal(err)
	}
	return page{name: "test", base: base, doc: doc, posts: posts}
}

func text(kind, value string) fixtureValue { return fixtureValue{Kind: kind, Value: value} }

func wantInts(t *testing.T, name string, got []int, want ...int) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func wantInt(t *testing.T, name string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", name, got, want)
	}
}

// listBody is two post cards in one list. The second card has no description,
// which is the optional field case.
const listBody = `
<div class="list">
  <div class="card">
    <a class="t" href="/alpha">Alpha</a>
    <p class="d">About alpha</p>
    <div class="tags"><span class="tag">news</span><span class="tag">tech</span></div>
  </div>
  <div class="card">
    <a class="t" href="/beta">Beta</a>
    <div class="tags"><span class="tag">music</span></div>
  </div>
</div>`

func listPosts() []fixturePost {
	return []fixturePost{
		{Link: "https://e.com/alpha", Texts: []fixtureValue{
			text("title", "Alpha"), text("description", "About alpha"),
			text("category", "news"), text("category", "tech"),
		}},
		{Link: "https://e.com/beta", Texts: []fixtureValue{
			text("title", "Beta"), text("category", "music"),
		}},
	}
}

// flatCards holds the title and the description of a card in one part and
// reads nothing of the tags.
func flatCards() []*Structure {
	return []*Structure{{Root: &Part{
		Name:  "card",
		Match: byClass("card"),
		Fields: []*Field{
			{Name: "title", Match: firstClass("t")},
			{Name: "description", Match: firstClass("d")},
		},
	}}}
}

// taggedCards adds a child part that matches one subtree per tag chip.
func taggedCards() []*Structure {
	return []*Structure{{Root: &Part{
		Name:  "card",
		Match: byClass("card"),
		Fields: []*Field{
			{Name: "title", Match: firstClass("t")},
			{Name: "description", Match: firstClass("d")},
		},
		Children: []*Part{{
			Name:   "tag",
			Match:  byClass("tag"),
			Fields: []*Field{{Name: "category", Match: self()}},
		}},
	}}}
}

func TestValuesOfANode(t *testing.T) {
	base, _ := url.Parse("https://e.com/posts/")
	doc, err := html.Parse(strings.NewReader(
		`<html><body><p id="p">before <b>inside</b> after` +
			`<img src="a.png" srcset="w1.png 1x, w2.png 2x" alt="A cat" title="t">` +
			`<img src="data:image/gif;base64,R0lGOD"></p>` +
			`<div id="h" hidden>hidden text</div>` +
			`<script>var x = 1;</script></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	var byID func(*html.Node, string) *html.Node
	byID = func(n *html.Node, id string) *html.Node {
		if attr(n, "id") == id {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if found := byID(c, id); found != nil {
				return found
			}
		}
		return nil
	}
	p := byID(doc, "p")

	// The text of the b element belongs to b, so p offers the two texts
	// written directly inside it and nothing of its children.
	wantValues(t, "p", values(p, base), "before", "after")
	wantValues(t, "b", values(p.FirstChild.NextSibling, base), "inside")

	img := firstTag("img")(p)
	wantValues(t, "img", values(img, base),
		"e.com/posts/a.png", "e.com/posts/w1.png", "e.com/posts/w2.png",
		"A cat", "t")

	// A data URI points at no page, so it offers nothing.
	wantValues(t, "data img", values(img.NextSibling, base))
	wantValues(t, "hidden", values(byID(doc, "h"), base))
	wantValues(t, "script", values(firstTag("script")(doc), base))
}

func wantValues(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("values of %s = %q, want %q", name, got, want)
	}
}

func TestExtractGivesOneInstancePerMatchedSubtree(t *testing.T) {
	p := testPage(t, listBody, listPosts()...)
	instances := Extract(taggedCards()[0], p.doc, p.base)
	wantInt(t, "instances", len(instances), 2)

	first := instances[0].Root
	wantInt(t, "fields read in the first card", len(first.Read), 2)
	wantValues(t, "title", first.Read[0].Values, "Alpha", "e.com/alpha")
	wantValues(t, "description", first.Read[1].Values, "About alpha")

	// One part produces one part instance per tag chip.
	wantInt(t, "tag part instances", len(first.Children), 2)
	wantValues(t, "first tag", first.Children[0].Read[0].Values, "news")
	wantValues(t, "second tag", first.Children[1].Read[0].Values, "tech")

	// The second card has no description, so that field reads nothing there
	// rather than reaching into the first card.
	second := instances[1].Root
	wantInt(t, "fields read in the second card", len(second.Read), 1)
	wantValues(t, "title", second.Read[0].Values, "Beta", "e.com/beta")
	wantInt(t, "tag part instances", len(second.Children), 1)
}

func TestRecallCountsPostsThatMatchAnInstance(t *testing.T) {
	p := testPage(t, listBody, listPosts()...)

	// The title and the link are the required kinds, and the flat structure
	// reads both of them, so every post matches. No structure reads the tags,
	// so no post matches exactly.
	flat := measure(p, flatCards())
	wantInt(t, "recall", flat.match, 2)
	wantInt(t, "recall+", flat.matchExact, 0)

	// With the tag part every recorded value is read, so exact matching
	// succeeds as well.
	tagged := measure(p, taggedCards())
	wantInt(t, "recall", tagged.match, 2)
	wantInt(t, "recall+", tagged.matchExact, 2)
}

func TestRecallMissesAPostWhoseRequiredValueIsAbsent(t *testing.T) {
	posts := listPosts()
	posts[0].Texts[0] = text("title", "A title the page does not show")
	p := testPage(t, listBody, posts...)

	r := measure(p, flatCards())
	wantInt(t, "recall", r.match, 1)
}

func TestMergeCountsThePostsOneInstanceStandsFor(t *testing.T) {
	p := testPage(t, listBody, listPosts()...)

	// One part per card: each instance stands for one post.
	wantInts(t, "merge", measure(p, flatCards()).merge, 1, 1)

	// One part over the whole list, with one field per post title. This is the
	// positional enumeration merge exists to report: recall is still 1.
	enumeration := []*Structure{{Root: &Part{
		Name:  "list",
		Match: byClass("list"),
		Fields: []*Field{
			{Name: "title 1", Match: nthClass("t", 0)},
			{Name: "title 2", Match: nthClass("t", 1)},
		},
	}}}
	r := measure(p, enumeration)
	wantInt(t, "recall", r.match, 2)
	wantInts(t, "merge", r.merge, 2)
	wantInts(t, "distrib", r.distrib, 1, 1)
}

func TestDistribCountsDistinctPartsAndNotPartInstances(t *testing.T) {
	p := testPage(t, listBody, listPosts()...)

	// Every value of a post sits in the root part, so one row covers it.
	wantInts(t, "distrib", measure(p, flatCards()).distrib, 1, 1)

	// With the tags in a child part, a post is spread over two parts. The
	// first post has two tag chips and the second has one, and both score 2,
	// because the review screen shows one row for the tag part either way.
	tagged := measure(p, taggedCards())
	wantInts(t, "distrib", tagged.distrib, 2, 2)
	wantInts(t, "distrib+", tagged.distribPlus, 2, 2)
}

func TestDistribIsTheSameWithTenTags(t *testing.T) {
	var chips strings.Builder
	var tags []fixtureValue
	for _, tag := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		chips.WriteString(`<span class="tag">` + tag + `</span>`)
		tags = append(tags, text("category", tag))
	}
	body := `<div class="card"><a class="t" href="/alpha">Alpha</a>` +
		`<div class="tags">` + chips.String() + `</div></div>`
	post := fixturePost{Link: "https://e.com/alpha",
		Texts: append([]fixtureValue{text("title", "Alpha")}, tags...)}

	r := measure(testPage(t, body, post), taggedCards())
	wantInt(t, "recall+", r.matchExact, 1)
	wantInts(t, "distrib+", r.distribPlus, 2)
}

func TestDistribCountsThePartsInBetween(t *testing.T) {
	body := `
<div class="card">
  <a class="t" href="/alpha">Alpha</a>
  <div class="meta"><div class="when"><time datetime="2026-01-02">Jan 2</time></div></div>
</div>`
	post := fixturePost{Link: "https://e.com/alpha", Texts: []fixtureValue{
		text("title", "Alpha"), text("pub-date", "2026-01-02"),
	}}

	// The meta part holds no value of the post, but the user passes through it
	// to reach the date, so it is counted.
	nested := []*Structure{{Root: &Part{
		Name:   "card",
		Match:  byClass("card"),
		Fields: []*Field{{Name: "title", Match: firstClass("t")}},
		Children: []*Part{{
			Name:  "meta",
			Match: byClass("meta"),
			Children: []*Part{{
				Name:   "when",
				Match:  byClass("when"),
				Fields: []*Field{{Name: "date", Match: firstTag("time")}},
			}},
		}},
	}}}

	r := measure(testPage(t, body, post), nested)
	wantInts(t, "distrib", r.distrib, 3)
	wantInts(t, "wrappers", r.wrappers, 0)
	wantInt(t, "depth", r.depth, 3)
}

func TestWrappersCountsThePartsOutsideTheSmallestSubtree(t *testing.T) {
	body := `
<div class="card">
  <div class="head"><span class="junk">Sponsored</span></div>
  <div class="body"><a class="t" href="/alpha">Alpha</a></div>
</div>`
	post := fixturePost{Link: "https://e.com/alpha",
		Texts: []fixtureValue{text("title", "Alpha")}}

	// Every value of the post sits in the body part, so distrib is 1. The card
	// part and the head part are read without reaching any value of the post.
	wrapped := []*Structure{{Root: &Part{
		Name:  "card",
		Match: byClass("card"),
		Children: []*Part{
			{
				Name:   "head",
				Match:  byClass("head"),
				Fields: []*Field{{Name: "label", Match: firstClass("junk")}},
			},
			{
				Name:   "body",
				Match:  byClass("body"),
				Fields: []*Field{{Name: "title", Match: firstClass("t")}},
			},
		},
	}}}

	r := measure(testPage(t, body, post), wrapped)
	wantInts(t, "distrib", r.distrib, 1)
	wantInts(t, "wrappers", r.wrappers, 2)
}

func TestDistribTakesTheCheapestInstance(t *testing.T) {
	p := testPage(t, listBody, listPosts()...)

	// The same posts are reached by two structures, one holding each post in a
	// single part and one spreading it over two. The user reviews the cheaper
	// one, so that is the one measured.
	both := append(flatCards(), taggedCards()...)
	r := measure(p, both)
	wantInt(t, "structures", r.structures, 2)
	wantInts(t, "distrib", r.distrib, 1, 1)

	// Exact matching is only reached by the structure that reads the tags, so
	// distrib+ reports that one.
	wantInts(t, "distrib+", r.distribPlus, 2, 2)
}

func TestDepthAndStructures(t *testing.T) {
	p := testPage(t, listBody, listPosts()...)
	wantInt(t, "depth", measure(p, flatCards()).depth, 1)
	wantInt(t, "depth", measure(p, taggedCards()).depth, 2)
	wantInt(t, "structures", measure(p, append(flatCards(), taggedCards()...)).structures, 2)
}

// blobBody is one post card whose body is two paragraphs, one of them holding
// a link of its own.
const blobBody = `
<div class="card">
  <a class="t" href="/alpha">Alpha</a>
  <div class="body"><p>First para</p><p>Second para <a href="/in">inner</a></p></div>
</div>`

func blobPost() fixturePost {
	return fixturePost{Link: "https://e.com/alpha", Texts: []fixtureValue{
		text("title", "Alpha"), text("body", "First para"),
		text("body", "Second para"), text("body", "inner"),
	}}
}

func TestBlobCoversTheBodyAndNothingElse(t *testing.T) {
	// The blob reads the run of paragraphs and everything below them, so the
	// text of the inner link belongs to it as well.
	clean := []*Structure{{Root: &Part{
		Name:  "card",
		Match: byClass("card"),
		Fields: []*Field{
			{Name: "title", Match: firstClass("t")},
			{Name: "body", Blob: childrenOf("body")},
		},
	}}}

	r := measure(testPage(t, blobBody, blobPost()), clean)
	wantInt(t, "blobs", r.blobs, 1)
	wantInt(t, "body values", r.bodyValues, 3)
	wantInt(t, "body values in a blob", r.bodyInBlob, 3)
	wantInts(t, "body split", r.bodySplit, 1)
	wantInt(t, "leak", r.leak, 0)
	wantInt(t, "non body values", r.nonBody, 2)
}

func TestLeakCountsTheValuesABlobSwallows(t *testing.T) {
	// This blob reaches up to the children of the card, so the title and the
	// link of the post end up inside it and the user can no longer tick them
	// on their own.
	wide := []*Structure{{Root: &Part{
		Name:   "card",
		Match:  byClass("card"),
		Fields: []*Field{{Name: "body", Blob: allChildren()}},
	}}}

	r := measure(testPage(t, blobBody, blobPost()), wide)
	wantInt(t, "body values in a blob", r.bodyInBlob, 3)
	wantInt(t, "leak", r.leak, 2)
	wantInt(t, "non body values", r.nonBody, 2)
}

func TestBodySplitCountsTheBlobsOneBodyIsSpreadOver(t *testing.T) {
	body := `
<div class="card">
  <a class="t" href="/alpha">Alpha</a>
  <div class="b1"><p>First para</p></div>
  <div class="b2"><p>Second para</p></div>
</div>`
	post := fixturePost{Link: "https://e.com/alpha", Texts: []fixtureValue{
		text("title", "Alpha"), text("body", "First para"), text("body", "Second para"),
	}}

	split := []*Structure{{Root: &Part{
		Name:  "card",
		Match: byClass("card"),
		Fields: []*Field{
			{Name: "title", Match: firstClass("t")},
			{Name: "body 1", Blob: childrenOf("b1")},
			{Name: "body 2", Blob: childrenOf("b2")},
		},
	}}}

	r := measure(testPage(t, body, post), split)
	wantInt(t, "blobs", r.blobs, 2)
	wantInt(t, "body values in a blob", r.bodyInBlob, 2)
	wantInts(t, "body split", r.bodySplit, 2)
}

func TestNoStructuresScoresZero(t *testing.T) {
	p := testPage(t, listBody, listPosts()...)
	r := measure(p, nil)
	wantInt(t, "recall", r.match, 0)
	wantInt(t, "structures", r.structures, 0)
	wantInt(t, "blobs", r.blobs, 0)
	wantInt(t, "body values", r.bodyValues, 0)
	wantInt(t, "non body values", r.nonBody, 8)
}
