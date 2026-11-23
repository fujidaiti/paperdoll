import 'dart:convert';

import 'package:html/dom.dart';
import 'package:html/parser.dart';

/// Builds the document the reader shows for an extracted article: the template
/// (`assets/reader/reader.html`) with its placeholders filled in.
///
/// Everything in the article comes from the original page, whose scripts can
/// rewrite it, so the content is sanitized with [sanitizeArticleHtml] and the
/// metadata is HTML-escaped. The placeholders are replaced in one pass, so a
/// placeholder inside a value is not replaced again.
///
/// Runs in an isolate with `compute`, which is why it takes a single record.
String renderArticle(
  ({
    String template,
    String? siteName,
    String? title,
    String? byline,
    String content,
  })
  article,
) {
  const escape = HtmlEscape();
  return article.template.replaceAllMapped(
    RegExp(r'\{\{(\w+)\}\}'),
    (match) => switch (match[1]) {
      'site_name' => escape.convert(article.siteName ?? ''),
      'title' => escape.convert(article.title ?? ''),
      'byline' => escape.convert(article.byline ?? ''),
      'content' => sanitizeArticleHtml(article.content),
      _ => match[0]!,
    },
  );
}

/// Keeps only article markup in [html] by an allowlist, so that a page cannot
/// run code, navigate, restyle the reader or imitate app UI through it.
///
/// Elements in [_removedElements] are removed with their contents, and other
/// elements not in [_allowedElements] are replaced with their contents.
/// Attributes not in [_allowedAttributes] are removed, as are links that are
/// not http, https or relative, and image sources that are not https or
/// relative (the reader's CSP allows https images only).
String sanitizeArticleHtml(String html) {
  final fragment = parseFragment(html);
  _sanitizeChildren(fragment);
  return fragment.outerHtml;
}

void _sanitizeChildren(Node parent) {
  for (final node in parent.nodes.toList()) {
    switch (node) {
      case Text():
        break;
      case Element(:final localName?) when _removedElements.contains(localName):
        node.remove();
      case Element(:final localName?) when _allowedElements.contains(localName):
        _sanitizeAttributes(node);
        _sanitizeChildren(node);
      case Element():
        _sanitizeChildren(node);
        for (final child in node.nodes.toList()) {
          parent.insertBefore(child, node);
        }
        node.remove();
      default:
        // Comments and other non-content nodes.
        node.remove();
    }
  }
}

void _sanitizeAttributes(Element element) {
  final attributes = element.attributes;
  for (final name in attributes.keys.toList()) {
    final value = attributes[name]!;
    final keep = switch (name) {
      'href' => _isAllowedUrl(value, {'http', 'https'}),
      'src' => _isAllowedUrl(value, {'https'}),
      'srcset' => _srcsetUrls(
        value,
      ).every((url) => _isAllowedUrl(url, {'https'})),
      final String name => _allowedAttributes.contains(name),
      _ => false,
    };
    if (!keep) {
      attributes.remove(name);
    }
  }
}

/// Whether [url] is relative or has one of [schemes].
bool _isAllowedUrl(String url, Set<String> schemes) {
  // Browsers remove tabs and newlines anywhere in a URL and ignore leading
  // control characters and spaces, so "java\tscript:" is a javascript: URL.
  final cleaned = url
      .replaceAll(RegExp('[\t\n\r]'), '')
      .replaceFirst(RegExp(r'^[\x00-\x20]+'), '');
  final scheme = RegExp(r'^([a-zA-Z][a-zA-Z0-9+.\-]*):')
      .firstMatch(cleaned)?[1]
      ?.toLowerCase();
  return scheme == null || schemes.contains(scheme);
}

/// The URLs of the image candidates in a `srcset` attribute, split as the
/// HTML spec describes: each URL runs up to the next whitespace, followed by
/// optional descriptors up to the next comma outside parentheses.
List<String> _srcsetUrls(String srcset) {
  final urls = <String>[];
  var i = 0;
  bool isSpace(int i) => ' \t\n\r\f'.contains(srcset[i]);
  while (true) {
    while (i < srcset.length && (isSpace(i) || srcset[i] == ',')) {
      i++;
    }
    if (i >= srcset.length) {
      return urls;
    }
    final start = i;
    while (i < srcset.length && !isSpace(i)) {
      i++;
    }
    final url = srcset.substring(start, i);
    if (url.endsWith(',')) {
      urls.add(url.replaceFirst(RegExp(r',+$'), ''));
      continue;
    }
    urls.add(url);
    var inParens = false;
    while (i < srcset.length && (inParens || srcset[i] != ',')) {
      if (srcset[i] == '(') {
        inParens = true;
      } else if (srcset[i] == ')') {
        inParens = false;
      }
      i++;
    }
  }
}

const _allowedElements = {
  'a', 'abbr', 'address', 'article', 'aside', 'b', 'bdi', 'bdo', //
  'blockquote', 'br', 'caption', 'cite', 'code', 'col', 'colgroup', 'dd',
  'del', 'details', 'dfn', 'div', 'dl', 'dt', 'em', 'figcaption', 'figure',
  'footer', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'header', 'hr', 'i', 'img',
  'ins', 'kbd', 'li', 'main', 'mark', 'ol', 'p', 'picture', 'pre', 'q', 'rp',
  'rt', 'ruby', 's', 'samp', 'section', 'small', 'source', 'span', 'strong',
  'sub', 'summary', 'sup', 'table', 'tbody', 'td', 'tfoot', 'th', 'thead',
  'time', 'tr', 'u', 'ul', 'var', 'wbr',
};

/// Elements removed with their contents: those that run code, load other
/// documents, change the page's behavior or styles, or hold text that is not
/// meant to be read as part of the article.
const _removedElements = {
  'script', 'style', 'meta', 'base', 'link', 'form', 'iframe', 'object', //
  'embed', 'svg', 'template', 'noscript', 'noembed', 'title', 'textarea',
  'select', 'video', 'audio', 'canvas',
};

const _allowedAttributes = {
  'href', 'src', 'srcset', 'alt', 'title', 'width', 'height', 'colspan', //
  'rowspan', 'lang', 'dir',
};
