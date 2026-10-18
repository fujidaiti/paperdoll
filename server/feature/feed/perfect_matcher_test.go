package feed

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// This file builds the matchers of a page from its fixture posts in the same
// way as a user of the client: the user selects a post, labels its fields,
// and excludes the posts that a group gets wrong. The client's code that each
// step follows is named in its comment.

// pageTree is a PageDocument with the parent of each node, which the client's
// PageNode has.
type pageTree struct {
	doc    *PageDocument
	nodes  []*PageNode // in document order
	parent map[*PageNode]*PageNode
}

func newPageTree(doc *PageDocument) *pageTree {
	t := &pageTree{doc: doc, parent: map[*PageNode]*PageNode{}}
	var visit func(n *PageNode)
	visit = func(n *PageNode) {
		t.nodes = append(t.nodes, n)
		for _, c := range n.Children {
			t.parent[c] = n
			visit(c)
		}
	}
	visit(doc.Root)
	return t
}

// contains reports whether n is a or one of its descendants.
func (t *pageTree) contains(a, n *PageNode) bool {
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
func (t *pageTree) selectionLevels(from, top *PageNode) []*PageNode {
	levels := []*PageNode{from}
	for n := from; n != top && t.parent[t.parent[n]] != nil; {
		n = t.parent[n]
		if len(n.Children) > 1 || n.Link != "" {
			levels = append(levels, n)
		}
	}
	return levels
}

// accepts reports whether n can provide a sensible value for f, as
// PostField.accepts of the client tells.
func accepts(f PostField, n *PageNode) bool {
	img, time := n.Kind == "img", n.Kind == "time"
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

// fieldNode returns the node in post that the user labels as f to get the
// value want, or nil when there is none. The user taps a line of the value
// and widens the selection until it shows the value: a line with the value
// itself, or for a body, a line of its text. A line that is not part of the
// value, such as the date above a body, would reach a wider node with the
// same text.
func (t *pageTree) fieldNode(post *PageNode, f PostField, want string) *PageNode {
	for _, line := range t.nodes {
		if !t.contains(post, line) {
			continue
		}
		if f == PostBody {
			if !line.isValue() || line.Kind == "img" || line.Kind == "time" {
				continue
			}
		} else if v, ok := f.valueOf(line); !ok || v != want || !accepts(f, line) {
			continue
		}
		for _, n := range t.selectionLevels(line, post) {
			if v, ok := f.valueOf(n); ok && v == want && accepts(f, n) {
				return n
			}
		}
	}
	return nil
}

// steps returns the path from post to n, as PostMatcher.withField of the
// client makes it.
func (t *pageTree) steps(post, n *PageNode) []FieldStep {
	steps := []FieldStep{}
	for ; n != post; n = t.parent[n] {
		nth := 0
		for _, c := range t.parent[n].Children {
			if c == n {
				break
			}
			if shape(c) == shape(n) {
				nth++
			}
		}
		steps = append([]FieldStep{{Shape: shape(n), Nth: nth}}, steps...)
	}
	return steps
}

// values returns the field values that m extracts from post.
func values(m *PostMatcher, post *PageNode) ExtractedPost {
	values := ExtractedPost{}
	for f, steps := range m.Fields {
		if n := resolve(post, steps); n != nil {
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
func withExcluded(m PostMatcher, doc *PageDocument, link string) PostMatcher {
	hp := hostPath(link)
	var kept []string
	for _, p := range m.matchPosts(doc) {
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

// perfectMatchers returns matchers that extract exactly posts from doc, built
// as a user of the client would build them. Until every post is extracted
// with its values, the user:
//
//  1. takes the first post that is not extracted with its values,
//  2. taps the line of its link and widens the selection until it holds all
//     values of the post, and marks it as a post of a new group, or edits
//     the group that it is a post of,
//  3. labels each field of the post (see fieldNode), and removes the labels
//     of the fields that the post does not have,
//  4. excludes the link of each post that a group gets wrong: a post that is
//     not in posts, or that has a value that differs from the one in posts.
//     Another group extracts the post with the right values.
//
// It fails when a step cannot be done, or when the steps come back to the
// same matchers.
func perfectMatchers(doc *PageDocument, posts []ExtractedPost) ([]PostMatcher, error) {
	t := newPageTree(doc)
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
			for _, n := range m.matchPosts(doc) {
				v := values(m, n)
				if m.isExcluded(n) || !isWrong(v) {
					continue
				}
				// The client asks before excluding other posts of the group
				// too, and the user agrees only when they are also wrong.
				excluded := withExcluded(*m, doc, v[PostLink])
				for _, p := range m.matchPosts(doc) {
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
		for _, p := range ExtractPosts(doc, matchers) {
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

		var start *PageNode
		for _, n := range t.nodes {
			if accepts(PostLink, n) && n.Link == link {
				start = n
				break
			}
		}
		if start == nil {
			return nil, fmt.Errorf("post %s: no node has the link", link)
		}
		var post *PageNode
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
			for _, p := range matchers[i].matchPosts(doc) {
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
		if m == nil {
			matchers = append(matchers, PostMatcher{Exclude: []string{}})
			m = &matchers[len(matchers)-1]
			for n := post; n != nil; n = t.parent[n] {
				m.Post = append([]string{shape(n)}, m.Post...)
			}
		}
		m.Fields = map[PostField][]FieldStep{}
		for f, v := range target {
			m.Fields[f] = t.steps(post, t.fieldNode(post, f, v))
		}
	}

	if got := ExtractPosts(doc, matchers); len(got) != len(posts) {
		return nil, fmt.Errorf("%d posts are extracted, want %d", len(got), len(posts))
	}
	return matchers, nil
}
