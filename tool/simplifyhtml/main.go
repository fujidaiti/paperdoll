// Command simplifyhtml converts the sample web pages in
// server/feature/feed/testdata into simplified document trees (JSON) that the
// client's page annotation screen loads as assets. It prototypes the cleanup
// the server will do before handing a page to the client.
//
// Usage (from the repository root):
//
//	go run ./tool/simplifyhtml [-in server/feature/feed/testdata] [-out client/assets/page_watch]
//
// Output format: one JSON file per page, plus index.json listing them.
//
//	{"url": "...", "title": "...", "root": Node}
//
// A Node is an object with these optional keys:
//
//	tag   element name, or "#text" for a run of text inside mixed content
//	cls   class names
//	text  text content; set only when the node has no child elements
//	href  absolute link target (a)
//	src   absolute image URL (img) or embedded page URL (iframe)
//	alt   image alt text (img)
//	dt    datetime attribute (time)
//	rich  the same text as "text", as a list of inline nodes; set only when
//	      the text has formatting worth keeping (see keptInlineTags)
//	ch    child nodes
//
// An inline node in "rich" is {"tag": "#text", "text": "..."}, {"tag": "br"},
// or {"tag": "b" | "i" | "code", "ch": [inline nodes]}.
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
	"strings"

	"golang.org/x/net/html"
)

type Node struct {
	Tag      string   `json:"tag"`
	Classes  []string `json:"cls,omitempty"`
	Text     string   `json:"text,omitempty"`
	Href     string   `json:"href,omitempty"`
	Src      string   `json:"src,omitempty"`
	Alt      string   `json:"alt,omitempty"`
	Datetime string   `json:"dt,omitempty"`
	Rich     []*Node  `json:"rich,omitempty"`
	Children []*Node  `json:"ch,omitempty"`
}

type Document struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Root  *Node  `json:"root"`
}

type IndexEntry struct {
	File  string `json:"file"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

func main() {
	in := flag.String("in", "server/feature/feed/testdata", "directory of sample HTML files")
	out := flag.String("out", "client/assets/page_watch", "output directory")
	flag.Parse()

	files, err := filepath.Glob(filepath.Join(*in, "*.html"))
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	var index []IndexEntry
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".html")
		raw, err := os.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		doc, err := simplify(raw, fallbackURL(name))
		if err != nil {
			log.Fatalf("%s: %v", f, err)
		}
		// Pages rendered by JavaScript have nothing to annotate.
		if len(doc.Root.Children) == 0 {
			fmt.Printf("%-50s skipped: empty body\n", filepath.Base(f))
			continue
		}
		data, err := json.Marshal(doc)
		if err != nil {
			log.Fatal(err)
		}
		outName := name + ".json"
		if err := os.WriteFile(filepath.Join(*out, outName), data, 0o644); err != nil {
			log.Fatal(err)
		}
		index = append(index, IndexEntry{File: outName, URL: doc.URL, Title: doc.Title})
		fmt.Printf("%-50s %8d -> %8d bytes\n", filepath.Base(f), len(raw), len(data))
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*out, "index.json"), data, 0o644); err != nil {
		log.Fatal(err)
	}
}

// fallbackURL guesses the page URL from a sample file name such as
// "developers.openai.com-blog", for pages without a canonical link.
func fallbackURL(name string) string {
	return "https://" + strings.ReplaceAll(name, "-", "/")
}

func simplify(raw []byte, fallback string) (*Document, error) {
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

	pageURL := fallback
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
	s := &simplifier{base: base}
	tree := s.convert(body)
	if tree == nil {
		tree = &Node{Tag: "body"}
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
}

// Formatting elements whose content is merged into the surrounding text.
var inlineTags = map[string]bool{
	"span": true, "b": true, "i": true, "em": true, "strong": true,
	"small": true, "code": true, "mark": true, "sup": true, "sub": true,
	"u": true, "s": true, "abbr": true, "cite": true, "q": true, "kbd": true,
	"br": true, "wbr": true, "font": true, "bdi": true, "bdo": true,
}

// Inline formatting that survives in "rich" text, by the tag it is written
// as. Other inline elements only contribute their text.
var keptInlineTags = map[string]string{
	"b": "b", "strong": "b", "i": "i", "em": "i", "code": "code", "kbd": "code",
	"br": "br",
}

// Elements whose own attributes make them meaningful, so that they are never
// replaced by their only child.
var semanticTags = map[string]bool{
	"a": true, "img": true, "time": true, "iframe": true,
}

var headingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

var displayNone = regexp.MustCompile(`display\s*:\s*none`)

// convert returns the simplified node for n, or nil when n has no visible
// content. Text and images are the only content; everything else is
// structure.
func (s *simplifier) convert(n *html.Node) *Node {
	if n.Type != html.ElementNode {
		return nil
	}
	tag := n.Data
	if droppedTags[tag] || hasAttr(n, "hidden") || displayNone.MatchString(attr(n, "style")) {
		return nil
	}
	node := &Node{Tag: tag, Classes: strings.Fields(attr(n, "class"))}
	switch tag {
	case "img":
		node.Src = s.imageSource(n)
		if node.Src == "" {
			return nil
		}
		node.Alt = normalizeSpace(attr(n, "alt"))
		return node
	case "iframe":
		// Embedded players and posts; blank or script frames carry nothing.
		node.Src = s.resolve(attr(n, "src"))
		if !strings.HasPrefix(node.Src, "http") {
			return nil
		}
		return node
	case "pre":
		// Preformatted text keeps its line breaks and indentation.
		node.Text = strings.Trim(textContent(n), "\n")
		if strings.TrimSpace(node.Text) == "" {
			return nil
		}
		return node
	case "a":
		node.Href = s.resolve(attr(n, "href"))
	case "time":
		node.Datetime = attr(n, "datetime")
	}

	if s.isInlineOnly(n) {
		node.Text = normalizeSpace(textContent(n))
		node.Rich = richText(inlineNodes(n))
		if node.Text == "" && node.Href == "" {
			return nil
		}
		return node
	}

	// Mixed content: element children are converted recursively, and runs of
	// text between them (including inline formatting) become #text nodes.
	var run strings.Builder
	var richRun []*Node
	flush := func() {
		if t := normalizeSpace(run.String()); t != "" {
			node.Children = append(node.Children, &Node{Tag: "#text", Text: t, Rich: richText(richRun)})
		}
		run.Reset()
		richRun = nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode:
			run.WriteString(c.Data)
			richRun = append(richRun, &Node{Tag: "#text", Text: c.Data})
		case c.Type == html.ElementNode && inlineTags[c.Data] && s.isInlineOnly(c):
			run.WriteString(" " + textContent(c) + " ")
			richRun = append(richRun, inlineNode(c)...)
		case c.Type == html.ElementNode:
			if child := s.convert(c); child != nil {
				flush()
				node.Children = append(node.Children, child)
			}
		}
	}
	flush()

	if len(node.Children) == 0 && node.Href == "" {
		return nil
	}
	// A plain wrapper around a single node adds depth without information, so
	// the two are merged. A semantic child replaces the wrapper; otherwise the
	// wrapper's tag is kept because list items (li, article) tend to identify
	// posts better than the generic containers inside them. Headings are kept
	// on both sides so that the client can show a post title, often a link,
	// as a heading, and find it by its level.
	if len(node.Children) == 1 && !semanticTags[tag] && !headingTags[tag] {
		child := node.Children[0]
		if semanticTags[child.Tag] || child.Tag == "#text" {
			return child
		}
		if !headingTags[child.Tag] {
			child.Tag = tag
		}
		child.Classes = append(node.Classes, child.Classes...)
		return child
	}
	// An element whose only content is a run of text holds it directly.
	if len(node.Children) == 1 && node.Children[0].Tag == "#text" {
		node.Text = node.Children[0].Text
		node.Rich = node.Children[0].Rich
		node.Children = nil
	}
	return node
}

// unwrapTextless replaces each container under n that has no text with its
// children. A post needs a title, and a field is a text, an image or a link,
// so such a container can be neither and only adds depth. Links are kept
// because they are values themselves.
func unwrapTextless(n *Node) {
	var children []*Node
	for _, c := range n.Children {
		unwrapTextless(c)
		if len(c.Children) > 0 && !semanticTags[c.Tag] && !hasText(c) {
			children = append(children, c.Children...)
		} else {
			children = append(children, c)
		}
	}
	n.Children = children
}

func hasText(n *Node) bool {
	return n.Text != "" || slices.ContainsFunc(n.Children, hasText)
}

// isInlineOnly reports whether n contains nothing but text and inline
// formatting elements.
func (s *simplifier) isInlineOnly(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if droppedTags[c.Data] {
			continue
		}
		if !inlineTags[c.Data] || !s.isInlineOnly(c) {
			return false
		}
	}
	return true
}

// inlineNodes converts the content of an inline-only element into inline
// nodes, before whitespace is normalized.
func inlineNodes(n *html.Node) []*Node {
	var nodes []*Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode:
			nodes = append(nodes, &Node{Tag: "#text", Text: c.Data})
		case c.Type == html.ElementNode && !droppedTags[c.Data]:
			nodes = append(nodes, inlineNode(c)...)
		}
	}
	return nodes
}

func inlineNode(n *html.Node) []*Node {
	tag, kept := keptInlineTags[n.Data]
	switch {
	case tag == "br":
		return []*Node{{Tag: "br"}}
	case kept:
		return []*Node{{Tag: tag, Children: inlineNodes(n)}}
	default:
		return inlineNodes(n)
	}
}

// richText collapses whitespace in inline nodes the same way normalizeSpace
// does for plain text, and drops empty nodes. It returns nil when no
// formatting remains, since "text" already holds the plain text.
func richText(nodes []*Node) []*Node {
	afterSpace := true // Leading whitespace is dropped.
	var last *Node     // The last text node, whose trailing space is dropped.
	var normalize func([]*Node) []*Node
	normalize = func(nodes []*Node) []*Node {
		var out []*Node
		for _, n := range nodes {
			switch n.Tag {
			case "#text":
				var b strings.Builder
				for _, r := range n.Text {
					if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
						if !afterSpace {
							b.WriteByte(' ')
						}
						afterSpace = true
						continue
					}
					b.WriteRune(r)
					afterSpace = false
				}
				if b.Len() > 0 {
					n.Text = b.String()
					last = n
					out = append(out, n)
				}
			case "br":
				if last != nil {
					last.Text = strings.TrimRight(last.Text, " ")
				}
				afterSpace = true
				out = append(out, n)
			default:
				if n.Children = normalize(n.Children); len(n.Children) > 0 {
					out = append(out, n)
				}
			}
		}
		return out
	}
	out := normalize(nodes)
	if last != nil {
		last.Text = strings.TrimRight(last.Text, " ")
	}
	// Line breaks at either end only add empty lines.
	for len(out) > 0 && out[0].Tag == "br" {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1].Tag == "br" {
		out = out[:len(out)-1]
	}
	for _, n := range out {
		if n.Tag != "#text" {
			return out
		}
	}
	return nil
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
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch {
			case c.Type == html.TextNode:
				b.WriteString(c.Data)
			case c.Type == html.ElementNode && c.Data == "br":
				b.WriteString(" ")
			case c.Type == html.ElementNode && !droppedTags[c.Data] && c.Data != "iframe":
				walk(c)
			}
		}
	}
	walk(n)
	return b.String()
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
