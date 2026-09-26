package api

import (
	"testing"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
	"github.com/google/go-cmp/cmp"
)

// The enumeration answers per item and the response answers per selector, so
// the transform is where the rows of the attribute screen are built. The case
// it has to get right is a group whose items do not all carry the same keys,
// which is how a group holds posts that have a description and posts that have
// none.
func TestPostGroups(t *testing.T) {
	// Ten posts. All of them carry a link and a title, and only the eighth
	// carries a description.
	var posts []feed.PostCandidate
	for i := range 10 {
		p := feed.PostCandidate{
			Links: []feed.Attribute{{Selector: ":scope", Value: "https://example.test/p/" + string(rune('0'+i))}},
			Texts: []feed.Attribute{{Selector: "div > h3", Value: "Post " + string(rune('0'+i))}},
		}
		if i == 7 {
			p.Texts = append(p.Texts, feed.Attribute{Selector: "div > p", Value: "The one post with a description"})
		}
		posts = append(posts, p)
	}

	got := postGroups([]feed.Group{{Selector: "html > body > ul > li", Posts: posts}})
	if len(got) != 1 {
		t.Fatalf("got %d groups, want 1", len(got))
	}
	g := got[0]
	if g.ID != 0 || g.Selector != "html > body > ul > li" || g.Count != 10 {
		t.Errorf("got id %d, key %q and count %d, want 0, the group key and 10", g.ID, g.Selector, g.Count)
	}
	// The first three items, plus the eighth, because it is the only one that
	// carries the description key. Without it that row would be empty
	// everywhere and the user could not tell what it is.
	if g.Sampled != 4 {
		t.Fatalf("got %d sampled items, want 4", g.Sampled)
	}
	want := []attributeCandidateSchema{
		{Selector: "div > h3", Values: []attributeValueSchema{
			{Value: "Post 0"}, {Value: "Post 1"}, {Value: "Post 2"}, {Value: "Post 7"},
		}},
		// The row of a key only one item carries holds an empty entry for the
		// others, so that index i of every row is item i of the sample.
		{Selector: "div > p", Values: []attributeValueSchema{
			{}, {}, {}, {Value: "The one post with a description"},
		}},
	}
	if d := cmp.Diff(want, g.Texts); d != "" {
		t.Errorf("text rows mismatch:\n%s", d)
	}
	if len(g.Links) != 1 || len(g.Links[0].Values) != g.Sampled {
		t.Errorf("got %d link rows with %d values, want 1 row of %d", len(g.Links), len(g.Links[0].Values), g.Sampled)
	}
	if len(g.Images) != 0 {
		t.Errorf("got %d image rows, want none", len(g.Images))
	}
}

// A group whose items all carry the same keys is sampled at sampleSize and no
// further.
func TestPostGroups_UniformItems(t *testing.T) {
	var posts []feed.PostCandidate
	for i := range 6 {
		posts = append(posts, feed.PostCandidate{
			Links: []feed.Attribute{{Selector: "a", Value: "https://example.test/p/" + string(rune('0'+i))}},
			Images: []feed.Attribute{{
				Selector: "img", Value: "https://example.test/i/" + string(rune('0'+i)), Alt: "An image",
			}},
		})
	}
	g := postGroups([]feed.Group{{Selector: "html > body > ul > li", Posts: posts}})[0]
	if g.Count != 6 || g.Sampled != sampleSize {
		t.Errorf("got count %d and %d sampled, want 6 and %d", g.Count, g.Sampled, sampleSize)
	}
	if len(g.Images) != 1 || g.Images[0].Values[0].Alt != "An image" {
		t.Errorf("the alt text of an image row is not reported: %+v", g.Images)
	}
}
