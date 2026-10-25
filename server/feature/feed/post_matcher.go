package feed

import (
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// PostField is an attribute of a post that the user maps to nodes of a page.
type PostField string

const (
	PostTitle PostField = "title"
	// PostBody is the text of a post as one line (see bodyText).
	PostBody      PostField = "body"
	PostLink      PostField = "link"
	PostThumbnail PostField = "thumbnail"
	PostPubDate   PostField = "pubDate"
)

// valueOf returns the value that f takes when it is mapped to n, or false
// when n has none, such as an image without alt text mapped to a title.
func (f PostField) valueOf(n *SemanticTreeNode) (string, bool) {
	switch f {
	case PostBody:
		t := bodyText(n)
		return t, t != ""
	case PostLink, PostThumbnail:
		return n.Link, n.Link != ""
	case PostTitle, PostPubDate:
		return n.Val, n.Val != ""
	}
	return "", false
}

// accepts reports whether f can take a value from n: a title from a text, a
// link from a node with a link other than an image, a thumbnail from an
// image, a pub date from a time, and a body from a node with text.
func (f PostField) accepts(n *SemanticTreeNode) bool {
	img, time := n.Tag == "img", n.Tag == "time"
	switch f {
	case PostTitle:
		return n.isValue() && !img && !time && n.Val != ""
	case PostBody:
		return !img && !time && bodyText(n) != ""
	case PostLink:
		return !img && n.Link != ""
	case PostThumbnail:
		return img
	case PostPubDate:
		return time
	}
	return false
}

// Step selects nodes by their tag and classes, as a step of a CSS descendant
// selector does: the descendants of a node that have Tag and every class in
// Cls. A step without Tag selects plain texts, which have no tag. When Nth is
// set, only the one at that zero-based index among them in document order is
// selected.
type Step struct {
	Tag string   `json:"tag,omitempty"`
	Cls []string `json:"cls,omitempty"`
	Nth *int     `json:"nth,omitempty"`
}

func (s Step) matches(n *SemanticTreeNode) bool {
	if n.Tag != s.Tag {
		return false
	}
	for _, c := range s.Cls {
		if !slices.Contains(n.Cls, c) {
			return false
		}
	}
	return true
}

// PostMatcher is a rule that finds the posts of one kind in a page and
// extracts their fields. The client builds matchers when the user annotates
// a page, and the server runs them on each fetch of the page.
//
// Post is the chain of steps to the post nodes: the first step selects
// descendants of the root, and each later step selects children of the nodes
// that the previous step selects. Each field has
// chains of steps from a post node to the node of its value, tried in order;
// an empty chain is the post node itself. A node that Post selects is a post
// only when its title and link are found (see hasFields), and when its link
// matches none of the patterns in Exclude.
type PostMatcher struct {
	Post    []Step                 `json:"post"`
	Fields  map[PostField][][]Step `json:"fields"`
	Exclude []string               `json:"exclude"`
}

// selectNodes returns the nodes that chain selects from start, in document
// order. With children, each step after the first selects among the children
// of the nodes that the previous step selects, instead of their descendants.
func selectNodes(start *SemanticTreeNode, chain []Step, children bool) []*SemanticTreeNode {
	nodes := []*SemanticTreeNode{start}
	for i, step := range chain {
		var next []*SemanticTreeNode
		seen := map[*SemanticTreeNode]bool{}
		for _, n := range nodes {
			var matches []*SemanticTreeNode
			var visit func(n *SemanticTreeNode)
			visit = func(n *SemanticTreeNode) {
				for _, c := range n.Children {
					if step.matches(c) {
						matches = append(matches, c)
					}
					if !children || i == 0 {
						visit(c)
					}
				}
			}
			visit(n)
			if step.Nth != nil {
				if *step.Nth >= len(matches) {
					continue
				}
				matches = matches[*step.Nth : *step.Nth+1]
			}
			for _, m := range matches {
				if !seen[m] {
					seen[m] = true
					next = append(next, m)
				}
			}
		}
		nodes = next
	}
	return nodes
}

// matchPosts returns the post nodes that m finds in tree. A node that
// contains another node that Post selects is left out, since it is a list of
// posts rather than a post.
func (m *PostMatcher) matchPosts(tree *SemanticTree) []*SemanticTreeNode {
	if len(m.Post) == 0 {
		return nil
	}
	selected := map[*SemanticTreeNode]bool{}
	for _, n := range selectNodes(tree.Root, m.Post, true) {
		selected[n] = true
	}
	var posts []*SemanticTreeNode
	// visit returns whether n is or contains a selected node.
	var visit func(n *SemanticTreeNode) bool
	visit = func(n *SemanticTreeNode) bool {
		inner := false
		for _, c := range n.Children {
			inner = visit(c) || inner
		}
		if selected[n] && !inner && m.hasFields(n) && !m.isExcluded(n) {
			posts = append(posts, n)
		}
		return inner || selected[n]
	}
	visit(tree.Root)
	return posts
}

// hasFields reports whether post has the title and the link, when they are
// labeled. The other fields are optional, since posts of one kind often lack
// some of them, such as the date of a featured post.
func (m *PostMatcher) hasFields(post *SemanticTreeNode) bool {
	for _, f := range []PostField{PostTitle, PostLink} {
		if _, ok := m.Fields[f]; ok && m.resolveField(post, f) == nil {
			return false
		}
	}
	return true
}

// resolveField returns the node of f in post: the first node that the first
// chain of f selects with a node that f accepts, or nil when there is none.
func (m *PostMatcher) resolveField(post *SemanticTreeNode, f PostField) *SemanticTreeNode {
	for _, chain := range m.Fields[f] {
		if n := resolveChain(post, f, chain); n != nil {
			return n
		}
	}
	return nil
}

// resolveChain returns the first node that chain selects from post and that
// f accepts, or nil when there is none.
func resolveChain(post *SemanticTreeNode, f PostField, chain []Step) *SemanticTreeNode {
	for _, n := range selectNodes(post, chain, false) {
		if f.accepts(n) {
			return n
		}
	}
	return nil
}

// isExcluded reports whether the link of post matches a pattern in Exclude.
func (m *PostMatcher) isExcluded(post *SemanticTreeNode) bool {
	n := m.resolveField(post, PostLink)
	if n == nil {
		return false
	}
	u, err := url.Parse(n.Link)
	if err != nil {
		return false
	}
	hostPath := u.Host + u.Path
	for _, p := range m.Exclude {
		if globMatches(p, hostPath) {
			return true
		}
	}
	return false
}

// globMatches reports whether the file glob pattern matches hostPath: "*"
// matches any characters except "/", "**" matches any characters, and "?"
// matches one character except "/".
func globMatches(pattern, hostPath string) bool {
	var re strings.Builder
	re.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch {
		case strings.HasPrefix(pattern[i:], "**"):
			re.WriteString(".*")
			i++
		case pattern[i] == '*':
			re.WriteString("[^/]*")
		case pattern[i] == '?':
			re.WriteString("[^/]")
		default:
			re.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
	}
	re.WriteString("$")
	return regexp.MustCompile(re.String()).MatchString(hostPath)
}

// ExtractedPost is a post that a matcher found in a page, by its field
// values.
type ExtractedPost map[PostField]string

// ExtractPosts returns the posts that matchers find in tree, in document order.
// Posts with the same link are merged into the first one, which takes the
// fields that it does not have from the later ones, since some pages split a
// post into sibling cards with the same link, such as one with the image and
// one with the title. Posts without a link are dropped.
func ExtractPosts(tree *SemanticTree, matchers []PostMatcher) []ExtractedPost {
	order := map[*SemanticTreeNode]int{}
	var number func(n *SemanticTreeNode)
	number = func(n *SemanticTreeNode) {
		order[n] = len(order)
		for _, c := range n.Children {
			number(c)
		}
	}
	number(tree.Root)

	type found struct {
		node   *SemanticTreeNode
		values ExtractedPost
	}
	var posts []found
	for _, m := range matchers {
		for _, n := range m.matchPosts(tree) {
			values := make(ExtractedPost)
			for field := range m.Fields {
				if v := m.resolveField(n, field); v != nil {
					if val, ok := field.valueOf(v); ok {
						values[field] = val
					}
				}
			}
			posts = append(posts, found{n, values})
		}
	}
	sort.SliceStable(posts, func(i, j int) bool { return order[posts[i].node] < order[posts[j].node] })

	var result []ExtractedPost
	byLink := map[string]ExtractedPost{}
	for _, p := range posts {
		link, ok := p.values[PostLink]
		if !ok {
			continue
		}
		first, ok := byLink[link]
		if !ok {
			byLink[link] = p.values
			result = append(result, p.values)
			continue
		}
		for field, v := range p.values {
			if _, ok := first[field]; !ok {
				first[field] = v
			}
		}
	}
	return result
}

// bodyText returns the text of the values in the subtree of n, joined with
// spaces as in the page, such as "Review’s" from the values "Review" and
// "’s". Images and times are left out, and links keep only their text. A
// body is a short summary shown below the title of a post, so its structure
// does not matter.
func bodyText(n *SemanticTreeNode) string {
	var texts []string
	var visit func(n *SemanticTreeNode)
	visit = func(n *SemanticTreeNode) {
		if n.Tag != "img" && n.Tag != "time" {
			texts = append(texts, n.Val)
		}
		for _, c := range n.Children {
			visit(c)
		}
	}
	visit(n)
	return normalizeSpace(closeGaps(strings.Join(texts, gap)))
}
