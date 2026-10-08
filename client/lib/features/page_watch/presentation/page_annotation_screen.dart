import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_colors.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/tokens/app_text_styles.dart';
import 'package:paperdoll/core/ui/widgets/app_divider.dart';
import 'package:paperdoll/core/ui/widgets/async_value_view.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/ui/widgets/heading_text.dart';
import 'package:paperdoll/features/page_watch/domain/page_document.dart';
import 'package:paperdoll/features/page_watch/domain/post_matcher.dart';
import 'package:paperdoll/features/page_watch/presentation/providers/page_watch_providers.dart';

/// Lets the user tell which parts of a web page are posts, so that the page
/// can be watched like a feed.
///
/// The user digs into the document tree and repeats two steps: mark a node
/// that holds one post, then label the nodes inside it that hold the post's
/// fields. Siblings that share the shape of the marked node become posts of
/// the same group automatically, with their fields found by the same
/// selectors. Each group results in one [PostMatcher].
///
/// A node labeled as the body of a post is converted as a whole, so the tree
/// stops there and the nodes inside it cannot be labeled.
class const PageAnnotationScreen({required final String file, super.key})
    extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final document = ref.watch(pageDocumentProvider(file));
    if (document case AsyncData(:final value)) {
      return _Annotator(document: value);
    }
    return Scaffold(
      appBar: AppBar(),
      body: AsyncValueView<PageDocument>(
        value: document,
        onRetry: () => ref.invalidate(pageDocumentProvider(file)),
        data: (_) => const SizedBox.shrink(),
      ),
    );
  }
}

/// Posts that share one shape, defined by [matcher]. The user labels fields
/// on [reference], one of the posts, and the field selectors are built
/// relative to it.
class _Group {
  new({required this.number, required this.matcher, required this.reference});

  /// A stable label that also picks the group's color.
  final int number;
  PostMatcher matcher;
  PageNode reference;

  Color get color => _groupColors[(number - 1) % _groupColors.length];
}

const _groupColors = [
  Color(0xFF1565C0),
  Color(0xFF2E7D32),
  Color(0xFFEF6C00),
  Color(0xFF6A1B9A),
  Color(0xFFC62828),
  Color(0xFF00838F),
];

/// Where the annotations of all groups land in the document, by node id.
class _Marks {
  final posts = <int, _Group>{};
  final fields = <int, List<(_Group, PostField)>>{};

  bool isBody(PageNode node) =>
      fields[node.id]?.any((m) => m.$2 == PostField.body) ?? false;
}

class const _Annotator({required final PageDocument document})
    extends StatefulWidget {
  @override
  State<_Annotator> createState() => _AnnotatorState();
}

class _AnnotatorState extends State<_Annotator> {
  late final _expanded = {widget.document.root.id};
  final _groups = <_Group>[];
  var _nextGroupNumber = 1;
  PageNode? _selected;

  /// The group whose fields are being labeled. While it is set, only the
  /// nodes inside its reference post can be selected.
  _Group? _editing;

  PageDocument get _document => widget.document;

  _Marks _computeMarks() {
    final marks = _Marks();
    for (final group in _groups) {
      for (final post in group.matcher.matchPosts(_document)) {
        marks.posts[post.id] = group;
        for (final field in PostField.values) {
          if (group.matcher.matchField(post, field) case final node?) {
            (marks.fields[node.id] ??= []).add((group, field));
          }
        }
      }
    }
    return marks;
  }

  /// Why [node] cannot become a post, or null if it can.
  String? _postRejection(PageNode node, _Marks marks) {
    if (node.parent == null) {
      return 'The whole page cannot be a post.';
    }
    if (node.isText) {
      return 'A text run cannot be a post. Select an element around it.';
    }
    final candidates = PostMatcher.fromNode(
      _document,
      node,
    ).matchPosts(_document);
    for (final MapEntry(key: id, value: group) in marks.posts.entries) {
      final post = _document.nodes[id];
      if (candidates.any(post.overlaps)) {
        return 'This overlaps with the posts of group ${group.number}.';
      }
    }
    return null;
  }

  /// Expands [node] and, unless the subtree is large, all of its descendants
  /// so that the user can see every field candidate at once.
  void _expandPost(PageNode node) {
    final containers = <int>[];
    void visit(PageNode n) {
      if (n.children.isNotEmpty) {
        containers.add(n.id);
        n.children.forEach(visit);
      }
    }

    visit(node);
    _expanded.addAll(containers.length <= 100 ? containers : [node.id]);
  }

  void _markAsPost(PageNode node) {
    final group = _Group(
      number: _nextGroupNumber++,
      matcher: PostMatcher.fromNode(_document, node),
      reference: node,
    );
    setState(() {
      _groups.add(group);
      _editing = group;
      _selected = null;
      _expandPost(node);
    });
  }

  void _editGroup(_Group group, PageNode post) {
    setState(() {
      group.reference = post;
      _editing = group;
      _selected = null;
      _expandPost(post);
    });
  }

  void _removeGroup(_Group group) {
    setState(() {
      _groups.remove(group);
      if (identical(_editing, group)) {
        _editing = null;
      }
      _selected = null;
    });
  }

  /// Why [field] cannot be labeled on [node] in the reference post of
  /// [group], or null if it can.
  String? _fieldRejection(_Group group, PostField field, PageNode node) {
    final matcher = group.matcher;
    if (field == PostField.body) {
      for (final other in PostField.values) {
        if (other == PostField.body) {
          continue;
        }
        if (matcher.matchField(group.reference, other) case final n?
            when node.contains(n)) {
          return 'A body cannot contain the ${other.label.toLowerCase()}.';
        }
      }
    } else if (matcher.matchField(group.reference, PostField.body)
        case final body? when body.contains(node)) {
      return 'This is part of the body.';
    }
    return null;
  }

  void _toggleField(_Group group, PostField field, PageNode node) {
    final current = group.matcher.matchField(group.reference, field);
    setState(() {
      group.matcher = group.matcher.withField(
        field,
        group.reference,
        identical(current, node) ? null : node,
      );
    });
  }

  void _onTapNode(PageNode node) {
    if (_editing case final group? when !group.reference.contains(node)) {
      return;
    }
    setState(() => _selected = identical(_selected, node) ? null : node);
  }

  void _toggleExpanded(PageNode node) {
    setState(() {
      if (!_expanded.remove(node.id)) {
        _expanded.add(node.id);
      }
    });
  }

  void _showPreview() {
    unawaited(
      showModalBottomSheet<void>(
        context: context,
        isScrollControlled: true,
        showDragHandle: true,
        builder: (context) => DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.8,
          maxChildSize: 0.95,
          builder: (context, controller) => _Preview(
            document: _document,
            groups: _groups,
            scrollController: controller,
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final marks = _computeMarks();
    final rows = <PageNode>[];
    void visit(PageNode node) {
      rows.add(node);
      if (_expanded.contains(node.id) && !marks.isBody(node)) {
        node.children.forEach(visit);
      }
    }

    // The root (body) cannot be a post, so its children are the top rows.
    _document.root.children.forEach(visit);

    final editing = _editing;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          _document.title.isEmpty ? _document.url : _document.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          IconButton(
            tooltip: 'Preview posts',
            icon: const Icon(Icons.visibility_outlined),
            onPressed: _groups.isNotEmpty && editing == null
                ? _showPreview
                : null,
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              itemCount: rows.length,
              itemBuilder: (context, index) {
                final node = rows[index];
                return _NodeRow(
                  key: ValueKey(node.id),
                  node: node,
                  marks: marks,
                  isExpanded: _expanded.contains(node.id),
                  isBody: marks.isBody(node),
                  isSelected: identical(node, _selected),
                  isDimmed:
                      editing != null && !editing.reference.contains(node),
                  onTap: () => _onTapNode(node),
                  onToggle: () => _toggleExpanded(node),
                );
              },
            ),
          ),
          const AppDivider(),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.all(spacingMd),
              child: SizedBox(
                width: double.infinity,
                child: editing == null
                    ? _buildBrowsePanel(marks)
                    : _buildEditPanel(editing),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBrowsePanel(_Marks marks) {
    final selected = _selected;
    if (selected == null) {
      final postCount = marks.posts.length;
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          const BodyText(
            'Expand the tree and select a node that holds exactly one post, '
            'such as a card or a list item.',
          ),
          if (_groups.isNotEmpty) ...[
            const Gap(spacingXs),
            CaptionText(
              '${_groups.length} group(s), $postCount post(s) found. '
              'Select a post to edit its group.',
            ),
          ],
        ],
      );
    }
    final group = marks.posts[selected.id];
    final rejection = group == null ? _postRejection(selected, marks) : null;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          selected.label,
          style: _tagStyle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        const Gap(spacingSm),
        if (group != null)
          Row(
            children: [
              Expanded(child: BodyText('A post of group ${group.number}.')),
              TextButton(
                onPressed: () => _removeGroup(group),
                child: const Text('Remove group'),
              ),
              const Gap(spacingSm),
              FilledButton(
                onPressed: () => _editGroup(group, selected),
                child: const Text('Edit fields'),
              ),
            ],
          )
        else if (rejection != null)
          CaptionText(rejection)
        else
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton(
              onPressed: () => _markAsPost(selected),
              child: const Text('Mark as post'),
            ),
          ),
      ],
    );
  }

  Widget _buildEditPanel(_Group group) {
    final selected = _selected;
    final hasLink = group.matcher.fields.containsKey(PostField.link);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          children: [
            _GroupDot(color: group.color),
            const Gap(spacingSm),
            Expanded(
              child: HeadingText(
                'Group ${group.number}: label the fields',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
        const Gap(spacingSm),
        Wrap(
          spacing: spacingXs,
          runSpacing: spacingXs,
          children: [
            for (final field in PostField.values)
              _FieldStatus(
                field: field,
                isSet: group.matcher.fields.containsKey(field),
              ),
          ],
        ),
        const Gap(spacingSm),
        if (selected == null)
          const CaptionText(
            'Select a node inside the highlighted post to label it.',
          )
        else ...[
          Text(
            selected.label,
            style: _tagStyle,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          const Gap(spacingXs),
          if (PostField.values.where((f) => f.accepts(selected)).isEmpty)
            const CaptionText('This node has no value to extract.')
          else ...[
            Wrap(
              spacing: spacingXs,
              runSpacing: spacingXs,
              children: [
                for (final field in PostField.values)
                  if (field.accepts(selected))
                    FilterChip(
                      label: Text(field.label),
                      selected: identical(
                        group.matcher.matchField(group.reference, field),
                        selected,
                      ),
                      onSelected:
                          _fieldRejection(group, field, selected) == null
                          ? (_) => _toggleField(group, field, selected)
                          : null,
                    ),
              ],
            ),
            for (final field in PostField.values)
              if (field.accepts(selected))
                if (_fieldRejection(group, field, selected)
                    case final rejection?) ...[
                  const Gap(spacingXs),
                  CaptionText(rejection),
                ],
          ],
        ],
        const Gap(spacingSm),
        Row(
          children: [
            TextButton(
              onPressed: () => _removeGroup(group),
              child: const Text('Remove group'),
            ),
            Expanded(
              child: hasLink
                  ? const SizedBox.shrink()
                  : const Align(
                      alignment: Alignment.centerRight,
                      child: CaptionText('A link is required.'),
                    ),
            ),
            const Gap(spacingSm),
            FilledButton(
              onPressed: hasLink
                  ? () => setState(() {
                      _editing = null;
                      _selected = null;
                    })
                  : null,
              child: const Text('Done'),
            ),
          ],
        ),
      ],
    );
  }
}

final _tagStyle = textCaption.copyWith(fontFamily: 'monospace');
const double _indent = 14;

/// One node of the document tree, indented by depth. Containers show a
/// chevron that expands or collapses their children.
class const _NodeRow({
  required final PageNode node,
  required final _Marks marks,
  required final bool isExpanded,
  required final bool isBody,
  required final bool isSelected,
  required final bool isDimmed,
  required final VoidCallback onTap,
  required final VoidCallback onToggle,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final postGroup = marks.posts[node.id];
    // The nearest post that contains this node tints the row with its color.
    _Group? enclosingGroup;
    for (PageNode? n = node; n != null; n = n.parent) {
      if (marks.posts[n.id] case final group?) {
        enclosingGroup = group;
        break;
      }
    }
    final background = switch ((isSelected, postGroup, enclosingGroup)) {
      (true, _, _) => colorAccent.withValues(alpha: 0.16),
      (_, final group?, _) => group.color.withValues(alpha: 0.18),
      (_, _, final group?) => group.color.withValues(alpha: 0.06),
      _ => colorBackground,
    };
    final fields = marks.fields[node.id] ?? const [];

    return Opacity(
      opacity: isDimmed ? 0.35 : 1,
      child: Material(
        color: background,
        child: InkWell(
          onTap: onTap,
          child: Padding(
            padding: EdgeInsets.only(
              left: (node.depth - 1) * _indent,
              right: spacingSm,
              top: spacingXs,
              bottom: spacingXs,
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SizedBox(
                  width: 36,
                  child: node.children.isEmpty || isBody
                      ? null
                      : IconButton(
                          visualDensity: VisualDensity.compact,
                          icon: Icon(
                            isExpanded
                                ? Icons.expand_more
                                : Icons.chevron_right,
                          ),
                          onPressed: onToggle,
                        ),
                ),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _NodeContent(
                        node: node,
                        isExpanded: isExpanded && !isBody,
                      ),
                      if (postGroup != null || fields.isNotEmpty) ...[
                        const Gap(spacingXs),
                        Wrap(
                          spacing: spacingXs,
                          runSpacing: spacingXs,
                          children: [
                            if (postGroup != null)
                              _Badge(
                                label: identical(postGroup.reference, node)
                                    ? 'Post · group ${postGroup.number}'
                                    : 'Post · group ${postGroup.number} (auto)',
                                color: postGroup.color,
                                isFilled: true,
                              ),
                            for (final (group, field) in fields)
                              _Badge(label: field.label, color: group.color),
                          ],
                        ),
                      ],
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// A preview of what a node shows on the page.
class const _NodeContent({
  required final PageNode node,
  required final bool isExpanded,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    if (node.isText) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: spacingSm),
        child: BodyText(
          node.text ?? '',
          maxLines: 3,
          overflow: TextOverflow.ellipsis,
        ),
      );
    }
    if (node.isImage) {
      return Row(
        children: [
          _Thumbnail(url: node.src!, size: 56),
          const Gap(spacingSm),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(node.label, style: _tagStyle, maxLines: 1),
                if (node.alt case final alt? when alt.isNotEmpty)
                  CaptionText(
                    alt,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
              ],
            ),
          ),
        ],
      );
    }
    if (node.isEmbed) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: spacingSm),
        child: Row(
          children: [
            const Icon(Icons.smart_display_outlined, color: colorTextMuted),
            const Gap(spacingSm),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(node.label, style: _tagStyle, maxLines: 1),
                  CaptionText(
                    node.src!,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
          ],
        ),
      );
    }
    final text = node.text;
    final href = node.href;
    final datetime = node.datetime;
    final preview = node.children.isNotEmpty && !isExpanded
        ? node.textContent
        : '';
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: spacingSm),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            node.children.isEmpty
                ? node.label
                : '${node.label}  (${node.children.length})',
            style: _tagStyle,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          if (text != null)
            BodyText(text, maxLines: 2, overflow: TextOverflow.ellipsis),
          if (href != null)
            Row(
              children: [
                const Icon(Icons.link, size: 14, color: colorAccent),
                const Gap(spacingXs),
                Expanded(
                  child: CaptionText(
                    href,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
          if (datetime != null) CaptionText('datetime: $datetime'),
          if (preview.isNotEmpty)
            CaptionText(preview, maxLines: 1, overflow: TextOverflow.ellipsis),
        ],
      ),
    );
  }
}

class const _Thumbnail({required final String url, required final double size})
    extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(spacingXs),
      child: Image.network(
        url,
        width: size,
        height: size,
        fit: BoxFit.cover,
        // Some images, such as SVG icons, cannot be decoded.
        errorBuilder: (context, error, stackTrace) => Container(
          width: size,
          height: size,
          color: colorSurface,
          child: const Icon(Icons.image_outlined, color: colorTextMuted),
        ),
      ),
    );
  }
}

class const _Badge({
  required final String label,
  required final Color color,
  final bool isFilled = false,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: spacingSm, vertical: 2),
      decoration: BoxDecoration(
        color: isFilled ? color : null,
        border: Border.all(color: color),
        borderRadius: BorderRadius.circular(spacingSm),
      ),
      child: Text(
        label,
        style: textCaption.copyWith(
          color: isFilled ? colorBackground : color,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class const _GroupDot({required final Color color}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 12,
      height: 12,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}

class const _FieldStatus({
  required final PostField field,
  required final bool isSet,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          isSet ? Icons.check_circle : Icons.radio_button_unchecked,
          size: 16,
          color: isSet ? colorAccent : colorTextMuted,
        ),
        const Gap(2),
        CaptionText(field.label),
        const Gap(spacingSm),
      ],
    );
  }
}

/// The posts extracted with the current matchers, and the matchers as they
/// would be sent to the server.
class const _Preview({
  required final PageDocument document,
  required final List<_Group> groups,
  required final ScrollController scrollController,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return ListView(
      controller: scrollController,
      padding: const EdgeInsets.all(spacingMd),
      children: [
        Row(
          children: [
            const Expanded(child: HeadingText('Extracted posts')),
            TextButton.icon(
              icon: const Icon(Icons.copy, size: 18),
              label: const Text('Copy matchers'),
              onPressed: () => unawaited(_copyMatchers(context)),
            ),
          ],
        ),
        for (final group in groups) ...[
          const Gap(spacingMd),
          Row(
            children: [
              _GroupDot(color: group.color),
              const Gap(spacingSm),
              Expanded(
                child: HeadingText(
                  'Group ${group.number} · '
                  '${group.matcher.matchPosts(document).length} post(s)',
                ),
              ),
            ],
          ),
          const Gap(spacingXs),
          Text(group.matcher.post.join(' > '), style: _tagStyle),
          for (final post in group.matcher.matchPosts(document)) ...[
            const AppDivider(),
            _PreviewPost(matcher: group.matcher, post: post),
          ],
        ],
      ],
    );
  }

  Future<void> _copyMatchers(BuildContext context) async {
    final messenger = ScaffoldMessenger.of(context);
    final payload = {
      'url': document.url,
      'matchers': [for (final group in groups) group.matcher.toJson()],
    };
    await Clipboard.setData(
      ClipboardData(text: const JsonEncoder.withIndent('  ').convert(payload)),
    );
    messenger.showSnackBar(
      const SnackBar(content: Text('Copied the matchers as JSON.')),
    );
  }
}

class const _PreviewPost({
  required final PostMatcher matcher,
  required final PageNode post,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    String? value(PostField field) => switch (matcher.matchField(post, field)) {
      final node? => field.valueOf(node),
      null => null,
    };
    final thumbnail = value(PostField.thumbnail);
    final description = value(PostField.description);
    final body = value(PostField.body);
    final pubDate = value(PostField.pubDate);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: spacingSm),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (thumbnail != null) ...[
            _Thumbnail(url: thumbnail, size: 64),
            const Gap(spacingSm),
          ],
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                HeadingText(
                  value(PostField.title) ?? '(no title)',
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
                if (pubDate != null) CaptionText(pubDate),
                if (description != null)
                  BodyText(
                    description,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                  ),
                if (body != null) ...[
                  const Gap(spacingXs),
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(spacingSm),
                    color: colorSurface,
                    child: Text(
                      body,
                      style: _tagStyle,
                      maxLines: 12,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  const Gap(spacingXs),
                ],
                CaptionText(
                  value(PostField.link) ?? '(no link)',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
