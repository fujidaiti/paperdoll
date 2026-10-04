package structures

import (
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// titleLength is the shortest text that can be read as the title of a post. A
// shorter one is the name of a tag, of an author or of a section of the site.
const titleLength = 15

// maxLevel bounds how deep the parts of one structure go. A deeper chain of
// parts costs the user a row to expand for every level, so a page that keeps
// repeating below this depth is described with fewer parts rather than with a
// tower of them.
const maxLevel = 4

// Build produces the structures of a page.
//
// It looks for sets of sibling elements that repeat, keeps the ones whose
// members each look like one post, and turns each of them into one structure.
// Inside those siblings, every place that holds a value becomes a field, named
// by the path of tags and class names that leads to it. A path that holds
// several nodes inside one sibling becomes a child part instead, because a
// field matcher points at most one node per subtree.
func Build(page *html.Node, base *url.URL) []*Structure {
	order := walkOrder(page)
	groups := candidates(page, base)
	groups = dropMerging(groups, order)
	groups = dropWrapped(groups, order)
	// Two sets of siblings under different parents, such as the cards of two
	// sections of the page, are reached by the same chain and would be offered
	// twice. The first of them is kept and the rest are dropped.
	var out []*Structure
	seen := map[string]bool{}
	for _, members := range groups {
		c := rootChain(members, page)
		found := c.find(page)
		if len(found) == 0 {
			continue
		}
		key := make([]string, len(found))
		for i, n := range found {
			key[i] = strconv.Itoa(order[n][0])
		}
		if seen[strings.Join(key, ",")] {
			continue
		}
		seen[strings.Join(key, ",")] = true

		// The parts and the fields describe what the chain found rather than
		// the set of siblings it was read from, because the chain is what runs
		// on a later build.
		c = trimClasses(c, page, found)
		root := buildPart(c.String(), found, base, 1)
		root.Match = func(n *html.Node) []*html.Node { return c.find(n) }
		out = append(out, &Structure{Root: root})
	}
	return out
}

// candidates returns the sets of sibling elements that can stand for a list of
// posts. The members of a set share a tag name, and two of them are enough to
// look like one post, because a list of post cards can also hold a promotion
// card or an empty slot without ceasing to be a post list.
func candidates(page *html.Node, base *url.URL) [][]*html.Node {
	post := postLike(page, base)
	var out [][]*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if !observable(n) {
			return
		}
		byTag := map[string][]*html.Node{}
		var tags []string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode || !observable(c) {
				continue
			}
			if _, seen := byTag[c.Data]; !seen {
				tags = append(tags, c.Data)
			}
			byTag[c.Data] = append(byTag[c.Data], c)
		}
		for _, tag := range tags {
			members := byTag[tag]
			if len(members) < 2 {
				continue
			}
			found := 0
			for _, m := range members {
				if post[m] {
					found++
				}
			}
			if found >= 2 {
				out = append(out, members)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(page)
	return out
}

// postLike reports, for every node, whether its subtree holds what a post
// holds: a link, a text long enough to be a title, and more than one place
// that carries a value. The last of the three is what tells a post card apart
// from a tag chip, which is a single link carrying a single name.
//
// The title has to be a text written in the page. A text carried by an
// attribute, such as the title attribute of a tag link, is read as a value but
// is not what a post shows as its title.
func postLike(page *html.Node, base *url.URL) map[*html.Node]bool {
	out := map[*html.Node]bool{}
	type found struct {
		link  bool
		title bool
		spots int
	}
	var walk func(*html.Node) found
	walk = func(n *html.Node) found {
		var f found
		if !observable(n) {
			return f
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sub := walk(c)
			f.link = f.link || sub.link
			f.title = f.title || sub.title
			f.spots = min(f.spots+sub.spots, 2)
			if c.Type == html.TextNode && len([]rune(flat(c.Data))) >= titleLength {
				f.title = true
			}
		}
		if n.Type == html.ElementNode {
			// Any element carrying an href counts, not only an anchor: a
			// page can write its post cards as a custom element.
			if resolve(base, attr(n, "href")) != "" {
				f.link = true
			}
			if len(values(n, base)) > 0 {
				f.spots = min(f.spots+1, 2)
			}
		}
		out[n] = f.link && f.title && f.spots >= 2
		return f
	}
	walk(page)
	return out
}

// walkOrder numbers every node by a walk from the page root, so that one node
// holding another is a question about two numbers.
func walkOrder(page *html.Node) map[*html.Node][2]int {
	out := map[*html.Node][2]int{}
	at := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		start := at
		at++
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		out[n] = [2]int{start, at}
	}
	walk(page)
	return out
}

// A reach is one candidate set, numbered so that the members sitting inside a
// node can be found and measured without walking the page again.
type reach struct {
	starts  []int
	covered []int // the nodes the members up to this one cover, summed
}

func reachOf(members []*html.Node, order map[*html.Node][2]int) reach {
	r := reach{starts: make([]int, len(members)), covered: make([]int, len(members)+1)}
	for i, m := range members {
		span := order[m]
		r.starts[i] = span[0]
		r.covered[i+1] = r.covered[i] + span[1] - span[0]
	}
	return r
}

// inside returns how many members of the set sit strictly inside the node, and
// how many nodes they cover between them.
func (r reach) inside(n *html.Node, order map[*html.Node][2]int) (count, covered int) {
	span := order[n]
	from := sort.SearchInts(r.starts, span[0]+1)
	to := sort.SearchInts(r.starts, span[1])
	return to - from, r.covered[to] - r.covered[from]
}

// dropMerging removes a set of siblings that holds a longer set filling one of
// its members. Such a member stands for a list of posts rather than for one
// post, which is what the merge metric reports.
//
// Two things have to hold for the inner set. It has to be longer, because the
// repeated blocks of a single post card, such as its tags, belong to one card
// while a list of cards runs over the whole page. And it has to fill the
// member it sits in, because a post card is mostly made of the post, while the
// tags of a card take a small corner of it.
func dropMerging(groups [][]*html.Node, order map[*html.Node][2]int) [][]*html.Node {
	reaches := make([]reach, len(groups))
	for i, g := range groups {
		reaches[i] = reachOf(g, order)
	}
	var out [][]*html.Node
	for i, outer := range groups {
		merging := false
		for j, inner := range groups {
			if i == j || len(inner) <= len(outer) {
				continue
			}
			for _, m := range outer {
				count, covered := reaches[j].inside(m, order)
				span := order[m]
				if count > 1 && covered*2 >= span[1]-span[0] {
					merging = true
					break
				}
			}
			if merging {
				break
			}
		}
		if !merging {
			out = append(out, outer)
		}
	}
	return out
}

// dropWrapped removes a set of siblings that sits inside another set, one
// member per member. Both describe the same posts, and the outer one reaches
// more of their values, so keeping both would only make the user read the same
// posts twice.
func dropWrapped(groups [][]*html.Node, order map[*html.Node][2]int) [][]*html.Node {
	reaches := make([]reach, len(groups))
	for i, g := range groups {
		reaches[i] = reachOf(g, order)
	}
	var out [][]*html.Node
	for i, inner := range groups {
		wrapped := false
		for j, outer := range groups {
			if i == j {
				continue
			}
			total, spread := 0, false
			for _, o := range outer {
				count, _ := reaches[i].inside(o, order)
				total += count
				if count > 1 {
					spread = true
				}
			}
			if total == len(inner) && !spread {
				wrapped = true
				break
			}
		}
		if !wrapped {
			out = append(out, inner)
		}
	}
	return out
}

// A step is one level of a chain: an element with this tag name that carries
// every class name of classes and none of not. No step uses a position, so a
// chain still reaches its element when a sibling is added or removed.
type step struct {
	tag     string
	classes []string
	not     []string
}

// A chain is a path of steps, each read as a direct child of what the step
// before it reached. An empty chain stands for the node the matcher receives.
type chain []step

func (c chain) find(root *html.Node) []*html.Node {
	at := []*html.Node{root}
	for _, s := range c {
		var next []*html.Node
		for _, n := range at {
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				if ch.Type == html.ElementNode && s.fits(ch) {
					next = append(next, ch)
				}
			}
		}
		at = next
	}
	return at
}

func (s step) fits(n *html.Node) bool {
	if n.Data != s.tag {
		return false
	}
	for _, name := range s.classes {
		if !carries(n, name) {
			return false
		}
	}
	for _, name := range s.not {
		if carries(n, name) {
			return false
		}
	}
	return true
}

func (s step) String() string {
	var out strings.Builder
	out.WriteString(s.tag)
	for _, name := range s.classes {
		out.WriteString("." + name)
	}
	for _, name := range s.not {
		out.WriteString(":not(." + name + ")")
	}
	return out.String()
}

func (c chain) String() string {
	if len(c) == 0 {
		return "self"
	}
	parts := make([]string, len(c))
	for i, s := range c {
		parts[i] = s.String()
	}
	return strings.Join(parts, " > ")
}

func classes(n *html.Node) []string { return strings.Fields(attr(n, "class")) }

func carries(n *html.Node, name string) bool {
	return slices.Contains(classes(n), name)
}

// rootChain reads the path from the page down to a set of siblings. The last
// step asks only for the class names every member carries, so that a member
// that carries an extra one is still reached.
func rootChain(members []*html.Node, page *html.Node) chain {
	var up []*html.Node
	for n := members[0].Parent; n != nil && n != page; n = n.Parent {
		up = append(up, n)
	}
	c := make(chain, 0, len(up)+1)
	for _, n := range slices.Backward(up) {
		c = append(c, step{tag: n.Data, classes: classes(n)})
	}
	shared := classes(members[0])
	for _, m := range members[1:] {
		shared = slices.DeleteFunc(slices.Clone(shared), func(name string) bool {
			return !carries(m, name)
		})
	}
	return append(c, step{tag: members[0].Data, classes: shared})
}

// trimClasses drops every class name the chain does not need to reach exactly
// the nodes it reaches now. A class name a build tool writes, such as
// style-1w7apwp, changes when the site is redeployed, so those are the ones
// tried first and the ones most likely to be dropped.
func trimClasses(c chain, page *html.Node, want []*html.Node) chain {
	out := make(chain, len(c))
	for i, s := range c {
		out[i] = step{tag: s.tag, classes: slices.Clone(s.classes)}
	}
	for i := range out {
		names := slices.Clone(out[i].classes)
		sort.SliceStable(names, func(a, b int) bool {
			return generatedClass(names[a]) && !generatedClass(names[b])
		})
		for _, name := range names {
			kept := out[i].classes
			out[i].classes = slices.DeleteFunc(slices.Clone(kept), func(c string) bool {
				return c == name
			})
			if !slices.Equal(out.find(page), want) {
				out[i].classes = kept
			}
		}
	}
	return out
}

// generatedClass reports whether a class name looks like one a build tool
// wrote, such as style-1w7apwp or storycard__header--474c6553bfa. A name that
// mixes letters and digits in a long run is the sign of it.
func generatedClass(name string) bool {
	for _, token := range strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	}) {
		if len(token) < 6 {
			continue
		}
		digits, letters := false, false
		for _, r := range token {
			switch {
			case r >= '0' && r <= '9':
				digits = true
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
				letters = true
			}
		}
		if digits && letters {
			return true
		}
	}
	return false
}

// A spot is one path inside the subtrees a part matched, together with the
// nodes it reaches in each of them. A spot that reaches one node per subtree
// can become a field, and a spot that reaches several in one subtree cannot,
// because a field matcher points at most one node per subtree.
type spot struct {
	c     chain
	nodes [][]*html.Node
}

// buildPart describes the subtrees one part matched. It walks them level by
// level, so that the nodes of every subtree that sit at the same path are seen
// together and become one field. The walk follows the page order, so that the
// fields of a part are offered in the order the page writes them.
func buildPart(name string, subtrees []*html.Node, base *url.URL, level int) *Part {
	part := &Part{Name: name}
	start := spot{nodes: make([][]*html.Node, len(subtrees))}
	for i, n := range subtrees {
		start.nodes[i] = []*html.Node{n}
	}
	var visit func(spot)
	visit = func(s spot) {
		most := 0
		for _, ns := range s.nodes {
			most = max(most, len(ns))
		}
		if most > 1 && level < maxLevel {
			var flat []*html.Node
			for _, ns := range s.nodes {
				flat = append(flat, ns...)
			}
			child := buildPart(s.c.String(), flat, base, level+1)
			child.Match = func(n *html.Node) []*html.Node { return s.c.find(n) }
			part.Children = append(part.Children, child)
			return
		}
		if f := s.field(base); f != nil {
			part.Fields = append(part.Fields, f)
		}
		for _, next := range s.children() {
			visit(next)
		}
	}
	visit(start)
	return part
}

// field returns the field of a spot, or nil when none of the nodes it reaches
// holds a value of its own. A node that holds no value is still walked
// through, because its children can hold one.
func (s spot) field(base *url.URL) *Field {
	f := &Field{Name: s.c.String()}
	for _, ns := range s.nodes {
		for _, n := range ns {
			if len(values(n, base)) > 0 {
				f.Samples = append(f.Samples, Sample{Nodes: []*html.Node{n}})
			}
		}
	}
	if len(f.Samples) == 0 {
		return nil
	}
	f.Match = func(root *html.Node) *html.Node {
		found := s.c.find(root)
		if len(found) == 0 {
			return nil
		}
		return found[0]
	}
	return f
}

// children returns the spots one level below. The children of one tag name are
// told apart by the class names they carry, and only a class name that appears
// under more than one subtree is used, so that a name a build tool writes per
// card is ignored.
func (s spot) children() []spot {
	type kid struct {
		at int
		n  *html.Node
	}
	byTag := map[string][]kid{}
	var tags []string
	for i, ns := range s.nodes {
		for _, n := range ns {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type != html.ElementNode || !observable(c) {
					continue
				}
				if _, seen := byTag[c.Data]; !seen {
					tags = append(tags, c.Data)
				}
				byTag[c.Data] = append(byTag[c.Data], kid{i, c})
			}
		}
	}

	var out []spot
	for _, tag := range tags {
		kids := byTag[tag]
		where := map[string]map[int]bool{}
		for _, k := range kids {
			for _, name := range classes(k.n) {
				if where[name] == nil {
					where[name] = map[int]bool{}
				}
				where[name][k.at] = true
			}
		}

		// Group the children by the class names that survive, keeping the
		// groups in the order the page writes them.
		groups := map[string][]kid{}
		signature := map[string][]string{}
		var keys []string
		for _, k := range kids {
			var sig []string
			for _, name := range classes(k.n) {
				if len(where[name]) > 1 || len(s.nodes) == 1 {
					sig = append(sig, name)
				}
			}
			sort.Strings(sig)
			key := strings.Join(sig, ".")
			if _, seen := groups[key]; !seen {
				keys = append(keys, key)
				signature[key] = sig
			}
			groups[key] = append(groups[key], k)
		}

		for _, key := range keys {
			// A group is told apart from its siblings by the class names they
			// carry and it does not, so that its matcher reaches its nodes
			// only.
			var not []string
			for _, other := range keys {
				for _, name := range signature[other] {
					if !slices.Contains(signature[key], name) && !slices.Contains(not, name) {
						not = append(not, name)
					}
				}
			}
			next := spot{
				c:     append(slices.Clone(s.c), step{tag: tag, classes: signature[key], not: not}),
				nodes: make([][]*html.Node, len(s.nodes)),
			}
			for _, k := range groups[key] {
				next.nodes[k.at] = append(next.nodes[k.at], k.n)
			}
			out = append(out, next)
		}
	}
	return out
}
