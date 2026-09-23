package feed

import (
	"io"
	"net/url"
	"strings"

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
	// Links, Texts and Images are the candidate values of the node, in
	// document order. A folded node holds the values of its whole subtree; a
	// node that has children holds only the values that sit directly on Node,
	// because everything else belongs to one of the children.
	Links  []string        `json:"links,omitempty"`
	Texts  []string        `json:"texts,omitempty"`
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
	n := semanticNode(root, &page)
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

	linkless := func(c *SemanticNode) bool { return len(c.Links) == 0 && len(c.Children) == 0 }
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

// SemanticImage is one image candidate. The alt text is kept because it is
// often the only place a card names the post it shows.
type SemanticImage struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

func semanticNode(n *html.Node, page *url.URL) *SemanticNode {
	links := itemLinks(n, page)
	distinct := map[string]bool{}
	for _, l := range links {
		distinct[l.Value] = true
	}

	if len(distinct) <= 1 {
		texts := itemTexts(n)
		images := itemImages(n, page)
		if len(links) == 0 && len(texts) == 0 && len(images) == 0 {
			// Nothing a post can be built from, so the subtree is dropped.
			return nil
		}
		node := &SemanticNode{Node: n}
		for _, a := range links {
			node.Links = append(node.Links, a.Value)
		}
		for _, a := range texts {
			node.Texts = append(node.Texts, a.Value)
		}
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
		if s := semanticNode(c, page); s != nil {
			children = append(children, s)
		}
	}

	// The values that sit on n itself. A link here is an element that wraps
	// several posts and links somewhere too, and a text here is a heading
	// written directly into the container, such as the section title above a
	// list. Both are kept because they describe the whole group.
	var own SemanticNode
	if u, ok := linkOf(n, page); ok {
		own.Links = append(own.Links, u.String())
	}
	var loose strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			loose.WriteString(c.Data)
			loose.WriteString(" ")
		}
	}
	if t := strings.Join(strings.Fields(loose.String()), " "); t != "" {
		own.Texts = append(own.Texts, t)
	}

	if len(own.Links) == 0 && len(own.Texts) == 0 && len(children) == 1 {
		return children[0]
	}
	return &SemanticNode{
		Node:     n,
		Links:    own.Links,
		Texts:    own.Texts,
		Children: children,
	}
}
