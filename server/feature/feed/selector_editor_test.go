package feed

import (
	"os"
	"strings"
	"testing"
)

// TestParseEditorPageRejectsWrongIDs checks that a page whose data-id of an
// element is not its position is rejected.
func TestParseEditorPageRejectsWrongIDs(t *testing.T) {
	raw, err := os.ReadFile("testdata/extract/list.html")
	if err != nil {
		t.Fatal(err)
	}
	page, err := SimplifyPage(raw, "https://example.com/")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := parseEditorPage(page); err != nil {
		t.Fatalf("the page with node IDs: %v", err)
	}
	// The body is always node 0.
	wrong := strings.Replace(page, `<body data-id="0">`, `<body data-id="1">`, 1)
	if wrong == page {
		t.Fatalf("the page has no body with the ID 0: %s", page)
	}
	if _, _, err := parseEditorPage(wrong); err == nil {
		t.Error("got no error for a wrong data-id")
	}
}
