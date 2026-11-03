package feed

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"

	"golang.org/x/net/html"
)

// This file builds the selectors of a page from its fixture posts in the same
// way as a user of the client: the user marks the first post that the selectors
// get wrong, as a post with its fields labeled, or as a post to exclude, and
// the selector editor generates the selectors again from all marks. The
// client's code that a step of the user follows is named in its comment.

// selectionLevels returns from and the nodes around it, from the narrowest to
// the widest, up to top, or up to the outermost node below the body if top is
// nil. A node around a single node without a link of its own is left out. It
// follows selectionLevels of the client, by which the user widens a
// selection.
func (t *pageIndex) selectionLevels(from, top *html.Node) []*html.Node {
	levels := []*html.Node{from}
	for n := from; n != top && n.Parent != t.body; {
		n = n.Parent
		if n.FirstChild != n.LastChild || linkOf(n) != "" {
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
func (t *pageIndex) fieldNode(post *html.Node, f PostField, want string) *html.Node {
	for _, line := range t.nodes {
		if !t.contains(post, line) {
			continue
		}
		if f == PostBody {
			if !isValue(line) || tagOf(line) == "img" || tagOf(line) == "time" {
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

// values returns the field values that m extracts from post.
func values(m *PostSelector, post *html.Node) ExtractedPost {
	values := make(ExtractedPost)
	for f := range m.Fields() {
		if n := m.resolveField(post, f, nil); n != nil {
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

// perfectSelectors returns selectors that extract exactly posts from page, a
// simplified page parsed by html.Parse, built
// as a user of the client would build them, and the number of marks that the
// user made. The selector editor generates the selectors from the posts that
// the user marked and the links that the user excluded (see
// generateSelectors), and the user repeats, until every post is extracted with its values:
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
func perfectSelectors(page *html.Node, posts []ExtractedPost) ([]PostSelector, int, error) {
	t := newPageIndex(pageBody(page))
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
	labeled := map[*html.Node]labeledPost{}
	excluded := map[string]bool{}
	// label marks post as a post with the nodes of the values in target.
	label := func(post *html.Node, target ExtractedPost) error {
		p := labeledPost{node: post, fields: make(map[PostField]*html.Node)}
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
		selectors := t.generateSelectors(labeledPosts, excluded, nil)

		// The selector that the user sees a node as found by is the first one.
		finder := map[*html.Node]*PostSelector{}
		for i := range selectors {
			for _, n := range selectors[i].selectPosts(t.body, nil) {
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
		for _, p := range ExtractPosts(page, selectors) {
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
			marked = append(marked, t.index[p.node])
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
					if hostPath(linkOf(p.fields[PostLink])) == hp {
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
		var start *html.Node
		for _, n := range t.nodes {
			if PostLink.accepts(n) && linkOf(n) == link {
				start = n
				break
			}
		}
		if start == nil {
			return nil, marks, fmt.Errorf("post %s: no node has the link", link)
		}
		var post *html.Node
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
