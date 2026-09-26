package feed

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

// This file reads a page that publishes no feed. It does not decide which
// elements of the page are posts. It enumerates every group of post-like
// elements and every value that can be read out of them, and the user resolves
// the ambiguity in the app. The design and the measurements behind it are in
// IDEA2.md, and the pages they were taken on are in testdata/.
//
// The second half of the file reads posts back out of a page with the keys the
// user picked. Both halves run the same enumeration, so the screen the user
// answered on and the posts the feed stores are produced by the same code.

// Group is one list of post-like items found in a page.
type Group struct {
	// Selector is the key of the group: the path from the document root to
	// its items, written by rootSelector. It only means something inside the
	// result of an enumeration of the same page; see "What a selector is" in
	// PLAN.md.
	Selector string
	Posts    []PostCandidate
}

// PostCandidate is one item of a group with every value that can be read out
// of it. The values are enumerated per post rather than per group, because two
// posts of one group may carry different ones: a group holds posts that have a
// description and posts that have none.
type PostCandidate struct {
	// Node is the element the item was read from. It stays attached to the
	// parsed document, so a caller can inspect the subtree.
	Node *html.Node
	// Links, Texts and Images hold the values of the item in document order.
	// An item that carries no link is not a post, so Links is never empty.
	Links  []Attribute
	Texts  []Attribute
	Images []Attribute
}

// Attribute is one value found inside a post, with the key that names it.
type Attribute struct {
	// Selector is the key of the element inside the item, written by
	// itemSelector.
	Selector string
	// Value is the resolved URL of a link and of an image, and the text of a
	// text. A <time> element that carries datetime reports the attribute
	// instead of the text, which is what makes a timestamp key usable at poll
	// time without a second extraction mode.
	Value string
	// Alt is the alt text of an image, and is empty on the other two kinds.
	Alt string
}

// EnumeratePostGroups parses an HTML page and returns every group of post-like
// elements in it, in document order, with nothing scored, cut, rejected or
// preselected. page is required: every URL is resolved against it, and a link
// that points back at the page itself is not a post.
//
// A group is a set of sibling elements whose subtrees have the same shape two
// levels deep, which is the approximation of the definition in IDEA2.md. A
// sibling that carries no link is left out of the group, because a post
// without a URL cannot be stored, and a group left with no item is not
// returned.
//
// Two reductions run at the end, and nothing else is removed:
//
//  1. A group whose URL set is identical to another group's is dropped, and
//     the one that comes first in the document is kept.
//  2. A group whose URL set is a strict subset of another group's is dropped.
//
// Both keep every URL reachable through some other group, so neither can break
// the recall requirement of IDEA2.md.
func EnumeratePostGroups(r io.Reader, page url.URL) ([]Group, error) {
	doc, err := sanitize(r)
	if err != nil {
		return nil, err
	}

	var groups []Group
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		// Group the element children by the shape of their subtree, keeping
		// the order in which the shapes first appear. Attributes are not part
		// of the shape, for the reason given on Group.Selector.
		bySig := map[string][]*html.Node{}
		var order []string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			s := signature(c)
			if _, ok := bySig[s]; !ok {
				order = append(order, s)
			}
			bySig[s] = append(bySig[s], c)
		}
		// Two shapes under one container can share an item tag, and the key
		// of a group is written from the tag alone, so the second and later of
		// them carry a shape number. Without it two groups of one page would
		// write the same key and the extraction could not tell them apart; see
		// rootSelector.
		shapes := map[string]int{}
		for _, s := range order {
			item, _, _ := strings.Cut(s, "(")
			shapes[item]++
			var posts []PostCandidate
			for _, m := range bySig[s] {
				// The links are built here because the reduction below needs
				// them. The texts and the images are built after the
				// reduction, for the groups that survive it, because most
				// groups of a large page do not.
				if ls := itemLinks(m, &page); len(ls) > 0 {
					posts = append(posts, PostCandidate{Node: m, Links: ls})
				}
			}
			if len(posts) > 0 {
				groups = append(groups, Group{
					Selector: rootSelector(n, s, shapes[item]),
					Posts:    posts,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				walk(c)
			}
		}
	}
	walk(doc)

	groups = reduce(groups)
	for i := range groups {
		for j := range groups[i].Posts {
			p := &groups[i].Posts[j]
			p.Texts = itemTexts(p.Node)
			p.Images = itemImages(p.Node, &page)
		}
	}
	return groups, nil
}

// reduce applies the two rules described on EnumeratePostGroups. It is what
// turns the raw enumeration into a list a person can read: on
// developer.apple.com the 968 raw groups come down to 95, and on
// paulgraham.com the 729 come down to 3. See the measurement table in
// IDEA2.md.
func reduce(groups []Group) []Group {
	// One URL per item, the first link it carries. Taking every link of the
	// item instead would make the group of the <body> element, which holds one
	// item carrying every link of the page, a superset of every other group,
	// and the second rule would then drop the whole page down to that one
	// group. The first link identifies the item well enough for this
	// comparison, and it is what the measurement table in IDEA2.md was taken
	// with.
	urls := make([]map[string]bool, len(groups))
	for i, g := range groups {
		s := map[string]bool{}
		for _, p := range g.Posts {
			s[p.Links[0].Value] = true
		}
		urls[i] = s
	}

	drop := make([]bool, len(groups))
	seen := map[string]bool{}
	for i := range groups {
		keys := make([]string, 0, len(urls[i]))
		for u := range urls[i] {
			keys = append(keys, u)
		}
		slices.Sort(keys)
		k := strings.Join(keys, "\n")
		if seen[k] {
			drop[i] = true
			continue
		}
		seen[k] = true
	}
	// A dropped group is still compared against, because it holds the same
	// URLs as the group that replaced it: a strict subset of a dropped group
	// is a strict subset of that one as well.
	for i := range groups {
		if drop[i] {
			continue
		}
		for j := range groups {
			if i == j || len(urls[i]) >= len(urls[j]) {
				continue
			}
			if subset(urls[i], urls[j]) {
				drop[i] = true
				break
			}
		}
	}

	var out []Group
	for i, g := range groups {
		if !drop[i] {
			out = append(out, g)
		}
	}
	return out
}

func subset(a, b map[string]bool) bool {
	for u := range a {
		if !b[u] {
			return false
		}
	}
	return true
}

// itemLinks returns every link of one item, including the item element itself,
// in document order. The whole card is a link on many pages, which is the case
// that reports the key ":scope".
func itemLinks(item *html.Node, page *url.URL) []Attribute {
	var out []Attribute
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if u, ok := linkOf(x, page); ok {
			out = append(out, Attribute{Selector: itemSelector(item, x), Value: u.String()})
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(item)
	return out
}

// itemTexts returns every text of one item, in document order.
//
// An element is reported when it holds text and no element child of it holds
// any. Reporting only these leaf blocks is what keeps the list short: without
// the rule, one heading inside a link inside a card produces three entries
// carrying the same string, and the user would have to choose between them.
func itemTexts(item *html.Node) []Attribute {
	var out []Attribute
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.ElementNode {
			if t := text(x); t != "" && !hasTextChild(x) {
				// The datetime attribute is a machine readable date, while
				// the text of the same element is often "two days ago".
				if x.Data == "time" {
					if d := attr(x, "datetime"); d != "" {
						t = d
					}
				}
				out = append(out, Attribute{Selector: itemSelector(item, x), Value: t})
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(item)
	return out
}

func hasTextChild(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && text(c) != "" {
			return true
		}
	}
	return false
}

// itemImages returns every image of one item, in document order, with the URL
// resolved against the page URL.
func itemImages(item *html.Node, page *url.URL) []Attribute {
	var out []Attribute
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.ElementNode && x.Data == "img" {
			// A lazily loaded image keeps its real source in data-src until
			// the page's script runs, and no script runs here. Until then
			// src is either missing or a placeholder written inline as a
			// data: URL, so both cases fall back to data-src.
			src := attr(x, "src")
			if src == "" || strings.HasPrefix(strings.ToLower(src), "data:") {
				src = attr(x, "data-src")
			}
			if src != "" && !strings.HasPrefix(strings.ToLower(src), "data:") {
				if u, err := page.Parse(src); err == nil {
					out = append(out, Attribute{
						Selector: itemSelector(item, x),
						Value:    u.String(),
						Alt:      attr(x, "alt"),
					})
				}
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(item)
	return out
}

// Post is one post read out of a page with a saved set of keys.
type Post struct {
	// URL is the value of the link key, resolved against the page URL. Every
	// post has one: an item where the link key reaches nothing is not
	// reported.
	URL url.URL
	// Title, Description, Timestamp and ImageURL are optional, because the
	// user is free to pick no key for them and because an item is free not to
	// carry the key that was picked.
	//
	// Timestamp is the date as the page writes it, either the datetime
	// attribute of a <time> element or the text of the element the user
	// picked. It is not parsed here: a page can date a post in a format this
	// package does not know.
	Title       string
	Description string
	Timestamp   string
	ImageURL    *url.URL
}

// Selectors is one saved set of keys for reading posts out of a page. Root and
// Link are required; the four others are empty when the user picked no element
// for them.
type Selectors struct {
	Root, Link, Title, Description, Image, Timestamp string
}

// ErrSelectors says a set of keys produces no post on the page that was
// fetched. Subscribing with such a set would create a feed that stays empty
// forever, so the request is rejected rather than saved.
var ErrSelectors = errors.New("the selectors produce no post")

// ExtractPosts enumerates the page and reads the posts that the given sets of
// keys name. It never matches a key against the DOM tree: a key is written by
// the enumerator and is read back by comparing it with the keys of another
// enumeration of the same page.
//
// The sets are read in the order they are given and the posts are deduplicated
// by URL across them, so a page that renders the same post as a grid and as a
// list reports it once. A set whose Root names no group, and a set whose Link
// is carried by no post of its group, fails with ErrSelectors.
func ExtractPosts(r io.Reader, page url.URL, sets []Selectors) ([]Post, error) {
	groups, err := EnumeratePostGroups(r, page)
	if err != nil {
		return nil, err
	}
	byRoot := map[string]Group{}
	for _, g := range groups {
		// The first group wins, which matters only when two groups of one
		// page write the same key. See TestEnumeratePostGroups_RootKeys.
		if _, ok := byRoot[g.Selector]; !ok {
			byRoot[g.Selector] = g
		}
	}

	var out []Post
	taken := map[string]bool{}
	for i, s := range sets {
		g, ok := byRoot[s.Root]
		if !ok {
			return nil, fmt.Errorf("%w: selector set %d names no group (%q)", ErrSelectors, i, s.Root)
		}
		linked := 0
		for _, c := range g.Posts {
			link, ok := valueOf(c.Links, s.Link)
			if !ok {
				// An item without a URL is not a post. This is the row of a
				// group that holds a separator or a promo tile between the
				// posts, and it is skipped rather than reported empty.
				continue
			}
			linked++
			u, err := url.Parse(link)
			if err != nil {
				continue
			}
			if taken[link] {
				continue
			}
			taken[link] = true

			p := Post{URL: *u}
			p.Title, _ = valueOf(c.Texts, s.Title)
			p.Description, _ = valueOf(c.Texts, s.Description)
			p.Timestamp, _ = valueOf(c.Texts, s.Timestamp)
			if v, ok := valueOf(c.Images, s.Image); ok {
				if iu, err := url.Parse(v); err == nil {
					p.ImageURL = iu
				}
			}
			out = append(out, p)
		}
		if linked == 0 {
			return nil, fmt.Errorf(
				"%w: the link of selector set %d (%q) is carried by no item of %q",
				ErrSelectors,
				i,
				s.Link,
				s.Root,
			)
		}
	}
	return out, nil
}

// valueOf returns the value the given key names in one list of attributes. An
// empty key means the user picked no element for that attribute, which is not
// the same as a key that the item does not carry.
func valueOf(as []Attribute, key string) (string, bool) {
	if key == "" {
		return "", false
	}
	for _, a := range as {
		if a.Selector == key {
			return a.Value, true
		}
	}
	return "", false
}

const (
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
//
// <button> is here because its text is an action label ("Read more", "Share"),
// a control label ("Menu", "Close") or a count, and never the title, the
// description or the date of a post. A card that is written as a <button>
// instead of a link carries no href, so it cannot become a post either way.
// Measured over the 25 saved pages, which hold 433 buttons on
// developer.apple.com, 100 on claude.com and 67 on qiita.com: removing them
// changes no group and no detected post, and only empties the title of four
// section links on bbc.com that are not posts.
var dropElements = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true,
	"svg": true, "math": true, "canvas": true, "iframe": true,
	"object": true, "embed": true, "head": true, "button": true,
}

// Elements that make <header> and <footer> belong to a part of the page rather
// than to the whole page. This is the list HTML itself uses to decide whether a
// <header> is the page banner.
var sectioning = map[string]bool{
	"article": true, "aside": true, "main": true, "nav": true, "section": true,
}

// sanitize parses a page and removes everything from it that cannot hold a
// post; see cleanup for what is removed and why.
func sanitize(r io.Reader) (*html.Node, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse page: %w", err)
	}
	cleanup(doc, false)
	return doc, nil
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

// rootSelector writes the key of a group: the path from the document root down
// to its items, for example "html > body > main > div:nth-of-type(2) > a". sig
// is the signature the group was built from, whose first word is the tag name
// of the items.
//
// shape is 1 for the first group of that item tag under this container and
// counts up for the ones after it. It is written into the key as
// ":nth-shape(n)", which is not a CSS pseudo-class: a key is not a CSS
// selector, and nothing ever compiles it. It exists because the rest of the
// key is built from tag names only, so two groups that hold different shapes
// of the same tag under one container would otherwise be impossible to tell
// apart when the saved key is read back.
func rootSelector(container *html.Node, sig string, shape int) string {
	item, _, _ := strings.Cut(sig, "(")
	if shape > 1 {
		item = fmt.Sprintf("%s:nth-shape(%d)", item, shape)
	}
	var parts []string
	for x := container; x != nil && x.Type == html.ElementNode; x = x.Parent {
		parts = append([]string{step(x)}, parts...)
	}
	return strings.Join(append(parts, item), " > ")
}

// itemSelector writes the key of one element inside a post item: the path of
// steps from the item down to it, for example "div > h3". The item element
// itself is ":scope", which happens when the whole card is a link.
func itemSelector(item, el *html.Node) string {
	if el == item {
		return ":scope"
	}
	var parts []string
	for x := el; x != nil && x != item; x = x.Parent {
		parts = append([]string{step(x)}, parts...)
	}
	return strings.Join(parts, " > ")
}

// step writes one element of a key: the tag name, followed by its position
// among the children of its parent that carry the same tag. The position is
// written only when the parent holds more than one of them, which keeps the
// common case readable and, more importantly, makes two items of the same
// shape write the same key.
func step(x *html.Node) string {
	if x.Parent == nil {
		return x.Data
	}
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
		return fmt.Sprintf("%s:nth-of-type(%d)", x.Data, nth)
	}
	return x.Data
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
