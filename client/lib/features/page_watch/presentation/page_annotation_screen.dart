import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/router/routes.dart';
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
import 'package:paperdoll/features/page_watch/presentation/post_fields_screen.dart';
import 'package:paperdoll/features/page_watch/presentation/providers/page_watch_providers.dart';
import 'package:paperdoll/features/page_watch/presentation/widgets/page_lines.dart';

/// Lets the user tell which parts of a web page are posts, so that the page
/// can be watched like a feed.
///
/// The page is shown as its content, line by line. The user repeats two
/// steps: select the area that holds one post, then label the content inside
/// it as the post's fields on [PostFieldsScreen]. An area is selected by
/// tapping a piece of content, which selects the element around it, and
/// widening the selection to the elements around that one. Areas that share
/// the shape of the selected post become posts of the same group
/// automatically, with their fields found by the same selectors. Each group
/// results in one [PostMatcher].
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

/// Posts that share one shape, defined by [matcher].
class _Group {
  new({required this.number, required this.matcher});

  /// A stable label that also picks the group's color.
  final int number;
  PostMatcher matcher;

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

/// Whether [node] or the content inside it has a link.
bool _hasLink(PageNode node) =>
    node.href != null || node.children.any(_hasLink);

/// The posts of all groups in the document, by node id.
class _Marks {
  final posts = <int, _Group>{};

  /// The post that contains [node], if any.
  PageNode? enclosingPost(PageNode node) {
    for (PageNode? n = node; n != null; n = n.parent) {
      if (posts.containsKey(n.id)) {
        return n;
      }
    }
    return null;
  }
}

class const _Annotator({required final PageDocument document})
    extends StatefulWidget {
  @override
  State<_Annotator> createState() => _AnnotatorState();
}

class _AnnotatorState extends State<_Annotator> {
  final _groups = <_Group>[];
  var _nextGroupNumber = 1;

  /// The element shown on the selection sheet, while the sheet is open.
  PageNode? _selected;

  PageDocument get _document => widget.document;

  _Marks _computeMarks() {
    final marks = _Marks();
    for (final group in _groups) {
      for (final post in group.matcher.matchPosts(_document)) {
        marks.posts[post.id] = group;
      }
    }
    return marks;
  }

  /// Why [node] cannot become a post, or null if it can.
  String? _postRejection(PageNode node, _Marks marks) {
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

  /// Opens the screen that labels the fields of [post], and returns the
  /// updated matcher, or null if the user left without finishing.
  Future<PostMatcher?> _labelFields({
    required PostMatcher matcher,
    required PageNode post,
    required int groupNumber,
  }) => context.pushNamed<PostMatcher>(
    routePostFieldsName,
    pathParameters: GoRouterState.of(context).pathParameters,
    extra: (
      document: _document,
      matcher: matcher,
      post: post,
      groupNumber: groupNumber,
    ),
  );

  Future<void> _markAsPost(PageNode node) async {
    final number = _nextGroupNumber;
    final matcher = await _labelFields(
      matcher: PostMatcher.fromNode(_document, node),
      post: node,
      groupNumber: number,
    );
    if (matcher == null || !mounted) {
      return;
    }
    setState(() {
      _nextGroupNumber++;
      _groups.add(_Group(number: number, matcher: matcher));
    });
  }

  Future<void> _editGroup(_Group group, PageNode post) async {
    final matcher = await _labelFields(
      matcher: group.matcher,
      post: post,
      groupNumber: group.number,
    );
    if (matcher == null || !mounted) {
      return;
    }
    setState(() {
      group.matcher = matcher;
    });
  }

  void _removeGroup(_Group group) {
    setState(() => _groups.remove(group));
  }

  Future<void> _onTapBlock(PageNode node, _Marks marks) async {
    // A tap inside a post selects the post.
    final target = marks.enclosingPost(node) ?? node;
    setState(() => _selected = target);
    await showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      // Keeps the selected area visible above the sheet.
      barrierColor: Colors.transparent,
      builder: (context) => _SelectionSheet(
        tapped: target,
        marks: marks,
        rejectionOf: (node) => _postRejection(node, marks),
        onSelect: (node) => setState(() => _selected = node),
        onMarkAsPost: (node) => unawaited(_markAsPost(node)),
        onEditGroup: (group, post) => unawaited(_editGroup(group, post)),
        onRemoveGroup: _removeGroup,
      ),
    );
    if (mounted) {
      setState(() => _selected = null);
    }
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
    final selected = _selected;
    final areas = <PageArea>[
      for (final MapEntry(key: id, value: group) in marks.posts.entries)
        (node: _document.nodes[id], color: group.color, linkOnly: false),
      if (selected != null) selectionArea(selected),
    ];

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
            onPressed: _groups.isNotEmpty ? _showPreview : null,
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(
                horizontal: spacingMd,
                vertical: spacingSm,
              ),
              child: PageBlock(
                node: _document.root,
                areas: areas,
                onTap: (node) => unawaited(_onTapBlock(node, marks)),
                // A post needs a link and a text for its title, and the whole
                // page cannot be a post.
                tappable: (node) =>
                    node.parent != null &&
                    _hasLink(node) &&
                    node.textContent.trim().isNotEmpty,
              ),
            ),
          ),
          _StatusFooter(
            groupCount: _groups.length,
            postCount: marks.posts.length,
          ),
        ],
      ),
    );
  }
}

/// How many groups and posts the page has so far.
class const _StatusFooter({
  required final int groupCount,
  required final int postCount,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        const AppDivider(),
        SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.all(spacingMd),
            child: SizedBox(
              width: double.infinity,
              child: CaptionText(
                '$groupCount group(s), $postCount post(s) found.',
              ),
            ),
          ),
        ),
      ],
    );
  }
}

/// What the selected element holds and what can be done with it. The
/// selection starts at [tapped], and can be widened to the elements around it
/// and narrowed back down to [tapped].
class const _SelectionSheet({
  required final PageNode tapped,
  required final _Marks marks,
  required final String? Function(PageNode node) rejectionOf,
  required final void Function(PageNode node) onSelect,
  required final void Function(PageNode node) onMarkAsPost,
  required final void Function(_Group group, PageNode post) onEditGroup,
  required final void Function(_Group group) onRemoveGroup,
}) extends StatefulWidget {
  @override
  State<_SelectionSheet> createState() => _SelectionSheetState();
}

class _SelectionSheetState extends State<_SelectionSheet> {
  late var _selected = widget.tapped;

  void _select(PageNode node) {
    setState(() => _selected = node);
    widget.onSelect(node);
  }

  /// Closes the sheet and then runs [action].
  void _closeThen(VoidCallback action) {
    Navigator.pop(context);
    action();
  }

  @override
  Widget build(BuildContext context) {
    final selected = _selected;
    final group = widget.marks.posts[selected.id];
    final rejection = group == null ? widget.rejectionOf(selected) : null;
    // A post is selected as a whole, and the whole page cannot be a post.
    final narrower = group == null
        ? narrowTowards(selected, widget.tapped)
        : null;
    final wider = selected.parent?.parent == null ? null : selected.parent;
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(spacingMd, 0, spacingMd, spacingMd),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            SelectionHeader(
              selected: selected,
              onNarrow: narrower == null ? null : () => _select(narrower),
              onWiden: wider == null ? null : () => _select(wider),
            ),
            const Gap(spacingSm),
            if (group != null)
              Row(
                children: [
                  Expanded(child: BodyText('A post of group ${group.number}.')),
                  TextButton(
                    onPressed: () =>
                        _closeThen(() => widget.onRemoveGroup(group)),
                    child: const Text('Remove group'),
                  ),
                  const Gap(spacingSm),
                  FilledButton(
                    onPressed: () =>
                        _closeThen(() => widget.onEditGroup(group, selected)),
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
                  onPressed: () =>
                      _closeThen(() => widget.onMarkAsPost(selected)),
                  child: const Text('Mark as post'),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

final _tagStyle = textCaption.copyWith(fontFamily: 'monospace');

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
              GroupDot(color: group.color),
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
