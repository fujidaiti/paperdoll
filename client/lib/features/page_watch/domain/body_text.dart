import 'package:paperdoll/features/page_watch/domain/semantic_tree.dart';

/// Marks the boundary between the texts of two values, which becomes a space
/// unless it is next to a punctuation mark.
const _gap = '\x00';

final _gapBeforePunct = RegExp("$_gap+([,.;:!?)\\]’'])");
final _gapAfterBracket = RegExp('([(\\[])$_gap+');

/// The text of the values in the subtree of [node] as one line, without the
/// alt texts of images and the times (see bodyText in
/// server/feature/feed/post_matcher.go).
String bodyText(SemanticTreeNode node) {
  final texts = <String>[];
  void visit(SemanticTreeNode n) {
    if (!n.isImage && !n.isTime) {
      texts.add(n.val ?? '');
    }
    n.children.forEach(visit);
  }

  visit(node);
  return texts
      .join(_gap)
      .replaceAllMapped(_gapBeforePunct, (m) => m[1]!)
      .replaceAllMapped(_gapAfterBracket, (m) => m[1]!)
      .replaceAll(_gap, ' ')
      .split(RegExp(r'\s+'))
      .where((w) => w.isNotEmpty)
      .join(' ');
}
