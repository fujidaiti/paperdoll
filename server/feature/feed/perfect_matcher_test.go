package feed

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
)

// This file builds the matchers of a page from its fixture posts in the same
// way as a user of the client: the user selects a post, labels its fields,
// and excludes the posts that a group gets wrong. The client's code that each
// step follows is named in its comment.

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

// withExample returns m widened to find post too, as a user of the client
// makes it by tapping a post that m does not find: each step keeps only the
// classes that it shares with post, or with the ancestor of post at its
// level. It returns nil when post or such an ancestor has another tag.
func (t *treeIndex) withExample(m PostMatcher, post *SemanticTreeNode) *PostMatcher {
	chain := slices.Clone(m.Post)
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
	m.Post = chain
	return &m
}

// withFields returns m with the chains that resolve the fields of post to the
// nodes that the user labels for the values in target, or nil when post does
// not have a labeled title or link of m, or a value of target. A chain is added after
// the others when no chain resolves the field in post, and before them when
// one resolves it to another node.
func (t *treeIndex) withFields(m PostMatcher, post *SemanticTreeNode, target ExtractedPost) *PostMatcher {
	fields := make(map[PostField][][]Step)
	for f, chains := range m.Fields {
		if _, ok := target[f]; !ok && (f == PostTitle || f == PostLink) {
			return nil
		}
		fields[f] = chains
	}
	m.Fields = fields
	for f, v := range target {
		n := t.fieldNode(post, f, v)
		if n == nil {
			return nil
		}
		got := m.resolveField(post, f)
		if got == n {
			continue
		}
		chain := t.fieldChain(post, f, n)
		if chain == nil {
			return nil
		}
		if got == nil {
			m.Fields[f] = append(slices.Clone(m.Fields[f]), chain)
		} else {
			m.Fields[f] = append([][]Step{chain}, m.Fields[f]...)
		}
	}
	return &m
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

// mergePattern returns the pattern that matches both pattern and hostPath,
// made by turning the segments that differ into "*", or "" when there is
// none. It follows _mergePattern of the client.
func mergePattern(pattern, hostPath string) string {
	a, b := strings.Split(pattern, "/"), strings.Split(hostPath, "/")
	if strings.Contains(pattern, "**") || strings.Contains(pattern, "?") || len(a) != len(b) || a[0] != b[0] {
		return ""
	}
	merged := make([]string, len(a))
	for i := range a {
		if a[i] != "*" && strings.Contains(a[i], "*") {
			return ""
		}
		merged[i] = a[i]
		if a[i] != b[i] {
			merged[i] = "*"
		}
	}
	return strings.Join(merged, "/")
}

// withExcluded returns m with a pattern that excludes link. link is merged
// into a pattern of m when the merged pattern excludes no post of m with
// another link. Otherwise the host and path of link are added, which also
// exclude the posts whose links differ from link only in the query or the
// fragment. It follows PostMatcher.withExcluded of the client.
func withExcluded(m PostMatcher, tree *SemanticTree, link string) PostMatcher {
	hp := hostPath(link)
	var kept []string
	for _, p := range m.matchPosts(tree) {
		if l := values(&m, p)[PostLink]; l != "" && l != link {
			kept = append(kept, hostPath(l))
		}
	}
	m.Exclude = slices.Clone(m.Exclude)
	for i, pattern := range m.Exclude {
		if merged := mergePattern(
			pattern,
			hp,
		); merged != "" &&
			!slices.ContainsFunc(kept, func(l string) bool { return globMatches(merged, l) }) {
			m.Exclude[i] = merged
			return m
		}
	}
	m.Exclude = append(m.Exclude, hp)
	return m
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
// as a user of the client would build them. Until every post is extracted
// with its values, the user:
//
//  1. takes the first post that is not extracted with its values,
//  2. taps the line of its link and widens the selection until it holds all
//     values of the post, and edits the group that it is a post of, or adds
//     it as an example to a group (see withExample), or marks it as a post
//     of a new group,
//  3. labels each field of the post that the group does not resolve to the
//     node of its value (see withFields),
//  4. excludes the link of each post that a group gets wrong: a post that is
//     not in posts, or that has a value that differs from the one in posts.
//     Another group extracts the post with the right values.
//
// It fails when a step cannot be done, or when the steps come back to the
// same matchers.
func perfectMatchers(tree *SemanticTree, posts []ExtractedPost) ([]PostMatcher, error) {
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

	var matchers []PostMatcher
	// The states of matchers so far. Editing a group can drop posts that
	// other groups then take, but coming back to a state would never end.
	seen := map[string]bool{}
	for {
		for i := range matchers {
			m := &matchers[i]
			for _, n := range m.matchPosts(tree) {
				v := values(m, n)
				if m.isExcluded(n) || !isWrong(v) {
					continue
				}
				// The client asks before excluding other posts of the group
				// too, and the user agrees only when they are also wrong.
				excluded := withExcluded(*m, tree, v[PostLink])
				for _, p := range m.matchPosts(tree) {
					if p != n && excluded.isExcluded(p) && !isWrong(values(m, p)) {
						return nil, fmt.Errorf(
							"group %d gets %s wrong, but excluding it also excludes %s",
							i+1,
							v[PostLink],
							values(m, p)[PostLink],
						)
					}
				}
				*m = excluded
			}
		}

		got := map[string]ExtractedPost{}
		for _, p := range ExtractPosts(tree, matchers) {
			got[p[PostLink]] = p
		}
		var target ExtractedPost
		for _, p := range posts {
			if !equalPosts(got[p[PostLink]], p) {
				target = p
				break
			}
		}
		if target == nil {
			break
		}
		link := target[PostLink]
		state, err := json.Marshal(matchers)
		if err != nil {
			return nil, err
		}
		if seen[string(state)] {
			return nil, fmt.Errorf("post %s is extracted as %v, want %v", link, got[link], target)
		}
		seen[string(state)] = true

		var start *SemanticTreeNode
		for _, n := range t.nodes {
			if PostLink.accepts(n) && n.Link == link {
				start = n
				break
			}
		}
		if start == nil {
			return nil, fmt.Errorf("post %s: no node has the link", link)
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
			return nil, fmt.Errorf("post %s: no node holds all of its values: %v", link, target)
		}
		// A post of a group is edited in that group. The client does not let
		// a new post overlap a post of a group.
		var m *PostMatcher
		for i := range matchers {
			for _, p := range matchers[i].matchPosts(tree) {
				switch {
				case p == post:
					m = &matchers[i]
				case t.contains(p, post) || t.contains(post, p):
					return nil, fmt.Errorf(
						"post %s overlaps with the post %s of group %d: %v",
						link,
						values(&matchers[i], p)[PostLink],
						i+1,
						matchers[i],
					)
				}
			}
		}
		if m != nil {
			edited := t.withFields(*m, post, target)
			if edited == nil {
				return nil, fmt.Errorf("post %s: cannot label its fields in its group: %v", link, target)
			}
			*m = *edited
			continue
		}
		// The user adds the post as an example to the first group that then
		// extracts it with its values, and gets no other post wrong that it
		// did not get wrong before.
		score := func(ms []PostMatcher) (right map[string]bool, wrong int) {
			right = map[string]bool{}
			for _, p := range ExtractPosts(tree, ms) {
				if isWrong(p) {
					wrong++
				} else {
					right[p[PostLink]] = true
				}
			}
			return right, wrong
		}
		right, wrong := score(matchers)
		for i := range matchers {
			widened := t.withExample(matchers[i], post)
			if widened == nil {
				continue
			}
			if widened = t.withFields(*widened, post, target); widened == nil {
				continue
			}
			trial := slices.Clone(matchers)
			trial[i] = *widened
			r, w := score(trial)
			if w > wrong || !r[link] ||
				slices.ContainsFunc(slices.Collect(maps.Keys(right)), func(l string) bool { return !r[l] }) {
				continue
			}
			m = &matchers[i]
			*m = *widened
			break
		}
		if m == nil {
			created := t.withFields(PostMatcher{Post: t.postChain(post), Exclude: []string{}}, post, target)
			if created == nil {
				return nil, fmt.Errorf("post %s: cannot label its fields: %v", link, target)
			}
			matchers = append(matchers, *created)
		}
	}
	return matchers, nil
}
