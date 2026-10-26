package feed

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
)

// This file builds the matchers of a page from its fixture posts in the same
// way as a user of the client: the user marks the first post that the matchers
// get wrong, as a post with its fields labeled, or as a post to exclude, and
// the client generates the matchers again from all marks. The client's code
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

// postChain returns the chain that selects post from the root: the step of
// post when it has classes, or else the steps of the nodes from its nearest
// ancestor with classes, or from the top of the tree below the root, down to
// post.
func (t *treeIndex) postChain(post *SemanticTreeNode) []Step {
	chain := []Step{t.step(post)}
	for n := post; len(chain[0].Cls) == 0 && t.parent[t.parent[n]] != nil; {
		n = t.parent[n]
		chain = append([]Step{t.step(n)}, chain...)
	}
	return chain
}

// fieldChain returns the chain that resolves f from post to n: the shortest
// tail of the steps of the nodes from post to n that does, and that has a
// step with classes when any of the nodes has classes. It falls back to all
// the steps, with Nth on the last one. It returns nil when no chain resolves
// to n.
func (t *treeIndex) fieldChain(post *SemanticTreeNode, f PostField, n *SemanticTreeNode) []Step {
	var path []Step
	for a := n; a != post; a = t.parent[a] {
		path = append([]Step{t.step(a)}, path...)
	}
	hasCls := func(chain []Step) bool {
		return slices.ContainsFunc(chain, func(s Step) bool { return len(s.Cls) > 0 })
	}
	for i := len(path); i >= 0; i-- {
		chain := append([]Step{}, path[i:]...)
		if (i == len(path) && len(path) > 0) || hasCls(path) && !hasCls(chain) {
			continue
		}
		if resolveChain(post, f, chain) == n {
			return chain
		}
	}
	if len(path) == 0 {
		return nil
	}
	for nth := 0; nth < len(t.nodes); nth++ {
		chain := slices.Clone(path)
		chain[len(chain)-1].Nth = &nth
		got := selectNodes(post, chain, false)
		if len(got) == 0 {
			return nil
		}
		if resolveChain(post, f, chain) == n {
			return chain
		}
	}
	return nil
}

// widen returns chain widened to find post too, as the client does when it
// groups posts by their shapes: each step keeps only the classes that it shares
// with post, or with the ancestor of post at its level. It returns nil when
// post or such an ancestor has another tag. It follows _widen of the client.
func (t *treeIndex) widen(chain []Step, post *SemanticTreeNode) []Step {
	chain = slices.Clone(chain)
	n := post
	for i := len(chain) - 1; i >= 0; i-- {
		if i < len(chain)-1 {
			n = t.parent[n]
		}
		if n == nil || n.Tag != chain[i].Tag {
			return nil
		}
		s := t.step(n)
		chain[i].Cls = slices.DeleteFunc(
			slices.Clone(chain[i].Cls),
			func(c string) bool { return !slices.Contains(s.Cls, c) },
		)
	}
	return chain
}

// chainOf returns the chain that selects every post of group: the one of its
// first post, widened to the others. It returns nil when they have different
// shapes. It follows _chainOf of the client.
func (t *treeIndex) chainOf(group []labeledPost) []Step {
	chain := t.postChain(group[0].node)
	for _, p := range group[1:] {
		if chain = t.widen(chain, p.node); chain == nil {
			return nil
		}
	}
	return chain
}

// withField returns m with the chain that resolves f from post to n. The chain
// is added after the others when none resolves f in post, and before them when
// one resolves it to another node. m is returned as it is when no chain
// resolves f to n. It follows PostMatcher._withField of the client.
func (t *treeIndex) withField(m PostMatcher, post *SemanticTreeNode, f PostField, n *SemanticTreeNode) PostMatcher {
	got := m.resolveField(post, f)
	if got == n {
		return m
	}
	chain := t.fieldChain(post, f, n)
	if chain == nil {
		return m
	}
	fields := maps.Clone(m.Fields)
	if got == nil {
		fields[f] = append(slices.Clone(fields[f]), chain)
	} else {
		fields[f] = append([][]Step{chain}, fields[f]...)
	}
	m.Fields = fields
	return m
}

// labeledPost is a post that the user labeled: its node and the nodes of its
// fields.
type labeledPost struct {
	node   *SemanticTreeNode
	fields map[PostField]*SemanticTreeNode
}

// generateMatchers returns the matchers that find the posts that the user
// labeled, and none of the posts whose links the user excluded, which excluded
// holds by host and path. The posts are grouped by their shapes in document
// order, and each group results in one matcher. A post that its group does not
// find with its fields is moved to a group of its own. It follows
// generateMatchers of the client.
func (t *treeIndex) generateMatchers(posts []labeledPost, excluded map[string]bool) []PostMatcher {
	exclude := slices.Sorted(maps.Keys(excluded))
	if exclude == nil {
		exclude = []string{}
	}
	index := map[*SemanticTreeNode]int{}
	for i, n := range t.nodes {
		index[n] = i
	}
	posts = slices.Clone(posts)
	slices.SortFunc(posts, func(a, b labeledPost) int { return index[a.node] - index[b.node] })

	var groups [][]labeledPost
	for _, p := range posts {
		i := slices.IndexFunc(groups, func(g []labeledPost) bool {
			return t.chainOf(append(slices.Clone(g), p)) != nil
		})
		if i < 0 {
			groups = append(groups, []labeledPost{p})
		} else {
			groups[i] = append(groups[i], p)
		}
	}

	var matchers []PostMatcher
	for i := 0; i < len(groups); i++ {
		group := groups[i]
		m := PostMatcher{
			Post:    t.chainOf(group),
			Fields:  make(map[PostField][][]Step),
			Exclude: slices.Clone(exclude),
		}
		for _, p := range group {
			for _, f := range slices.Sorted(maps.Keys(p.fields)) {
				m = t.withField(m, p.node, f, p.fields[f])
			}
		}
		found := m.matchPosts(t.tree)
		isFound := true
		for _, p := range group {
			if !slices.Contains(found, p.node) {
				isFound = false
			}
			for f, n := range p.fields {
				if m.resolveField(p.node, f) != n {
					isFound = false
				}
			}
		}
		if len(group) > 1 && !isFound {
			groups[i] = group[:len(group)-1]
			groups = append(groups, []labeledPost{group[len(group)-1]})
			i--
			continue
		}
		matchers = append(matchers, m)
	}
	return matchers
}

// values returns the field values that m extracts from post.
func values(m *PostMatcher, post *SemanticTreeNode) ExtractedPost {
	values := make(ExtractedPost)
	for f := range m.Fields {
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

// perfectMatchers returns matchers that extract exactly posts from tree, built
// as a user of the client would build them, and the number of marks that the
// user made. The client generates the matchers from the posts that the user
// marked and the links that the user excluded (see generateMatchers), and the
// user repeats, until every post is extracted with its values:
//
//  1. If a node that the matchers find is wrong, the user marks it. A node
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
func perfectMatchers(tree *SemanticTree, posts []ExtractedPost) ([]PostMatcher, int, error) {
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
		matchers := t.generateMatchers(labeledPosts, excluded)

		// The matcher that the user sees a node as found by is the first one.
		finder := map[*SemanticTreeNode]*PostMatcher{}
		for i := range matchers {
			for _, n := range matchers[i].matchPosts(tree) {
				if finder[n] == nil {
					finder[n] = &matchers[i]
				}
			}
		}
		for n := range labeled {
			if finder[n] == nil {
				return nil, marks, fmt.Errorf(
					"the labeled post %s is not found: %v",
					values(&matchers[0], n)[PostLink],
					matchers,
				)
			}
		}
		got := map[string]ExtractedPost{}
		for _, p := range ExtractPosts(tree, matchers) {
			if excluded[hostPath(p[PostLink])] {
				return nil, marks, fmt.Errorf("the excluded link %s is extracted", p[PostLink])
			}
			got[p[PostLink]] = p
		}

		state, err := json.Marshal(matchers)
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
			return nil, marks, fmt.Errorf("the marks come back to the same state, with matchers %s", state)
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
			return matchers, marks, nil
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
