package feed

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestExtractPosts checks the posts that ExtractPosts finds for each behaviour
// in testdata/extract, on pages that are already simplified.
func TestExtractPosts(t *testing.T) {
	runCases(t, "extract", "posts.json", func(t *testing.T, sections map[string]string) {
		doc, _ := parseCasePage(t, sections["page.html"])
		var rules ExtractionRules
		if err := json.Unmarshal([]byte(sections["rules.json"]), &rules); err != nil {
			t.Fatal(err)
		}
		checkJSON(t, sections["posts.json"], ExtractPosts(doc, rules))
	})
}

func TestPostSelectorJSON(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		m := PostSelector{
			Box:     []Step{{Tag: "a", Cls: []string{"card"}}},
			Title:   []Step{{Tag: "h3"}},
			Link:    []Step{},
			PubDate: []Step{{Tag: "time", Nth: new(0)}},
		}

		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"box":[{"tag":"a","cls":["card"]}],` +
			`"title":[{"tag":"h3"}],"link":[],"pubDate":[{"tag":"time","nth":0}]}`
		if string(data) != want {
			t.Errorf("JSON is %s, want %s", data, want)
		}

		// The empty path of the link stays empty instead of becoming nil, which
		// would mean that the selector has no link.
		var got PostSelector
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(m, got); diff != "" {
			t.Errorf("selector differs (-want +got):\n%s", diff)
		}
	})

	errors := []struct{ name, json string }{
		{"unknown field", `{"box":[{"tag":"article"}],"title":[],"link":[],"author":[]}`},
		{"no title", `{"box":[{"tag":"article"}],"link":[]}`},
		{"no link", `{"box":[{"tag":"article"}],"title":[]}`},
	}
	for _, tt := range errors {
		t.Run(tt.name, func(t *testing.T) {
			var m PostSelector
			if err := json.Unmarshal([]byte(tt.json), &m); err == nil {
				t.Errorf("no error for %s", tt.json)
			}
		})
	}
}

func TestExtractionRulesJSON(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		rules := ExtractionRules{
			Selectors: []PostSelector{{
				Box:   []Step{{Tag: "article"}},
				Title: []Step{{Tag: "h2"}},
				Link:  []Step{{Tag: "a"}},
			}},
			ExcludeLinks: []string{"example.com/newsletters"},
			ExcludeNodes: [][]Step{{{Tag: "nav", Nth: new(0)}}},
		}

		data, err := json.Marshal(rules)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"selectors":[{"box":[{"tag":"article"}],"title":[{"tag":"h2"}],"link":[{"tag":"a"}]}],` +
			`"excludeLinks":["example.com/newsletters"],` +
			`"excludeNodes":[[{"tag":"nav","nth":0}]]}`
		if string(data) != want {
			t.Errorf("JSON is %s, want %s", data, want)
		}

		var got ExtractionRules
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(rules, got); diff != "" {
			t.Errorf("rules differ (-want +got):\n%s", diff)
		}
	})

	// Rules without selectors and exclusions keep the selectors as an array, and
	// leave out the empty lists.
	t.Run("empty", func(t *testing.T) {
		data, err := json.Marshal(ExtractionRules{Selectors: []PostSelector{}})
		if err != nil {
			t.Fatal(err)
		}
		if want := `{"selectors":[]}`; string(data) != want {
			t.Errorf("JSON is %s, want %s", data, want)
		}
	})
}

// TestBodyText checks the body of the only child of the body of each page in
// testdata/body.
func TestBodyText(t *testing.T) {
	runCases(t, "body", "want.txt", func(t *testing.T, sections map[string]string) {
		doc, _ := parseCasePage(t, sections["page.html"])
		want := strings.TrimSpace(sections["want.txt"])
		if got := bodyText(pageBody(doc).FirstChild); got != want {
			t.Errorf("bodyText is %q, want %q", got, want)
		}
	})
}
