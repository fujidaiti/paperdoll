// Package scoredtree is an experiment that stands beside the selection in the
// feed package and is not used by it.
//
// The selection there scores a group of siblings and offers the groups whose
// score is high enough. This package asks the question one level down: how
// much does a single node look like a post? Grouping is answered afterwards,
// from the labelled nodes, because siblings that are posts and sit under the
// same parent are already a list.
//
// The tree is read from the JSON files the feed package writes under
// testdata/semantic/, so that nothing here has to parse HTML and no file of
// the feed package has to change.
package scoredtree

import (
	"encoding/json"
	"net/url"
	"os"
	"strings"
)

// Node is one node of the semantic tree, read from the JSON the feed package
// writes. It carries the two fields this experiment adds: the score of the
// node and whether the score passed the threshold.
type Node struct {
	// Score is how much the node looks like a post, between 0 and 1. It is
	// written by ScoreNodes and is 0 until then. The root has no score,
	// because a score is only meaningful next to the siblings it was read
	// with. It is written before the values so that it is readable in a file
	// whose nodes hold long texts.
	Score float64 `json:"score"`
	// Post is whether Score passed the threshold ScoreNodes was given.
	Post bool `json:"post,omitempty"`

	Link     string  `json:"link,omitempty"`
	Texts    []Text  `json:"texts,omitempty"`
	Images   []Image `json:"images,omitempty"`
	Children []*Node `json:"children,omitempty"`

	// Parent is not part of the JSON. It is filled in while loading, because
	// grouping reads it.
	Parent *Node `json:"-"`
}

type Text struct {
	Tag      string `json:"tag"`
	Value    string `json:"value"`
	Datetime string `json:"datetime,omitempty"`
}

type Image struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

// Load reads one semantic tree from the JSON file the feed package wrote.
func Load(path string) (*Node, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root Node
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	link(&root, nil)
	return &root, nil
}

func link(n *Node, parent *Node) {
	n.Parent = parent
	for _, c := range n.Children {
		link(c, n)
	}
}

// Walk returns every node of the tree, parents before children.
func (n *Node) Walk() []*Node {
	out := []*Node{n}
	for _, c := range n.Children {
		out = append(out, c.Walk()...)
	}
	return out
}

// Links returns every link of the subtree, with repeats.
func (n *Node) Links() []string {
	var out []string
	for _, x := range n.Walk() {
		if x.Link != "" {
			out = append(out, x.Link)
		}
	}
	return out
}

func (n *Node) subtreeTexts() []Text {
	var out []Text
	for _, x := range n.Walk() {
		out = append(out, x.Texts...)
	}
	return out
}

func (n *Node) hasImage() bool {
	for _, x := range n.Walk() {
		if len(x.Images) > 0 {
			return true
		}
	}
	return false
}

// Normalize is the form a link is compared with a fixture URL in. The query
// and the fragment are kept, because developer.apple.com identifies a post by
// ?id= and some pages identify one by #anchor.
func Normalize(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	out := strings.ToLower(u.Host) + strings.TrimRight(u.Path, "/")
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		out += "#" + u.Fragment
	}
	return out
}
