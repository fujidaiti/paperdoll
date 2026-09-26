package scoredtree

// FoldCards folds every card of the tree into a single node.
//
// BuildSemanticTree keeps a subtree split as soon as it carries two distinct
// links, so a card that links to its category, to its author or to a "read
// more" page stays a subtree even though it stands for one post. That costs
// the user twice: the card has to be opened and read through instead of being
// read at once, and the extra links and the texts written on them are offered
// as values of the post.
//
// A card is recognised from what its siblings agree on. Most children of one
// node carry a link of the same URL template, which is the template of the
// posts, so a child that carries exactly one link of that template stands for
// exactly one post. Every other link it carries cannot be that post and is
// taken out, which leaves one link and lets the card be folded.
func FoldCards(root *Node) {
	var walk func(parent *Node)
	walk = func(parent *Node) {
		members := parent.Children
		if len(members) == 0 {
			return
		}
		links := groupLinks(members, candidatesOf(members))
		var order []string
		counts := map[string]int{}
		for _, l := range links {
			if l == "" {
				continue
			}
			t := urlTemplate(l)
			if counts[t] == 0 {
				order = append(order, t)
			}
			counts[t]++
		}
		shape, agreed := "", 0
		for _, t := range order {
			if counts[t] > agreed {
				shape, agreed = t, counts[t]
			}
		}
		// Two siblings that happen to carry a link of the same template say
		// nothing; a list says it about most of its items.
		if agreed < minAgreed {
			shape = ""
		}

		for i, m := range members {
			if len(m.Children) == 0 {
				continue
			}
			if shape != "" && links[i] != "" && urlTemplate(links[i]) == shape {
				var ofShape map[string]bool
				all := distinctLinks(m)
				for l := range all {
					if urlTemplate(l) == shape {
						if ofShape == nil {
							ofShape = map[string]bool{}
						}
						ofShape[l] = true
					}
				}
				// A subtree that carries many links is a part of the page
				// holding several cards, not a card that names its category.
				if len(ofShape) == 1 && len(all) <= maxCardLinks {
					stripLinks(m, ofShape)
					if len(distinctLinks(m)) <= 1 {
						fold(m)
						continue
					}
				}
			}
			walk(m)
		}
	}
	walk(root)
}

const (
	// maxCardLinks is how many distinct links a subtree may carry and still be
	// read as a single card.
	maxCardLinks = 10
	// minAgreed is how many children have to carry a link of the same URL
	// template before that template is read as the template of the posts.
	minAgreed = 3
	// keepTextLen is the length of a text that makes a link a post of its own.
	// A category or an author is a word or a name, while a second post inside
	// the card is written with its title, so taking its link out would lose a
	// post the page lists.
	keepTextLen = 40
)

// stripLinks takes out the links of a subtree that are not allowed, so that
// the subtree is left with the one link it stands for. The texts of the nodes
// the links sat on are kept, because a card writes its title on the link and
// a date beside it.
func stripLinks(n *Node, allowed map[string]bool) {
	for _, c := range n.Children {
		long := false
		for _, t := range c.Texts {
			if len(t.Value) >= keepTextLen {
				long = true
			}
		}
		if c.Link != "" && !allowed[c.Link] && !long {
			c.Link = ""
		}
		stripLinks(c, allowed)
	}
}

// fold moves every value of the subtree into n, in document order, and leaves
// n without children.
func fold(n *Node) {
	var texts []Text
	var images []Image
	link := n.Link
	var walk func(*Node)
	walk = func(x *Node) {
		if x != n {
			if link == "" {
				link = x.Link
			}
			texts = append(texts, x.Texts...)
			images = append(images, x.Images...)
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	n.Link = link
	n.Texts = append(n.Texts, texts...)
	n.Images = append(n.Images, images...)
	n.Children = nil
}
