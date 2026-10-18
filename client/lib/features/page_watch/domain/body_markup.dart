import 'package:paperdoll/features/page_watch/domain/semantic_tree.dart';

/// Converts the values in the subtree of [node] into the marked-up text of a
/// post body, one block per value, separated by a blank line:
///
/// - `# heading` (one `#` per level, h1 to h6)
/// - `![alt](url)` for an image
/// - `[text](url)` for a text with a link
/// - the plain text of other values
String bodyMarkup(SemanticTreeNode node) => _blocks(node).join('\n\n');

List<String> _blocks(SemanticTreeNode node) {
  if (!node.isValue) {
    return [for (final child in node.children) ..._blocks(child)];
  }
  final val = node.val ?? '';
  if (node.isImage) {
    return ['![$val](${node.link})'];
  }
  final text = switch (node.link) {
    final link? => '[${val.isEmpty ? link : val}]($link)',
    null => val,
  };
  if (text.isEmpty) {
    return const [];
  }
  return [
    switch (node.headingLevel) {
      final level? => '${'#' * level} $text',
      null => text,
    },
  ];
}
