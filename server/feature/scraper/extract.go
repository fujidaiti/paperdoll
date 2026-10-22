package scraper

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"codeberg.org/readeck/go-readability/v2"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

// Page is what Extract finds in a fetched web page.
type Page struct {
	// Title comes from the page's metadata (e.g. og:title, <title>). Empty if
	// the page has none.
	Title string
	// Description comes from the page's metadata (e.g. og:description), or
	// from the first paragraph of its content if the page has none. Empty if
	// neither exists. Truncated to 500 characters.
	Description string
	// Content is a sanitized HTML document holding the page's readable
	// content. Empty if none could be extracted, e.g. for a page whose body
	// is rendered by JavaScript.
	Content string
}

// Extract parses an HTML page once and returns its metadata and readable
// content. The metadata is returned even if the content can't be extracted.
func Extract(r io.Reader, pageURL url.URL) (Page, error) {
	baseUrl := new(pageURL)
	baseUrl.Path = ""
	baseUrl.RawPath = ""
	baseUrl.Fragment = ""
	baseUrl.RawFragment = ""
	baseUrl.RawQuery = ""
	baseUrl.User = nil

	// Parse the raw page rather than a sanitized one: sanitizing first would
	// strip the <meta> tags the metadata is read from. The extracted content
	// is sanitized below instead.
	parser := readability.NewParser()
	// Readability ranks JSON-LD metadata above og:title, <title>, and
	// og:description, but some sites put unsuitable data there, e.g. the whole
	// article as the description.
	// TODO(#212): Use JSON-LD metadata with sanity checks.
	parser.DisableJSONLD = true
	article, err := parser.Parse(r, baseUrl)
	if err != nil {
		return Page{}, err
	}

	page := Page{Title: strings.TrimSpace(article.Title())}
	if article.Node == nil {
		// If the page has no description metadata, Article.Excerpt looks for
		// the first paragraph in Article.Node, which panics with a nil pointer
		// dereference if there's no content. Call it on a copy with empty
		// content instead to get the description from the metadata only.
		meta := article
		meta.Node = &html.Node{Type: html.DocumentNode}
		page.Description = truncate(meta.Excerpt())
		return page, nil
	}
	// Falls back to the content's first paragraph if the page has no
	// description metadata.
	page.Description = truncate(article.Excerpt())

	var buf bytes.Buffer
	if err := article.RenderHTML(&buf); err != nil {
		return Page{}, err
	}
	content := bluemonday.UGCPolicy().SanitizeReader(&buf)
	if content.Len() > 0 {
		page.Content = fmt.Sprintf(contentTemplate, content)
	}
	return page, nil
}

// truncate shortens a description to at most 500 characters, replacing the
// last one with an ellipsis if it's longer. Descriptions are only shown as
// short previews, and some pages put the whole article in them. Characters are
// counted as Unicode code points, not words, so that it works for languages
// without spaces between words.
func truncate(s string) string {
	const maxLen = 500
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen-1]) + "…"
}

const contentTemplate = `
<!DOCTYPE html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
%s
</body>
</html>
`
