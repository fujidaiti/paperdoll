import 'package:paperdoll/features/page_watch/domain/page_document.dart';

/// Converts the subtree of [node] into the marked-up text of a post body.
///
/// Blocks are separated by a blank line:
///
/// - a paragraph of inline text
/// - `# heading` (one `#` per level, h1 to h6)
/// - `- item` and `1. item` for list items, one per line
/// - `> quote` for each line of a block quote
/// - a code block fenced by three backticks
/// - `@[embed](url)` for an embedded page such as a video player
///
/// Inline text uses `**bold**`, `_italic_`, `` `code` ``, `[text](url)` for
/// links and `![alt](url)` for images. A line break is a newline.
String bodyMarkup(PageNode node) => _blocks(node).join('\n\n');

List<String> _blocks(PageNode node) {
  if (node.isEmbed) {
    return ['@[embed](${node.src})'];
  }
  switch (node.tag) {
    case 'pre':
      return ['```\n${node.text}\n```'];
    case 'h1' || 'h2' || 'h3' || 'h4' || 'h5' || 'h6':
      final text = _inline(node);
      return [if (text.isNotEmpty) '${'#' * int.parse(node.tag[1])} $text'];
    case 'ul' || 'ol':
      final items = [
        for (final child in node.children)
          if (_blocks(child).join('\n') case final item when item.isNotEmpty)
            item.replaceAll('\n', '\n  '),
      ];
      return [
        if (items.isNotEmpty)
          [
            for (final (i, item) in items.indexed)
              '${node.tag == 'ol' ? '${i + 1}.' : '-'} $item',
          ].join('\n'),
      ];
    case 'blockquote':
      final quote = node.children.isEmpty
          ? _inline(node)
          : _containerBlocks(node).join('\n\n');
      return [
        if (quote.isNotEmpty)
          quote.split('\n').map((line) => '> $line'.trimRight()).join('\n'),
      ];
  }
  // A link around blocks, such as a card, cannot be written as inline text,
  // so only its content is kept.
  if (node.children.isEmpty || (_isInline(node) && _hasInlineContent(node))) {
    final text = _inline(node);
    return [if (text.isNotEmpty) text];
  }
  return _containerBlocks(node);
}

/// The blocks of a container, where each run of consecutive inline children
/// becomes one paragraph.
List<String> _containerBlocks(PageNode node) {
  final blocks = <String>[];
  final paragraph = <String>[];
  void flush() {
    if (paragraph.isNotEmpty) {
      blocks.add(paragraph.join(' '));
      paragraph.clear();
    }
  }

  for (final child in node.children) {
    if (_isInline(child) && _hasInlineContent(child)) {
      if (_inline(child) case final text when text.isNotEmpty) {
        paragraph.add(text);
      }
    } else {
      flush();
      blocks.addAll(_blocks(child));
    }
  }
  flush();
  return blocks;
}

const _inlineTags = {
  '#text', 'a', 'img', 'time', 'span', 'b', 'strong', 'i', 'em', 'code', //
  'small', 'sup', 'sub', 'mark', 'abbr', 'cite', 'q', 'u', 's',
};

bool _isInline(PageNode node) => _inlineTags.contains(node.tag);

bool _hasInlineContent(PageNode node) =>
    node.children.every((c) => _isInline(c) && _hasInlineContent(c));

String _inline(PageNode node) {
  if (node.isImage) {
    return '![${node.alt ?? ''}](${node.src})';
  }
  final text = node.children.isEmpty
      ? switch (node.rich) {
          final rich? => rich.map(_rich).join(),
          null => node.text ?? '',
        }
      : node.children.map(_inline).where((t) => t.isNotEmpty).join(' ');
  return switch (node.href) {
    final href? => '[${text.isEmpty ? href : text}]($href)',
    null => text,
  };
}

String _rich(InlineNode node) {
  final text = node.children.map(_rich).join();
  return switch (node.tag) {
    '#text' => node.text ?? '',
    'br' => '\n',
    'b' => '**$text**',
    'i' => '_${text}_',
    'code' => '`$text`',
    _ => text,
  };
}
