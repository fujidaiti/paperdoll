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
// get wrong, as a post with its fields marked, or as a post to exclude, and
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

// fieldNode returns the node in post that the user marks as f to get the
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

// perfectRules returns rules that extract exactly posts from page, a
// simplified page parsed by html.Parse, built
// as a user of the client would build them, and the number of marks that the
// user made. The selector editor generates the rules from the posts that
// the user marked and the links that the user excluded (see
// generateRules), and the user repeats, until every post is extracted with its values:
//
//  1. If a node that the selectors find is wrong, the user marks it. A node
//     whose link is not in posts is marked as not a post, which excludes the
//     host and path of its link, and unmarks the posts that the user marked
//     with them. A node that has a value that differs from the one in posts is
//     marked as a post with the nodes of the values.
//  2. Otherwise, the user takes the first post that is not extracted with its
//     values, taps the line of its link, widens the selection until it holds
//     all values of the post, and marks the nodes of the values as its fields.
//
// In every round, every marked post must be found and no excluded link must
// be extracted. It fails when a step cannot be done, when a guarantee does not
// hold, or when the marks come back to the same state.
func perfectRules(page *html.Node, posts []ExtractedPost) (ExtractionRules, int, error) {
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
	marked := map[*html.Node]markedPost{}
	excluded := map[string]bool{}
	// mark marks post as a post with the nodes of the values in target.
	mark := func(post *html.Node, target ExtractedPost) error {
		p := markedPost{box: post, fields: make(map[PostField]*html.Node)}
		for f, v := range target {
			if p.fields[f] = t.fieldNode(post, f, v); p.fields[f] == nil {
				return fmt.Errorf("post %s: no node of %s has the value %q", target[PostLink], f, v)
			}
		}
		marked[post] = p
		delete(excluded, hostPath(target[PostLink]))
		return nil
	}

	marks := 0
	seen := map[string]bool{}
	for {
		var markedPosts []markedPost
		for _, p := range marked {
			markedPosts = append(markedPosts, p)
		}
		rules := t.generateRules(markedPosts, excluded, nil)

		// The selector that the user sees a node as found by is the first one.
		finder := map[*html.Node]*PostSelector{}
		exclude := rules.compileExclude()
		for i := range rules.Selectors {
			for _, n := range rules.Selectors[i].selectPosts(t.body, nil, exclude) {
				if finder[n] == nil {
					finder[n] = &rules.Selectors[i]
				}
			}
		}
		for n := range marked {
			if finder[n] == nil {
				return ExtractionRules{}, marks, fmt.Errorf(
					"the marked post %s is not found: %v",
					values(&rules.Selectors[0], n)[PostLink],
					rules,
				)
			}
		}
		got := map[string]ExtractedPost{}
		for _, p := range ExtractPosts(page, rules) {
			if excluded[hostPath(p[PostLink])] {
				return ExtractionRules{}, marks, fmt.Errorf("the excluded link %s is extracted", p[PostLink])
			}
			got[p[PostLink]] = p
		}

		state, err := json.Marshal(rules)
		if err != nil {
			return ExtractionRules{}, marks, err
		}
		var boxes []int
		for _, p := range markedPosts {
			boxes = append(boxes, t.index[p.box])
		}
		slices.Sort(boxes)
		key := fmt.Sprint(string(state), slices.Sorted(maps.Keys(excluded)), boxes)
		if seen[key] {
			return ExtractionRules{}, marks, fmt.Errorf("the marks come back to the same state, with rules %s", state)
		}
		seen[key] = true

		// A node of a post that has a value that differs from the one in
		// posts, and that does not hold the values in posts, such as a card of
		// another title with the link of the post, cannot be marked. The
		// user cannot exclude it either, since that also excludes the post.
		acted := false
		for _, n := range t.nodes {
			m := finder[n]
			if m == nil || !isWrong(values(m, n)) {
				continue
			}
			link := values(m, n)[PostLink]
			if w, ok := want[link]; ok {
				if mark(n, w) != nil {
					continue
				}
				marks++
			} else {
				hp := hostPath(link)
				excluded[hp] = true
				for n, p := range marked {
					if hostPath(linkOf(p.fields[PostLink])) == hp {
						delete(marked, n)
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
			return rules, marks, nil
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
			return ExtractionRules{}, marks, fmt.Errorf("post %s: no node has the link", link)
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
			return ExtractionRules{}, marks, fmt.Errorf("post %s: no node holds all of its values: %v", link, target)
		}
		// The client does not let a new post overlap a post that it finds.
		for n, m := range finder {
			if n != post && (t.contains(n, post) || t.contains(post, n)) {
				return ExtractionRules{}, marks, fmt.Errorf(
					"post %s overlaps with the post %s: %v", link, values(m, n)[PostLink], m)
			}
		}
		marks++
		if err := mark(post, target); err != nil {
			return ExtractionRules{}, marks, err
		}
	}
}
