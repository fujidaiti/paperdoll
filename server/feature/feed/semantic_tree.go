package feed

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type SemanticTreeNode struct {
	Kind     string              `json:"kind,omitempty"`
	Val      string              `json:"val,omitempty"`
	Link     string              `json:"link,omitempty"`
	Children []*SemanticTreeNode `json:"ch,omitempty"`
}

// isValue reports whether n is a value, since a container always has children.
func (n *SemanticTreeNode) isValue() bool { return len(n.Children) == 0 }

// SemanticTree is a web page simplified into a semantic tree, which post
// matchers run on (see PostMatcher).
//
// A tree is serialized as JSON:
//
//	{"url": "...", "title": "...", "root": SemanticTreeNode}
//
// A SemanticTreeNode is either a value or a container:
//
//	value      {"kind"?, "val"?, "link"?}
//	container  {"link"?, "ch"}
//
//	kind  "h1".."h6" for a heading, "img" for an image, "time" for a time
//	      element or a text that is only a date
//	val   plain text; the alt text for an image, which may be missing; the
//	      timestamp of a time in RFC 3339 and UTC, at 00:00:00 when the page
//	      gives only a date
//	link  link target, or the image URL for an image; http and https only
//	ch    child nodes
//
// Text formatting and line breaks are dropped: a value is plain text on one
// line. A value keeps a link when the link covers the whole text. A link that
// covers only part of a text splits it into values: the text before the link,
// the link, and the text after it.
type SemanticTree struct {
	URL   string            `json:"url"`
	Title string            `json:"title"`
	Root  *SemanticTreeNode `json:"root"`
}

// BuildSemanticTree converts the HTML page raw into a semantic tree.
// fallbackURL is the URL of the page when it has no canonical link.
func BuildSemanticTree(raw []byte, fallbackURL string) (*SemanticTree, error) {
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

	s := &treeBuilder{base: base}
	tree := s.convert(body)
	if tree == nil || tree.isValue() {
		tree = &SemanticTreeNode{
			Children: slices.DeleteFunc([]*SemanticTreeNode{tree}, func(n *SemanticTreeNode) bool { return n == nil }),
		}
	}
	unwrapTextless(tree)
	return &SemanticTree{URL: base.String(), Title: title, Root: tree}, nil
}

type treeBuilder struct {
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
// surrounding text. Links are included, but a link splits the text around it
// (see hasInnerLink).
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

// Classes that hide an element visually and leave it only to screen readers.
var screenReaderClasses = map[string]bool{
	"screen-reader-text": true, "screen-reader-only": true, "sr-only": true,
	"m-sr-only": true, "visually-hidden": true, "visuallyhidden": true,
}

// cssSize matches a CSS width or height in pixels.
var cssSize = regexp.MustCompile(`(?:^|[;\s])(width|height)\s*:\s*(\d+)px`)

// isSmall reports whether the image n shows no content because of its size:
// it is one pixel wide or high, such as a tracking pixel or a spacer, or it is
// at most 92 pixels on both sides, such as an icon, a bullet or an avatar. The
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
	return w == 1 || h == 1 || w > 0 && w <= 92 && h > 0 && h <= 92
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

// convert returns the node for the element n, or nil when n has no visible
// content.
func (s *treeBuilder) convert(n *html.Node) *SemanticTreeNode {
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
		return &SemanticTreeNode{Kind: "img", Val: alt, Link: src}
	case "time":
		// The datetime attribute is preferred over the displayed text, but some
		// pages put a displayed text in it, such as "SEP 23, 2026". A time
		// element without a full date is handled as any other element.
		if val, ok := parseDate(attr(n, "datetime")); ok {
			return &SemanticTreeNode{Kind: "time", Val: val}
		}
		if val, ok := parseDate(textContent(n)); ok {
			return &SemanticTreeNode{Kind: "time", Val: val}
		}
	}
	// Separate items laid out side by side, such as "<span>Video</span>
	// <span>Title</span> <span>0:48</span>", are split into values, so that
	// each can be a field of a post. A heading or a link stays one value,
	// since it is one title.
	split := !headingTags[n.Data] && n.Data != "a"
	separate := split && countApart(slices.Collect(n.ChildNodes())) >= 2
	if isTextOnly(n) && !hasDateText(n) && !s.hasInnerLink(n) && !separate {
		return s.textValue(n)
	}

	// Mixed content: element children are converted recursively, and runs of
	// text and inline elements between them become values.
	var children []*SemanticTreeNode
	var run []*html.Node
	flush := func() {
		if split && countApart(run) >= 2 {
			var part []*html.Node
			flushPart := func() {
				if t := normalizeSpace(joinText(part)); t != "" {
					children = append(children, &SemanticTreeNode{Val: t})
				}
				part = nil
			}
			for _, c := range run {
				if c.Type == html.ElementNode && !formattingTags[c.Data] {
					flushPart()
					if v := s.textValue(c); v != nil {
						children = append(children, v)
					}
				} else {
					part = append(part, c)
				}
			}
			flushPart()
			run = nil
			return
		}
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
		// A run of a single element, such as a span, is a value of its own.
		if len(elems) == 1 && strings.TrimSpace(text.String()) == "" {
			if v := s.textValue(elems[0]); v != nil {
				children = append(children, v)
			}
		} else {
			if t := normalizeSpace(joinText(run)); t != "" {
				children = append(children, &SemanticTreeNode{Val: t})
			}
		}
		run = nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode && isDate(c.Data):
			flush()
			val, _ := parseDate(c.Data)
			children = append(children, &SemanticTreeNode{Kind: "time", Val: val})
		case c.Type == html.TextNode:
			run = append(run, c)
		case c.Type == html.ElementNode && (c.Data == "br" || c.Data == "wbr"):
			// A line break only separates words, so that an element followed
			// by a break is still a value of its own.
			if c.Data == "br" {
				run = append(run, &html.Node{Type: html.TextNode, Data: " "})
			}
		case c.Type == html.ElementNode && inlineTags[c.Data] && isTextOnly(c) && !hasDateText(c) && !s.hasLink(c):
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
	// Wrappers stay containers even around a single node, so that a post keeps
	// its depth when it lacks some of its values. A heading is not a wrapper:
	// it gives its level to the text inside it.
	if kind != "" && len(children) == 1 && children[0].isValue() && children[0].Kind == "" {
		children[0].Kind = kind
		return children[0]
	}
	return &SemanticTreeNode{Link: link, Children: children}
}

// textValue returns the value for the text-only element n, or nil when it
// has no text. Elements that are the only content of their parent are merged
// into one value, which takes a link or a heading level from any of them, so
// that "<h2><a>Title</a></h2>" becomes one heading with a link.
func (s *treeBuilder) textValue(n *html.Node) *SemanticTreeNode {
	text := normalizeSpace(textContent(n))
	if text == "" {
		return nil
	}
	v := &SemanticTreeNode{Val: text}
	for e := n; e != nil; e = onlyChild(e) {
		if e.Data == "a" && v.Link == "" {
			v.Link = s.resolve(attr(e, "href"))
		}
		if headingTags[e.Data] && v.Kind == "" {
			v.Kind = e.Data
		}
	}
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

// hasInnerLink reports whether a link under n covers only part of its text,
// such as a link in a sentence, or a title link followed by ", date". The
// link then becomes a value of its own, so that it can be a field of a post.
func (s *treeBuilder) hasInnerLink(n *html.Node) bool {
	e := n
	for {
		if e.Data == "a" && s.resolve(attr(e, "href")) != "" {
			return false
		}
		next := onlyChild(e)
		if next == nil {
			return s.hasLink(e)
		}
		e = next
	}
}

// hasLink reports whether n or an element under it, apart from dropped
// elements, is a link.
func (s *treeBuilder) hasLink(n *html.Node) bool {
	if n.Data == "a" && s.resolve(attr(n, "href")) != "" {
		return true
	}
	for c := range n.ChildNodes() {
		if c.Type == html.ElementNode && !isDropped(c) && s.hasLink(c) {
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
func unwrapTextless(n *SemanticTreeNode) {
	var children []*SemanticTreeNode
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

func hasText(n *SemanticTreeNode) bool {
	if n.isValue() {
		return n.Kind != "img" && n.Val != ""
	}
	return slices.ContainsFunc(n.Children, hasText)
}

// countApart returns the number of elements in nodes that are laid out apart
// from the text next to them, which are those other than formatting, and
// that have text.
func countApart(nodes []*html.Node) int {
	count := 0
	for _, n := range nodes {
		if n.Type == html.ElementNode && !isDropped(n) && !formattingTags[n.Data] && n.Data != "br" &&
			strings.TrimSpace(textContent(n)) != "" {
			count++
		}
	}
	return count
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
func (s *treeBuilder) imageSource(n *html.Node) string {
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
func (s *treeBuilder) resolve(ref string) string {
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
	return closeGaps(b.String())
}

// closeGaps replaces the gaps in text with spaces, except before closing
// punctuation and after opening brackets.
func closeGaps(text string) string {
	text = gapBeforePunct.ReplaceAllString(text, "$1")
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
