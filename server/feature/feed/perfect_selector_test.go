package feed

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"testing"
)

// This file builds the selectors of a page from its fixture posts in the same
// way as a user of the client: the user marks the first post that the selectors
// get wrong, as a post with its fields labeled, or as a post to exclude, and
// the client generates the selectors again from all marks. The client's code
// that each step follows is named in its comment.

// treeIndex is a SemanticTree with the parent of each node, which the
// client's SemanticTreeNode has, and the number of nodes with each class.
type treeIndex struct {
	tree    *SemanticTree
	nodes   []*SemanticTreeNode // in document order
	parent  map[*SemanticTreeNode]*SemanticTreeNode
	classes map[string]int
}

func newTreeIndex(tree *SemanticTree) *treeIndex {
	t := &treeIndex{tree: tree, parent: map[*SemanticTreeNode]*SemanticTreeNode{}, classes: map[string]int{}}
	var visit func(n *SemanticTreeNode)
	visit = func(n *SemanticTreeNode) {
		t.nodes = append(t.nodes, n)
		for _, c := range n.Cls {
			t.classes[c]++
		}
		for _, c := range n.Children {
			t.parent[c] = n
			visit(c)
		}
	}
	visit(tree.Root)
	return t
}

// contains reports whether n is a or one of its descendants.
func (t *treeIndex) contains(a, n *SemanticTreeNode) bool {
	for ; n != nil; n = t.parent[n] {
		if n == a {
			return true
		}
	}
	return false
}

// selectionLevels returns from and the nodes around it, from the narrowest to
// the widest, up to top, or up to the outermost node below the root if top is
// nil. A node around a single node without a link of its own is left out. It
// follows selectionLevels of the client, by which the user widens a
// selection.
func (t *treeIndex) selectionLevels(from, top *SemanticTreeNode) []*SemanticTreeNode {
	levels := []*SemanticTreeNode{from}
	for n := from; n != top && t.parent[t.parent[n]] != nil; {
		n = t.parent[n]
		if len(n.Children) > 1 || n.Link != "" {
			levels = append(levels, n)
		}
	}
	return levels
}

// fieldNode returns the node in post that the user labels as f to get the
// value want, or nil when there is none. The user taps a line of the value
// and widens the selection until it shows the value: a line with the value
// itself, or for a body, a line of its text. A line that is not part of the
// value, such as the date above a body, would reach a wider node with the
// same text.
func (t *treeIndex) fieldNode(post *SemanticTreeNode, f PostField, want string) *SemanticTreeNode {
	for _, line := range t.nodes {
		if !t.contains(post, line) {
			continue
		}
		if f == PostBody {
			if !line.isValue() || line.Tag == "img" || line.Tag == "time" {
				continue
			}
		} else if v, ok := f.valueOf(line); !ok || v != want || !f.accepts(line) {
			continue
		}
		for _, n := range t.selectionLevels(line, post) {
			if v, ok := f.valueOf(n); ok && v == want && f.accepts(n) {
				return n
			}
		}
	}
	return nil
}

// step returns the step of n: its tag and the classes that other nodes have
// too. A class of n alone, such as an ID, cannot be shared by two posts.
func (t *treeIndex) step(n *SemanticTreeNode) Step {
	s := Step{Tag: n.Tag}
	for _, c := range n.Cls {
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
// of the list. It follows _postClasses of the client.
func postClasses(cls []string) []string {
	cls = slices.DeleteFunc(slices.Clone(cls), variedPostClass.MatchString)
	if kinds := slices.DeleteFunc(slices.Clone(cls), func(c string) bool {
		return !postKindClass.MatchString(c)
	}); len(kinds) > 0 {
		return kinds
	}
	return cls
}

func TestPostClasses(t *testing.T) {
	for _, tt := range []struct{ cls, want []string }{
		{[]string{"post", "other"}, []string{"post"}},
		{[]string{"type-post", "other"}, []string{"type-post"}},
		{[]string{"post-card", "other"}, []string{"post-card"}},
		{[]string{"masonry-card", "other"}, []string{"masonry-card"}},
		{[]string{"card__inner", "other"}, []string{"card__inner"}},
		{[]string{"storyCard__linked", "other"}, []string{"storyCard__linked"}},
		{[]string{"IndexCardStyled", "other"}, []string{"IndexCardStyled"}},
		{[]string{"AdPost", "other"}, []string{"AdPost"}},
		{[]string{"adpost", "other"}, []string{"adpost", "other"}},
		{[]string{"compost", "other"}, []string{"compost", "other"}},
		{[]string{"greencard", "other"}, []string{"greencard", "other"}},
		{[]string{"postcard", "other"}, []string{"postcard", "other"}},
		{[]string{"cardinal", "other"}, []string{"cardinal", "other"}},
		{[]string{"post-9818", "other"}, []string{"other"}},
		{[]string{"category-security", "other"}, []string{"other"}},
		{[]string{"tag-whatsapp", "other"}, []string{"other"}},
		{[]string{"has-post-thumbnail", "other"}, []string{"other"}},
		{[]string{"post", "type-post", "hentry", "post-9818", "category-security"}, []string{"post", "type-post"}},
		{[]string{"post-9818", "category-security"}, []string{}},
	} {
		if got := postClasses(tt.cls); !slices.Equal(got, tt.want) {
			t.Errorf("postClasses(%q) = %q, want %q", tt.cls, got, tt.want)
		}
	}
}

// postChain returns the chain that selects post from the root: the step of
// post when it has classes left (see postClasses), or else the steps of the
// nodes from its nearest ancestor with classes, or from the top of the tree
// below the root, down to post.
func (t *treeIndex) postChain(post *SemanticTreeNode) []Step {
	s := t.step(post)
	s.Cls = postClasses(s.Cls)
	chain := []Step{s}
	for n := post; len(chain[0].Cls) == 0 && t.parent[t.parent[n]] != nil; {
		n = t.parent[n]
		chain = append([]Step{t.step(n)}, chain...)
	}
	return chain
}

// pathOf returns the steps of the nodes from post, exclusive, down to n.
func (t *treeIndex) pathOf(post, n *SemanticTreeNode) []Step {
	var path []Step
	for a := n; a != post; a = t.parent[a] {
		path = append([]Step{t.step(a)}, path...)
	}
	return path
}

// fieldChain returns the chain that resolves f to n in post: the shortest
// tail of the path from post to n that does, and that has a step with classes
// when the path has one. It falls back to the whole path, with Nth on the last
// step. It returns false when no chain resolves to n. It follows _fieldChain
// of the client.
func (t *treeIndex) fieldChain(f PostField, post, n *SemanticTreeNode) ([]Step, bool) {
	path := t.pathOf(post, n)
	hasCls := func(chain []Step) bool {
		return slices.ContainsFunc(chain, func(s Step) bool { return len(s.Cls) > 0 })
	}
	for i := len(path); i >= 0; i-- {
		chain := append([]Step{}, path[i:]...)
		if (i == len(path) && len(path) > 0) || hasCls(path) && !hasCls(chain) {
			continue
		}
		if resolveChain(post, f, chain) == n {
			return chain, true
		}
	}
	if len(path) == 0 {
		return nil, false
	}
	for nth := 0; nth < len(t.nodes); nth++ {
		chain := slices.Clone(path)
		chain[len(chain)-1].Nth = &nth
		if len(selectNodes(post, chain, false)) == 0 {
			return nil, false
		}
		if resolveChain(post, f, chain) == n {
			return chain, true
		}
	}
	return nil, false
}

// labeledPost is a post that the user labeled: its node and the nodes of its
// fields.
type labeledPost struct {
	node   *SemanticTreeNode
	fields map[PostField]*SemanticTreeNode
}

// setChain sets the chain of f in m.
func setChain(m *PostSelector, f PostField, chain []Step) {
	switch f {
	case PostTitle:
		m.Title = chain
	case PostLink:
		m.Link = chain
	case PostBody:
		m.Body = chain
	case PostThumbnail:
		m.Thumbnail = chain
	case PostPubDate:
		m.PubDate = chain
	}
}

// selectorOf returns the selector that finds post with its fields, or false when
// no chain resolves its title or link. A field that no chain resolves is left
// out. It follows _selectorOf of the client.
func (t *treeIndex) selectorOf(post labeledPost, exclude []string) (PostSelector, bool) {
	m := PostSelector{Post: t.postChain(post.node), Exclude: exclude}
	for f, n := range post.fields {
		if chain, ok := t.fieldChain(f, post.node, n); ok {
			setChain(&m, f, chain)
		}
	}
	return m, m.Title != nil && m.Link != nil
}

// sameChains reports whether a and b have the same steps, with the classes of
// each step in any order.
func sameChains(a, b []Step) bool {
	return slices.EqualFunc(a, b, func(x, y Step) bool {
		return x.Tag == y.Tag &&
			(x.Nth == nil) == (y.Nth == nil) && (x.Nth == nil || *x.Nth == *y.Nth) &&
			len(x.Cls) == len(y.Cls) &&
			!slices.ContainsFunc(x.Cls, func(c string) bool { return !slices.Contains(y.Cls, c) })
	})
}

// merge returns the selector that finds the posts of a and of b with their
// fields, or false when they cannot merge. They merge when their post chains
// have the same tags and the same classes except on the last step, and when
// they have the same title and link chains, and no other field with
// different chains. The last step of the merged post chain keeps the classes
// that a and b share, and the merge fails when no step has classes left, since
// such a chain selects any node of its tags. It follows _merge of the client.
//
// The merged selector finds the posts of a and of b without a check on the
// tree: its post chain selects their nodes, and their fields resolve by the
// same chains as before.
func merge(a, b PostSelector) (PostSelector, bool) {
	if len(a.Post) != len(b.Post) {
		return PostSelector{}, false
	}
	last := len(a.Post) - 1
	for i := range a.Post {
		if a.Post[i].Tag != b.Post[i].Tag || i < last && !sameChains(a.Post[i:i+1], b.Post[i:i+1]) {
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
	if !sameChains(a.Title, b.Title) || !sameChains(a.Link, b.Link) {
		return PostSelector{}, false
	}
	m := PostSelector{Post: post, Title: a.Title, Link: a.Link, Exclude: a.Exclude}
	for _, f := range []PostField{PostBody, PostThumbnail, PostPubDate} {
		chain, ok := a.chainOf(f)
		if other, okOther := b.chainOf(f); !ok {
			chain = other
		} else if okOther && !sameChains(chain, other) {
			return PostSelector{}, false
		}
		setChain(&m, f, chain)
	}
	return m, true
}

// generateSelectors returns the selectors that find the posts that the user
// labeled, and none of the posts whose links the user excluded, which excluded
// holds by host and path. Each post starts with its own selector (see
// selectorOf), in document order, and the first two selectors that can merge
// (see merge) are replaced by the merged one, until no two can merge. A post
// whose title or link no chain resolves gets no selector. It follows
// generateSelectors of the client.
func (t *treeIndex) generateSelectors(posts []labeledPost, excluded map[string]bool) []PostSelector {
	exclude := slices.Sorted(maps.Keys(excluded))
	index := map[*SemanticTreeNode]int{}
	for i, n := range t.nodes {
		index[n] = i
	}
	posts = slices.Clone(posts)
	slices.SortFunc(posts, func(a, b labeledPost) int { return index[a.node] - index[b.node] })

	var selectors []PostSelector
	for _, p := range posts {
		if m, ok := t.selectorOf(p, exclude); ok {
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

// values returns the field values that m extracts from post.
func values(m *PostSelector, post *SemanticTreeNode) ExtractedPost {
	values := make(ExtractedPost)
	for f := range m.Fields() {
		if n := m.resolveField(post, f); n != nil {
			if v, ok := f.valueOf(n); ok {
				values[f] = v
			}
		}
	}
	return values
}

func hostPath(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return link
	}
	return u.Host + u.Path
}

func equalPosts(a, b ExtractedPost) bool {
	if len(a) != len(b) {
		return false
	}
	for f, v := range a {
		if w, ok := b[f]; !ok || w != v {
			return false
		}
	}
	return true
}

// perfectSelectors returns selectors that extract exactly posts from tree, built
// as a user of the client would build them, and the number of marks that the
// user made. The client generates the selectors from the posts that the user
// marked and the links that the user excluded (see generateSelectors), and the
// user repeats, until every post is extracted with its values:
//
//  1. If a node that the selectors find is wrong, the user marks it. A node
//     whose link is not in posts is marked as not a post, which excludes the
//     host and path of its link, and unmarks the posts that the user marked
//     with them. A node that has a value that differs from the one in posts is
//     labeled as a post with the nodes of the values.
//  2. Otherwise, the user takes the first post that is not extracted with its
//     values, taps the line of its link, widens the selection until it holds
//     all values of the post, and labels the nodes of the values as its fields.
//
// In every round, every labeled post must be found and no excluded link must
// be extracted. It fails when a step cannot be done, when a guarantee does not
// hold, or when the marks come back to the same state.
func perfectSelectors(tree *SemanticTree, posts []ExtractedPost) ([]PostSelector, int, error) {
	t := newTreeIndex(tree)
	want := map[string]ExtractedPost{}
	for _, p := range posts {
		want[p[PostLink]] = p
	}
	isWrong := func(v ExtractedPost) bool {
		p, ok := want[v[PostLink]]
		if !ok {
			return true
		}
		for f, val := range v {
			if w, ok := p[f]; !ok || w != val {
				return true
			}
		}
		return false
	}
	index := map[*SemanticTreeNode]int{}
	for i, n := range t.nodes {
		index[n] = i
	}

	labeled := map[*SemanticTreeNode]labeledPost{}
	excluded := map[string]bool{}
	// label marks post as a post with the nodes of the values in target.
	label := func(post *SemanticTreeNode, target ExtractedPost) error {
		p := labeledPost{node: post, fields: make(map[PostField]*SemanticTreeNode)}
		for f, v := range target {
			if p.fields[f] = t.fieldNode(post, f, v); p.fields[f] == nil {
				return fmt.Errorf("post %s: no node of %s has the value %q", target[PostLink], f, v)
			}
		}
		labeled[post] = p
		delete(excluded, hostPath(target[PostLink]))
		return nil
	}

	marks := 0
	seen := map[string]bool{}
	for {
		var labeledPosts []labeledPost
		for _, p := range labeled {
			labeledPosts = append(labeledPosts, p)
		}
		selectors := t.generateSelectors(labeledPosts, excluded)

		// The selector that the user sees a node as found by is the first one.
		finder := map[*SemanticTreeNode]*PostSelector{}
		for i := range selectors {
			for _, n := range selectors[i].matchPosts(tree) {
				if finder[n] == nil {
					finder[n] = &selectors[i]
				}
			}
		}
		for n := range labeled {
			if finder[n] == nil {
				return nil, marks, fmt.Errorf(
					"the labeled post %s is not found: %v",
					values(&selectors[0], n)[PostLink],
					selectors,
				)
			}
		}
		got := map[string]ExtractedPost{}
		for _, p := range ExtractPosts(tree, selectors) {
			if excluded[hostPath(p[PostLink])] {
				return nil, marks, fmt.Errorf("the excluded link %s is extracted", p[PostLink])
			}
			got[p[PostLink]] = p
		}

		state, err := json.Marshal(selectors)
		if err != nil {
			return nil, marks, err
		}
		var marked []int
		for _, p := range labeledPosts {
			marked = append(marked, index[p.node])
		}
		slices.Sort(marked)
		key := fmt.Sprint(string(state), slices.Sorted(maps.Keys(excluded)), marked)
		if seen[key] {
			return nil, marks, fmt.Errorf("the marks come back to the same state, with selectors %s", state)
		}
		seen[key] = true

		// A node of a post that has a value that differs from the one in
		// posts, and that does not hold the values in posts, such as a card of
		// another title with the link of the post, cannot be labeled. The
		// user cannot exclude it either, since that also excludes the post.
		acted := false
		for _, n := range t.nodes {
			m := finder[n]
			if m == nil || !isWrong(values(m, n)) {
				continue
			}
			link := values(m, n)[PostLink]
			if w, ok := want[link]; ok {
				if label(n, w) != nil {
					continue
				}
				marks++
			} else {
				hp := hostPath(link)
				excluded[hp] = true
				for n, p := range labeled {
					if hostPath(p.fields[PostLink].Link) == hp {
						delete(labeled, n)
					}
				}
				marks++
			}
			acted = true
			break
		}
		if acted {
			continue
		}

		var target ExtractedPost
		for _, p := range posts {
			if !equalPosts(got[p[PostLink]], p) {
				target = p
				break
			}
		}
		if target == nil {
			return selectors, marks, nil
		}
		link := target[PostLink]
		var start *SemanticTreeNode
		for _, n := range t.nodes {
			if PostLink.accepts(n) && n.Link == link {
				start = n
				break
			}
		}
		if start == nil {
			return nil, marks, fmt.Errorf("post %s: no node has the link", link)
		}
		var post *SemanticTreeNode
		for _, n := range t.selectionLevels(start, nil) {
			holds := true
			for f, v := range target {
				if t.fieldNode(n, f, v) == nil {
					holds = false
					break
				}
			}
			if holds {
				post = n
				break
			}
		}
		if post == nil {
			return nil, marks, fmt.Errorf("post %s: no node holds all of its values: %v", link, target)
		}
		// The client does not let a new post overlap a post that it finds.
		for n, m := range finder {
			if n != post && (t.contains(n, post) || t.contains(post, n)) {
				return nil, marks, fmt.Errorf(
					"post %s overlaps with the post %s: %v", link, values(m, n)[PostLink], m)
			}
		}
		marks++
		if err := label(post, target); err != nil {
			return nil, marks, err
		}
	}
}
