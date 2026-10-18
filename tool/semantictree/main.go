// Command semantictree converts sample web pages into semantic trees (JSON),
// which post matchers run on (see MATCHER.md), with the page simplifier of
// the server (feed.SimplifyPage; feed.PageDocument describes the format).
//
// Usage (from the repository root):
//
//	go run ./tool/semantictree [-in server/feature/feed/testdata/source] -out DIR
//
// The trees of the sample pages are also in testdata, written by:
//
//	go run ./tool/semantictree -out tool/semantictree/testdata/source
//	go run ./tool/semantictree -in server/feature/feed/testdata/robustness -out tool/semantictree/testdata/robustness
//
// Output: one JSON file per page, plus index.json listing them as
// [{"file": "...", "url": "...", "title": "..."}].
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fujidaiti/paperdoll/server/feature/feed"
)

func main() {
	in := flag.String("in", "server/feature/feed/testdata/source", "directory of sample HTML files")
	out := flag.String("out", "", "output directory (required)")
	flag.Parse()
	if *out == "" {
		log.Fatal("-out is required")
	}

	files, err := filepath.Glob(filepath.Join(*in, "*.html"))
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	type IndexEntry struct {
		File  string `json:"file"`
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	var index []IndexEntry
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".html")
		raw, err := os.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		doc, err := feed.SimplifyPage(raw, "https://"+strings.ReplaceAll(name, "-", "/"))
		if err != nil {
			log.Fatalf("%s: %v", f, err)
		}
		// Pages rendered by JavaScript have nothing to annotate.
		if len(doc.Root.Children) == 0 {
			fmt.Printf("%-50s skipped: empty body\n", filepath.Base(f))
			continue
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(doc); err != nil {
			log.Fatal(err)
		}
		data := buf.Bytes()
		if err := os.WriteFile(filepath.Join(*out, name+".json"), data, 0o644); err != nil {
			log.Fatal(err)
		}
		index = append(index, IndexEntry{File: name + ".json", URL: doc.URL, Title: doc.Title})
		fmt.Printf("%-50s %8d -> %8d bytes\n", filepath.Base(f), len(raw), len(data))
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*out, "index.json"), data, 0o644); err != nil {
		log.Fatal(err)
	}
}
