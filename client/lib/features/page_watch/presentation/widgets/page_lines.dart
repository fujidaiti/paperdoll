import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_colors.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/tokens/app_text_styles.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/features/page_watch/domain/page_document.dart';
import 'package:paperdoll/features/page_watch/domain/post_matcher.dart';

// Widgets that show a page as its content, shared by the screen that selects
// posts and the screen that labels the fields of a post.

/// Elements that flow within a line of text rather than starting a new one.
const _inlineTags = {
  '#text', 'a', 'time', 'span', 'b', 'strong', 'i', 'em', 'code', 'small', //
  'abbr', 'cite', 'q', 'mark', 'sup', 'sub', 'u', 's',
};

/// Splits the content of [nodes] into lines. A line is a node without
/// children, or a run of such sibling nodes inside mixed content, such as the
/// text and links of one paragraph.
///
/// A link around other content, such as a card, has no text of its own, so
/// it gets a line that shows its target before its content.
List<List<PageNode>> layOutLines(Iterable<PageNode> nodes) {
  final lines = <List<PageNode>>[];
  void visit(PageNode node) {
    if (node.children.isEmpty || node.href != null) {
      lines.add([node]);
    }
    if (node.children.isEmpty) {
      return;
    }
    final isMixed = node.children.any((c) => c.isText);
    List<PageNode>? run;
    for (final child in node.children) {
      if (isMixed &&
          child.children.isEmpty &&
          _inlineTags.contains(child.tag)) {
        if (run == null) {
          lines.add(run = [child]);
        } else {
          run.add(child);
        }
      } else {
        run = null;
        visit(child);
      }
    }
  }

  nodes.forEach(visit);
  return lines;
}

/// The node of the first line of [node].
PageNode firstLeaf(PageNode node) {
  var n = node;
  while (n.children.isNotEmpty && n.href == null) {
    n = n.children.first;
  }
  return n;
}

/// A short description of what [node] holds, without tag names.
String describeNode(PageNode node) {
  if (node.isImage) {
    return 'An image';
  }
  if (node.isEmbed) {
    return 'Embedded content';
  }
  if (node.children.isEmpty) {
    return node.href != null ? 'A link' : 'A text';
  }
  var texts = 0;
  var links = 0;
  var images = 0;
  var embeds = 0;
  void visit(PageNode n) {
    if (n.href != null) {
      links++;
    }
    if (n.isImage) {
      images++;
    } else if (n.isEmbed) {
      embeds++;
    } else if (n.children.isEmpty) {
      texts++;
    }
    n.children.forEach(visit);
  }

  visit(node);
  String count(int n, String noun) => '$n $noun${n == 1 ? '' : 's'}';
  final counts = [
    if (texts > 0) count(texts, 'text'),
    if (links > 0) count(links, 'link'),
    if (images > 0) count(images, 'image'),
    if (embeds > 0) count(embeds, 'embed'),
  ];
  return 'An area with ${counts.join(', ')}';
}

/// An annotated area of the page, drawn as a translucent tint of `color`
/// over its lines.
///
/// When `linkOnly` is true and `node` is a link around other content, only
/// the line that shows the link's target is tinted, because the area is about
/// the link rather than its content.
typedef PageArea = ({PageNode node, Color color, bool linkOnly});

bool _isLinkOnly(PageArea area) =>
    area.linkOnly && area.node.href != null && area.node.children.isNotEmpty;

PageNode _lastLeaf(PageNode node) {
  var n = node;
  while (n.children.isNotEmpty) {
    n = n.children.last;
  }
  return n;
}

/// One line of the page. Each of [areas] that covers the line tints it, so
/// that nested areas look darker. An area that covers only a piece of the
/// line, such as a link in a paragraph, tints the background of that piece
/// instead.
///
/// A line is at least as tall as a touch target, and a tap anywhere on it
/// reports the whole line, because the pieces of a line are too small to tap
/// one by one. A long press on a link opens it in the in-app browser.
class const PageLine({
  required final List<PageNode> nodes,
  required final List<PageArea> areas,
  required final void Function(List<PageNode> line) onTap,
  super.key,
}) extends StatefulWidget {
  @override
  State<PageLine> createState() => _PageLineState();
}

class _PageLineState extends State<PageLine> {
  /// Tells which piece of a line with several pieces is under the finger.
  ///
  /// A span takes only one recognizer, so a long press is handled by the
  /// whole line, which needs to know the piece under the finger. The tap
  /// recognizer reports it on tap down, which comes before a long press. A
  /// tap on a piece wins over the line's own tap, so it reports the line too.
  late final _recognizers = [
    if (widget.nodes.length > 1)
      for (final node in widget.nodes)
        TapGestureRecognizer()
          ..onTapDown = ((_) => _pressed = node)
          ..onTap = () => widget.onTap(widget.nodes),
  ];

  /// The piece of the line under the finger, or null when the line has
  /// several pieces and the finger is on none of them.
  PageNode? _pressed;

  @override
  void dispose() {
    for (final recognizer in _recognizers) {
      recognizer.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final nodes = widget.nodes;
    final first = nodes.first;
    // Outer areas first, so that inner tints are drawn over outer ones.
    final boxes =
        widget.areas
            .where(
              (a) => _isLinkOnly(a)
                  ? identical(first, a.node)
                  : nodes.every(a.node.contains),
            )
            .toList()
          ..sort((a, b) => a.node.depth.compareTo(b.node.depth));

    Widget content;
    if (nodes.length == 1) {
      content = _Content(node: first);
    } else {
      content = Text.rich(
        TextSpan(
          children: [
            for (final (i, node) in nodes.indexed) ...[
              if (i > 0) const TextSpan(text: ' '),
              _textSpan(
                node,
                recognizer: _recognizers[i],
                highlight: widget.areas
                    .where(
                      (a) =>
                          !_isLinkOnly(a) &&
                          a.node.contains(node) &&
                          !boxes.contains(a),
                    )
                    .map((a) => a.color.withValues(alpha: _tintAlpha))
                    .fold<Color?>(
                      null,
                      (below, tint) =>
                          below == null ? tint : Color.alphaBlend(tint, below),
                    ),
              ),
            ],
          ],
        ),
      );
    }
    content = _frameLine(first, content);
    // The tints are painted behind the content rather than wrapped around
    // it, so that annotating does not change the layout. A tint leaves a thin
    // gap at the ends of its area to keep adjacent areas apart.
    content = Stack(
      children: [
        for (final area in boxes)
          Positioned.fill(
            top: nodes.contains(firstLeaf(area.node)) ? 2 : 0,
            bottom: _isLinkOnly(area) || nodes.contains(_lastLeaf(area.node))
                ? 2
                : 0,
            child: ColoredBox(color: area.color.withValues(alpha: _tintAlpha)),
          ),
        content,
      ],
    );
    return Listener(
      onPointerDown: (_) => _pressed = nodes.length == 1 ? first : null,
      child: InkWell(
        onTap: () => widget.onTap(nodes),
        onLongPress: nodes.any((n) => n.href != null)
            ? () {
                if (_pressed?.href case final href?) {
                  unawaited(openInAppLink(context, href));
                }
              }
            : null,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: spacingMd),
          child: content,
        ),
      ),
    );
  }
}

const _tintAlpha = 0.14;

/// [content] laid out as a line that starts with [first].
///
/// Lines touch each other so that the tint of an area is continuous; the
/// space between lines is inside the innermost tint.
Widget _frameLine(PageNode first, Widget content) => Container(
  constraints: const BoxConstraints(minHeight: kMinInteractiveDimension),
  alignment: Alignment.centerLeft,
  padding: EdgeInsets.only(
    top: _headingLevel(first) != null ? spacingMd : spacingXs,
    bottom: spacingXs,
  ),
  child: content,
);

/// A node with children shown as one block: its own lines and, nested in
/// it, a block for each child with children. A tap anywhere on the block
/// that is not on a nested block reports [node], so the block is the unit
/// that is selected. The lines themselves cannot be tapped. A block whose
/// node is not [tappable] cannot be tapped either, so a tap on it reaches
/// the block around it, but its nested blocks can be tapped.
///
/// Each of [areas] whose node is [node] tints the block, so that nested
/// areas look darker. A long press on a link opens it in the in-app browser.
class const PageBlock({
  required final PageNode node,
  required final List<PageArea> areas,
  required final void Function(PageNode node) onTap,
  required final bool Function(PageNode node) tappable,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    // The same lines as [layOutLines], without descending into children.
    final items = <Object>[
      // A link around other content shows its target first.
      if (node.href != null) [node],
    ];
    final isMixed = node.children.any((c) => c.isText);
    List<PageNode>? run;
    for (final child in node.children) {
      if (child.children.isNotEmpty) {
        run = null;
        items.add(child);
      } else if (isMixed && _inlineTags.contains(child.tag)) {
        if (run == null) {
          items.add(run = [child]);
        } else {
          run.add(child);
        }
      } else {
        run = null;
        items.add([child]);
      }
    }
    Widget content = Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final item in items)
          if (item case final PageNode child)
            PageBlock(
              key: ValueKey(child.id),
              node: child,
              areas: areas,
              onTap: onTap,
              tappable: tappable,
            )
          else if (item case final List<PageNode> line)
            _BlockLine(key: ValueKey(line.first.id), nodes: line),
      ],
    );
    // A tint leaves a thin gap at the ends of its block to keep adjacent
    // areas apart.
    content = Stack(
      children: [
        for (final area in areas)
          if (identical(area.node, node))
            Positioned.fill(
              top: 2,
              bottom: 2,
              child: ColoredBox(
                color: area.color.withValues(alpha: _tintAlpha),
              ),
            ),
        content,
      ],
    );
    if (tappable(node)) {
      content = InkWell(onTap: () => onTap(node), child: content);
    }
    return content;
  }
}

/// A line inside a [PageBlock]. It does not take taps, so that a tap reaches
/// the block, but a long press on a link opens it.
class const _BlockLine({required final List<PageNode> nodes, super.key})
    extends StatefulWidget {
  @override
  State<_BlockLine> createState() => _BlockLineState();
}

class _BlockLineState extends State<_BlockLine> {
  /// The long press recognizers of the links of a line with several pieces.
  /// A quick tap is not a long press, so it still reaches the block.
  late final _recognizers = [
    for (final node in widget.nodes)
      if (widget.nodes.length > 1 && node.href != null)
        LongPressGestureRecognizer()
          ..onLongPress = () => unawaited(openInAppLink(context, node.href!))
      else
        null,
  ];

  @override
  void dispose() {
    for (final recognizer in _recognizers) {
      recognizer?.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final nodes = widget.nodes;
    final first = nodes.first;
    if (nodes.length > 1) {
      return _frameLine(
        first,
        Text.rich(
          TextSpan(
            children: [
              for (final (i, node) in nodes.indexed) ...[
                if (i > 0) const TextSpan(text: ' '),
                _textSpan(node, recognizer: _recognizers[i]),
              ],
            ],
          ),
        ),
      );
    }
    return GestureDetector(
      onLongPress: first.href == null
          ? null
          : () => unawaited(openInAppLink(context, first.href!)),
      child: _frameLine(first, _Content(node: first)),
    );
  }
}

/// The level of the heading that [node] belongs to, if any.
int? _headingLevel(PageNode node) {
  for (PageNode? n = node; n != null; n = n.parent) {
    if (n.tag case 'h1' || 'h2' || 'h3' || 'h4' || 'h5' || 'h6') {
      return int.parse(n.tag[1]);
    }
  }
  return null;
}

bool _isInBlockquote(PageNode node) {
  for (PageNode? n = node; n != null; n = n.parent) {
    if (n.tag == 'blockquote') {
      return true;
    }
  }
  return false;
}

/// The text of [node], styled the way the page would show it. The
/// [recognizer] is set on every leaf span because a tap only reaches the
/// span under it. A list item starts with a bullet unless [bullet] is false.
TextSpan _textSpan(
  PageNode node, {
  GestureRecognizer? recognizer,
  Color? highlight,
  bool bullet = true,
}) {
  var style = switch (_headingLevel(node)) {
    1 => textTitle.copyWith(fontSize: 26),
    2 => textTitle,
    3 => textHeading.copyWith(fontWeight: FontWeight.bold),
    final _? => textBody.copyWith(fontWeight: FontWeight.bold),
    null => textBody,
  };
  if (node.tag == 'time' || node.tag == 'small') {
    style = textCaption;
  }
  if (node.tag case 'b' || 'strong') {
    style = style.copyWith(fontWeight: FontWeight.bold);
  }
  if (node.tag case 'i' || 'em' || 'cite' || 'q') {
    style = style.copyWith(fontStyle: FontStyle.italic);
  }
  if (_isInBlockquote(node)) {
    style = style.copyWith(fontStyle: FontStyle.italic, color: colorTextMuted);
  }
  if (node.tag case 'pre' || 'code') {
    style = style.copyWith(fontFamily: 'monospace');
  }
  if (highlight != null) {
    style = style.copyWith(backgroundColor: highlight);
  }
  // The text of a link without text is its target.
  var text = node.text;
  if (node.href case final href?) {
    style = style.copyWith(decoration: TextDecoration.underline);
    text ??= href;
  }
  final isListItem =
      bullet &&
      (node.tag == 'li' ||
          (node.parent?.tag == 'li' &&
              identical(node.parent!.children.first, node)));

  TextSpan rich(InlineNode inline) => switch (inline.tag) {
    '#text' => TextSpan(text: inline.text, recognizer: recognizer),
    'br' => TextSpan(text: '\n', recognizer: recognizer),
    final tag => TextSpan(
      style: switch (tag) {
        'b' => const TextStyle(fontWeight: FontWeight.bold),
        'i' => const TextStyle(fontStyle: FontStyle.italic),
        'code' => const TextStyle(fontFamily: 'monospace'),
        _ => null,
      },
      children: [for (final child in inline.children) rich(child)],
    ),
  };

  return TextSpan(
    style: style,
    children: [
      if (isListItem) TextSpan(text: '•  ', recognizer: recognizer),
      if (node.rich case final inlines?)
        for (final inline in inlines) rich(inline)
      else
        TextSpan(text: text ?? '', recognizer: recognizer),
    ],
  );
}

/// A node without children shown as it would appear on the page.
class const _Content({required final PageNode node}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    if (node.isImage) {
      return ConstrainedBox(
        constraints: const BoxConstraints(maxHeight: 200),
        child: Image.network(
          node.src!,
          fit: BoxFit.contain,
          alignment: Alignment.centerLeft,
          // Some images, such as SVG icons, cannot be decoded.
          errorBuilder: (context, error, stackTrace) => Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.image_outlined, color: colorTextMuted),
              if (node.alt case final alt? when alt.isNotEmpty) ...[
                const Gap(spacingXs),
                Flexible(child: CaptionText(alt)),
              ],
            ],
          ),
        ),
      );
    }
    if (node.isEmbed) {
      return Container(
        width: double.infinity,
        padding: const EdgeInsets.all(spacingMd),
        color: colorSurface,
        child: Row(
          children: [
            const Icon(Icons.smart_display_outlined, color: colorTextMuted),
            const Gap(spacingSm),
            Expanded(
              child: CaptionText(
                Uri.tryParse(node.src!)?.host ?? node.src!,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
      );
    }
    if (node.tag == 'pre') {
      return Container(
        width: double.infinity,
        padding: const EdgeInsets.all(spacingSm),
        color: colorSurface,
        child: Text.rich(_textSpan(node)),
      );
    }
    return Text.rich(_textSpan(node));
  }
}

class const GroupDot({required final Color color, super.key})
    extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 12,
      height: 12,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}

/// The color of the tint over the content labeled as [field].
Color fieldColor(PostField field) => switch (field) {
  PostField.title => const Color(0xFF1565C0),
  PostField.description => const Color(0xFF00897B),
  PostField.body => const Color(0xFF6A1B9A),
  PostField.link => const Color(0xFFEF6C00),
  PostField.thumbnail => const Color(0xFFD81B60),
  PostField.pubDate => const Color(0xFF2E7D32),
};

/// The tint of the selection. It is gray so that it differs from the colors
/// of groups and fields.
PageArea selectionArea(PageNode node, {bool linkOnly = false}) =>
    (node: node, color: colorText, linkOnly: linkOnly);

/// What the selection holds, with buttons to widen it to the element around
/// it or narrow it back down. A null callback disables its button.
class const SelectionHeader({
  required final PageNode selected,
  required final VoidCallback? onNarrow,
  required final VoidCallback? onWiden,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              BodyText(
                describeNode(selected),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              if (selected.textContent case final text when text.isNotEmpty)
                CaptionText(text, maxLines: 1, overflow: TextOverflow.ellipsis),
            ],
          ),
        ),
        IconButton(
          tooltip: 'Narrow the selection',
          icon: const Icon(Icons.unfold_less),
          onPressed: onNarrow,
        ),
        IconButton(
          tooltip: 'Widen the selection',
          icon: const Icon(Icons.unfold_more),
          onPressed: onWiden,
        ),
      ],
    );
  }
}

/// Shows the line of [pieces], which are too small to tap on the page,
/// zoomed in a sheet, and returns the piece the user picks, or null if the
/// user closes the sheet.
Future<PageNode?> pickLinePiece(BuildContext context, List<PageNode> pieces) {
  return showModalBottomSheet<PageNode>(
    context: context,
    showDragHandle: true,
    builder: (context) => SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(spacingMd, 0, spacingMd, spacingMd),
        child: SizedBox(
          width: double.infinity,
          child: _ZoomedLine(
            pieces: pieces,
            onPick: (piece) => Navigator.pop(context, piece),
          ),
        ),
      ),
    ),
  );
}

/// A line shown larger, in the same flow as on the page, so that each piece
/// is a touch target: the lines are at least [kMinInteractiveDimension] tall,
/// and a piece narrower than that is widened.
class const _ZoomedLine({
  required final List<PageNode> pieces,
  required final void Function(PageNode piece) onPick,
}) extends StatefulWidget {
  @override
  State<_ZoomedLine> createState() => _ZoomedLineState();
}

class _ZoomedLineState extends State<_ZoomedLine> {
  static const _scaler = TextScaler.linear(1.5);

  late final _recognizers = [
    for (final piece in widget.pieces)
      TapGestureRecognizer()..onTap = () => widget.onPick(piece),
  ];

  @override
  void dispose() {
    for (final recognizer in _recognizers) {
      recognizer.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Each piece has a background so that the user can tell where it ends.
    final background = colorText.withValues(alpha: 0.06);
    return Text.rich(
      TextSpan(
        children: [
          for (final (i, piece) in widget.pieces.indexed) ...[
            if (i > 0) const TextSpan(text: ' '),
            switch (_textSpan(
              piece,
              recognizer: _recognizers[i],
              highlight: background,
              bullet: false,
            )) {
              final span when _isNarrow(span) => WidgetSpan(
                alignment: PlaceholderAlignment.middle,
                // The text scaler scales the widget too, so its size is set
                // before scaling.
                child: GestureDetector(
                  onTap: () => widget.onPick(piece),
                  child: ConstrainedBox(
                    constraints: BoxConstraints.tight(
                      Size.square(kMinInteractiveDimension / _scaler.scale(1)),
                    ),
                    child: Center(child: Text.rich(span)),
                  ),
                ),
              ),
              final span => span,
            },
          ],
        ],
      ),
      textScaler: _scaler,
      // The strut is scaled with the text, which makes lines 48 px tall.
      strutStyle: StrutStyle(
        fontSize: textBody.fontSize,
        height: kMinInteractiveDimension / _scaler.scale(textBody.fontSize!),
      ),
    );
  }

  bool _isNarrow(TextSpan span) {
    final painter = TextPainter(
      text: span,
      textScaler: _scaler,
      textDirection: TextDirection.ltr,
    )..layout();
    final width = painter.width;
    painter.dispose();
    return width < kMinInteractiveDimension;
  }
}

/// The child of [selected] on the way down to [tapped], if [selected] was
/// widened from [tapped].
PageNode? narrowTowards(PageNode selected, PageNode? tapped) {
  if (tapped == null ||
      identical(tapped, selected) ||
      !selected.contains(tapped)) {
    return null;
  }
  var n = tapped;
  while (!identical(n.parent, selected)) {
    n = n.parent!;
  }
  return n;
}
