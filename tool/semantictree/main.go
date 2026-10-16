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
//	      element or a text that is only a date
//	val   plain text; the alt text for an image, which may be missing; the
//	      timestamp of a time in RFC 3339 and UTC, at 00:00:00 when the page
//	      gives only a date
//	link  link target, or the image URL for an image; http and https only
//	ch    child nodes
//
// Text formatting, line breaks and links inside text are dropped: a value is
// plain text on one line, and keeps a link only when the link covers the
// whole text.
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
	"time"

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

	// An image that appears many times in the page, such as a placeholder of a
	// lazy-loaded image, a default thumbnail or an icon, shows nothing specific
	// to a post. An image that appears twice can still be a post thumbnail,
	// such as a picture for light and dark modes or a post listed twice.
	s := &simplifier{base: base, repeatedImages: map[string]bool{}}
	count := map[string]int{}
	for n := range body.Descendants() {
		if n.Type == html.ElementNode && n.Data == "img" {
			if src := s.imageSource(n); src != "" {
				count[src]++
			}
		}
	}
	for src, c := range count {
		if c >= 3 {
			s.repeatedImages[src] = true
		}
	}
	tree := s.convert(body)
	if tree == nil || tree.isValue() {
		tree = &Node{Sig: "body", Children: slices.DeleteFunc([]*Node{tree}, func(n *Node) bool { return n == nil })}
	}
	unwrapTextless(tree)
	return &Document{URL: base.String(), Title: title, Root: tree}, nil
}

type simplifier struct {
	base           *url.URL
	repeatedImages map[string]bool
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

// groups reports whether n, an element with the single child only, groups
// the parts of one field, such as the paragraphs of a body or the links of a
// tag list. Such an element stays a container, so that a field is the same
// node whether it has one part or many. Sections and articles also wrap whole
// pages and posts, so they stay only around a value: merged into a container,
// they still leave a container.
func groups(n *html.Node, only *Node) bool {
	switch n.Data {
	case "blockquote", "ul", "ol":
		return true
	case "section", "article":
		return only.isValue()
	}
	return false
}

var headingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

var displayNone = regexp.MustCompile(`display\s*:\s*none`)

// Classes that hide an element visually and leave it only to screen readers.
var screenReaderClasses = map[string]bool{
	"screen-reader-text": true, "screen-reader-only": true, "sr-only": true,
	"m-sr-only": true, "visually-hidden": true, "visuallyhidden": true,
}

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
// them, "File-module__<hash>__name" by CSS modules, and other names that end
// with a hash, such as "card__title--8e57666e…" or "blog-date__y06Tq". A
// part of at least five letters and digits with a digit in it counts as a
// hash, which also removes the ids in names such as "post-24394" that differ
// between posts.
var (
	styledClass = regexp.MustCompile(`^(.+)-sc-[0-9a-f]+-\d+$`)
	styleHash   = regexp.MustCompile(`^[A-Za-z]+$`)
	moduleClass = regexp.MustCompile(`^(.+-module)__[A-Za-z0-9_-]{6}__(.+)$`)
	namePart    = regexp.MustCompile(`[A-Za-z0-9]+`)
	className   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
)

// stableClasses returns the class names of the attribute value class
// without their generated hashes. Names with other characters, such as
// "md:grid-cols-2", "w-[72%]" or "px-g1.75", are dropped: they are utility
// classes that only set styles.
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
		// A class whose second part is a hash, such as "astro-73gi4scu", is
		// generated as a whole.
		for i, loc := range namePart.FindAllStringIndex(c, -1) {
			if part := c[loc[0]:loc[1]]; i > 0 && len(part) >= 5 && strings.ContainsAny(part, "0123456789") {
				c = strings.TrimRight(c[:loc[0]], "-_")
				if i == 1 {
					c = ""
				}
				break
			}
		}
		if !className.MatchString(c) {
			continue
		}
		if !slices.Contains(classes, c) {
			classes = append(classes, c)
		}
	}
	return classes
}

// isDropped reports whether the element n and its content are never shown.
// A "hidden" class counts only without a responsive class such as
// "xl:block", which shows the element on some screens.
func isDropped(n *html.Node) bool {
	if droppedTags[n.Data] || hasAttr(n, "hidden") || displayNone.MatchString(attr(n, "style")) {
		return true
	}
	classes := strings.Fields(attr(n, "class"))
	for _, c := range classes {
		if screenReaderClasses[strings.ToLower(c)] {
			return true
		}
	}
	return slices.Contains(classes, "hidden") && !slices.ContainsFunc(classes, func(c string) bool {
		return strings.Contains(c, ":")
	})
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
		if src == "" || isSmall(n) || s.repeatedImages[src] {
			return nil
		}
		alt := normalizeSpace(attr(n, "alt"))
		return &Node{Sig: signature(n), Kind: "img", Val: alt, Link: src}
	case "time":
		// The datetime attribute is preferred over the displayed text, but some
		// pages put a displayed text in it, such as "SEP 23, 2026". A time
		// element without a full date is handled as any other element.
		if val, ok := parseDate(attr(n, "datetime")); ok {
			return &Node{Sig: signature(n), Kind: "time", Val: val}
		}
		if val, ok := parseDate(textContent(n)); ok {
			return &Node{Sig: signature(n), Kind: "time", Val: val}
		}
	}
	if isTextOnly(n) && !hasDateText(n) {
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
			if t := normalizeSpace(joinText(run)); t != "" {
				children = append(children, &Node{Val: t})
			}
		}
		run = nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode && isDate(c.Data):
			flush()
			val, _ := parseDate(c.Data)
			children = append(children, &Node{Kind: "time", Val: val})
		case c.Type == html.TextNode:
			run = append(run, c)
		case c.Type == html.ElementNode && (c.Data == "br" || c.Data == "wbr"):
			// A line break only separates words, so that a link followed by
			// a break is still a value of its own.
			if c.Data == "br" {
				run = append(run, &html.Node{Type: html.TextNode, Data: " "})
			}
		case c.Type == html.ElementNode && inlineTags[c.Data] && isTextOnly(c) && !hasDateText(c):
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
	// link or a kind, since a node holds only one of each, or the wrapper
	// groups the parts of a field.
	if len(children) == 1 && !groups(n, children[0]) {
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

// Date formats of texts, with the month name in any letter case.
var dateLayouts = []string{
	"January 2, 2006", "Jan 2, 2006", "2 January 2006", "2 Jan 2006",
	"1/2/2006", "2006/1/2", "2006-01-02", "2006年1月2日", time.RFC3339,
}

// parseDate returns the timestamp of the text s in RFC 3339 and UTC, with
// the time 00:00:00 when s has none. It fails unless s is only a date with a
// year, a month and a day, such as "Sep 25, 2026", and not a text that
// contains a date.
func parseDate(s string) (string, bool) {
	s = normalizeSpace(s)
	for _, layout := range dateLayouts {
		t, err := time.Parse(layout, s)
		if err != nil {
			continue
		}
		return t.UTC().Format(time.RFC3339), true
	}
	return "", false
}

func isDate(s string) bool {
	_, ok := parseDate(s)
	return ok
}

// hasDateText reports whether a text under n, apart from those under dropped
// elements, is a date. Such texts become values of their own.
func hasDateText(n *html.Node) bool {
	for c := range n.ChildNodes() {
		switch {
		case c.Type == html.TextNode && isDate(c.Data):
			return true
		case c.Type == html.ElementNode && !isDropped(c) && hasDateText(c):
			return true
		}
	}
	return false
}

// onlyChild returns the only child element of n, or nil when n has other
// content.
func onlyChild(n *html.Node) *html.Node {
	var only *html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode && strings.TrimSpace(c.Data) != "":
			return nil
		case c.Type == html.ElementNode && !isDropped(c) && c.Data != "br" && c.Data != "wbr":
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

// resolve returns the absolute URL of ref, or "" unless it is a web page or
// file, since a post links to one, unlike links such as mailto: or feed:.
func (s *simplifier) resolve(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	u, err := s.base.Parse(ref)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.String()
}

// textContent concatenates the text under n, skipping dropped elements.
func textContent(n *html.Node) string {
	return joinText(slices.Collect(n.ChildNodes()))
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
// dropped elements. The text of each element other than formatting is set
// apart by spaces, and line breaks become spaces.
func joinText(nodes []*html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			b.WriteString(n.Data)
		case n.Type == html.ElementNode && n.Data == "br":
			b.WriteString(" ")
		case n.Type == html.ElementNode && !isDropped(n):
			apart := !formattingTags[n.Data]
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
