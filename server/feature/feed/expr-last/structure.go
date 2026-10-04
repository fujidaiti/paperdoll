// Package structures is an experiment that builds post structures for a page.
// DEF.md in this directory defines what a structure is, what the algorithm
// receives and produces, and what the metrics measure. Nothing in this
// directory is used by the feed package.
package structures

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// A Structure is the shape of one post. It is one item of what the algorithm
// outputs for a page, and it is a tree of parts.
type Structure struct {
	Root *Part
}

// A Part is one node of a structure's tree. It holds a set of fields and a
// part matcher.
type Part struct {
	Name     string
	Fields   []*Field
	Children []*Part
	Match    PartMatcher
}

// A Field stands for one attribute of a post, such as the title. Exactly one
// of Match and Blob is set: a field with Blob set is a blob, which reads a run
// of sibling nodes instead of a single node.
type Field struct {
	Name    string
	Match   FieldMatcher
	Blob    BlobMatcher
	Samples []Sample
}

// A Sample is one fragment the algorithm found in the input page: one node, or
// a run of consecutive sibling nodes when the field is a blob. Samples are
// kept so that the review screen can show what was found. No metric reads
// them.
type Sample struct {
	Nodes []*html.Node
}

// A PartMatcher receives a DOM subtree, which is the whole page for a root
// part and one subtree the parent part matched for any other part. It returns
// the roots of non-overlapping subtrees inside it, in page order.
type PartMatcher func(root *html.Node) []*html.Node

// A FieldMatcher receives one subtree that the part holding the field matched,
// and returns at most one node inside it. nil means the field found nothing
// there, which is how an absent description is represented.
type FieldMatcher func(root *html.Node) *html.Node

// A BlobMatcher receives one subtree that the part holding the field matched,
// and returns a run of consecutive sibling nodes under one parent. Everything
// below those nodes belongs to the blob as well. nil means the blob found
// nothing there.
type BlobMatcher func(root *html.Node) []*html.Node

// Depth returns the length of the deepest chain of parts in the structure. A
// structure of a single part has depth 1.
func (s *Structure) Depth() int {
	var depth func(*Part) int
	depth = func(p *Part) int {
		best := 0
		for _, c := range p.Children {
			if d := depth(c); d > best {
				best = d
			}
		}
		return best + 1
	}
	if s == nil || s.Root == nil {
		return 0
	}
	return depth(s.Root)
}

// urlAttrs are the observable attributes whose value is a URL. Their value is
// resolved against the page URL.
var urlAttrs = []string{"href", "src", "data-src", "poster", "srcset"}

// textAttrs are the remaining observable attributes. Their value is read as
// text.
var textAttrs = []string{"alt", "title", "datetime", "content", "value"}

// hiddenElements never carry a value a user would observe, so no value is read
// from them.
var hiddenElements = map[string]bool{
	"script": true, "style": true, "noscript": true,
	"template": true, "head": true,
}

// values returns what one node offers: one value for each text written
// directly inside it, in document order, followed by one value for each
// observable attribute it carries. Text written inside a child node belongs to
// that child and is not returned here.
//
// Values are read from element nodes. The text of a text node belongs to the
// element around it, so a text node offers nothing of its own and a node a
// user cannot observe offers nothing at all.
func values(n *html.Node, base *url.URL) []string {
	if !observable(n) {
		return nil
	}
	var out []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.TextNode {
			continue
		}
		if s := flat(c.Data); s != "" {
			out = append(out, s)
		}
	}
	if n.Type != html.ElementNode {
		return out
	}
	for _, key := range urlAttrs {
		raw := attr(n, key)
		if raw == "" {
			continue
		}
		if key == "srcset" {
			// A srcset holds several candidates separated by commas, each
			// being a URL followed by an optional descriptor such as "2x".
			for candidate := range strings.SplitSeq(raw, ",") {
				parts := strings.Fields(candidate)
				if len(parts) == 0 {
					continue
				}
				if u := resolve(base, parts[0]); u != "" {
					out = append(out, u)
				}
			}
			continue
		}
		if u := resolve(base, raw); u != "" {
			out = append(out, u)
		}
	}
	for _, key := range textAttrs {
		if s := flat(attr(n, key)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// observable reports whether a user can see what the node carries. Only the
// node itself is checked, not the nodes above it.
func observable(n *html.Node) bool {
	if n == nil || n.Type == html.CommentNode || n.Type == html.DoctypeNode {
		return false
	}
	if n.Type != html.ElementNode {
		return true
	}
	if hiddenElements[n.Data] {
		return false
	}
	if hasAttr(n, "hidden") || strings.EqualFold(attr(n, "aria-hidden"), "true") {
		return false
	}
	style := strings.ToLower(strings.ReplaceAll(attr(n, "style"), " ", ""))
	return !strings.Contains(style, "display:none") &&
		!strings.Contains(style, "visibility:hidden")
}

// hasAttr reports whether the node carries the attribute at all. The hidden
// attribute is written without a value, so its presence is what counts.
func hasAttr(n *html.Node, key string) bool {
	if n == nil {
		return false
	}
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return true
		}
	}
	return false
}

func attr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// flat collapses runs of whitespace to one space and trims the ends, which is
// the normalization every text value is built with.
func flat(s string) string { return strings.Join(strings.Fields(s), " ") }

// resolve turns a raw attribute value into an absolute URL and normalizes it.
// It returns an empty string for a value that points at no page, such as an
// inline data URI.
func resolve(base *url.URL, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "data:") {
		return ""
	}
	u, err := base.Parse(raw)
	if err != nil {
		return ""
	}
	return normalizeURL(u)
}

// normalizeURL is the normalization every URL value is built with. The scheme
// is dropped and the host is lowered, because a page and a fixture can write
// the same link with a different scheme or a different case. Percent escapes
// are decoded for the same reason.
func normalizeURL(u *url.URL) string {
	out := strings.ToLower(u.Host) + strings.TrimSuffix(u.Path, "/")
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		out += "#" + u.Fragment
	}
	if decoded, err := url.PathUnescape(out); err == nil {
		out = decoded
	}
	return flat(out)
}
