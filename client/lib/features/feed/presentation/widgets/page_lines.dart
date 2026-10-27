import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_colors.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/tokens/app_text_styles.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/features/feed/domain/post_matcher.dart';
import 'package:paperdoll/features/feed/domain/semantic_tree.dart';

// Widgets that show a page as its content, shared by the screen that selects
// posts and the screen that labels the fields of a post.

/// Splits the content of [nodes] into lines: one line for each value, and
/// one for each container with a link, such as a card, which shows its target
/// before its content.
List<SemanticTreeNode> layOutLines(Iterable<SemanticTreeNode> nodes) {
  final lines = <SemanticTreeNode>[];
  void visit(SemanticTreeNode node) {
    if (node.isValue || node.link != null) {
      lines.add(node);
    }
    node.children.forEach(visit);
  }

  nodes.forEach(visit);
  return lines;
}

/// The node of the first line of [node].
SemanticTreeNode firstLeaf(SemanticTreeNode node) {
  var n = node;
  while (n.children.isNotEmpty && n.link == null) {
    n = n.children.first;
  }
  return n;
}

/// A short description of what [node] holds.
String describeNode(SemanticTreeNode node) {
  if (node.isImage) {
    return 'An image';
  }
  if (node.isValue) {
    return node.link != null ? 'A link' : 'A text';
  }
  var texts = 0;
  var links = 0;
  var images = 0;
  void visit(SemanticTreeNode n) {
    if (n.pageLink != null) {
      links++;
    }
    if (n.isImage) {
      images++;
    } else if (n.isValue) {
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
  ];
  return counts.join(', ');
}

/// An annotated area of the page, drawn as a translucent tint of `color`
/// over its lines.
///
/// When `linkOnly` is true and `node` is a link around other content, only
/// the line that shows the link's target is tinted, because the area is about
/// the link rather than its content.
typedef PageArea = ({SemanticTreeNode node, Color color, bool linkOnly});

bool _isLinkOnly(PageArea area) =>
    area.linkOnly && area.node.link != null && !area.node.isValue;

SemanticTreeNode _lastLeaf(SemanticTreeNode node) {
  var n = node;
  while (n.children.isNotEmpty) {
    n = n.children.last;
  }
  return n;
}

/// One line of the page. Each of [areas] that covers the line tints it, so
/// that nested areas look darker.
///
/// A line is at least as tall as a touch target. A long press on a link opens
/// it in the in-app browser.
class const PageLine({
  required final SemanticTreeNode node,
  required final List<PageArea> areas,
  required final void Function(SemanticTreeNode node) onTap,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    // Outer areas first, so that inner tints are drawn over outer ones.
    final boxes =
        areas
            .where(
              (a) => _isLinkOnly(a)
                  ? identical(node, a.node)
                  : a.node.contains(node),
            )
            .toList()
          ..sort((a, b) => a.node.depth.compareTo(b.node.depth));
    // The tints are painted over the content rather than wrapped around it,
    // so that annotating does not change the layout, and so that images do
    // not hide them. A tint leaves a thin gap at the ends of its area to keep
    // adjacent areas apart.
    final content = Stack(
      children: [
        _frameLine(node, _Content(node: node)),
        for (final area in boxes)
          Positioned.fill(
            top: identical(firstLeaf(area.node), node) ? 2 : 0,
            bottom: _isLinkOnly(area) || identical(_lastLeaf(area.node), node)
                ? 2
                : 0,
            child: IgnorePointer(
              child: ColoredBox(
                color: area.color.withValues(alpha: _tintAlpha),
              ),
            ),
          ),
      ],
    );
    return InkWell(
      onTap: () {
        onTap(node);
        // A tap can show a sheet below the page, which then covers the line
        // if it was near the bottom, so the line is scrolled back into view.
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (context.mounted) {
            Scrollable.ensureVisible(
              context,
              duration: const Duration(milliseconds: 200),
              alignmentPolicy: ScrollPositionAlignmentPolicy.keepVisibleAtEnd,
            );
          }
        });
      },
      onLongPress: switch (node.pageLink) {
        final link? => () => unawaited(openInAppLink(context, link)),
        null => null,
      },
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: spacingMd),
        child: content,
      ),
    );
  }
}

const _tintAlpha = 0.14;

/// [content] laid out as a line that starts with [first].
///
/// Lines touch each other so that the tint of an area is continuous; the
/// space between lines is inside the innermost tint.
Widget _frameLine(SemanticTreeNode first, Widget content) => Container(
  constraints: const BoxConstraints(minHeight: kMinInteractiveDimension),
  alignment: Alignment.centerLeft,
  padding: EdgeInsets.only(
    top: first.headingLevel != null ? spacingMd : spacingXs,
    bottom: spacingXs,
  ),
  child: content,
);

/// The text of [node], styled the way the page would show it. The text of a
/// link without text, such as a card, is its target.
TextSpan _textSpan(SemanticTreeNode node) {
  var style = switch (node.headingLevel) {
    1 => textTitle.copyWith(fontSize: 26),
    2 => textTitle,
    3 => textHeading.copyWith(fontWeight: FontWeight.bold),
    final _? => textBody.copyWith(fontWeight: FontWeight.bold),
    null => node.isTime ? textCaption : textBody,
  };
  var text = node.isValue ? node.val : null;
  if (node.link case final link?) {
    style = style.copyWith(decoration: TextDecoration.underline);
    text ??= link;
  }
  return TextSpan(text: text ?? '', style: style);
}

/// A value, or the link line of a container, shown as it would appear on the
/// page.
class const _Content({required final SemanticTreeNode node})
    extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    if (node.isImage) {
      const maxHeight = 200.0;
      // The image is decoded at the size it is shown rather than at its full
      // size, which saves memory on pages with many large images.
      final ratio = MediaQuery.devicePixelRatioOf(context);
      return ConstrainedBox(
        constraints: const BoxConstraints(maxHeight: maxHeight),
        child: Image(
          image: ResizeImage(
            NetworkImage(node.link!),
            width: (MediaQuery.sizeOf(context).width * ratio).round(),
            height: (maxHeight * ratio).round(),
            policy: ResizeImagePolicy.fit,
          ),
          fit: BoxFit.contain,
          alignment: Alignment.centerLeft,
          // Some images, such as SVG icons, cannot be decoded.
          errorBuilder: (context, error, stackTrace) => Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.image_outlined, color: colorTextMuted),
              if (node.val case final alt? when alt.isNotEmpty) ...[
                const Gap(spacingXs),
                Flexible(child: CaptionText(alt)),
              ],
            ],
          ),
        ),
      );
    }
    return Text.rich(_textSpan(node));
  }
}

/// The color of the tint over the content labeled as [field].
Color fieldColor(PostField field) => switch (field) {
  PostField.title => const Color(0xFF1565C0),
  PostField.body => const Color(0xFF6A1B9A),
  PostField.link => const Color(0xFFEF6C00),
  PostField.thumbnail => const Color(0xFFD81B60),
  PostField.pubDate => const Color(0xFF2E7D32),
};

/// The tint of the selection. It is gray so that it differs from the colors
/// of fields.
PageArea selectionArea(SemanticTreeNode node, {bool linkOnly = false}) =>
    (node: node, color: colorText, linkOnly: linkOnly);

/// A sheet at the bottom of a screen that shows the selection and what can be
/// done with it.
///
/// It is laid out below the page instead of being set as
/// Scaffold.bottomSheet, so that it does not cover the end of the page, which
/// can still be scrolled and tapped while the sheet is shown.
class const SelectionSheet({required final Widget child, super.key})
    extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    // Material 3 bottom sheets have no shadow, and an elevation shadow falls
    // below the sheet, so the shadow is cast upwards here to show the edge of
    // the sheet against the page.
    return DecoratedBox(
      decoration: const BoxDecoration(
        // The corner radius of Material 3 bottom sheets.
        borderRadius: BorderRadius.vertical(top: Radius.circular(28)),
        boxShadow: [
          BoxShadow(
            color: Color(0x33000000),
            blurRadius: 16,
            offset: Offset(0, -4),
          ),
        ],
      ),
      child: BottomSheet(
        enableDrag: false,
        onClosing: () {},
        builder: (context) => SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.all(spacingMd),
            child: child,
          ),
        ),
      ),
    );
  }
}

/// A slider that moves the selection through [levels], the elements from the
/// narrowest to the widest that can be selected. [selected] is one of
/// [levels].
class const SelectionSlider({
  required final SemanticTreeNode selected,
  required final List<SemanticTreeNode> levels,
  required final void Function(SemanticTreeNode node) onSelect,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final canMove = levels.length > 1;
    return Row(
      children: [
        const Icon(Icons.unfold_less, color: colorTextMuted),
        Expanded(
          child: Slider(
            value: canMove ? levels.indexOf(selected).toDouble() : 0,
            max: canMove ? levels.length - 1 : 1,
            divisions: canMove ? levels.length - 1 : null,
            onChanged: canMove
                ? (value) => onSelect(levels[value.round()])
                : null,
          ),
        ),
        const Icon(Icons.unfold_more, color: colorTextMuted),
      ],
    );
  }
}

/// [from] and the elements around it, from the narrowest to the widest, up
/// to [top], or up to the outermost element below the root if [top] is null.
///
/// An element around a single element, without a link line of its own, looks
/// the same as that element on the page, so it is left out. Otherwise moving
/// the slider would change nothing that the user can see.
List<SemanticTreeNode> selectionLevels(
  SemanticTreeNode from, {
  SemanticTreeNode? top,
}) {
  final levels = [from];
  var n = from;
  while (!identical(n, top) && n.parent?.parent != null) {
    n = n.parent!;
    if (n.children.length > 1 || n.link != null) {
      levels.add(n);
    }
  }
  return levels;
}
