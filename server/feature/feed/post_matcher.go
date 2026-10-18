package feed

import (
	"net/url"
	"regexp"
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

// FieldStep is one step of the path from a post node to a field: the child
// with Shape at the zero-based index Nth among the children with that shape.
type FieldStep struct {
	Shape string `json:"shape"`
	Nth   int    `json:"nth"`
}

// PostMatcher is a rule that finds the posts of one shape in a page and
// extracts their fields (see tool/semantictree/MATCHER.md). The client builds
// matchers when the user annotates a page, and the server runs them on each
// fetch of the page.
//
// Post is the shape path of the post nodes from the root, and each field is
// the path of steps from a post node to its value. A node at Post is a post
// only when all labeled fields are found, and when its link matches none of
// the patterns in Exclude.
type PostMatcher struct {
	Post    []string                  `json:"post"`
	Fields  map[PostField][]FieldStep `json:"fields"`
	Exclude []string                  `json:"exclude"`
}

// shape returns the shape of n, such as "card", "h2" or "txt+link".
func shape(n *SemanticTreeNode) string {
	if !n.isValue() {
		if n.Link != "" {
			return "card"
		}
		return "box"
	}
	s := n.Kind
	if s == "" {
		s = "txt"
	}
	if n.Link != "" {
		s += "+link"
	}
	return s
}

// matchPosts returns the post nodes that m finds in doc.
func (m *PostMatcher) matchPosts(doc *SemanticTree) []*SemanticTreeNode {
	if len(m.Post) == 0 || shape(doc.Root) != m.Post[0] {
		return nil
	}
	nodes := []*SemanticTreeNode{doc.Root}
	for _, s := range m.Post[1:] {
		var next []*SemanticTreeNode
		for _, n := range nodes {
			for _, c := range n.Children {
				if shape(c) == s {
					next = append(next, c)
				}
			}
		}
		nodes = next
	}
	var posts []*SemanticTreeNode
	for _, n := range nodes {
		if m.hasFields(n) && !m.isExcluded(n) {
			posts = append(posts, n)
		}
	}
	return posts
}

func (m *PostMatcher) hasFields(post *SemanticTreeNode) bool {
	for _, steps := range m.Fields {
		if resolve(post, steps) == nil {
			return false
		}
	}
	return true
}

// isExcluded reports whether the link of post matches a pattern in Exclude.
func (m *PostMatcher) isExcluded(post *SemanticTreeNode) bool {
	steps, ok := m.Fields[PostLink]
	if !ok {
		return false
	}
	n := resolve(post, steps)
	if n == nil || n.Link == "" {
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

// resolve returns the node at steps from n, or nil when there is none.
func resolve(n *SemanticTreeNode, steps []FieldStep) *SemanticTreeNode {
	for _, step := range steps {
		var match *SemanticTreeNode
		i := 0
		for _, c := range n.Children {
			if shape(c) != step.Shape {
				continue
			}
			if i == step.Nth {
				match = c
				break
			}
			i++
		}
		if match == nil {
			return nil
		}
		n = match
	}
	return n
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

// ExtractPosts returns the posts that matchers find in doc, in document order.
// Posts with the same link are merged into the first one, which takes the
// fields that it does not have from the later ones, since some pages split a
// post into sibling cards with the same link, such as one with the image and
// one with the title. Posts without a link are dropped.
func ExtractPosts(doc *SemanticTree, matchers []PostMatcher) []ExtractedPost {
	order := map[*SemanticTreeNode]int{}
	var number func(n *SemanticTreeNode)
	number = func(n *SemanticTreeNode) {
		order[n] = len(order)
		for _, c := range n.Children {
			number(c)
		}
	}
	number(doc.Root)

	type found struct {
		node   *SemanticTreeNode
		values ExtractedPost
	}
	var posts []found
	for _, m := range matchers {
		for _, n := range m.matchPosts(doc) {
			values := ExtractedPost{}
			for field, steps := range m.Fields {
				if v := resolve(n, steps); v != nil {
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
// spaces. Images and times are left out, and links keep only their text. A
// body is a short summary shown below the title of a post, so its structure
// does not matter.
func bodyText(n *SemanticTreeNode) string {
	var texts []string
	var visit func(n *SemanticTreeNode)
	visit = func(n *SemanticTreeNode) {
		if n.Kind != "img" && n.Kind != "time" {
			texts = append(texts, n.Val)
		}
		for _, c := range n.Children {
			visit(c)
		}
	}
	visit(n)
	return normalizeSpace(strings.Join(texts, " "))
}
