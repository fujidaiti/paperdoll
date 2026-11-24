// HTML fragments are concatenated without whitespace on purpose, so that the
// expected output is exact.
// ignore_for_file: missing_whitespace_between_adjacent_strings

import 'package:flutter_test/flutter_test.dart';
import 'package:paperdoll/features/reader/presentation/article_html.dart';

void main() {
  group('sanitizeArticleHtml', () {
    test('keeps article markup', () {
      const html =
          '<h2>Heading</h2>'
          '<p>Some <em>text</em> with <a href="https://a.example/x" title="t">a link</a>'
          ' and <a href="/relative">another</a>.</p>'
          '<figure><picture>'
          '<source srcset="https://a.example/1.webp 1x, /2.webp 2x">'
          '<img src="https://a.example/1.jpg" srcset="https://a.example/1.jpg 640w, b.jpg 1280w" alt="Alt" width="640" height="480">'
          '</picture><figcaption>Caption</figcaption></figure>'
          '<table><thead><tr><th colspan="2">Head</th></tr></thead>'
          '<tbody><tr><td rowspan="2">1</td><td>2</td></tr></tbody></table>'
          '<ul><li>One</li></ul><blockquote lang="fr" dir="ltr">Quote</blockquote>'
          '<pre><code>code()</code></pre>';

      expect(sanitizeArticleHtml(html), html);
    });

    test('removes code, forms and embedded documents with their contents', () {
      const html =
          '<p>Kept</p>'
          '<script>alert(1)</script>'
          '<style>p { color: red }</style>'
          '<meta http-equiv="refresh" content="0; url=https://evil.example">'
          '<base href="https://evil.example/">'
          '<link rel="stylesheet" href="https://evil.example/s.css">'
          '<form action="https://evil.example"><input name="q">Form text</form>'
          '<iframe src="https://evil.example"></iframe>'
          '<object data="x.swf">Object text</object>'
          '<embed src="x.swf">'
          '<noscript><img src="https://a.example/i.jpg"></noscript>'
          '<svg><a href="javascript:alert(1)"><text>SVG text</text></a></svg>';

      expect(sanitizeArticleHtml(html), '<p>Kept</p>');
    });

    test('removes event handlers, styles and classes', () {
      const html =
          '<p onclick="alert(1)" style="position:fixed" class="app-bar" '
          'id="x" data-x="1">Text</p>'
          '<img src="https://a.example/i.jpg" onerror="alert(1)">';

      expect(
        sanitizeArticleHtml(html),
        '<p>Text</p><img src="https://a.example/i.jpg">',
      );
    });

    test('removes links and sources with other schemes', () {
      const html =
          '<a href="javascript:alert(1)">1</a>'
          '<a href=" JavaScript:alert(1)">2</a>'
          '<a href="java&#x09;script:alert(1)">3</a>'
          '<a href="data:text/html,<script>alert(1)</script>">4</a>'
          '<a href="mailto:a@example.com">5</a>'
          '<a href="http://a.example/">6</a>'
          '<img src="http://a.example/i.jpg">'
          '<img src="data:image/png;base64,AAAA">'
          '<img srcset="https://a.example/1.jpg 1x, http://a.example/2.jpg 2x">'
          '<img srcset="https://a.example/1.jpg 1x,data:image/png;base64,AAAA 2x">';

      expect(
        sanitizeArticleHtml(html),
        '<a>1</a><a>2</a><a>3</a><a>4</a><a>5</a>'
        '<a href="http://a.example/">6</a><img><img><img><img>',
      );
    });

    test('replaces unknown elements with their contents', () {
      const html =
          '<custom-card><p>Inside</p></custom-card>'
          '<font color="red">Font</font>'
          '<button>Button</button>';

      expect(sanitizeArticleHtml(html), '<p>Inside</p>FontButton');
    });

    test('removes comments', () {
      expect(sanitizeArticleHtml('<p>A<!-- B --></p>'), '<p>A</p>');
    });
  });

  group('renderArticle', () {
    const template =
        '<header><p>{{site_name}}</p><h1>{{title}}</h1><p>{{byline}}</p>'
        '</header><article>{{content}}</article>';

    test('fills the template', () {
      final html = renderArticle((
        template: template,
        siteName: 'Coastal Notes',
        title: 'Tide Pools',
        byline: 'Mina Sato',
        content: '<p>Body</p><script>alert(1)</script>',
      ));

      expect(
        html,
        '<header><p>Coastal Notes</p><h1>Tide Pools</h1><p>Mina Sato</p>'
        '</header><article><p>Body</p></article>',
      );
    });

    test('escapes the metadata', () {
      final html = renderArticle((
        template: template,
        siteName: '<img src=x onerror=alert(1)>',
        title: '"Tom" & <b>Jerry</b>',
        byline: null,
        content: '',
      ));

      expect(
        html,
        '<header><p>&lt;img src=x onerror=alert(1)&gt;</p>'
        '<h1>&quot;Tom&quot; &amp; &lt;b&gt;Jerry&lt;&#47;b&gt;</h1><p></p>'
        '</header><article></article>',
      );
    });

    test('does not replace placeholders inside values', () {
      final html = renderArticle((
        template: template,
        siteName: null,
        title: '{{content}}',
        byline: null,
        content: '<p>{{title}}</p>',
      ));

      expect(
        html,
        '<header><p></p><h1>{{content}}</h1><p></p>'
        '</header><article><p>{{title}}</p></article>',
      );
    });
  });
}
