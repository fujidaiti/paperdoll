package scoredtree

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// ScoreNodes writes into every node of the tree how much it looks like a post,
// and marks the nodes whose score reaches threshold.
//
// The score of a node is read together with its siblings, not from the node
// alone. Half of it is what the node holds, which is a long text, a date, an
// image and a heading, and the other half is how well the node agrees with its
// siblings, which is whether it carries a link of the shape they share, whether
// it looks like them and whether it carries exactly one link of that shape.
// The second half is what makes the number comparable between pages: a page
// whose cards carry no date scores low on content everywhere, so a threshold
// read from content alone cannot hold for every page.
//
// The root is left at 0, because it has no siblings to be read against.
func ScoreNodes(root *Node, threshold float64) {
	for _, parent := range root.Walk() {
		members := parent.Children
		if len(members) == 0 {
			continue
		}
		cands := candidatesOf(members)
		links := groupLinks(members, cands)

		// The URL shape and the look most of the siblings agree on. A member
		// that matches them is a card of the same list; a menu entry or a
		// section title next to a list matches neither.
		var order []string
		shapes := map[string]int{}
		sigs := map[string]int{}
		for i, l := range links {
			sigs[nodeShape(members[i])]++
			if l == "" {
				continue
			}
			t := urlTemplate(l)
			if shapes[t] == 0 {
				order = append(order, t)
			}
			shapes[t]++
		}
		shape, agreed := "", 0
		for _, t := range order {
			if shapes[t] > agreed {
				shape, agreed = t, shapes[t]
			}
		}
		modalSig, modalCount := "", 0
		for s, c := range sigs {
			if c > modalCount || (c == modalCount && s < modalSig) {
				modalSig, modalCount = s, c
			}
		}

		size := float64(len(members)) / 8.0
		if size > 1 {
			size = 1
		}

		for i, m := range members {
			// How many links of the shape the siblings agreed on this member
			// holds. One is what a post needs: a container holds a whole list
			// and so holds many, and a tag or an author holds none.
			ofShape := 0
			for l := range cands[i] {
				if urlTemplate(l) == shape {
					ofShape++
				}
			}
			agreement := 0.0
			if links[i] != "" && urlTemplate(links[i]) == shape {
				agreement += 0.5
			}
			if nodeShape(m) == modalSig {
				agreement += 0.5
			}
			one := 0.0
			if ofShape == 1 {
				one = 1
			}
			linked := 0.0
			if links[i] != "" {
				linked = 1
			}

			m.Score = 0.5*content(m) + 0.2*agreement + 0.2*one + 0.1*size*linked
			m.Post = m.Score >= threshold
		}
	}
	root.Score = 0
	root.Post = false
}

// content is what the node holds, read without looking at any other node. The
// weights are the ones the group score uses, because they describe a card and
// a card is what a member of a list is.
func content(n *Node) float64 {
	longest, hasHead, hasDate := 0, false, false
	for _, t := range n.subtreeTexts() {
		if len(t.Value) > longest {
			longest = len(t.Value)
		}
		if headingTags[t.Tag] {
			hasHead = true
		}
		if t.Datetime != "" || dateText.MatchString(t.Value) {
			hasDate = true
		}
	}
	text := float64(longest) / 40.0
	if text > 1 {
		text = 1
	}
	out := 0.4 * text
	if hasDate {
		out += 0.3
	}
	if n.hasImage() {
		out += 0.2
	}
	if hasHead {
		out += 0.1
	}
	return out
}

// Group is one list of posts, which is one row of the selection screen. Links
// holds the link every member stands for, in the order of the members.
type Group struct {
	Members []*Node
	Links   []string
}

// Groups reads the lists of posts out of a tree ScoreNodes has already marked.
//
// The nodes marked as posts under one parent are one list, because a post
// never contains another post, so two posts that are siblings belong to the
// same list. The lists of two different parents are then merged when their
// members agree on the same URL shape, which is what a page that writes its
// sections as separate containers needs: bbc.com holds its posts under 24
// parents and shows one list.
//
// Groups are returned largest first, which is the order the selection screen
// would show them in.
func Groups(root *Node) []Group {
	byParent := map[*Node][]*Node{}
	var order []*Node
	for _, n := range root.Walk() {
		if !n.Post || n.Parent == nil {
			continue
		}
		if byParent[n.Parent] == nil {
			order = append(order, n.Parent)
		}
		byParent[n.Parent] = append(byParent[n.Parent], n)
	}

	// The URL shape a set of siblings agreed on, which is what the sets are
	// merged by. A set whose members carry no link has none and is kept on
	// its own, because there is nothing to merge it by.
	type part struct {
		members []*Node
		links   []string
		shape   string
	}
	var parts []part
	for _, p := range order {
		members := byParent[p]
		links := groupLinks(members, candidatesOf(members))
		counts := map[string]int{}
		shape, best := "", 0
		for _, l := range links {
			if l == "" {
				continue
			}
			t := urlTemplate(l)
			counts[t]++
			if counts[t] > best || (counts[t] == best && t < shape) {
				shape, best = t, counts[t]
			}
		}
		parts = append(parts, part{members, links, shape})
	}

	var out []Group
	merged := map[string]int{}
	for _, p := range parts {
		if p.shape != "" {
			if at, ok := merged[p.shape]; ok {
				out[at].Members = append(out[at].Members, p.members...)
				out[at].Links = append(out[at].Links, p.links...)
				continue
			}
			merged[p.shape] = len(out)
		}
		out = append(out, Group{Members: p.members, Links: p.links})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return len(out[i].Members) > len(out[j].Members)
	})
	return out
}

// The tags that carry no meaning of their own. A title written in a div and
// the same title written in a span are the same value, so a card that names
// its author in a span still looks like a card that has no author.
var neutralTags = map[string]bool{
	"p": true, "span": true, "div": true, "li": true, "em": true,
	"strong": true, "b": true, "i": true, "small": true, "a": true,
}

var headingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

// dateText matches the ways a date is written in a card. A date is far more
// often a plain string than a <time> element, so the string is what is read.
var dateText = regexp.MustCompile(`(?i)\b(\d{4}-\d{2}-\d{2}|\d{4}/\d{1,2}/\d{1,2}|` +
	`(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?\s+\d{1,2}|` +
	`\d{1,2}\s+(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)|` +
	`\d+\s+(minute|hour|day|week|month|year)s?\s+ago|\d{4}年\d{1,2}月)`)

var (
	numberSegment = regexp.MustCompile(`^\d+$`)
	hashSegment   = regexp.MustCompile(`^[0-9a-fA-F]{7,}$`)
)

// nodeShape is what a node looks like without its values. Two cards of the
// same list have the same shape even when one of them has no category.
func nodeShape(n *Node) string {
	tags := map[string]bool{}
	for _, t := range n.Texts {
		if neutralTags[t.Tag] {
			tags["text"] = true
			continue
		}
		tags[t.Tag] = true
	}
	names := make([]string, 0, len(tags))
	for t := range tags {
		names = append(names, t)
	}
	sort.Strings(names)
	link := "0"
	if n.Link != "" {
		link = "1"
	}
	return link + ":" + strings.Join(names, ",")
}

// candidatesOf reads the candidate links of every member of a group. How many
// members carry a link is counted once for the whole group, because the test
// that drops a banner repeated on every card is the only one that looks
// outside the member.
func candidatesOf(members []*Node) []map[string]bool {
	carried := map[string]int{}
	for _, m := range members {
		for l := range distinctLinks(m) {
			carried[l]++
		}
	}
	out := make([]map[string]bool, len(members))
	for i, m := range members {
		out[i] = candidateLinks(m, carried)
	}
	return out
}

// groupLinks returns the link every member stands for. The members are read
// twice: once on their own, and once knowing the URL shape most of them agreed
// on. The second pass is what an item written as running text needs, because
// the body of such an item links to other pages and only the agreement of the
// other members says which link is the item itself.
func groupLinks(members []*Node, cands []map[string]bool) []string {
	first := make([]string, len(members))
	for i, m := range members {
		first[i] = representativeLink(m, cands[i], "")
	}

	var order []string
	count := map[string]int{}
	for _, l := range first {
		if l == "" {
			continue
		}
		t := urlTemplate(l)
		if count[t] == 0 {
			order = append(order, t)
		}
		count[t]++
	}
	shape, agreed := "", 0
	for _, t := range order {
		if count[t] > agreed {
			shape, agreed = t, count[t]
		}
	}
	if agreed < 2 {
		return first
	}
	out := make([]string, len(members))
	for i, m := range members {
		out[i] = representativeLink(m, cands[i], shape)
	}
	return out
}

// representativeLink returns the link a member stands for. Among the candidate
// links, the post is the one the card repeats, then the one the longest text
// sits on. When shape is set, a candidate of that shape is preferred.
func representativeLink(n *Node, allowed map[string]bool, shape string) string {
	var found []*Node
	for _, x := range n.Walk() {
		if x.Link != "" {
			found = append(found, x)
		}
	}
	if len(found) == 0 {
		return ""
	}
	if shape != "" {
		same := map[string]bool{}
		for l := range allowed {
			if urlTemplate(l) == shape {
				same[l] = true
			}
		}
		if len(same) > 0 {
			allowed = same
		}
	}

	var order []string
	count := map[string]int{}
	longest := map[string]int{}
	for _, x := range found {
		if !allowed[x.Link] {
			continue
		}
		if count[x.Link] == 0 {
			order = append(order, x.Link)
		}
		count[x.Link]++
		for _, t := range x.Texts {
			if len(t.Value) > longest[x.Link] {
				longest[x.Link] = len(t.Value)
			}
		}
	}
	if len(order) == 0 {
		return found[0].Link
	}
	best := order[0]
	for _, l := range order[1:] {
		if count[l] > count[best] ||
			(count[l] == count[best] && longest[l] > longest[best]) {
			best = l
		}
	}
	return best
}

// candidateLinks returns the links of a subtree that could be the post the
// subtree is about. A card carries more than the post it shows, and each kind
// of extra link is told apart by how it sits next to the others:
//
//   - a link whose path another link of the same card extends is the author
//     page, the category or the section the post belongs to;
//   - a shape that occurs more than once in one card, such as five links to
//     /tags/<name>, cannot be the single post of that card;
//   - a link that several members of the same group carry is a banner or a
//     menu repeated on every card.
//
// When every link fails one of the tests, all of them are returned, because a
// card that stands for a post always stands for one of its links.
func candidateLinks(n *Node, carried map[string]int) map[string]bool {
	links := distinctLinks(n)
	if len(links) <= 1 {
		return links
	}

	paths := map[string]map[string]bool{}
	shapes := map[string]int{}
	for l := range links {
		host, path := hostAndPath(l)
		if paths[host] == nil {
			paths[host] = map[string]bool{}
		}
		paths[host][path] = true
		shapes[urlTemplate(l)]++
	}

	out := map[string]bool{}
	for l := range links {
		host, path := hostAndPath(l)
		if extended(paths[host], path) || shapes[urlTemplate(l)] > 1 {
			continue
		}
		// carried counts this member too, so it is taken out before the test.
		if carried[l]-1 >= 2 {
			continue
		}
		out[l] = true
	}
	if len(out) == 0 {
		return links
	}
	return out
}

func distinctLinks(n *Node) map[string]bool {
	out := map[string]bool{}
	for _, x := range n.Walk() {
		if x.Link != "" {
			out[x.Link] = true
		}
	}
	return out
}

// extended reports whether one of the paths continues path with a further
// segment, which is what makes path a page about the posts under it.
func extended(paths map[string]bool, path string) bool {
	for p := range paths {
		if len(p) > len(path) && strings.HasPrefix(p, path+"/") {
			return true
		}
	}
	return false
}

func hostAndPath(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", raw
	}
	return strings.ToLower(u.Host), "/" + strings.Trim(u.Path, "/")
}

// urlTemplate is the shape of a URL: the host, one token per path segment and
// the names of the query parameters. Two links of the same list normally share
// it, and a link that belongs to something else normally does not.
func urlTemplate(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(u.Host))
	for s := range strings.SplitSeq(strings.Trim(u.Path, "/"), "/") {
		if s == "" {
			continue
		}
		b.WriteString("/")
		switch {
		case numberSegment.MatchString(s):
			b.WriteString("{num}")
		case hashSegment.MatchString(s) && strings.ContainsAny(s, "0123456789"):
			b.WriteString("{hash}")
		default:
			b.WriteString("{slug}")
		}
	}
	if q := u.Query(); len(q) > 0 {
		names := make([]string, 0, len(q))
		for k := range q {
			names = append(names, k)
		}
		sort.Strings(names)
		b.WriteString("?")
		b.WriteString(strings.Join(names, ","))
	}
	return b.String()
}
