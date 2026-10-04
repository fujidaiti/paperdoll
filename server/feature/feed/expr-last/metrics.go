package structures

import (
	"net/url"

	"golang.org/x/net/html"
)

// An Instance is one subtree the root part's matcher found. It stands for one
// post.
type Instance struct {
	Structure *Structure
	Root      *PartInstance
}

// A PartInstance is one subtree a part's matcher found, together with the
// values the fields of that part read there. Its children are the part
// instances of the child parts. One part can produce several part instances
// inside one instance, such as one per tag chip.
type PartInstance struct {
	Part     *Part
	Node     *html.Node
	Read     []FieldRead
	Children []*PartInstance
}

// A FieldRead is what one field read in one part instance. A field that found
// nothing there produces no FieldRead.
type FieldRead struct {
	Field  *Field
	Values []string
}

// Extract applies one structure to a page and returns one instance per subtree
// the root part's matcher finds. It is the three step procedure DEF.md
// describes, and it is what runs both at review time and at polling time.
func Extract(s *Structure, page *html.Node, base *url.URL) []*Instance {
	if s == nil || s.Root == nil || s.Root.Match == nil {
		return nil
	}
	var out []*Instance
	for _, node := range s.Root.Match(page) {
		out = append(out, &Instance{Structure: s, Root: instantiate(s.Root, node, base)})
	}
	return out
}

// instantiate reads the fields of one part in one subtree it matched, then
// runs the matcher of every child part inside that same subtree.
func instantiate(p *Part, node *html.Node, base *url.URL) *PartInstance {
	pi := &PartInstance{Part: p, Node: node}
	for _, f := range p.Fields {
		var read []string
		switch {
		case f.Blob != nil:
			for _, n := range f.Blob(node) {
				read = append(read, subtreeValues(n, base)...)
			}
		case f.Match != nil:
			if n := f.Match(node); n != nil {
				read = values(n, base)
			}
		}
		if len(read) > 0 {
			pi.Read = append(pi.Read, FieldRead{Field: f, Values: read})
		}
	}
	for _, child := range p.Children {
		if child.Match == nil {
			continue
		}
		for _, sub := range child.Match(node) {
			pi.Children = append(pi.Children, instantiate(child, sub, base))
		}
	}
	return pi
}

// subtreeValues returns the values of one node and of every node below it, in
// document order. A blob observes its run of sibling nodes this way.
func subtreeValues(n *html.Node, base *url.URL) []string {
	if !observable(n) {
		return nil
	}
	out := values(n, base)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, subtreeValues(c, base)...)
	}
	return out
}

// valueSet returns every value the instance holds, which is what a fixture
// post is compared against.
func (in *Instance) valueSet() map[string]bool {
	out := map[string]bool{}
	var walk func(*PartInstance)
	walk = func(pi *PartInstance) {
		for _, r := range pi.Read {
			for _, v := range r.Values {
				out[v] = true
			}
		}
		for _, c := range pi.Children {
			walk(c)
		}
	}
	walk(in.Root)
	return out
}

// cost is the review cost of one fixture post read from one instance: the
// distrib and wrappers of DEF.md.
//
// want holds the values of the post that the instance carries. The part
// instances holding at least one of them are the holders, and the smallest
// subtree of the instance that covers them is the holders together with the
// part instances in between, up to their lowest common ancestor. distrib is
// how many distinct parts that subtree uses, and wrappers is how many distinct
// parts of the instance it leaves out.
func (in *Instance) cost(want map[string]bool) (distrib, wrappers int) {
	// The tree is flattened so that a part instance can be walked upwards.
	var nodes []*PartInstance
	parent := map[*PartInstance]*PartInstance{}
	depth := map[*PartInstance]int{}
	var walk func(*PartInstance, *PartInstance, int)
	walk = func(pi, up *PartInstance, d int) {
		nodes = append(nodes, pi)
		parent[pi] = up
		depth[pi] = d
		for _, c := range pi.Children {
			walk(c, pi, d+1)
		}
	}
	walk(in.Root, nil, 0)

	var holders []*PartInstance
	for _, pi := range nodes {
		for _, r := range pi.Read {
			if holdsAny(r.Values, want) {
				holders = append(holders, pi)
				break
			}
		}
	}
	if len(holders) == 0 {
		return 0, len(partsOf(nodes))
	}

	top := holders[0]
	for _, h := range holders[1:] {
		a, b := top, h
		for depth[a] > depth[b] {
			a = parent[a]
		}
		for depth[b] > depth[a] {
			b = parent[b]
		}
		for a != b {
			a, b = parent[a], parent[b]
		}
		top = a
	}

	inside := map[*PartInstance]bool{}
	for _, h := range holders {
		for pi := h; pi != nil && !inside[pi]; pi = parent[pi] {
			inside[pi] = true
			if pi == top {
				break
			}
		}
	}
	var covered []*PartInstance
	for _, pi := range nodes {
		if inside[pi] {
			covered = append(covered, pi)
		}
	}
	return len(partsOf(covered)), len(partsOf(nodes)) - len(partsOf(covered))
}

// partsOf returns the distinct parts the part instances belong to. The metrics
// count parts and not part instances, because the review screen shows one row
// per part however many subtrees that part matches.
func partsOf(nodes []*PartInstance) map[*Part]bool {
	out := map[*Part]bool{}
	for _, pi := range nodes {
		out[pi.Part] = true
	}
	return out
}

func holdsAny(have []string, want map[string]bool) bool {
	for _, v := range have {
		if want[v] {
			return true
		}
	}
	return false
}

// blobReads returns, for every value some blob of the page read, the blob
// fields that read it.
func blobReads(instances []*Instance) map[string]map[*Field]bool {
	out := map[string]map[*Field]bool{}
	for _, in := range instances {
		var walk func(*PartInstance)
		walk = func(pi *PartInstance) {
			for _, r := range pi.Read {
				if r.Field.Blob == nil {
					continue
				}
				for _, v := range r.Values {
					if out[v] == nil {
						out[v] = map[*Field]bool{}
					}
					out[v][r.Field] = true
				}
			}
			for _, c := range pi.Children {
				walk(c)
			}
		}
		walk(in.Root)
	}
	return out
}

// blobFields returns the blob fields of the structures of a page.
func blobFields(ss []*Structure) map[*Field]bool {
	out := map[*Field]bool{}
	for _, s := range ss {
		var walk func(*Part)
		walk = func(p *Part) {
			for _, f := range p.Fields {
				if f.Blob != nil {
					out[f] = true
				}
			}
			for _, c := range p.Children {
				walk(c)
			}
		}
		if s.Root != nil {
			walk(s.Root)
		}
	}
	return out
}
