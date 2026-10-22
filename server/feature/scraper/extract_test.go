package scraper

import (
	"bytes"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name            string
		fixture         string
		wantTitle       string
		wantDescription string
		// wantContent is a substring the content must contain, or empty if
		// the content must be empty.
		wantContent string
	}{
		{
			name:            "head only page",
			fixture:         "testdata/head_only.html",
			wantTitle:       "Head Only Article",
			wantDescription: "The description from og:description.",
			wantContent:     "",
		},
		{
			name:            "title tag only page",
			fixture:         "testdata/title_tag_only.html",
			wantTitle:       "Tom & Jerry's Article",
			wantDescription: "Fish & chips.",
			wantContent:     "",
		},
		{
			// See https://github.com/fujidaiti/paperdoll/issues/212.
			name:            "JSON-LD is ignored",
			fixture:         "testdata/json_ld.html",
			wantTitle:       "JSON-LD Article | Example Site",
			wantDescription: "The description from og:description.",
			wantContent:     "",
		},
		{
			name:      "description falls back to the first paragraph",
			fixture:   "testdata/no_description.html",
			wantTitle: "No Description Article",
			wantDescription: "This is the first paragraph of a test article used by the scraper tests. " +
				"It contains several sentences so the readability extractor considers the page substantial enough to extract.",
			wantContent: "This is the second paragraph",
		},
		{
			// The sanitization itself is checked below for every case.
			name:            "unsafe markup",
			fixture:         "testdata/unsafe_markup.html",
			wantTitle:       "Unsafe Markup Article",
			wantDescription: "An article with unsafe markup.",
			wantContent:     "This is the first paragraph",
		},
		{
			name:            "no title, description, or content",
			fixture:         "testdata/no_context.html",
			wantTitle:       "",
			wantDescription: "",
			wantContent:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			page, err := Extract(bytes.NewReader(b), url.URL{Scheme: "http", Host: "example.test", Path: "/article"})
			if err != nil {
				t.Fatalf("Extract returned an unexpected error: %v", err)
			}
			if page.Title != tt.wantTitle {
				t.Errorf("got title %q, want %q", page.Title, tt.wantTitle)
			}
			if page.Description != tt.wantDescription {
				t.Errorf("got description %q, want %q", page.Description, tt.wantDescription)
			}
			if tt.wantContent == "" {
				if page.Content != "" {
					t.Errorf("got content %q, want empty", page.Content)
				}
			} else if !strings.Contains(page.Content, tt.wantContent) {
				t.Errorf("got content %q, want it to contain %q", page.Content, tt.wantContent)
			}
			for _, unsafe := range []string{"onclick", "javascript:"} {
				if strings.Contains(page.Content, unsafe) {
					t.Errorf("got content %q, want %q to be sanitized away", page.Content, unsafe)
				}
			}
		})
	}
}
