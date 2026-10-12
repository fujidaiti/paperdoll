// Command semantictree converts sample web pages into semantic trees (JSON),
// a prototype of the page format that post matchers will run on.
//
// Usage (from the repository root):
//
//	go run ./tool/semantictree -out DIR [-in server/feature/feed/testdata/source]
//
// The trees of the sample pages are in testdata, written by:
//
//	go run ./tool/semantictree -out tool/semantictree/testdata/source
//	go run ./tool/semantictree -in server/feature/feed/testdata/robustness -out tool/semantictree/testdata/robustness
//
// Output format: one JSON file per page.
//
//	{"url": "...", "title": "...", "root": Node}
//
// A Node is either a value or a container:
//
//	value      {"sig", "kind"?, "val"?, "link"?}
//	container  {"sig", "link"?, "ch"}
//
//	sig   the elements the node was made from, outermost first, such as
//	      "h2.title > a"; empty for text between elements
//	kind  "h1".."h6" for a heading, "img" for an image, "time" for a time
//	val   plain text; the alt text for an image, which may be missing; the
//	      datetime attribute for a time, or its text if it has none
//	link  link target, or the image URL for an image
//	ch    child nodes
//
// Text formatting and links inside text are dropped: a value is plain text,
// and keeps a link only when the link covers the whole text.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Node struct {
	Sig      string  `json:"sig,omitempty"`
	Kind     string  `json:"kind,omitempty"`
	Val      string  `json:"val,omitempty"`
	Link     string  `json:"link,omitempty"`
	Children []*Node `json:"ch,omitempty"`
}

// isValue reports whether n is a value, since a container always has children.
func (n *Node) isValue() bool { return len(n.Children) == 0 }

type Document struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Root  *Node  `json:"root"`
}

func main() {
	in := flag.String("in", "server/feature/feed/testdata/source", "directory of sample HTML files")
	out := flag.String("out", "", "output directory")
	flag.Parse()
	if *out == "" {
		log.Fatal("-out is required")
	}

	files, err := filepath.Glob(filepath.Join(*in, "*.html"))
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".html")
		raw, err := os.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		doc, err := simplify(raw, "https://"+strings.ReplaceAll(name, "-", "/"))
		if err != nil {
			log.Fatalf("%s: %v", f, err)
		}
		// Pages rendered by JavaScript have nothing to annotate.
		if len(doc.Root.Children) == 0 {
			fmt.Printf("%-50s skipped: empty body\n", filepath.Base(f))
			continue
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(doc); err != nil {
			log.Fatal(err)
		}
		data := buf.Bytes()
		if err := os.WriteFile(filepath.Join(*out, name+".json"), data, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-50s %8d -> %8d bytes\n", filepath.Base(f), len(raw), len(data))
	}
}

func simplify(raw []byte, fallbackURL string) (*Document, error) {
	root, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	var head, body *html.Node
	for n := range root.Descendants() {
		if n.Type != html.ElementNode {
			continue
		}
		if n.Data == "head" && head == nil {
			head = n
		}
		if n.Data == "body" && body == nil {
			body = n
		}
	}
	if body == nil {
		return nil, fmt.Errorf("no body")
	}

	pageURL := fallbackURL
	title := ""
	if head != nil {
		for n := range head.Descendants() {
			if n.Type != html.ElementNode {
				continue
			}
			switch {
			case n.Data == "title" && title == "" && n.FirstChild != nil:
				title = normalizeSpace(n.FirstChild.Data)
			case n.Data == "link" && attr(n, "rel") == "canonical" && attr(n, "href") != "":
				pageURL = attr(n, "href")
			}
		}
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, err
	}
	// The main content of a page holds its posts, and the rest is site
	// navigation, sidebars and footers. A page can have several mains, of
	// which only one is shown, so all of them are kept.
	var mains []*html.Node
	var findMains func(n *html.Node)
	findMains = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode || isDropped(c) {
				continue
			}
			if c.Data == "main" || strings.EqualFold(attr(c, "role"), "main") {
				mains = append(mains, c)
			} else {
				findMains(c)
			}
		}
	}
	findMains(body)
	if len(mains) > 0 {
		body = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: body.DataAtom, Attr: body.Attr}
		for _, m := range mains {
			m.Parent.RemoveChild(m)
			body.AppendChild(m)
		}
	}

	s := &simplifier{base: base}
	tree := s.convert(body)
	if tree == nil || tree.isValue() {
		tree = &Node{Sig: "body", Children: slices.DeleteFunc([]*Node{tree}, func(n *Node) bool { return n == nil })}
	}
	unwrapTextless(tree)
	return &Document{URL: base.String(), Title: title, Root: tree}, nil
}

type simplifier struct {
	base *url.URL
}

// Elements that never carry post content.
var droppedTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true,
	"svg": true, "canvas": true, "video": true, "audio": true,
	"form": true, "input": true, "button": true, "select": true,
	"textarea": true, "label": true, "dialog": true, "link": true,
	"meta": true, "source": true, "object": true, "embed": true, "map": true,
	"iframe": true,
}

// Elements that only format text, so that their content is part of the
// surrounding text. Links are included, since a link inside text is dropped.
var inlineTags = map[string]bool{
	"a":    true,
	"span": true, "b": true, "i": true, "em": true, "strong": true,
	"small": true, "code": true, "mark": true, "sup": true, "sub": true,
	"u": true, "s": true, "abbr": true, "cite": true, "q": true, "kbd": true,
	"br": true, "wbr": true, "font": true, "bdi": true, "bdo": true,
}

var headingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

var displayNone = regexp.MustCompile(`display\s*:\s*none`)

// cssSize matches a CSS width or height in pixels.
var cssSize = regexp.MustCompile(`(?:^|[;\s])(width|height)\s*:\s*(\d+)px`)

// isSmall reports whether the image n shows no content because of its size:
// it is one pixel wide or high, such as a tracking pixel or a spacer, or it is
// at most 96 pixels on both sides, such as an icon, a bullet or an avatar. The
// size comes from the attributes or the inline style. A size of zero counts as
// unknown, since some pages size their images by CSS after setting zero in the
// attributes.
func isSmall(n *html.Node) bool {
	size := map[string]int{}
	for _, key := range []string{"width", "height"} {
		if v, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(attr(n, key)), "px")); err == nil {
			size[key] = v
		}
	}
	for _, m := range cssSize.FindAllStringSubmatch(attr(n, "style"), -1) {
		size[m[1]], _ = strconv.Atoi(m[2])
	}
	w, h := size["width"], size["height"]
	return w == 1 || h == 1 || w > 0 && w <= 96 && h > 0 && h <= 96
}

// Class names generated from hashes, which change when a site is rebuilt:
// "Name-sc-<hash>-N" by styled-components, followed by classes that are only
// hashes of their styles, such as "hygVWX", one for each of the classes before
// them, and "File-module__<hash>__name" by CSS modules.
var (
	styledClass = regexp.MustCompile(`^(.+)-sc-[0-9a-f]+-\d+$`)
	styleHash   = regexp.MustCompile(`^[A-Za-z]+$`)
	moduleClass = regexp.MustCompile(`^(.+-module)__[A-Za-z0-9_-]{6}__(.+)$`)
)

// stableClasses returns the class names of the attribute value class
// without their generated hashes.
func stableClasses(class string) []string {
	var classes []string
	hashes := 0 // The number of hashes expected next.
	for c := range strings.FieldsSeq(class) {
		if hashes > 0 && styleHash.MatchString(c) {
			hashes--
			continue
		}
		if m := styledClass.FindStringSubmatch(c); m != nil {
			c = m[1]
			hashes++
		} else {
			hashes = 0
			if m := moduleClass.FindStringSubmatch(c); m != nil {
				c = m[1] + "__" + m[2]
			}
		}
		if !slices.Contains(classes, c) {
			classes = append(classes, c)
		}
	}
	return classes
}

// isDropped reports whether the element n and its content are never shown.
func isDropped(n *html.Node) bool {
	return droppedTags[n.Data] || hasAttr(n, "hidden") || displayNone.MatchString(attr(n, "style"))
}

// signature returns the signature of the element n alone, such as
// "div.card.large".
func signature(n *html.Node) string {
	return strings.Join(append([]string{n.Data}, stableClasses(attr(n, "class"))...), ".")
}

// joinSig returns the signature of a node made from an outer element with
// signature outer and the node inner.
func joinSig(outer, inner string) string {
	if inner == "" {
		return outer
	}
	return outer + " > " + inner
}

// convert returns the node for the element n, or nil when n has no visible
// content.
func (s *simplifier) convert(n *html.Node) *Node {
	if n.Type != html.ElementNode || isDropped(n) {
		return nil
	}
	switch n.Data {
	case "img":
		src := s.imageSource(n)
		if src == "" || isSmall(n) {
			return nil
		}
		alt := normalizeSpace(attr(n, "alt"))
		return &Node{Sig: signature(n), Kind: "img", Val: alt, Link: src}
	case "time":
		// A machine-readable timestamp is preferred over the displayed text.
		val := strings.TrimSpace(attr(n, "datetime"))
		if val == "" {
			val = normalizeSpace(textContent(n))
		}
		if val == "" {
			return nil
		}
		return &Node{Sig: signature(n), Kind: "time", Val: val}
	case "pre":
		// Preformatted text keeps its line breaks and indentation.
		text := strings.Trim(joinText(slices.Collect(n.ChildNodes()), false), "\n")
		if strings.TrimSpace(text) == "" {
			return nil
		}
		return &Node{Sig: signature(n), Val: text}
	}
	if isTextOnly(n) {
		return s.textValue(n)
	}

	// Mixed content: element children are converted recursively, and runs of
	// text and inline elements between them become values.
	var children []*Node
	var run []*html.Node
	flush := func() {
		var elems []*html.Node
		for _, c := range run {
			if c.Type == html.ElementNode {
				elems = append(elems, c)
			}
		}
		var text strings.Builder
		for _, c := range run {
			if c.Type == html.TextNode {
				text.WriteString(c.Data)
			}
		}
		// A run of a single element, such as a link, is a value of its own.
		if len(elems) == 1 && strings.TrimSpace(text.String()) == "" {
			if v := s.textValue(elems[0]); v != nil {
				children = append(children, v)
			}
		} else {
			if t := normalizeSpace(joinText(run, true)); t != "" {
				children = append(children, &Node{Val: t})
			}
		}
		run = nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode:
			run = append(run, c)
		case c.Type == html.ElementNode && inlineTags[c.Data] && isTextOnly(c):
			run = append(run, c)
		case c.Type == html.ElementNode:
			if child := s.convert(c); child != nil {
				flush()
				children = append(children, child)
			}
		}
	}
	flush()
	if len(children) == 0 {
		return nil
	}

	link := ""
	if n.Data == "a" {
		link = s.resolve(attr(n, "href"))
	}
	kind := ""
	if headingTags[n.Data] {
		kind = n.Data
	}
	// A wrapper around a single node is merged into it, unless both have a
	// link or a kind, since a node holds only one of each.
	if len(children) == 1 {
		c := children[0]
		conflict := link != "" && c.Link != "" || kind != "" && (c.Kind != "" || !c.isValue())
		if !conflict {
			c.Sig = joinSig(signature(n), c.Sig)
			if c.Link == "" {
				c.Link = link
			}
			if c.Kind == "" {
				c.Kind = kind
			}
			return c
		}
	}
	return &Node{Sig: signature(n), Link: link, Children: children}
}

// textValue returns the value for the text-only element n, or nil when it
// has no text. Elements that are the only content of their parent are merged
// into one value, which takes a link or a heading level from any of them, so
// that "<h2><a>Title</a></h2>" becomes one heading with a link.
func (s *simplifier) textValue(n *html.Node) *Node {
	text := normalizeSpace(textContent(n))
	if text == "" {
		return nil
	}
	v := &Node{Val: text}
	var sigs []string
	for e := n; e != nil; e = onlyChild(e) {
		sigs = append(sigs, signature(e))
		if e.Data == "a" && v.Link == "" {
			v.Link = s.resolve(attr(e, "href"))
		}
		if headingTags[e.Data] && v.Kind == "" {
			v.Kind = e.Data
		}
	}
	v.Sig = strings.Join(sigs, " > ")
	return v
}

// onlyChild returns the only child element of n, or nil when n has other
// content.
func onlyChild(n *html.Node) *html.Node {
	var only *html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode && strings.TrimSpace(c.Data) != "":
			return nil
		case c.Type == html.ElementNode && !droppedTags[c.Data] && c.Data != "br" && c.Data != "wbr":
			if only != nil {
				return nil
			}
			only = c
		}
	}
	return only
}

// unwrapTextless replaces each container under n that has no text and no
// link with its children. A post needs a title, and a field is a text, an
// image or a link, so such a container can be neither and only adds depth.
func unwrapTextless(n *Node) {
	var children []*Node
	for _, c := range n.Children {
		unwrapTextless(c)
		if !c.isValue() && c.Link == "" && !hasText(c) {
			children = append(children, c.Children...)
		} else {
			children = append(children, c)
		}
	}
	n.Children = children
}

func hasText(n *Node) bool {
	if n.isValue() {
		return n.Kind != "img" && n.Val != ""
	}
	return slices.ContainsFunc(n.Children, hasText)
}

// isTextOnly reports whether n contains nothing but text and inline
// elements.
func isTextOnly(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || isDropped(c) {
			continue
		}
		if !inlineTags[c.Data] || !isTextOnly(c) {
			return false
		}
	}
	return true
}

// imageSource picks a usable absolute URL for an img, preferring real URLs
// over inline data placeholders used by lazy loading.
func (s *simplifier) imageSource(n *html.Node) string {
	candidates := []string{
		attr(n, "src"),
		attr(n, "data-src"),
		firstSrcset(attr(n, "srcset")),
		firstSrcset(attr(n, "data-srcset")),
	}
	for _, c := range candidates {
		if c == "" || strings.HasPrefix(c, "data:") {
			continue
		}
		return s.resolve(c)
	}
	return ""
}

func firstSrcset(v string) string {
	first, _, _ := strings.Cut(v, ",")
	fields := strings.Fields(first)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func (s *simplifier) resolve(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "javascript:") {
		return ""
	}
	u, err := s.base.Parse(ref)
	if err != nil {
		return ""
	}
	return u.String()
}

// textContent concatenates the text under n, skipping dropped elements.
func textContent(n *html.Node) string {
	return joinText(slices.Collect(n.ChildNodes()), true)
}

// Elements that format part of a text, which can be part of a word, such as
// "un<b>believ</b>able". Other elements, such as spans, are often laid out
// apart from the text next to them, with no space between them in the HTML.
var formattingTags = map[string]bool{
	"b": true, "i": true, "em": true, "strong": true,
	"small": true, "code": true, "mark": true, "sup": true, "sub": true,
	"u": true, "s": true, "abbr": true, "cite": true, "q": true, "kbd": true,
	"font": true, "bdi": true, "bdo": true, "wbr": true,
}

// Spaces added around an element are written as gaps first, and dropped
// before closing punctuation and after opening brackets, as in
// "<span>Tools</span>, <span>Kiro</span>".
const gap = "\x00"

var (
	gapBeforePunct  = regexp.MustCompile(gap + `+([,.;:!?)\]’'])`)
	gapAfterBracket = regexp.MustCompile(`([(\[])` + gap + `+`)
)

// joinText concatenates the text of nodes and their descendants, skipping
// dropped elements. When spaced is true, the text of each element other than
// formatting is set apart by spaces.
func joinText(nodes []*html.Node, spaced bool) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			b.WriteString(n.Data)
		case n.Type == html.ElementNode && n.Data == "br":
			b.WriteString(" ")
		case n.Type == html.ElementNode && !isDropped(n):
			apart := spaced && !formattingTags[n.Data]
			if apart {
				b.WriteString(gap)
			}
			for c := range n.ChildNodes() {
				walk(c)
			}
			if apart {
				b.WriteString(gap)
			}
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	text := gapBeforePunct.ReplaceAllString(b.String(), "$1")
	text = gapAfterBracket.ReplaceAllString(text, "$1")
	return strings.ReplaceAll(text, gap, " ")
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}
