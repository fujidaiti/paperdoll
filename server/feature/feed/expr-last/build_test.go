package structures

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// cardsBody is two post cards in one list. Each card holds a link, a title
// long enough to be read as one, a description and a list of tags.
const cardsBody = `
<div class="list">
  <div class="card">
    <a class="t" href="/alpha">The first long post title</a>
    <p class="d">A description of the first post</p>
    <div class="tags"><span class="tag">news</span><span class="tag">tech</span></div>
  </div>
  <div class="card">
    <a class="t" href="/beta">The second long post title</a>
    <p class="d">A description of the second post</p>
    <div class="tags"><span class="tag">music</span></div>
  </div>
</div>`

func cardsPosts() []fixturePost {
	return []fixturePost{
		{Link: "https://e.com/alpha", Texts: []fixtureValue{
			text("title", "The first long post title"),
			text("description", "A description of the first post"),
			text("category", "news"), text("category", "tech"),
		}},
		{Link: "https://e.com/beta", Texts: []fixtureValue{
			text("title", "The second long post title"),
			text("description", "A description of the second post"),
			text("category", "music"),
		}},
	}
}

func TestBuildGivesOneStructurePerList(t *testing.T) {
	p := testPage(t, cardsBody, cardsPosts()...)
	ss := Build(p.doc, p.base)
	wantInt(t, "structures", len(ss), 1)

	instances := Extract(ss[0], p.doc, p.base)
	wantInt(t, "instances", len(instances), 2)

	var names []string
	for _, f := range ss[0].Root.Fields {
		names = append(names, f.Name)
	}
	wantValues(t, "fields", names, "a.t", "p.d")
}

// A path that holds several nodes inside one card cannot be a field, because a
// field matcher points at most one node per subtree.
func TestBuildMakesAChildPartForARepeatedPath(t *testing.T) {
	p := testPage(t, cardsBody, cardsPosts()...)
	ss := Build(p.doc, p.base)
	wantInt(t, "depth", ss[0].Depth(), 2)
	wantInt(t, "children", len(ss[0].Root.Children), 1)

	instances := Extract(ss[0], p.doc, p.base)
	wantInt(t, "tags of the first card", len(instances[0].Root.Children), 2)
	wantInt(t, "tags of the second card", len(instances[1].Root.Children), 1)
}

// Every value of a post has to be reachable through one instance, which is
// what the recall and merge metrics report.
func TestBuildReachesEveryPostOnce(t *testing.T) {
	p := testPage(t, cardsBody, cardsPosts()...)
	r := measure(p, Build(p.doc, p.base))
	wantInt(t, "matched posts", r.match, 2)
	wantInt(t, "exactly matched posts", r.matchExact, 2)
	wantInts(t, "merge", r.merge, 1, 1)
}

// A list of tag chips repeats as a post list does, but a chip is a single link
// carrying a single name, so it is not offered as a structure.
func TestBuildSkipsAListOfChips(t *testing.T) {
	p := testPage(t, `
<div class="chips">
  <a class="chip" href="/tag/news">Technology and science</a>
  <a class="chip" href="/tag/music">Music and performance</a>
  <a class="chip" href="/tag/food">Food and restaurants</a>
</div>`)
	wantInt(t, "structures", len(Build(p.doc, p.base)), 0)
}

// A section holding a whole list of cards must not be offered, because one of
// its instances would stand for several posts.
func TestBuildSkipsTheContainerOfAList(t *testing.T) {
	var body strings.Builder
	body.WriteString(`<div class="page">`)
	for _, section := range []string{"news", "trends"} {
		body.WriteString(`<section class="sec">`)
		for _, name := range []string{"one", "two", "three"} {
			body.WriteString(`<div class="card"><a class="t" href="/` +
				section + "/" + name + `">The ` + section + " " + name +
				` post title</a><p class="d">A description</p></div>`)
		}
		body.WriteString(`</section>`)
	}
	body.WriteString(`</div>`)

	p := testPage(t, body.String())
	var instances []*Instance
	for _, s := range Build(p.doc, p.base) {
		instances = append(instances, Extract(s, p.doc, p.base)...)
	}
	wantInt(t, "instances", len(instances), 6)
	for _, in := range instances {
		wantInt(t, "titles in one instance", strings.Count(
			strings.Join(sortedValues(in), " "), "post title"), 1)
	}
}

func sortedValues(in *Instance) []string {
	var out []string
	for v := range in.valueSet() {
		out = append(out, v)
	}
	return out
}

// A class name a build tool writes changes when the site is redeployed, so a
// chain must not ask for one it does not need.
func TestBuildDropsAClassNameItDoesNotNeed(t *testing.T) {
	card := func(style string) string {
		return `<div class="list"><div class="card ` + style + `">` +
			`<a class="t" href="/alpha">The first long post title</a>` +
			`<p class="d">A description of the first post</p></div>` +
			`<div class="card ` + style + `">` +
			`<a class="t" href="/beta">The second long post title</a>` +
			`<p class="d">A description of the second post</p></div></div>`
	}
	p := testPage(t, card("style-1a2b3c"))
	ss := Build(p.doc, p.base)
	wantInt(t, "structures", len(ss), 1)

	later, err := html.Parse(strings.NewReader(
		"<html><body>" + card("style-9z8y7x") + "</body></html>"))
	if err != nil {
		t.Fatal(err)
	}
	wantInt(t, "instances on a later build",
		len(Extract(ss[0], later, p.base)), 2)
}
