import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/error/domain_error.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/core/ui/tokens/app_colors.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/app_divider.dart';
import 'package:paperdoll/core/ui/widgets/async_value_view.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/ui/widgets/heading_text.dart';
import 'package:paperdoll/core/util/date_format.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/page_watch/data/matcher_repository.dart';
import 'package:paperdoll/features/page_watch/domain/page_document.dart';
import 'package:paperdoll/features/page_watch/domain/post_matcher.dart';
import 'package:paperdoll/features/page_watch/presentation/post_fields_screen.dart';
import 'package:paperdoll/features/page_watch/presentation/providers/page_watch_providers.dart';
import 'package:paperdoll/features/page_watch/presentation/widgets/page_lines.dart';

/// Lets the user tell which parts of a web page are posts, so that the page
/// can be watched like a feed.
///
/// The page is shown as its content, line by line, except that each post
/// found so far is shown as a card of the values extracted from it. The user
/// repeats two
/// steps: select the area that holds one post, then label the content inside
/// it as the post's fields on [PostFieldsScreen]. An area is selected by
/// tapping a line, which selects the narrowest element around it that can be
/// a post, and
/// widening the selection to the elements around that one. Areas at the same
/// shape path as the selected post become posts of the same group
/// automatically, with their fields found at the same positions. Each group
/// results in one [PostMatcher]. A post of a group that is not a post, such as
/// a promotion, can be excluded by its link.
///
/// The page is the one of the unstructured feed [feedId], and the matchers
/// saved for the feed are shown as the initial groups. Saving replaces them.
class const PageAnnotationScreen({required final int feedId, super.key})
    extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final matchers = ref.watch(feedMatchersProvider(feedId));
    if (matchers case AsyncData(:final value)) {
      return _Annotator(
        feedId: feedId,
        document: value.page,
        matchers: value.matchers,
      );
    }
    return Scaffold(
      appBar: AppBar(),
      body: AsyncValueView<FeedMatchers>(
        value: matchers,
        onRetry: () => ref.invalidate(feedMatchersProvider(feedId)),
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

/// Whether [node] or the content inside it has a link to a page.
bool _hasLink(PageNode node) =>
    node.pageLink != null || node.children.any(_hasLink);

/// The posts of all groups in the document, and the excluded ones with the
/// pattern that excludes them, by node id.
class _Marks {
  final posts = <int, _Group>{};
  final excluded = <int, ({_Group group, String pattern})>{};

  /// The post of a group that contains [node], if any.
  PageNode? postOf(PageNode node) {
    for (PageNode? n = node; n != null; n = n.parent) {
      if (posts.containsKey(n.id)) {
        return n;
      }
    }
    return null;
  }

  /// The post or excluded post that contains [node], if any.
  PageNode? enclosingPost(PageNode node) {
    for (PageNode? n = node; n != null; n = n.parent) {
      if (posts.containsKey(n.id) || excluded.containsKey(n.id)) {
        return n;
      }
    }
    return null;
  }
}

class const _Annotator({
  required final int feedId,
  required final PageDocument document,
  required final List<PostMatcher> matchers,
}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<_Annotator> createState() => _AnnotatorState();
}

class _AnnotatorState extends ConsumerState<_Annotator> {
  late final _groups = [
    for (final (i, matcher) in widget.matchers.indexed)
      _Group(number: i + 1, matcher: matcher),
  ];
  late var _nextGroupNumber = _groups.length + 1;
  var _saving = false;

  /// The element shown on the selection sheet, while the sheet is open.
  PageNode? _selected;

  /// The content the user tapped last. The selection can be narrowed back
  /// down to it after being widened.
  PageNode? _tapped;

  void _select(PageNode? node, {PageNode? tapped}) {
    setState(() {
      _selected = node;
      _tapped = tapped ?? _tapped;
    });
  }

  PageDocument get _document => widget.document;

  /// The lines of the whole page. They are built lazily by a list, since a
  /// page can have thousands of them.
  late final _lines = layOutLines([_document.root]);

  /// [line] or the narrowest element around it that can be a post, if any.
  /// A post needs a link and a text for its title, and the whole page cannot
  /// be a post.
  PageNode? _selectableElement(PageNode line) {
    for (var n = line; n.parent != null; n = n.parent!) {
      if (_hasLink(n) && n.textContent.trim().isNotEmpty) {
        return n;
      }
    }
    return null;
  }

  _Marks _computeMarks() {
    final marks = _Marks();
    for (final group in _groups) {
      for (final post in group.matcher.matchPosts(_document)) {
        marks.posts[post.id] = group;
      }
      for (final post in group.matcher.matchExcluded(_document)) {
        marks.excluded[post.id] = (
          group: group,
          pattern: group.matcher.excludedBy(post)!,
        );
      }
    }
    return marks;
  }

  /// Why [node] cannot become a post, or null if it can.
  ///
  /// Only [node] itself is checked. Other nodes at its shape path can be posts
  /// of another group, since the labeled fields decide which of them are
  /// posts, such as cards whose date is at a different position.
  String? _postRejection(PageNode node, _Marks marks) {
    for (final MapEntry(key: id, value: group) in marks.posts.entries) {
      if (_document.nodes[id].overlaps(node)) {
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
      matcher: PostMatcher.fromNode(node).withEvidentFields(node),
      post: node,
      groupNumber: number,
    );
    if (matcher == null || !mounted) {
      return;
    }
    setState(() {
      _nextGroupNumber++;
      _groups.add(_Group(number: number, matcher: matcher));
      _selected = null;
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
      _selected = null;
    });
  }

  /// Excludes the link of [post] from [group], or tells the user why it
  /// cannot be excluded.
  void _exclude(_Group group, PageNode post) {
    final link = group.matcher.matchField(post, PostField.link)!.link!;
    final matcher = group.matcher.withExcluded(_document, link);
    if (matcher == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text(
            'This cannot be excluded, because a post of the group has a link '
            'with the same host and path.',
          ),
        ),
      );
      return;
    }
    setState(() {
      group.matcher = matcher;
      _selected = null;
    });
  }

  void _restore(_Group group, String pattern) {
    setState(() {
      group.matcher = group.matcher.withoutExcluded(pattern);
      _selected = null;
    });
  }

  void _removeGroup(_Group group) {
    setState(() {
      _groups.remove(group);
      _selected = null;
    });
  }

  Future<void> _save() async {
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    setState(() => _saving = true);
    try {
      await ref.read(matcherRepositoryProvider).save(widget.feedId, [
        for (final g in _groups) g.matcher,
      ]);
      ref.invalidate(feedMatchersProvider(widget.feedId));
      messenger.showSnackBar(
        const SnackBar(
          key: AppDebugKey.saveMatchersSuccessSnackBar,
          content: Text('Saved the posts of the page.'),
        ),
      );
      router.pop();
    } on DomainError catch (error) {
      messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
    } finally {
      if (mounted) {
        setState(() => _saving = false);
      }
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
    // The lines of a post are replaced by its card. They are next to each
    // other, since the lines are in document order.
    final items = <PageNode>[];
    for (final line in _lines) {
      final post = marks.postOf(line);
      if (post == null) {
        items.add(line);
      } else if (items.isEmpty || !identical(items.last, post)) {
        items.add(post);
      }
    }
    final areas = <PageArea>[
      for (final id in marks.excluded.keys)
        (node: _document.nodes[id], color: colorTextMuted, linkOnly: false),
      if (selected != null) selectionArea(selected),
    ];

    return Scaffold(
      key: AppDebugKey.pageAnnotationScreen,
      appBar: AppBar(
        title: Text(
          _document.title.isEmpty ? _document.url : _document.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          TextButton(
            key: AppDebugKey.saveMatchersButton,
            onPressed: _groups.isEmpty || _saving
                ? null
                : () => unawaited(_save()),
            child: const Text('Save'),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              padding: const EdgeInsets.symmetric(vertical: spacingSm),
              itemCount: items.length,
              itemBuilder: (context, index) {
                final line = items[index];
                if (marks.posts[line.id] case final group?) {
                  return _PostCard(
                    key: ValueKey(line.id),
                    values: {
                      for (final field in PostField.values)
                        if (group.matcher.matchField(line, field)
                            case final node?)
                          field: ?field.valueOf(node),
                    },
                    group: group,
                    isSelected: selected?.contains(line) ?? false,
                    onTap: () => _select(line, tapped: line),
                  );
                }
                return PageLine(
                  key: ValueKey(line.id),
                  node: line,
                  areas: areas,
                  onTap: (line) {
                    // A tap on a line selects the narrowest element around it
                    // that can be a post, or the post that contains it.
                    final element = _selectableElement(line);
                    if (element != null) {
                      final target = marks.enclosingPost(element) ?? element;
                      _select(target, tapped: target);
                    }
                  },
                );
              },
            ),
          ),
          if (selected != null)
            SelectionSheet(child: _buildSheet(selected, marks))
          else if (_groups.isNotEmpty)
            _StatusFooter(
              groupCount: _groups.length,
              postCount: marks.posts.length,
              onPreview: _showPreview,
            ),
        ],
      ),
    );
  }

  Widget _buildSheet(PageNode selected, _Marks marks) {
    final group = marks.posts[selected.id];
    final excluded = marks.excluded[selected.id];
    return _SelectionSheet(
      selected: selected,
      levels: selectionLevels(_tapped ?? selected),
      onSelect: _select,
      group: group,
      excluded: excluded,
      rejection: group == null && excluded == null
          ? _postRejection(selected, marks)
          : null,
      onCancel: () => _select(null),
      onMarkAsPost: () => unawaited(_markAsPost(selected)),
      onEditGroup: group == null
          ? null
          : () => unawaited(_editGroup(group, selected)),
      onExclude: group == null ? null : () => _exclude(group, selected),
      onRestore: excluded == null
          ? null
          : () => _restore(excluded.group, excluded.pattern),
      onRemoveGroup: group == null ? null : () => _removeGroup(group),
    );
  }
}

/// How many groups and posts the page has so far, with a button to preview
/// the posts.
class const _StatusFooter({
  required final int groupCount,
  required final int postCount,
  required final VoidCallback onPreview,
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
            padding: const EdgeInsets.symmetric(
              horizontal: spacingMd,
              vertical: spacingSm,
            ),
            child: Row(
              children: [
                Expanded(
                  child: CaptionText(
                    '$groupCount group(s), $postCount post(s) found.',
                  ),
                ),
                TextButton(
                  onPressed: onPreview,
                  child: const Text('Preview posts'),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

/// What the selected element holds and what can be done with it. The
/// element is a post of [group] if it is not null, a post excluded from a
/// group if [excluded] is not null, and cannot become a post for the reason
/// in [rejection] if it is not null.
class const _SelectionSheet({
  required final PageNode selected,
  required final _Group? group,
  required final ({_Group group, String pattern})? excluded,
  required final String? rejection,
  required final List<PageNode> levels,
  required final void Function(PageNode node) onSelect,
  required final VoidCallback onCancel,
  required final VoidCallback onMarkAsPost,
  required final VoidCallback? onEditGroup,
  required final VoidCallback? onExclude,
  required final VoidCallback? onRestore,
  required final VoidCallback? onRemoveGroup,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final group = this.group;
    final excluded = this.excluded;
    final rejection = this.rejection;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          children: [
            Expanded(
              child: BodyText(
                describeNode(selected),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            IconButton(
              icon: const Icon(Icons.close),
              tooltip: 'Cancel',
              onPressed: onCancel,
            ),
          ],
        ),
        // A post is selected as a whole.
        if (group == null && excluded == null)
          SelectionSlider(
            selected: selected,
            levels: levels,
            onSelect: onSelect,
          ),
        const Gap(spacingSm),
        if (rejection != null) ...[
          CaptionText(rejection),
          const Gap(spacingSm),
        ],
        if (excluded != null) ...[
          CaptionText(
            'Excluded from group ${excluded.group.number} by '
            '${excluded.pattern}.',
          ),
          const Gap(spacingSm),
        ],
        Row(
          children: [
            const Spacer(),
            if (group != null) ...[
              // Actions that are used rarely are kept in a menu, since the
              // row is too narrow for all of them.
              PopupMenuButton<VoidCallback>(
                icon: const Icon(Icons.more_vert),
                tooltip: 'More',
                onSelected: (action) => action(),
                itemBuilder: (context) => [
                  if (onRemoveGroup case final onRemoveGroup?)
                    PopupMenuItem(
                      value: onRemoveGroup,
                      child: const Text('Remove group'),
                    ),
                ],
              ),
              TextButton(onPressed: onExclude, child: const Text('Exclude')),
              const Gap(spacingSm),
              FilledButton(
                onPressed: onEditGroup,
                child: const Text('Edit fields'),
              ),
            ] else if (excluded != null)
              FilledButton(onPressed: onRestore, child: const Text('Restore'))
            else if (rejection == null)
              FilledButton(
                onPressed: onMarkAsPost,
                child: const Text('Mark as post'),
              ),
          ],
        ),
      ],
    );
  }
}

/// A post found by [group], shown as a card of the field [values] extracted
/// from it, so that the user can check them without reading the content of
/// the post. The fields without a value are left out, and the card has the
/// color of the group.
///
/// A long press opens the link of the post in the in-app browser.
class const _PostCard({
  required final Map<PostField, String> values,
  required final _Group group,
  final bool isSelected = false,
  final VoidCallback? onTap,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    String? value(PostField field) => switch (values[field]) {
      final v? when v.isNotEmpty => v,
      _ => null,
    };
    final thumbnail = value(PostField.thumbnail);
    final link = value(PostField.link);
    final pubDate = value(PostField.pubDate);
    final body = value(PostField.body);
    final date = pubDate == null ? null : DateTime.tryParse(pubDate);
    return Padding(
      padding: const EdgeInsets.symmetric(
        horizontal: spacingMd,
        vertical: spacingXs,
      ),
      child: Material(
        color: isSelected
            ? colorText.withValues(alpha: 0.08)
            : group.color.withValues(alpha: 0.06),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(spacingSm),
          side: BorderSide(
            color: isSelected ? colorText : group.color,
            width: isSelected ? 2 : 1,
          ),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap == null
              ? null
              : () {
                  onTap!();
                  // The sheet shown below the page can cover the card, so
                  // the card is scrolled back into view, as a line is.
                  WidgetsBinding.instance.addPostFrameCallback((_) {
                    if (context.mounted) {
                      Scrollable.ensureVisible(
                        context,
                        duration: const Duration(milliseconds: 200),
                        alignmentPolicy:
                            ScrollPositionAlignmentPolicy.keepVisibleAtEnd,
                      );
                    }
                  });
                },
          onLongPress: link == null
              ? null
              : () => unawaited(openInAppLink(context, link)),
          child: Padding(
            padding: const EdgeInsets.all(spacingSm),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
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
                            maxLines: 3,
                            overflow: TextOverflow.ellipsis,
                          ),
                          if (pubDate != null)
                            CaptionText(
                              date == null ? pubDate : formatDate(date),
                            ),
                        ],
                      ),
                    ),
                  ],
                ),
                if (body != null) ...[
                  const Gap(spacingXs),
                  BodyText(
                    _plainText(body),
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
                if (link != null) ...[
                  const Gap(spacingXs),
                  CaptionText(
                    link,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// The text of the body [markup] (see `bodyMarkup`) without the markup:
/// the images are left out, and the headings and links become their text.
String _plainText(String markup) => markup
    .replaceAll(RegExp(r'!\[[^\]]*\]\([^)]*\)'), '')
    .replaceAllMapped(RegExp(r'\[([^\]]*)\]\([^)]*\)'), (m) => m[1]!)
    .replaceAll(RegExp('^#+ ', multiLine: true), '')
    .split('\n')
    .where((line) => line.trim().isNotEmpty)
    .join('\n');

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

/// The posts extracted with the current matchers, merged by link.
class const _Preview({
  required final PageDocument document,
  required final List<_Group> groups,
  required final ScrollController scrollController,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final posts = extractPosts(document, [for (final g in groups) g.matcher]);
    return ListView(
      controller: scrollController,
      // The cards have their own side padding.
      padding: const EdgeInsets.symmetric(vertical: spacingMd),
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: spacingMd),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              HeadingText('${posts.length} post(s)'),
              for (final group in groups)
                if (group.matcher.exclude.isNotEmpty)
                  Row(
                    children: [
                      GroupDot(color: group.color),
                      const Gap(spacingSm),
                      Expanded(
                        child: CaptionText(
                          'Group ${group.number} excludes '
                          '${group.matcher.exclude.join(', ')}',
                        ),
                      ),
                    ],
                  ),
            ],
          ),
        ),
        const Gap(spacingSm),
        for (final post in posts)
          _PostCard(values: post.values, group: groups[post.matcher]),
      ],
    );
  }
}
