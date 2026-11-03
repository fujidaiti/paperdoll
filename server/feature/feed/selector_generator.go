package feed

import (
	"maps"
	"regexp"
	"slices"

	"golang.org/x/net/html"
)

// The functions below generate the post selectors of a page from the posts
// that a user labeled in a selector editor (see SelectorEditor).

// pageIndex is the body of a simplified page with its page nodes in document
// order, and the number of nodes with each class. The page nodes are the body
// and every node that childrenOf yields below it, and the position of a node
// in nodes is its node ID (see withNodeIDs).
type pageIndex struct {
	body    *html.Node
	nodes   []*html.Node
	index   map[*html.Node]int
	classes map[string]int
}

func newPageIndex(body *html.Node) *pageIndex {
	t := &pageIndex{body: body, index: map[*html.Node]int{}, classes: map[string]int{}}
	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		t.index[n] = len(t.nodes)
		t.nodes = append(t.nodes, n)
		for _, c := range classesOf(n) {
			t.classes[c]++
		}
		for c := range childrenOf(n) {
			visit(c)
		}
	}
	visit(body)
	return t
}

// contains reports whether n is a or one of its descendants.
func (t *pageIndex) contains(a, n *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == a {
			return true
		}
	}
	return false
}

// step returns the step of n: its tag and the classes that other nodes have
// too. A class of n alone, such as an ID, cannot be shared by two posts.
func (t *pageIndex) step(n *html.Node) Step {
	s := Step{Tag: tagOf(n)}
	for _, c := range classesOf(n) {
		if t.classes[c] >= 2 {
			s.Cls = append(s.Cls, c)
		}
	}
	return s
}

var (
	// variedPostClass matches the classes that differ between the posts of a
	// list, such as the ID and the categories of a WordPress post.
	variedPostClass = regexp.MustCompile(`^(post-\d+|category-.+|tag-.+|has-post-thumbnail)$`)
	// postKindClass matches the classes with the word "post" or "card", which
	// name the kind of item rather than its layout or state. A word starts at
	// the start of the class, after "-" or "_", or with an uppercase letter,
	// and ends at the end of the class, before "-" or "_", or before an
	// uppercase letter.
	postKindClass = regexp.MustCompile(`(^|[-_])(post|card)($|[-_A-Z])|(Post|Card)($|[-_A-Z])`)
)

// postClasses returns the classes of cls that a post node keeps in its step:
// the ones that postKindClass matches, or all when none does, without the
// ones that variedPostClass matches. A step built from a single post would
// otherwise keep the classes of that post alone, and miss the other posts
// of the list.
func postClasses(cls []string) []string {
	cls = slices.DeleteFunc(slices.Clone(cls), variedPostClass.MatchString)
	if kinds := slices.DeleteFunc(slices.Clone(cls), func(c string) bool {
		return !postKindClass.MatchString(c)
	}); len(kinds) > 0 {
		return kinds
	}
	return cls
}

// postPath returns the path that selects post from the body: the step of
// post when it has classes left (see postClasses), or else the steps of the
// nodes from its nearest ancestor with classes, or from the top of the page
// below the body, down to post.
func (t *pageIndex) postPath(post *html.Node) []Step {
	s := t.step(post)
	s.Cls = postClasses(s.Cls)
	path := []Step{s}
	for n := post; len(path[0].Cls) == 0 && n.Parent != t.body; {
		n = n.Parent
		path = append([]Step{t.step(n)}, path...)
	}
	return path
}

// positionPath returns the path that selects n by its position (see
// ExcludeNodes of PostSelector).
func (t *pageIndex) positionPath(n *html.Node) []Step {
	var path []Step
	for ; n != t.body; n = n.Parent {
		nth := 0
		for s := n.PrevSibling; s != nil; s = s.PrevSibling {
			if tagOf(s) == tagOf(n) {
				nth++
			}
		}
		path = append([]Step{{Tag: tagOf(n), Nth: &nth}}, path...)
	}
	return path
}

// fullPathOf returns the steps of the nodes from post, exclusive, down to n.
func (t *pageIndex) fullPathOf(post, n *html.Node) []Step {
	var fullPath []Step
	for a := n; a != post; a = a.Parent {
		fullPath = append([]Step{t.step(a)}, fullPath...)
	}
	return fullPath
}

// fieldPath returns the path that resolves f to n in post: the shortest
// tail of the full path from post to n that does, and that has a step with
// classes when the full path has one. It falls back to the full path, with Nth
// on the last step. It returns false when no path resolves to n. The subtrees
// of the excluded nodes in skip are left out, as when the selector runs.
func (t *pageIndex) fieldPath(f PostField, post, n *html.Node, skip map[*html.Node]bool) ([]Step, bool) {
	fullPath := t.fullPathOf(post, n)
	hasCls := func(path []Step) bool {
		return slices.ContainsFunc(path, func(s Step) bool { return len(s.Cls) > 0 })
	}
	for i := len(fullPath); i >= 0; i-- {
		path := append([]Step{}, fullPath[i:]...)
		// The empty path resolves only to the post node itself.
		if (i == len(fullPath) && len(fullPath) > 0) || hasCls(fullPath) && !hasCls(path) {
			continue
		}
		if resolvePath(post, f, path, skip) == n {
			return path, true
		}
	}
	if len(fullPath) == 0 {
		return nil, false
	}
	for nth := 0; nth < len(t.nodes); nth++ {
		path := slices.Clone(fullPath)
		path[len(path)-1].Nth = &nth
		if len(selectNodes(post, path, false, skip)) == 0 {
			return nil, false
		}
		if resolvePath(post, f, path, skip) == n {
			return path, true
		}
	}
	return nil, false
}

// labeledPost is a post that the user labeled: its node and the nodes of its
// fields.
type labeledPost struct {
	node   *html.Node
	fields map[PostField]*html.Node
}

// setPath sets the path of f in m.
func setPath(m *PostSelector, f PostField, path []Step) {
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
	}
}

// selectorOf returns the selector that finds post with its fields, or false when
// no path resolves its title or link. A field that no path resolves is left
// out. exclude and excludeNodes are the ExcludePatterns and ExcludeNodes of
// the selector, and skip holds the nodes of excludeNodes.
func (t *pageIndex) selectorOf(
	post labeledPost,
	exclude []string,
	excludeNodes [][]Step,
	skip map[*html.Node]bool,
) (PostSelector, bool) {
	m := PostSelector{Post: t.postPath(post.node), ExcludePatterns: exclude, ExcludeNodes: excludeNodes}
	for f, n := range post.fields {
		if path, ok := t.fieldPath(f, post.node, n, skip); ok {
			setPath(&m, f, path)
		}
	}
	return m, m.Title != nil && m.Link != nil
}

// samePaths reports whether a and b have the same steps, with the classes of
// each step in any order.
func samePaths(a, b []Step) bool {
	return slices.EqualFunc(a, b, func(x, y Step) bool {
		return x.Tag == y.Tag &&
			(x.Nth == nil) == (y.Nth == nil) && (x.Nth == nil || *x.Nth == *y.Nth) &&
			len(x.Cls) == len(y.Cls) &&
			!slices.ContainsFunc(x.Cls, func(c string) bool { return !slices.Contains(y.Cls, c) })
	})
}

// merge returns the selector that finds the posts of a and of b with their
// fields, or false when they cannot merge. They merge when their post paths
// have the same tags and the same classes except on the last step, and when
// they have the same title and link paths, and no other field with
// different paths. The last step of the merged post path keeps the classes
// that a and b share, and the merge fails when no step has classes left, since
// such a path selects any node of its tags.
//
// The merged selector finds the posts of a and of b without a check on the
// page: its post path selects their nodes, and their fields resolve by the
// same paths as before.
func merge(a, b PostSelector) (PostSelector, bool) {
	if len(a.Post) != len(b.Post) {
		return PostSelector{}, false
	}
	last := len(a.Post) - 1
	for i := range a.Post {
		if a.Post[i].Tag != b.Post[i].Tag || i < last && !samePaths(a.Post[i:i+1], b.Post[i:i+1]) {
			return PostSelector{}, false
		}
	}
	post := slices.Clone(a.Post)
	post[last].Cls = slices.DeleteFunc(
		slices.Clone(post[last].Cls),
		func(c string) bool { return !slices.Contains(b.Post[last].Cls, c) },
	)
	if !slices.ContainsFunc(post, func(s Step) bool { return len(s.Cls) > 0 }) {
		return PostSelector{}, false
	}
	if !samePaths(a.Title, b.Title) || !samePaths(a.Link, b.Link) {
		return PostSelector{}, false
	}
	m := PostSelector{
		Post:            post,
		Title:           a.Title,
		Link:            a.Link,
		ExcludePatterns: a.ExcludePatterns,
		ExcludeNodes:    a.ExcludeNodes,
	}
	for _, f := range []PostField{PostBody, PostThumbnail, PostPubDate} {
		path, ok := a.pathOf(f)
		if other, okOther := b.pathOf(f); !ok {
			path = other
		} else if okOther && !samePaths(path, other) {
			return PostSelector{}, false
		}
		setPath(&m, f, path)
	}
	return m, true
}

// generateSelectors returns the selectors that find the posts that the user
// labeled, and none of the posts whose links the user excluded, which excluded
// holds by host and path, and nothing in the subtrees of excludedNodes. The
// selectors depend on the posts, the links and the nodes, not on the order in
// which the user marked them. A link of excluded must not be the link of a
// post, and a post and its fields must not be in the subtree of an excluded
// node.
//
// Each post starts with its own selector (see selectorOf), in document order,
// and the first two selectors that can merge (see merge) are replaced by the
// merged one, until no two can merge. A post whose title or link no path
// resolves gets no selector.
func (t *pageIndex) generateSelectors(
	posts []labeledPost,
	excluded map[string]bool,
	excludedNodes map[*html.Node]bool,
) []PostSelector {
	exclude := slices.Sorted(maps.Keys(excluded))
	nodes := slices.SortedFunc(maps.Keys(excludedNodes), func(a, b *html.Node) int { return t.index[a] - t.index[b] })
	var excludeNodes [][]Step
	for _, n := range nodes {
		excludeNodes = append(excludeNodes, t.positionPath(n))
	}
	posts = slices.Clone(posts)
	slices.SortFunc(posts, func(a, b labeledPost) int { return t.index[a.node] - t.index[b.node] })

	var selectors []PostSelector
	for _, p := range posts {
		if m, ok := t.selectorOf(p, exclude, excludeNodes, excludedNodes); ok {
			selectors = append(selectors, m)
		}
	}
merging:
	for {
		for i := range selectors {
			for j := i + 1; j < len(selectors); j++ {
				if m, ok := merge(selectors[i], selectors[j]); ok {
					selectors[i] = m
					selectors = slices.Delete(selectors, j, j+1)
					continue merging
				}
			}
		}
		return selectors
	}
}

// labelFoundPosts returns every post that selectors find, with the nodes of
// its fields, in document order, so that the selectors can be generated again
// from them, since saved selectors do not keep the marks of the user. All of
// them are needed, since a merged selector keeps only the classes that all of
// its posts share, and fewer posts would make the selectors find fewer posts.
// A post that several selectors find takes each field from the first selector
// that finds it.
func (t *pageIndex) labelFoundPosts(selectors []PostSelector) []labeledPost {
	var posts []labeledPost
	byNode := map[*html.Node]labeledPost{}
	for i := range selectors {
		m := &selectors[i]
		skip := m.excludedNodes(t.body)
		for _, n := range m.selectPosts(t.body, skip) {
			p, ok := byNode[n]
			if !ok {
				p = labeledPost{node: n, fields: make(map[PostField]*html.Node)}
				byNode[n] = p
				posts = append(posts, p)
			}
			for f := range m.Fields() {
				if _, ok := p.fields[f]; ok {
					continue
				}
				if v := m.resolveField(n, f, skip); v != nil {
					p.fields[f] = v
				}
			}
		}
	}
	slices.SortFunc(posts, func(a, b labeledPost) int { return t.index[a.node] - t.index[b.node] })
	return posts
}
