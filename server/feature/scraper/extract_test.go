package scraper

import (
	"bytes"
	"net/url"
	"os"
	"strings"
	"testing"
)

// TODO(#218): Cover every title and description source and their priority, so
// that the parser can be replaced without changing the behavior.
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
			// Truncated to 500 characters, the last of which is the ellipsis.
			name:      "long description",
			fixture:   "testdata/long_description.html",
			wantTitle: "Long Description Page",
			wantDescription: "毎朝の通勤時間を使って、ニュースやブログ記事をまとめて読む習慣を続けている人は多いでしょう。" +
				"しかし、気になった記事をブラウザのタブに開いたままにしておくと、いつの間にか数十個のタブが溜まり、どれが重要だったのか分からなくなってしまいます。" +
				"この記事では、後で読みたい記事を一か所に集め、空いた時間に落ち着いて読むための仕組みづくりについて紹介します。" +
				"まず大切なのは、記事を見つけた瞬間に迷わず保存できることです。" +
				"共有メニューから一度タップするだけで保存できれば、読むかどうかの判断を後回しにでき、目の前の作業に集中できます。" +
				"次に、保存した記事を読むときには、広告やポップアップに邪魔されない読みやすい表示が欠かせません。" +
				"本文だけを抜き出して表示すれば、小さな画面でも文章に集中できます。" +
				"さらに、読み終えた記事をアーカイブしておけば、リストは常に未読の記事だけになり、次に何を読むべきかがすぐに分かります。" +
				"最後に、こうした習慣を長く続けるためのコツとして、保存する記事の数を一日に数本までに絞ること、週末にまとめて読み返す時間を設けること、そして読む価値がないと感じた記事は迷わず削除することを挙げてお…",
			wantContent: "",
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
			for _, unsafe := range []string{"<script", "alert(", "onclick", "javascript:"} {
				if strings.Contains(page.Content, unsafe) {
					t.Errorf("got content %q, want %q to be sanitized away", page.Content, unsafe)
				}
			}
		})
	}
}
