package feed

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

// TestResolveMarks checks that the marks of each case in testdata/editor with
// an error.txt are rejected with that message.
func TestResolveMarks(t *testing.T) {
	runCases(t, "editor", "error.txt", func(t *testing.T, sections map[string]string) {
		_, index := parseCasePage(t, sections["page.html"])
		var marks Marks
		if err := json.Unmarshal([]byte(sections["marks.json"]), &marks); err != nil {
			t.Fatal(err)
		}
		_, _, _, err := index.resolveMarks(marks)
		if !errors.Is(err, ErrInvalidMarks) {
			t.Fatalf("got the error %v, want ErrInvalidMarks", err)
		}
		if want := strings.TrimSpace(sections["error.txt"]); err.Error() != want {
			t.Errorf("got the error %q, want %q", err, want)
		}
	})
}

// TestEditorState checks what the rules generated from the marks of each case
// in testdata/editor with a state.json find.
func TestEditorState(t *testing.T) {
	runCases(t, "editor", "state.json", func(t *testing.T, sections map[string]string) {
		doc, index := parseCasePage(t, sections["page.html"])
		var marks Marks
		if err := json.Unmarshal([]byte(sections["marks.json"]), &marks); err != nil {
			t.Fatal(err)
		}
		posts, excluded, excludedNodes, err := index.resolveMarks(marks)
		if err != nil {
			t.Fatal(err)
		}
		rules := index.generateRules(posts, excluded, excludedNodes)
		checkJSON(t, sections["state.json"], index.editorState(doc, rules))
	})
}

// TestParseEditorPage checks that a page whose data-id of an element is not its
// position is rejected.
func TestParseEditorPage(t *testing.T) {
	raw, err := os.ReadFile("testdata/webpage/go.dev-blog/simplified.golden.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(renderGoldenPage(t, raw))
	if _, _, err := parseEditorPage(page); err != nil {
		t.Fatalf("the page with node IDs: %v", err)
	}
	wrong := strings.Replace(page, `data-id="1"`, `data-id="2"`, 1)
	if wrong == page {
		t.Fatalf("the page has no node 1: %s", page)
	}
	if _, _, err := parseEditorPage(wrong); err == nil {
		t.Error("got no error for a wrong data-id")
	}
}
