package feed

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// The functions below read a node of a simplified page (see SimplifyPage).
// The client reads its nodes by the same rules.

// tagOf returns the tag of the element n, or "" for a text node.
func tagOf(n *html.Node) string {
	if n.Type != html.ElementNode {
		return ""
	}
	return n.Data
}

func classesOf(n *html.Node) []string {
	return strings.Fields(attr(n, "class"))
}

// isValue reports whether n is a value: a text node, or an element without
// child elements. Other elements are containers.
func isValue(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			return false
		}
	}
	return true
}

// childrenOf returns the child nodes of the container n, or none when n is a
// value, since the text of a value element is a part of it rather than a node
// of its own.
func childrenOf(n *html.Node) iter.Seq[*html.Node] {
	if isValue(n) {
		return func(func(*html.Node) bool) {}
	}
	return n.ChildNodes()
}

// textOf returns the text of the value n: the alt text of an image, or else
// the text in n.
func textOf(n *html.Node) string {
	switch {
	case n.Type == html.TextNode:
		return n.Data
	case n.Data == "img":
		return attr(n, "alt")
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return b.String()
}

// linkOf returns the link of n: the URL of an image, or else the link target.
func linkOf(n *html.Node) string {
	if tagOf(n) == "img" {
		return attr(n, "src")
	}
	return attr(n, "href")
}

// pageBody returns the body of page, a simplified page parsed by html.Parse,
// which always adds a body.
func pageBody(page *html.Node) *html.Node {
	for n := range page.Descendants() {
		if n.Type == html.ElementNode && n.Data == "body" {
			return n
		}
	}
	return nil
}

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
func (f PostField) valueOf(n *html.Node) (string, bool) {
	var v string
	switch f {
	case PostBody:
		v = bodyText(n)
	case PostLink, PostThumbnail:
		v = linkOf(n)
	case PostTitle, PostPubDate:
		v = textOf(n)
	}
	return v, v != ""
}

// accepts reports whether f can take a value from n: a title from a text, a
// link from a node with a link other than an image, a thumbnail from an
// image, a pub date from a time, and a body from a node with text.
func (f PostField) accepts(n *html.Node) bool {
	img, time := tagOf(n) == "img", tagOf(n) == "time"
	switch f {
	case PostTitle:
		return isValue(n) && !img && !time && textOf(n) != ""
	case PostBody:
		return !img && !time && bodyText(n) != ""
	case PostLink:
		return !img && linkOf(n) != ""
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

func (s Step) matches(n *html.Node) bool {
	if tagOf(n) != s.Tag {
		return false
	}
	cls := classesOf(n)
	for _, c := range s.Cls {
		if !slices.Contains(cls, c) {
			return false
		}
	}
	return true
}

// PostSelector is a rule that finds the posts of one kind in a page and
// extracts their fields. A selector editor generates selectors from the posts
// that the user labels on a page (see SelectorEditor), and the server runs
// them on each fetch of the page.
//
// Post is the path of steps to the post nodes: the first step selects
// descendants of the root, and each later step selects children of the nodes
// that the previous step selects. Each field has one path of steps from a
// post node to the node of its value; an empty path is the post node itself.
// Title and Link are required, since they define the kind of posts. Body,
// Thumbnail and PubDate are nil when the posts have none. A node that Post
// selects is a post only when its title and link are found (see hasFields),
// when its link matches none of the patterns in ExcludePatterns, and when it
// contains no other such node (see candidates).
//
// ExcludeNodes holds the paths of the nodes that the user excluded with their
// subtrees, such as a menu. Each step of such a path selects children, from
// the body to the node, and has the tag of its node and Nth, the index of the
// node among the children of its parent with the same tag, so that a path
// selects a node by its position in the page. The selector runs as if the
// subtrees of these nodes were not in the page.
type PostSelector struct {
	Post            []Step
	Title           []Step
	Link            []Step
	Body            []Step
	Thumbnail       []Step
	PubDate         []Step
	ExcludePatterns []string
	ExcludeNodes    [][]Step
}

// pathOf returns the path of f, or false when m has none.
func (m *PostSelector) pathOf(f PostField) ([]Step, bool) {
	var path []Step
	switch f {
	case PostTitle:
		path = m.Title
	case PostLink:
		path = m.Link
	case PostBody:
		path = m.Body
	case PostThumbnail:
		path = m.Thumbnail
	case PostPubDate:
		path = m.PubDate
	}
	return path, path != nil
}

// Fields returns the path of each field that has one.
func (m *PostSelector) Fields() map[PostField][]Step {
	fields := make(map[PostField][]Step)
	for _, f := range []PostField{PostTitle, PostBody, PostLink, PostThumbnail, PostPubDate} {
		if path, ok := m.pathOf(f); ok {
			fields[f] = path
		}
	}
	return fields
}

// postSelectorJSON is the JSON of PostSelector, whose fields hold the path of
// each field that has one.
type postSelectorJSON struct {
	Post            []Step               `json:"post"`
	Fields          map[PostField][]Step `json:"fields"`
	ExcludePatterns []string             `json:"excludePatterns,omitempty"`
	ExcludeNodes    [][]Step             `json:"excludeNodes,omitempty"`
}

func (m PostSelector) MarshalJSON() ([]byte, error) {
	return json.Marshal(postSelectorJSON{
		Post:            m.Post,
		Fields:          m.Fields(),
		ExcludePatterns: m.ExcludePatterns,
		ExcludeNodes:    m.ExcludeNodes,
	})
}

func (m *PostSelector) UnmarshalJSON(data []byte) error {
	var j postSelectorJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*m = PostSelector{Post: j.Post, ExcludePatterns: j.ExcludePatterns, ExcludeNodes: j.ExcludeNodes}
	for f, path := range j.Fields {
		// An empty path is the post node itself, which nil would leave out.
		if path == nil {
			path = []Step{}
		}
		switch f {
		case PostTitle:
			m.Title = path
		case PostLink:
			m.Link = path
		case PostBody:
			m.Body = path
		case PostThumbnail:
			m.Thumbnail = path
		case PostPubDate:
			m.PubDate = path
		default:
			return fmt.Errorf("unknown field %q", f)
		}
	}
	if m.Title == nil || m.Link == nil {
		return fmt.Errorf("no path of the title or the link")
	}
	return nil
}

// selectNodes returns the nodes that path selects from start, in document
// order. With children, each step after the first selects among the children
// of the nodes that the previous step selects, instead of their descendants.
// The subtrees of the nodes in skip are left out.
func selectNodes(start *html.Node, path []Step, children bool, skip map[*html.Node]bool) []*html.Node {
	nodes := []*html.Node{start}
	for i, step := range path {
		var next []*html.Node
		seen := map[*html.Node]bool{}
		for _, n := range nodes {
			var matches []*html.Node
			var visit func(n *html.Node)
			visit = func(n *html.Node) {
				for c := range childrenOf(n) {
					if skip[c] {
						continue
					}
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

// excludedNodes returns the nodes in body that the paths of ExcludeNodes
// select. A path that selects no node is left out.
func (m *PostSelector) excludedNodes(body *html.Node) map[*html.Node]bool {
	excluded := map[*html.Node]bool{}
paths:
	for _, path := range m.ExcludeNodes {
		n := body
		for _, step := range path {
			var next *html.Node
			i := 0
			for c := range childrenOf(n) {
				if !step.matches(c) {
					continue
				}
				if step.Nth == nil || i == *step.Nth {
					next = c
					break
				}
				i++
			}
			if next == nil {
				continue paths
			}
			n = next
		}
		if n != body {
			excluded[n] = true
		}
	}
	return excluded
}

// selectPosts returns the post nodes that m finds in body, the body of a
// simplified page: the candidates (see candidates) whose links match none of
// the patterns in ExcludePatterns. skip holds the excluded nodes (see
// excludedNodes).
func (m *PostSelector) selectPosts(body *html.Node, skip map[*html.Node]bool) []*html.Node {
	exclude := m.compileExclude()
	return slices.DeleteFunc(m.candidates(body, skip), func(n *html.Node) bool {
		_, excluded := m.excludedBy(n, exclude, skip)
		return excluded
	})
}

// candidates returns the nodes in body that Post selects and that have the
// title and the link, including the ones whose links are excluded. A node that
// contains another such node is left out, since it is a list of posts rather
// than a post. Other nodes that Post selects inside a post, such as a footer
// with the same classes, do not matter. The subtrees of the nodes in skip are
// left out.
func (m *PostSelector) candidates(body *html.Node, skip map[*html.Node]bool) []*html.Node {
	if len(m.Post) == 0 {
		return nil
	}
	selected := map[*html.Node]bool{}
	for _, n := range selectNodes(body, m.Post, true, skip) {
		selected[n] = true
	}
	var posts []*html.Node
	// visit returns whether n is or contains a selected node with the title
	// and the link.
	var visit func(n *html.Node) bool
	visit = func(n *html.Node) bool {
		inner := false
		for c := range childrenOf(n) {
			if !skip[c] {
				inner = visit(c) || inner
			}
		}
		isPost := selected[n] && m.hasFields(n, skip)
		if isPost && !inner {
			posts = append(posts, n)
		}
		return inner || isPost
	}
	visit(body)
	return posts
}

// hasFields reports whether post has the title and the link. The other fields
// are optional, since posts of one kind often lack some of them, such as the
// date of a featured post.
func (m *PostSelector) hasFields(post *html.Node, skip map[*html.Node]bool) bool {
	return m.resolveField(post, PostTitle, skip) != nil && m.resolveField(post, PostLink, skip) != nil
}

// resolveField returns the node of f in post: the first node that the path
// of f selects outside the subtrees of skip and that f accepts, or nil when
// there is none.
func (m *PostSelector) resolveField(post *html.Node, f PostField, skip map[*html.Node]bool) *html.Node {
	path, ok := m.pathOf(f)
	if !ok {
		return nil
	}
	return resolvePath(post, f, path, skip)
}

// resolvePath returns the first node that path selects from post outside the
// subtrees of skip and that f accepts, or nil when there is none.
func resolvePath(post *html.Node, f PostField, path []Step, skip map[*html.Node]bool) *html.Node {
	for _, n := range selectNodes(post, path, false, skip) {
		if f.accepts(n) {
			return n
		}
	}
	return nil
}

// compileExclude returns the regexps of the patterns in ExcludePatterns.
func (m *PostSelector) compileExclude() []*regexp.Regexp {
	exclude := make([]*regexp.Regexp, len(m.ExcludePatterns))
	for i, p := range m.ExcludePatterns {
		exclude[i] = globRegexp(p)
	}
	return exclude
}

// excludedBy returns the first pattern in ExcludePatterns that matches the
// host and path of the link of post, or false when none does. exclude holds
// the compiled patterns (see compileExclude), and skip the excluded nodes.
func (m *PostSelector) excludedBy(post *html.Node, exclude []*regexp.Regexp, skip map[*html.Node]bool) (string, bool) {
	n := m.resolveField(post, PostLink, skip)
	if n == nil {
		return "", false
	}
	u, err := url.Parse(linkOf(n))
	if err != nil {
		return "", false
	}
	hostPath := u.Host + u.Path
	for i, re := range exclude {
		if re.MatchString(hostPath) {
			return m.ExcludePatterns[i], true
		}
	}
	return "", false
}

// globRegexp compiles the file glob pattern into a regexp that matches a
// whole host and path: "*" matches any characters except "/", "**" matches
// any characters, and "?" matches one character except "/".
func globRegexp(pattern string) *regexp.Regexp {
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
	return regexp.MustCompile(re.String())
}

// ExtractedPost is a post that a selector found in a page, by its field
// values.
type ExtractedPost map[PostField]string

// ExtractPosts returns the posts that selectors find in page, a simplified
// page parsed by html.Parse, in document order.
// Posts with the same link are merged into the first one, which takes the
// fields that it does not have from the later ones, since some pages split a
// post into sibling cards with the same link, such as one with the image and
// one with the title. Posts without a link are dropped.
func ExtractPosts(page *html.Node, selectors []PostSelector) []ExtractedPost {
	body := pageBody(page)
	order := map[*html.Node]int{}
	for n := range body.Descendants() {
		order[n] = len(order)
	}

	type found struct {
		node   *html.Node
		values ExtractedPost
	}
	var posts []found
	for _, m := range selectors {
		skip := m.excludedNodes(body)
		for _, n := range m.selectPosts(body, skip) {
			values := make(ExtractedPost)
			for field := range m.Fields() {
				if v := m.resolveField(n, field, skip); v != nil {
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
func bodyText(n *html.Node) string {
	var texts []string
	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if tag := tagOf(n); tag == "img" || tag == "time" {
			return
		}
		if n.Type == html.TextNode {
			texts = append(texts, n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return normalizeSpace(closeGaps(strings.Join(texts, gap)))
}
