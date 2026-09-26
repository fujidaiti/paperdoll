package feed

import (
	"io"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// This file is an experiment that stands beside the enumeration in
// unstructured_feed.go and is not used by it.
//
// The enumeration groups siblings by the shape of their markup, so a list
// whose items are written differently, for example one card without a
// category, falls apart into two groups. The semantic tree asks a different
// question: which values belong to the same post? It keeps only what a post is
// made of, which is a link, texts and images, and drops every element that
// carries no such value.

// SemanticNode is one node of the semantic tree. It holds every candidate
// value found in the part of the document it covers, and it has children only
// when that part covers more than one post.
type SemanticNode struct {
	// Node is the element of the parsed document the node was folded from.
	Node *html.Node `json:"-"`
	// Link, Texts and Images are the candidate values of the node. A folded
	// node holds the values of its whole subtree; a node that has children
	// holds only the values that sit directly on Node, because everything else
	// belongs to one of the children. The texts and the images are in document
	// order.
	//
	// Link is a single URL, not a list, because a subtree is folded only when
	// it carries at most one distinct link. A page often repeats that link,
	// for example as an empty element covering a whole card, and the repeats
	// say nothing about the post.
	Link   string          `json:"link,omitempty"`
	Texts  []SemanticText  `json:"texts,omitempty"`
	Images []SemanticImage `json:"images,omitempty"`
	// Children are the parts of the subtree that hold a post of their own.
	Children []*SemanticNode `json:"children,omitempty"`
}

// BuildSemanticTree parses a page and folds it into a semantic tree. page is
// required, because every URL is resolved against it.
//
// A subtree is folded into a single node when it carries at most one distinct
// link, on the assumption that a link is what identifies a post: everything
// around a single link describes that one post, whatever markup holds it. So
// <a href="u"><div><div><p>text</p></div></div></a> becomes one node with the
// link "u" and the text "text", and a card whose title, date and image sit in
// three wrappers becomes one node as well.
//
// A subtree that carries two or more distinct links describes more than one
// post, so it stays a node with one child per part that carries a post. A
// wrapper that adds no value of its own and holds a single child is replaced
// by that child, so the tree holds no step a reader has to look through.
func BuildSemanticTree(r io.Reader, page url.URL) (*SemanticNode, error) {
	return buildSemanticTree(r, page, false)
}

// BuildSemanticTreeWithHeadings is BuildSemanticTree with tag escalation
// turned on, and is an experiment that stands beside it.
//
// Folding a subtree keeps the element name of the innermost element a text was
// read from, so the title of a post written as <h1><a>title</a></h1> arrives
// as the text of a link and the heading is lost. Escalation reports such a
// text under the heading instead, which is what says the text is a title
// rather than any other text of the card.
func BuildSemanticTreeWithHeadings(r io.Reader, page url.URL) (*SemanticNode, error) {
	return buildSemanticTree(r, page, true)
}

func buildSemanticTree(r io.Reader, page url.URL, escalate bool) (*SemanticNode, error) {
	doc, err := sanitize(r)
	if err != nil {
		return nil, err
	}
	root := doc
	for root != nil && root.Type != html.ElementNode {
		root = root.FirstChild
	}
	if root == nil {
		return nil, nil
	}
	n := semanticNode(root, &page, escalate)
	mergeLinkless(n)
	return n, nil
}

// mergeLinkless moves the values of the nodes that carry no link into their
// parent, on the assumption that a value next to a link describes the post
// that link points at, the way a tag, a banner or a date does. A node that
// carries no link is never a post of its own, so leaving it in the tree as a
// child would offer the user a post that does not exist.
//
// A node that carries no link also has no children, because a node only gets
// children when its subtree holds two links or more, so the nodes moved here
// are always leaves. The values are moved only when the parent keeps a child
// that carries a link: a parent whose children all carry no link holds no post
// to attach them to.
//
// This is what a page that lists its posts as running text needs, rather than
// as cards: on developer.apple.com the image and the date of an item sit
// beside the link instead of inside it.
func mergeLinkless(n *SemanticNode) {
	if n == nil {
		return
	}
	for _, c := range n.Children {
		mergeLinkless(c)
	}

	linkless := func(c *SemanticNode) bool { return c.Link == "" && len(c.Children) == 0 }
	linked := false
	for _, c := range n.Children {
		if !linkless(c) {
			linked = true
			break
		}
	}
	if !linked {
		return
	}

	children := n.Children
	n.Children = nil
	for _, c := range children {
		if linkless(c) {
			n.Texts = append(n.Texts, c.Texts...)
			n.Images = append(n.Images, c.Images...)
			continue
		}
		n.Children = append(n.Children, c)
	}
}

// SemanticText is one text candidate, kept with the facts about the element it
// was read from. Folding a subtree into one node throws the markup away, and
// these are the parts of it that say what the text is: a title is written as a
// heading or as the text of the link itself far more often than as anything
// else, and a date is often written as a relative phrase such as "2 days ago"
// while the element carries the real date in an attribute.
type SemanticText struct {
	// Tag is the name of the element the text was read from.
	Tag string `json:"tag"`
	// Value is the text as the reader sees it.
	Value string `json:"value"`
	// Datetime is the machine readable date of a <time> element, if it has
	// one. Value is kept as it is, because the two say different things and
	// only one of them can be shown to the reader.
	Datetime string `json:"datetime,omitempty"`
}

// semanticTexts returns every text of one subtree, in document order. It is a
// copy of itemTexts that keeps the element name and the datetime attribute
// instead of a selector; the enumeration is left as it is for now.
//
// An element whose element children hold no text is reported as one text,
// which keeps the list short: without the rule, one heading inside a link
// inside a card produces three entries carrying the same string.
//
// An element that holds text directly while one of its element children also
// holds text is reported once per run of text between the children, so that
// nothing written beside a child is lost and the order of the list stays the
// order of the document. This is what
// <a href="u">Learn more about <span>age ratings</span></a> needs, and what a
// date written after the links of a card needs. A run that holds no letter and
// no digit is dropped, because it is a separator such as "," or "·" between
// two children rather than a value of the post.
//
// escalate turns tag escalation on: a text is reported under the heading it
// sits inside, anywhere between the folded subtree and the element the text
// was read from, instead of under that element. The highest ranked heading of
// the path wins, so a text inside an <h2> inside an <h1> is reported as an
// <h1>. Without it the heading is lost as soon as the title is written inside
// a link, which is how a post list normally writes it.
func semanticTexts(n *html.Node, escalate bool) []SemanticText {
	var out []SemanticText
	// heading is the heading the element sits inside, and is only ever set
	// while escalate is on.
	var walk func(x *html.Node, heading string)
	walk = func(x *html.Node, heading string) {
		if escalate && headingTags[x.Data] && (heading == "" || x.Data < heading) {
			heading = x.Data
		}
		tag := x.Data
		if heading != "" {
			tag = heading
		}
		if !hasTextChild(x) {
			// No element child holds text, so no descendant does, and the
			// whole text of x is one block.
			if t := text(x); t != "" {
				out = append(out, SemanticText{
					Tag:      tag,
					Value:    t,
					Datetime: attr(x, "datetime"),
				})
			}
			return
		}
		var run strings.Builder
		flush := func() {
			t := strings.Join(strings.Fields(run.String()), " ")
			run.Reset()
			if strings.IndexFunc(t, func(r rune) bool {
				return unicode.IsLetter(r) || unicode.IsDigit(r)
			}) < 0 {
				return
			}
			out = append(out, SemanticText{
				Tag:      tag,
				Value:    t,
				Datetime: attr(x, "datetime"),
			})
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				run.WriteString(c.Data)
				run.WriteString(" ")
				continue
			}
			if c.Type == html.ElementNode {
				flush()
				walk(c, heading)
			}
		}
		flush()
	}
	walk(n, "")
	return out
}

// SemanticImage is one image candidate. The alt text is kept because it is
// often the only place a card names the post it shows.
type SemanticImage struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

func semanticNode(n *html.Node, page *url.URL, escalate bool) *SemanticNode {
	links := itemLinks(n, page)
	distinct := map[string]bool{}
	for _, l := range links {
		distinct[l.Value] = true
	}

	if len(distinct) <= 1 {
		texts := semanticTexts(n, escalate)
		images := itemImages(n, page)
		if len(links) == 0 && len(texts) == 0 && len(images) == 0 {
			// Nothing a post can be built from, so the subtree is dropped.
			return nil
		}
		node := &SemanticNode{Node: n}
		if len(links) > 0 {
			node.Link = links[0].Value
		}
		node.Texts = append(node.Texts, texts...)
		for _, a := range images {
			node.Images = append(node.Images, SemanticImage{URL: a.Value, Alt: a.Alt})
		}
		return node
	}

	var children []*SemanticNode
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if s := semanticNode(c, page, escalate); s != nil {
			children = append(children, s)
		}
	}

	// The values that sit on n itself. A link here is an element that wraps
	// several posts and links somewhere too, and a text here is a heading
	// written directly into the container, such as the section title above a
	// list. Both are kept because they describe the whole group.
	var own SemanticNode
	if u, ok := linkOf(n, page); ok {
		own.Link = u.String()
	}
	var loose strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			loose.WriteString(c.Data)
			loose.WriteString(" ")
		}
	}
	if t := strings.Join(strings.Fields(loose.String()), " "); t != "" {
		own.Texts = append(own.Texts, SemanticText{Tag: n.Data, Value: t})
	}

	if own.Link == "" && len(own.Texts) == 0 && len(children) == 1 {
		return children[0]
	}
	return &SemanticNode{
		Node:     n,
		Link:     own.Link,
		Texts:    own.Texts,
		Children: children,
	}
}
