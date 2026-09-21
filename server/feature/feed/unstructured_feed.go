package feed

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// This file detects a post list in a page that publishes no feed. The method
// and the measurements behind every constant here are described in
// docs/html-feed-detection.md. The pages it was measured on are in testdata/.
//
// The result is a list of candidates rather than a single one, because a page
// often splits one post list into several containers, for example a hero item
// above the rest. The caller decides which candidates are worth keeping: this
// file only reports what looks like a post list and how strongly.

// Post is one item of a detected post list.
type Post struct {
	// Node is the element the post was detected from. It stays attached to
	// the parsed document, so a caller can inspect the subtree to decide
	// whether the post is worth keeping.
	Node *html.Node
	// URL is the first link found in the item, resolved against the page URL.
	// Every post has one; an item without a link is not a post.
	URL url.URL
	// Title, Timestamp and ImageURL are best effort. A page is free not to
	// carry them, so each may be empty or nil.
	Title string
	// Timestamp is the date as the page writes it, either the datetime
	// attribute of a <time> element or the text a date pattern matched. It is
	// not parsed here: a page can date a post in a format this package does
	// not know, and the caller is in a better position to decide what to do
	// with a date it cannot read.
	Timestamp string
	ImageURL  *url.URL
}

// PostList is a group of posts that share one container and one structure.
type PostList struct {
	// ID distinguishes the lists of one page. It is assigned in the order the
	// lists are returned, starting at 1, and has no meaning beyond that: it
	// lets a caller tell which posts were rendered together, so that posts of
	// one page can be grouped again later.
	ID int
	// Selector matches the post elements of this list from the document root.
	// It is built from tag names and positions only, because the saved pages
	// identify their lists by hashed CSS module names or by Tailwind utility
	// classes, and both change whenever the site is rebuilt. It may match more
	// elements than this list holds when the container mixes several item
	// shapes.
	Selector string
	// Score is how strongly the group looks like a post list. It is only
	// comparable between the lists of the same page.
	Score float64
	Posts []Post
	// menu says the items hold a link and almost no text, which describes a
	// row of tags, a genre list or a site menu as well as it describes a list
	// of bare post links. See the rejection in DetectPostLists.
	menu bool
}

// DetectPostLists parses an HTML page and returns the post lists found in it,
// in descending score order, with an empty result when the page holds none.
// pageURL is required: relative links are resolved against it, and a link that
// points back at the page itself is not a post.
//
// A post that appears in more than one list, which happens when a page renders
// the same items as a grid and as a list, is kept in the higher scoring one
// only.
func DetectPostLists(r io.Reader, pageURL url.URL) ([]PostList, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse page: %w", err)
	}
	cleanup(doc, false)

	var lists []PostList
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		// Group the children by structure. Two children belong to the same
		// group when their subtrees have the same shape two levels deep, for
		// example article(section(section)). Attributes are not part of the
		// shape, for the reason given on PostList.Selector.
		groups := map[string][]*html.Node{}
		var order []string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			s := signature(c)
			if _, ok := groups[s]; !ok {
				order = append(order, s)
			}
			groups[s] = append(groups[s], c)
		}
		// A list whose items differ in shape, for example one card with a
		// thumbnail and one without, is split into several groups, so evaluate
		// all children together as well. None of the saved pages needs this
		// any more, because the structural signature already unifies the two
		// card variants on developer.apple.com. It is kept as a fallback.
		if len(order) > 1 {
			var all []*html.Node
			for _, s := range order {
				all = append(all, groups[s]...)
			}
			groups["*"] = all
			order = append(order, "*")
		}

		for _, s := range order {
			g := groups[s]
			if len(g) < minMembers {
				continue
			}
			// Only members that carry a link are items. Spacer rows,
			// separators and promo tiles between posts are dropped here
			// rather than disqualifying the whole group.
			var posts []Post
			seen := map[string]bool{}
			dated, totalText := 0, 0
			for _, m := range g {
				ls := links(m, &pageURL)
				if len(ls) == 0 {
					continue
				}
				p := Post{Node: m}
				t := text(m)
				totalText += len(t)
				p.Title, p.URL = titleAndURL(m, ls, &pageURL, t)
				p.Timestamp = timestamp(m, t)
				p.ImageURL = image(m, &pageURL)
				if p.Timestamp != "" {
					dated++
				}
				posts = append(posts, p)
				seen[p.URL.String()] = true
			}
			if len(posts) < minMembers || len(seen) < minMembers {
				continue
			}
			avgText := totalText / len(posts)

			// Each item contributes its first link only, so a real list scores
			// close to its item count and a block of repeated links cannot
			// inflate its score.
			score := float64(len(seen))
			if len(seen)*10 < len(posts)*8 {
				score *= 0.5 // the items share one target: not a post list
			}
			menu := avgText < minItemText
			if menu {
				score *= 0.3 // link and nothing else: a menu, not a post
			}
			if dated == len(posts) {
				score *= 2.0
			}
			lists = append(lists, PostList{Selector: selector(n, s), Score: score, Posts: posts, menu: menu})
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				walk(c)
			}
		}
	}
	walk(doc)
	if len(lists) == 0 {
		return nil, nil
	}

	sort.SliceStable(lists, func(i, j int) bool { return lists[i].Score > lists[j].Score })

	// Keep every list that scores at least unionFraction of the best one. A
	// page often splits one post list into several containers, for example the
	// hero items and the rest, or the two halves of deepmind.google/blog, so
	// returning the best list alone returns part of the posts. Removing the
	// duplicated URLs afterwards also replaces a merge step: a page that
	// renders the same posts twice, as claude.com does with its grid and its
	// list, collapses by itself.
	//
	// A post does not contain other posts, so a list whose items hold the
	// items of another kept list is dropped. A page whose top level sections
	// repeat the same shape, as github.blog does, produces a group of those
	// sections, and that group can win the page while the real post lists sit
	// inside its items. Only the kept lists are compared, because almost every
	// post card holds some repeated group of its own, for example a row of
	// tags, and those groups score far too low to be returned.
	//
	// Dropping a list lowers the best score, which lowers the cut and can let
	// another list in, so the two steps run until the set stops changing.
	for {
		// Every list that scores at least unionFraction of the best one.
		var kept []PostList
		cut := lists[0].Score * unionFraction
		for _, l := range lists {
			if l.Score < cut {
				break
			}
			kept = append(kept, l)
		}
		item := map[*html.Node][]int{}
		for i, l := range kept {
			for _, p := range l.Posts {
				item[p.Node] = append(item[p.Node], i)
			}
		}
		// holds reports whether the subtree below n, n itself excluded, holds
		// an item of a kept list other than i. The item itself is excluded
		// because the group of all children of a container repeats the items
		// of the groups it was built from.
		var holds func(int, *html.Node) bool
		holds = func(i int, n *html.Node) bool {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				for _, j := range item[c] {
					if j != i {
						return true
					}
				}
				if holds(i, c) {
					return true
				}
			}
			return false
		}
		// A page that writes its posts as bare links, as paulgraham.com does,
		// produces one menu list and nothing else, so a menu list is only
		// dropped while a list with real item text is kept beside it.
		real := false
		for _, l := range kept {
			if !l.menu {
				real = true
				break
			}
		}
		var flat []PostList
		for i, l := range kept {
			if l.menu && real {
				continue
			}
			nested := false
			for _, p := range l.Posts {
				if holds(i, p.Node) {
					nested = true
					break
				}
			}
			if !nested {
				flat = append(flat, l)
			}
		}
		if len(flat) == 0 {
			return nil, nil
		}
		done := len(flat) == len(kept)
		lists = flat
		if done {
			break
		}
	}

	taken := map[string]bool{}
	var out []PostList
	for _, l := range lists {
		var posts []Post
		for _, p := range l.Posts {
			if u := p.URL.String(); !taken[u] {
				taken[u] = true
				posts = append(posts, p)
			}
		}
		if len(posts) == 0 {
			continue
		}
		l.Posts = posts
		l.ID = len(out) + 1
		out = append(out, l)
	}
	return out, nil
}

const (
	// minMembers is how many items a group must hold to be a list. Three is
	// the smallest number that still describes a repetition. A single item
	// displayed on its own above the list, which some pages use for their
	// newest post, is therefore not detected on its own.
	minMembers = 3
	// minItemText is the average item text length, in bytes, below which the
	// group is treated as a menu rather than a post list.
	minItemText = 25
	// unionFraction is the share of the best score a list must reach to be
	// returned. Measured on the 25 saved pages: the lowest group that is a
	// real post list sits at 27% (github.blog/ai-and-ml) and the highest group
	// that is not sits at 17% (the sidebar on developers.openai.com), so 25%
	// keeps every real post. It does not separate the two cleanly on every
	// page; see docs/html-feed-detection.md.
	unionFraction = 0.25
	// sigDepth is how many levels of the subtree the signature covers. Two is
	// the measured optimum: one and two both rank every correct list first,
	// two gives the wider margins, and three or more splits a list whose items
	// differ in their own content, for example one post with two paragraphs
	// and the next with three.
	sigDepth = 2
)

// Elements that carry no structure a post list can be built from. They are
// removed with their contents, so that they cannot become part of an item
// signature and so that a caller reading a returned Node does not have to skip
// them. <svg> is here because one inline icon can be hundreds of path
// elements, and <template> because its contents are not part of the page.
var dropElements = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true,
	"svg": true, "math": true, "canvas": true, "iframe": true,
	"object": true, "embed": true, "head": true,
}

// Elements that make <header> and <footer> belong to a part of the page rather
// than to the whole page. This is the list HTML itself uses to decide whether a
// <header> is the page banner.
var sectioning = map[string]bool{
	"article": true, "aside": true, "main": true, "nav": true, "section": true,
}

// cleanup removes, in place, everything that cannot hold a post: the site
// navigation, the page banner and footer, hidden elements, and the elements
// above. Without it the best groups on most pages are navigation flyouts and
// footer link lists, and the two pages that hold no post list produce footer
// groups instead of nothing.
//
// inSection says whether n is inside one of the sectioning elements above.
// <header> and <footer> are dropped outside of them only, because both tags are
// also used inside a post: on cursor.com every post card holds its image in a
// <header>.
func cleanup(n *html.Node, inSection bool) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if c.Type == html.CommentNode || c.Type == html.DoctypeNode {
			n.RemoveChild(c)
			continue
		}
		if c.Type == html.ElementNode {
			role := strings.ToLower(strings.TrimSpace(attr(c, "role")))
			isChrome := c.Data == "nav" ||
				role == "navigation" || role == "contentinfo" || role == "banner" ||
				(!inSection && (c.Data == "header" || c.Data == "footer")) ||
				named(c, "footer") || named(c, "nav") || named(c, "navigation")
			if dropElements[c.Data] || isChrome {
				n.RemoveChild(c)
				continue
			}
			// A hidden element is dropped with its contents, unless the
			// element itself is a link. A card is often covered by an empty
			// <a> that repeats the card's URL, and that <a> is marked
			// aria-hidden="true" so that a screen reader does not read the
			// same link twice. claude.com has 30 of these and
			// deepmind.google has 26, and the URL is the one thing a post
			// cannot do without. A hidden container is still dropped with
			// everything below it, because a hidden menu holds many links.
			if hidden(c) && attr(c, "href") == "" {
				n.RemoveChild(c)
				continue
			}
			cleanup(c, inSection || sectioning[c.Data])
		}
	}
}

// named reports whether one of the element's class names is exactly the given
// word. A page region often names itself: diggersfactory.com writes its page
// footer as <section class="footer">, which neither the tag test nor the role
// test above reaches, and its genre menu and its policy menu sit inside it.
//
// Only whole words are compared. A class name that merely contains the word,
// such as "featured-external-links-pattern__list" on github.blog, says nothing
// about the region, and a post card is free to carry a class like "post-links".
// The word list is short on purpose: a site that builds its class names from
// Tailwind utilities or from hashed CSS module names, which is most of the
// saved pages, names nothing at all, so this test cannot carry the work that
// the score does.
func named(n *html.Node, word string) bool {
	return slices.Contains(strings.Fields(strings.ToLower(attr(n, "class"))), word)
}

// hidden reports whether an element is not shown to the reader. Hiding through
// a class name, such as Tailwind's "hidden lg:flex" or "sr-only", cannot be
// detected without knowing the site's stylesheet.
func hidden(n *html.Node) bool {
	if attr(n, "hidden") != "" || strings.EqualFold(attr(n, "aria-hidden"), "true") {
		return true
	}
	style := strings.ToLower(attr(n, "style"))
	return strings.Contains(strings.ReplaceAll(style, " ", ""), "display:none")
}

// signature describes "the same kind of element" as the shape of its subtree:
// the tag name, followed by the signatures of its element children in
// parentheses, down to sigDepth levels.
func signature(n *html.Node) string {
	var shape func(*html.Node, int) string
	shape = func(x *html.Node, depth int) string {
		if depth == 0 {
			return x.Data
		}
		var kids []string
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				kids = append(kids, shape(c, depth-1))
			}
		}
		if len(kids) == 0 {
			return x.Data
		}
		return x.Data + "(" + strings.Join(kids, "|") + ")"
	}
	return shape(n, sigDepth)
}

// selector builds a CSS selector that matches the items of a group: the path
// of the container from the document root, followed by the item's tag name.
// The group of all children of a container, which has no single tag name, ends
// in "*".
func selector(container *html.Node, sig string) string {
	item, _, _ := strings.Cut(sig, "(")
	var parts []string
	for x := container; x != nil && x.Type == html.ElementNode; x = x.Parent {
		part := x.Data
		// The position is only added when the parent holds more than one
		// element of that tag, which keeps the common case readable.
		if x.Parent != nil {
			nth, total := 0, 0
			for c := x.Parent.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == x.Data {
					total++
					if c == x {
						nth = total
					}
				}
			}
			if total > 1 {
				part = fmt.Sprintf("%s:nth-of-type(%d)", x.Data, nth)
			}
		}
		parts = append([]string{part}, parts...)
	}
	return strings.Join(append(parts, item), " > ")
}

// link is an element that carries an accepted href, together with the target
// resolved against the page URL. The element is kept because the title is read
// from it: see titleAndURL.
type link struct {
	node *html.Node
	url  url.URL
}

// linkOf reports whether one element carries a link a post may use, and
// returns the target resolved against the page URL.
func linkOf(x *html.Node, page *url.URL) (url.URL, bool) {
	// Any element that carries href, not only <a>. blog.google builds its
	// post cards from custom elements such as <uni-simple-article-card
	// href="..."> with no <a> around them, so an <a> only rule finds nothing
	// there. <link> is excluded because it points at assets and at alternate
	// language versions.
	if x.Type != html.ElementNode || x.Data == "link" {
		return url.URL{}, false
	}
	h := attr(x, "href")
	if h == "" {
		return url.URL{}, false
	}
	u, err := page.Parse(h)
	if err != nil {
		return url.URL{}, false
	}
	u.Fragment = ""
	// The query string can carry the post identity (developer.apple.com uses
	// /news/?id=<id>), so it is kept and the self link test compares path and
	// query together.
	self := u.Host == page.Host && u.Path == page.Path && u.RawQuery == page.RawQuery
	bare := (u.Path == "" || u.Path == "/") && u.RawQuery == ""
	// A cross-origin target is accepted: a post list may link to another site.
	if (u.Scheme != "http" && u.Scheme != "https") || self || bare {
		return url.URL{}, false
	}
	return *u, true
}

// links returns every link below n, including n itself, in document order.
func links(n *html.Node, page *url.URL) []link {
	var out []link
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if u, ok := linkOf(x, page); ok {
			out = append(out, link{node: x, url: u})
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
			b.WriteString(" ")
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// titleAndURL picks the title and the URL of one item together. Picking them
// independently pairs a heading with the link of something else: github.blog
// renders a category badge link above the post heading, so the post was
// reported under its category URL, and every card of the list reported the
// same one.
//
// ls holds the links of the item in document order, and is not empty. full is
// the item text, which the caller has already collected.
func titleAndURL(item *html.Node, ls []link, page *url.URL, full string) (string, url.URL) {
	h := heading(item)
	if h == nil {
		// No heading: the title is the text of the element the URL comes
		// from, so that the two still describe the same thing. A card often
		// wraps its whole contents in that link, in which case the text is the
		// item text and the cut below applies to it.
		t := text(ls[0].node)
		if t == "" {
			t = full
		}
		// A title is a line, not a paragraph. An item that holds its full text
		// would otherwise produce a title of several kilobytes.
		const maxTitle = 120
		if len(t) > maxTitle {
			cut := strings.LastIndex(t[:maxTitle], " ")
			if cut < maxTitle/2 {
				cut = maxTitle
			}
			t = strings.TrimSpace(t[:cut])
		}
		return t, ls[0].url
	}

	// The heading decides the URL. Its own link comes first, then the link the
	// heading sits inside, which covers a card wrapped in one link and covers
	// the custom elements on blog.google.
	if below := links(h, page); len(below) > 0 {
		return text(h), below[0].url
	}
	for x := h; x != nil; x = x.Parent {
		if u, ok := linkOf(x, page); ok {
			return text(h), u
		}
		if x == item {
			break
		}
	}
	// The heading carries no link at all. The item keeps the URL it would have
	// had without this rule, so no item can lose its URL and stop being a post.
	return text(h), ls[0].url
}

// heading returns the first h1-h6 of the item that holds text.
func heading(item *html.Node) *html.Node {
	var found *html.Node
	var walk func(*html.Node) bool
	walk = func(x *html.Node) bool {
		if x.Type == html.ElementNode && len(x.Data) == 2 && x.Data[0] == 'h' && x.Data[1] >= '1' && x.Data[1] <= '6' {
			if text(x) != "" {
				found = x
				return true
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(item)
	return found
}

// dateRe matches the date formats the saved pages write. A written month is
// accepted with or without a day, because some posts are dated by month only.
// It is deliberately loose, which is why the text it matches is kept as it is
// rather than parsed; see Post.Timestamp.
var dateRe = regexp.MustCompile(`(?i)\b(` + strings.Join([]string{
	`\d{4}-\d{2}-\d{2}`,                         // 2026-09-16
	`\d{1,2}\s+` + months + `[a-z]*\.?\s+\d{4}`, // 16 September 2026
	months + `[a-z]*\.?\s+\d{1,2}(,?\s+\d{4})?`, // Sep 16, 2026
	`\d{4}/\d{1,2}/\d{1,2}`,                     // 2026/9/16
	`\d{1,2}/\d{1,2}/\d{4}`,                     // 9/16/2026
	`\d{4}年\d{1,2}月\d{1,2}日`,                    // 2026年9月16日
	months + `[a-z]*\.?\s+\d{4}`,                // September 2026
}, "|") + `)\b`)

const months = `(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)`

// timestamp returns the date the item carries, as the page writes it: the
// datetime attribute of a <time> element if the item has one, otherwise the
// text a date pattern matched. full is the item text, which the caller has
// already collected.
//
// The result is not parsed, so it can be anything from "2026-09-16T10:00:00Z"
// to "16 September 2026". It is also a weak signal on its own: the pattern
// reads "Octoverse 2025" on github.blog as a month and a year. A group where
// every item carries one is still far more likely to be a post list than a
// group where none does, which is what the score uses it for.
func timestamp(item *html.Node, full string) string {
	var raw string
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if raw != "" {
			return
		}
		if x.Type == html.ElementNode && x.Data == "time" {
			if v := attr(x, "datetime"); v != "" {
				raw = v
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(item)
	if raw == "" {
		raw = dateRe.FindString(full)
	}
	return strings.TrimSpace(raw)
}

// image returns the first image of the item, resolved against the page URL.
func image(item *html.Node, page *url.URL) *url.URL {
	var found *url.URL
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if found != nil {
			return
		}
		if x.Type == html.ElementNode && x.Data == "img" {
			src := attr(x, "src")
			if src == "" {
				// A lazily loaded image keeps its real source in data-src
				// until the page's script runs, and no script runs here.
				src = attr(x, "data-src")
			}
			if src != "" && !strings.HasPrefix(strings.ToLower(src), "data:") {
				if u, err := page.Parse(src); err == nil {
					found = u
				}
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(item)
	return found
}
