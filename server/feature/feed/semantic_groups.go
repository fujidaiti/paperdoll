package feed

import (
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// This file reads posts out of the semantic tree built by semantic_tree.go. It
// stands beside the enumeration in unstructured_feed.go and is not used by it.
//
// The tree answers which values belong to the same post. What is still missing
// is which node is a list of posts, and which link of such a node is the post
// rather than the author, the category or a tag. Neither question can be
// answered from one node alone: a list of tags inside a card carries one link
// per member and one shape, exactly like a list of posts, and the link of a
// card is only recognisable next to the links of the other cards of the same
// list. So both are answered by comparing the members of a group with each
// other.

// PostGroup is one list of posts found on a page, which is one row of the
// selection screen. Links holds one link per member, in the same order, and is
// empty for a member that carries none.
type PostGroup struct {
	Members []*SemanticNode
	Links   []string
	Score   float64
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

// SelectPostGroups returns the lists of posts of a page, best first.
//
// A group is a node with two or more children, and its score says how much it
// looks like a list of posts. A group is offered when its score reaches alpha
// times the best score of the same page: the scores of two pages are not
// comparable, because a page whose cards carry no date scores lower
// everywhere, so the threshold has to be relative.
//
// A featured card at the top of a list is a group of its own and the list sits
// inside it, so two groups that are both offered often overlap. The members
// that hold another offered group are taken out of their group, which offers
// the featured card and the list beside it as two rows instead of one of them
// replacing the other. A group that lost members is scored again, because what
// is left of it is not what was scored.
func SelectPostGroups(root *SemanticNode, alpha float64) []PostGroup {
	return selectPostGroups(root, alpha, false)
}

// SelectPostGroupsWithHeadings is SelectPostGroups with the heading rule
// turned on, and is an experiment that stands beside it.
//
// candidateLinks drops a URL shape that occurs more than once inside one card,
// because five links to /tags/<name> cannot be the single post of that card.
// A page that prints whole articles rather than a list of cards links from the
// body of a post to other posts of the same site, so the post's own link is
// dropped by that test before anything else can read it. The heading rule says
// that a link whose text is a heading is the post whatever the other links of
// the card look like, which puts such a link back.
func SelectPostGroupsWithHeadings(root *SemanticNode, alpha float64) []PostGroup {
	return selectPostGroups(root, alpha, true)
}

func selectPostGroups(root *SemanticNode, alpha float64, headings bool) []PostGroup {
	var candidates []*SemanticNode
	var walk func(*SemanticNode)
	walk = func(n *SemanticNode) {
		if n == nil {
			return
		}
		if len(n.Children) >= 2 {
			candidates = append(candidates, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	if len(candidates) == 0 {
		return nil
	}

	type scored struct {
		node  *SemanticNode
		score float64
	}
	all := make([]scored, 0, len(candidates))
	for _, n := range candidates {
		all = append(all, scored{n, groupScore(groupFeatures(n.Children, headings))})
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].score > all[j].score })

	taken := map[*SemanticNode]bool{}
	var above []scored
	for _, g := range all {
		if g.score < alpha*all[0].score {
			break
		}
		above = append(above, g)
		taken[g.node] = true
	}

	var kept []PostGroup
	for _, g := range above {
		var members []*SemanticNode
		for _, m := range g.node.Children {
			if holdsGroup(m, taken) {
				continue
			}
			members = append(members, m)
		}
		if len(members) == 0 {
			continue
		}
		p := PostGroup{Members: members}
		if len(members) > 1 {
			p.Score = groupScore(groupFeatures(members, headings))
		}
		kept = append(kept, p)
	}

	best := 0.0
	for _, g := range kept {
		if len(g.Members) > 1 && g.Score > best {
			best = g.Score
		}
	}

	var out []PostGroup
	for _, g := range kept {
		if len(g.Members) > 1 && best > 0 && g.Score < alpha*best {
			continue
		}
		g.Links = groupLinks(g.Members, headings)
		out = append(out, g)
	}
	return out
}

func holdsGroup(n *SemanticNode, taken map[*SemanticNode]bool) bool {
	if taken[n] {
		return true
	}
	for _, c := range n.Children {
		if holdsGroup(c, taken) {
			return true
		}
	}
	return false
}

// GroupLinks returns the link every member of a group stands for, in the order
// of the members.
//
// The group is read twice. The first pass reads every member on its own, and
// the second reads it again knowing the URL shape most members agreed on. The
// second pass is what an item written as running text needs: the body of such
// an item links to other pages, and only the agreement of the other members
// says which link is the item itself.
func GroupLinks(members []*SemanticNode) []string {
	return groupLinks(members, false)
}

func groupLinks(members []*SemanticNode, headings bool) []string {
	return groupLinksWith(members, candidatesOf(members), headings)
}

// candidatesOf reads the candidate links of every member of a group.
//
// How many members carry a link is counted once for the whole group, because
// the test that drops a banner repeated on every card is the only one that
// looks outside the member, and counting it per member would read every
// sibling again for each of them.
func candidatesOf(members []*SemanticNode) []map[string]bool {
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

func groupLinksWith(members []*SemanticNode, cands []map[string]bool, headings bool) []string {
	first := make([]string, len(members))
	for i, m := range members {
		first[i] = representativeLink(m, cands[i], "", headings)
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
		out[i] = representativeLink(m, cands[i], shape, headings)
	}
	return out
}

// representativeLink returns the link a member of a group stands for. Among
// the candidate links, the post is the one the card repeats, then the one the
// longest text sits on. When shape is set, a candidate of that shape is
// preferred over every other one.
//
// headings reads the links whose text is a heading before the candidates are
// read, so that such a link is the post even when candidateLinks dropped it.
func representativeLink(n *SemanticNode, allowed map[string]bool, shape string, headings bool) string {
	found := linkedNodes(n)
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
	if headings {
		if headed := headedLinks(found, shape); len(headed) > 0 {
			allowed = headed
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
// of extra link is told apart by how it sits next to the others rather than by
// its own text:
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
func candidateLinks(n *SemanticNode, carried map[string]int) map[string]bool {
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

// headedLinks returns the links of nodes whose own text is a heading, which is
// how a list of posts writes the title of an item. When shape is set and some
// of them have it, only those are returned.
func headedLinks(found []*SemanticNode, shape string) map[string]bool {
	out := map[string]bool{}
	for _, x := range found {
		for _, t := range x.Texts {
			if headingTags[t.Tag] {
				out[x.Link] = true
				break
			}
		}
	}
	if shape == "" || len(out) == 0 {
		return out
	}
	same := map[string]bool{}
	for l := range out {
		if urlTemplate(l) == shape {
			same[l] = true
		}
	}
	if len(same) > 0 {
		return same
	}
	return out
}

func distinctLinks(n *SemanticNode) map[string]bool {
	out := map[string]bool{}
	for _, x := range linkedNodes(n) {
		out[x.Link] = true
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

// groupStats is what a group of siblings looks like, read without knowing
// which links are posts. Every value is read from the members as whole
// subtrees, because a card is often still split into several nodes.
type groupStats struct {
	n           int
	linkRate    float64
	uniqRate    float64
	tplRate     float64
	sigRate     float64
	oneLinkRate float64
	medText     int
	headRate    float64
	imgRate     float64
	dateRate    float64
}

func groupFeatures(members []*SemanticNode, headings bool) groupStats {
	n := len(members)
	cands := candidatesOf(members)
	links := groupLinksWith(members, cands, headings)

	var order []string
	shapes := map[string]int{}
	distinct := map[string]bool{}
	linked := 0
	for _, l := range links {
		if l == "" {
			continue
		}
		linked++
		distinct[l] = true
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

	sigs := map[string]int{}
	modalSig := 0
	texts := make([]int, 0, n)
	one, heads, images, dates := 0, 0, 0, 0
	for i, m := range members {
		s := nodeShape(m)
		sigs[s]++
		if sigs[s] > modalSig {
			modalSig = sigs[s]
		}

		// How many links of the shape the group agreed on the member holds.
		// One is what a row of the selection screen needs: a page container
		// holds a whole list per member, and a member of a list of tags holds
		// none of the shape the posts share.
		// The heading rule is read here as well, so that a member whose post
		// link candidateLinks dropped is still counted as holding one link of
		// the shape the group agreed on.
		ofShape := 0
		cand := cands[i]
		if headings {
			if headed := headedLinks(linkedNodes(m), ""); len(headed) > 0 {
				cand = headed
			}
		}
		for l := range cand {
			if urlTemplate(l) == shape {
				ofShape++
			}
		}
		if ofShape == 1 {
			one++
		}

		longest := 0
		hasHead, hasDate := false, false
		for _, t := range subtreeTexts(m) {
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
		texts = append(texts, longest)
		if hasHead {
			heads++
		}
		if hasDate {
			dates++
		}
		if subtreeHasImage(m) {
			images++
		}
	}
	sort.Ints(texts)

	f := groupStats{
		n:           n,
		linkRate:    float64(linked) / float64(n),
		oneLinkRate: float64(one) / float64(n),
		sigRate:     float64(modalSig) / float64(n),
		medText:     texts[n/2],
		headRate:    float64(heads) / float64(n),
		imgRate:     float64(images) / float64(n),
		dateRate:    float64(dates) / float64(n),
	}
	if linked > 0 {
		f.uniqRate = float64(len(distinct)) / float64(n)
		f.tplRate = float64(agreed) / float64(n)
	}
	return f
}

// nodeShape is what a node looks like without its values. Two cards of the
// same list have the same shape even when one of them has no category. The
// tags that carry no meaning of their own are reported under one name, and the
// number of children and the presence of an image are left out, because a card
// that is still split into several nodes would otherwise never match one that
// is not.
func nodeShape(n *SemanticNode) string {
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

// groupScore says how much a group of siblings looks like a list of posts. No
// single value decides it: a list of tags inside a card has one link per
// member and one shape, like a list of posts, and is told apart only by what
// the members hold.
func groupScore(f groupStats) float64 {
	text := float64(f.medText) / 40.0
	if text > 1 {
		text = 1
	}
	content := 0.4*text + 0.3*f.dateRate + 0.2*f.imgRate + 0.1*f.headRate
	shape := 0.5*f.tplRate + 0.5*f.sigRate
	size := float64(f.n) / 8.0
	if size > 1 {
		size = 1
	}
	return 0.5*content + 0.2*shape + 0.2*f.oneLinkRate +
		0.1*size*f.linkRate*f.uniqRate
}

func linkedNodes(n *SemanticNode) []*SemanticNode {
	var out []*SemanticNode
	var walk func(*SemanticNode)
	walk = func(x *SemanticNode) {
		if x.Link != "" {
			out = append(out, x)
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

func subtreeTexts(n *SemanticNode) []SemanticText {
	out := append([]SemanticText(nil), n.Texts...)
	for _, c := range n.Children {
		out = append(out, subtreeTexts(c)...)
	}
	return out
}

func subtreeHasImage(n *SemanticNode) bool {
	if len(n.Images) > 0 {
		return true
	}
	return slices.ContainsFunc(n.Children, subtreeHasImage)
}
