package feed

import (
	"maps"
	"regexp"
	"slices"

	"golang.org/x/net/html"
)

// The functions below generate the post selectors of a page from the posts
// that a user marked in a selector editor (see SelectorEditor).

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
// ExcludeNodes of ExtractionRules).
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

// markedPost is a post that the user marked: its box and the nodes of its
// fields.
type markedPost struct {
	box    *html.Node
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
// out. skip holds the excluded nodes.
func (t *pageIndex) selectorOf(post markedPost, skip map[*html.Node]bool) (PostSelector, bool) {
	m := PostSelector{Box: t.postPath(post.box)}
	for f, n := range post.fields {
		if path, ok := t.fieldPath(f, post.box, n, skip); ok {
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
	if len(a.Box) != len(b.Box) {
		return PostSelector{}, false
	}
	last := len(a.Box) - 1
	for i := range a.Box {
		if a.Box[i].Tag != b.Box[i].Tag || i < last && !samePaths(a.Box[i:i+1], b.Box[i:i+1]) {
			return PostSelector{}, false
		}
	}
	post := slices.Clone(a.Box)
	post[last].Cls = slices.DeleteFunc(
		slices.Clone(post[last].Cls),
		func(c string) bool { return !slices.Contains(b.Box[last].Cls, c) },
	)
	if !slices.ContainsFunc(post, func(s Step) bool { return len(s.Cls) > 0 }) {
		return PostSelector{}, false
	}
	if !samePaths(a.Title, b.Title) || !samePaths(a.Link, b.Link) {
		return PostSelector{}, false
	}
	m := PostSelector{Box: post, Title: a.Title, Link: a.Link}
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

// generateRules returns the rules whose selectors find the posts that the
// user marked, and none of the posts whose links the user excluded, which
// excluded holds by host and path, and nothing in the subtrees of
// excludedNodes. The rules depend on the posts, the links and the nodes, not
// on the order in which the user marked them. A link of excluded must not be
// the link of a post, and a post and its fields must not be in the subtree of
// an excluded node.
//
// Each post starts with its own selector (see selectorOf), in document order,
// and the first two selectors that can merge (see merge) are replaced by the
// merged one, until no two can merge. A post whose title or link no path
// resolves gets no selector.
func (t *pageIndex) generateRules(
	posts []markedPost,
	excluded map[string]bool,
	excludedNodes map[*html.Node]bool,
) ExtractionRules {
	rules := ExtractionRules{ExcludeLinks: slices.Sorted(maps.Keys(excluded))}
	nodes := slices.SortedFunc(maps.Keys(excludedNodes), func(a, b *html.Node) int { return t.index[a] - t.index[b] })
	for _, n := range nodes {
		rules.ExcludeNodes = append(rules.ExcludeNodes, t.positionPath(n))
	}
	posts = slices.Clone(posts)
	slices.SortFunc(posts, func(a, b markedPost) int { return t.index[a.box] - t.index[b.box] })

	selectors := []PostSelector{}
	for _, p := range posts {
		if m, ok := t.selectorOf(p, excludedNodes); ok {
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
		rules.Selectors = selectors
		return rules
	}
}

// marksFor returns the marks that a user would make to get targets, posts in
// document order, and the rules generated from them, excluded and
// excludedNodes (see generateRules). It marks the first unmarked target that
// the rules of the marks so far do not find with the nodes of its fields,
// until there is none, so it marks each target at most once. A post that only
// the rules find is not marked. The targets must meet the conditions of
// generateRules on the posts.
func (t *pageIndex) marksFor(
	targets []markedPost,
	excluded map[string]bool,
	excludedNodes map[*html.Node]bool,
) ([]markedPost, ExtractionRules) {
	marks := []markedPost{}
	marked := map[*html.Node]bool{}
	for {
		rules := t.generateRules(marks, excluded, excludedNodes)
		found := map[*html.Node]markedPost{}
		for _, p := range t.markFoundPosts(rules) {
			found[p.box] = p
		}
		i := slices.IndexFunc(targets, func(p markedPost) bool {
			f, ok := found[p.box]
			return !marked[p.box] && (!ok || !maps.Equal(f.fields, p.fields))
		})
		if i < 0 {
			return marks, rules
		}
		marks = append(marks, targets[i])
		marked[targets[i].box] = true
	}
}

// markFoundPosts returns every post that rules find, with the nodes of its
// fields, in document order. A post that several selectors find takes each
// field from the first selector that finds it.
func (t *pageIndex) markFoundPosts(rules ExtractionRules) []markedPost {
	var posts []markedPost
	byNode := map[*html.Node]markedPost{}
	skip := rules.excludedNodes(t.body)
	excluded := rules.excludedLinks()
	for i := range rules.Selectors {
		m := &rules.Selectors[i]
		for _, n := range m.selectPosts(t.body, skip, excluded) {
			p, ok := byNode[n]
			if !ok {
				p = markedPost{box: n, fields: make(map[PostField]*html.Node)}
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
	slices.SortFunc(posts, func(a, b markedPost) int { return t.index[a.box] - t.index[b.box] })
	return posts
}
