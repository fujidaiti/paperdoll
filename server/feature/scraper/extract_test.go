package scraper

import (
	"bytes"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestExtract_HeadOnlyPage(t *testing.T) {
	b, err := os.ReadFile("testdata/head_only.html")
	if err != nil {
		t.Fatal(err)
	}

	page, err := Extract(bytes.NewReader(b), url.URL{Scheme: "http", Host: "example.test", Path: "/article"})
	if err != nil {
		t.Fatalf("Extract returned an unexpected error: %v", err)
	}
	if page.Title != "Head Only Article" {
		t.Errorf("got title %q, want og:title %q", page.Title, "Head Only Article")
	}
	if page.Description != "The description from og:description." {
		t.Errorf("got description %q, want og:description %q", page.Description, "The description from og:description.")
	}
	if page.Content != "" {
		t.Errorf("got content %q, want empty for a page without readable content", page.Content)
	}
}

func TestExtract_TitleTagOnlyPage(t *testing.T) {
	b, err := os.ReadFile("testdata/title_tag_only.html")
	if err != nil {
		t.Fatal(err)
	}

	page, err := Extract(bytes.NewReader(b), url.URL{Scheme: "http", Host: "example.test", Path: "/article"})
	if err != nil {
		t.Fatalf("Extract returned an unexpected error: %v", err)
	}
	if page.Title != "Tom & Jerry's Article" {
		t.Errorf("got title %q, want the decoded <title> %q", page.Title, "Tom & Jerry's Article")
	}
	if page.Description != "Fish & chips." {
		t.Errorf("got description %q, want the decoded description meta %q", page.Description, "Fish & chips.")
	}
	if page.Content != "" {
		t.Errorf("got content %q, want empty for a page without readable content", page.Content)
	}
}

func TestExtract_Article(t *testing.T) {
	b, err := os.ReadFile("testdata/article.html")
	if err != nil {
		t.Fatal(err)
	}

	page, err := Extract(bytes.NewReader(b), url.URL{Scheme: "http", Host: "example.test", Path: "/article"})
	if err != nil {
		t.Fatalf("Extract returned an unexpected error: %v", err)
	}
	if page.Title != "Regular Article" {
		t.Errorf("got title %q, want %q", page.Title, "Regular Article")
	}
	// Not the first paragraph: the description only comes from metadata.
	if page.Description != "" {
		t.Errorf("got description %q, want empty for a page without description metadata", page.Description)
	}
	if !strings.Contains(page.Content, "This is the first paragraph") {
		t.Errorf("got content %q, want it to contain the article's paragraphs", page.Content)
	}
	for _, unsafe := range []string{"onclick", "javascript:"} {
		if strings.Contains(page.Content, unsafe) {
			t.Errorf("got content %q, want %q to be sanitized away", page.Content, unsafe)
		}
	}
}
