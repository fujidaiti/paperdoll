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
	// Title and Description come from the page's metadata (e.g. og:title,
	// <title>, og:description). Empty if the page has none.
	Title       string
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
	// TODO: Use JSON-LD metadata with sanity checks (#212).
	parser.DisableJSONLD = true
	article, err := parser.Parse(r, baseUrl)
	if err != nil {
		return Page{}, err
	}

	// Article.Excerpt falls back to the content's first paragraph when the
	// page has no description metadata, and panics if there's no content.
	// Call it on a copy with empty content to get the metadata only.
	meta := article
	meta.Node = &html.Node{Type: html.DocumentNode}
	page := Page{
		Title:       strings.TrimSpace(article.Title()),
		Description: meta.Excerpt(),
	}
	if article.Node == nil {
		return page, nil
	}

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
