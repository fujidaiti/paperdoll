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
import 'package:paperdoll/features/page_watch/domain/post_matcher.dart';
import 'package:paperdoll/features/page_watch/domain/semantic_tree.dart';
import 'package:paperdoll/features/page_watch/presentation/post_fields_screen.dart';
import 'package:paperdoll/features/page_watch/presentation/providers/page_watch_providers.dart';
import 'package:paperdoll/features/page_watch/presentation/widgets/page_lines.dart';

/// Lets the user tell which parts of a web page are posts, so that the page
/// can be watched like a feed.
///
/// The page is shown as its content, line by line, except that each post
/// found so far is shown as a card of the values extracted from it. The user
/// marks a post by selecting the area that holds it and labeling the content
/// inside it as the post's fields on [PostFieldsScreen]. An area is selected
/// by tapping a line, which selects the narrowest element around it that can
/// be a post, and widening the selection to the elements around that one.
///
/// The matchers are generated from the posts that the user marked and the
/// links that the user marked as not posts (see [generateMatchers]), so the
/// posts that look like a marked one are found automatically. The user never
/// sees the matchers: a card is either a post that the user marked or one that
/// was found automatically, and a post that is not one can be marked as such.
///
/// The page is the one of the unstructured feed [feedId]. The matchers saved
/// for the feed are shown as one post for each of them. Saving replaces them.
class const PageAnnotationScreen({required final int feedId, super.key})
    extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final matchers = ref.watch(feedMatchersProvider(feedId));
    if (matchers case AsyncData(:final value)) {
      return _Annotator(
        feedId: feedId,
        tree: value.tree,
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

/// Whether [node] or the content inside it has a link to a page.
bool _hasLink(SemanticTreeNode node) =>
    node.pageLink != null || node.children.any(_hasLink);

/// The posts that the matchers find in the tree, by node id, with the
/// matcher that finds each, and the excluded ones, by node id, with the host
/// and path that excludes them.
class _Marks {
  final posts = <int, PostMatcher>{};
  final excluded = <int, String>{};

  /// The post that contains [node], if any.
  SemanticTreeNode? postOf(SemanticTreeNode node) {
    for (SemanticTreeNode? n = node; n != null; n = n.parent) {
      if (posts.containsKey(n.id)) {
        return n;
      }
    }
    return null;
  }

  /// The post or excluded post that contains [node], if any.
  SemanticTreeNode? enclosingPost(SemanticTreeNode node) {
    for (SemanticTreeNode? n = node; n != null; n = n.parent) {
      if (posts.containsKey(n.id) || excluded.containsKey(n.id)) {
        return n;
      }
    }
    return null;
  }
}

class const _Annotator({
  required final int feedId,
  required final SemanticTree tree,
  required final List<PostMatcher> matchers,
}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<_Annotator> createState() => _AnnotatorState();
}

class _AnnotatorState extends ConsumerState<_Annotator> {
  /// The posts that the user marked, by node id.
  late final _labeled = {
    for (final p in sampleLabeledPosts(widget.tree, widget.matchers))
      p.node.id: p,
  };

  /// The host and path of the links that the user marked as not posts.
  late final _excluded = {for (final m in widget.matchers) ...m.exclude};

  late List<PostMatcher> _matchers;
  late _Marks _marks;
  var _saving = false;

  @override
  void initState() {
    super.initState();
    _update();
  }

  /// Generates the matchers again after the marks changed.
  void _update() {
    _matchers = generateMatchers(_tree, _labeled.values, _excluded);
    _marks = _Marks();
    for (final matcher in _matchers) {
      for (final post in matcher.matchPosts(_tree)) {
        _marks.posts.putIfAbsent(post.id, () => matcher);
      }
      for (final post in matcher.matchExcluded(_tree)) {
        _marks.excluded[post.id] = matcher.excludedBy(post)!;
      }
    }
  }

  /// The element shown on the selection sheet, while the sheet is open.
  SemanticTreeNode? _selected;

  /// The content the user tapped last. The selection can be narrowed back
  /// down to it after being widened.
  SemanticTreeNode? _tapped;

  void _select(SemanticTreeNode? node, {SemanticTreeNode? tapped}) {
    setState(() {
      _selected = node;
      _tapped = tapped ?? _tapped;
    });
  }

  SemanticTree get _tree => widget.tree;

  /// The lines of the whole page. They are built lazily by a list, since a
  /// page can have thousands of them.
  late final _lines = layOutLines([_tree.root]);

  /// [line] or the narrowest element around it that can be a post, if any.
  /// A post needs a link and a text for its title, and the whole page cannot
  /// be a post.
  SemanticTreeNode? _selectableElement(SemanticTreeNode line) {
    for (var n = line; n.parent != null; n = n.parent!) {
      if (_hasLink(n) && n.textContent.trim().isNotEmpty) {
        return n;
      }
    }
    return null;
  }

  /// Why [node] cannot become a post, or null if it can.
  ///
  /// Only [node] itself is checked. Other nodes that its matcher would select
  /// can be posts of another matcher, since the labeled fields decide which of
  /// them are posts, such as cards whose title is in a different element.
  String? _postRejection(SemanticTreeNode node) {
    for (final id in _marks.posts.keys) {
      if (_tree.nodes[id].overlaps(node)) {
        return 'This overlaps with a post.';
      }
    }
    return null;
  }

  /// The nodes that [matcher] labels as the fields of [post].
  PostFields _fieldsOf(PostMatcher matcher, SemanticTreeNode post) => {
    for (final field in PostField.values)
      field: ?matcher.matchField(post, field),
  };

  /// Opens the screen that labels the fields of [post], and marks [post] as a
  /// post with them, or does nothing if the user left without finishing.
  Future<void> _markAsPost(SemanticTreeNode post, PostFields fields) async {
    final labeled = await context.pushNamed<PostFields>(
      routePostFieldsName,
      pathParameters: GoRouterState.of(context).pathParameters,
      extra: (post: post, fields: fields),
    );
    if (labeled == null || !mounted) {
      return;
    }
    setState(() {
      _labeled[post.id] = (node: post, fields: labeled);
      _excluded.remove(hostPathOf(labeled[PostField.link]!.link!));
      _update();
      _selected = null;
    });
  }

  /// Marks the link of [post] as not a post, which also unmarks the posts that
  /// the user marked with the same host and path.
  void _markAsNotPost(SemanticTreeNode post) {
    final hostPath = hostPathOf(
      _marks.posts[post.id]!.matchField(post, PostField.link)!.link!,
    );
    setState(() {
      _excluded.add(hostPath);
      _labeled.removeWhere(
        (_, p) => hostPathOf(p.fields[PostField.link]!.link!) == hostPath,
      );
      _update();
      _selected = null;
    });
  }

  void _restore(String hostPath) {
    setState(() {
      _excluded.remove(hostPath);
      _update();
      _selected = null;
    });
  }

  void _clearMark(SemanticTreeNode post) {
    setState(() {
      _labeled.remove(post.id);
      _update();
      _selected = null;
    });
  }

  Future<void> _save() async {
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    setState(() => _saving = true);
    try {
      await ref.read(matcherRepositoryProvider).save(widget.feedId, _matchers);
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
            tree: _tree,
            matchers: _matchers,
            scrollController: controller,
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final marks = _marks;
    final selected = _selected;
    // The lines of a post are replaced by its card. They are next to each
    // other, since the lines are in document order.
    final items = <SemanticTreeNode>[];
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
        (node: _tree.nodes[id], color: colorTextMuted, linkOnly: false),
      if (selected != null) selectionArea(selected),
    ];

    return Scaffold(
      key: AppDebugKey.pageAnnotationScreen,
      appBar: AppBar(
        title: Text(
          _tree.title.isEmpty ? _tree.url : _tree.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          TextButton(
            key: AppDebugKey.saveMatchersButton,
            onPressed: _matchers.isEmpty || _saving
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
                if (marks.posts[line.id] case final matcher?) {
                  return _PostCard(
                    key: ValueKey(line.id),
                    values: {
                      for (final field in PostField.values)
                        if (matcher.matchField(line, field) case final node?)
                          field: ?field.valueOf(node),
                    },
                    isMarked: _labeled.containsKey(line.id),
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
            SelectionSheet(child: _buildSheet(selected))
          else if (_matchers.isNotEmpty)
            _StatusFooter(
              postCount: marks.posts.length,
              onPreview: _showPreview,
            ),
        ],
      ),
    );
  }

  Widget _buildSheet(SemanticTreeNode selected) {
    final matcher = _marks.posts[selected.id];
    final excluded = _marks.excluded[selected.id];
    final rejection = matcher == null && excluded == null
        ? _postRejection(selected)
        : null;
    return _SelectionSheet(
      selected: selected,
      levels: selectionLevels(_tapped ?? selected),
      onSelect: _select,
      isPost: matcher != null,
      isMarked: _labeled.containsKey(selected.id),
      isExcluded: excluded != null,
      rejection: rejection,
      onCancel: () => _select(null),
      onMarkAsPost: () =>
          unawaited(_markAsPost(selected, evidentFields(selected))),
      onEditFields: matcher == null
          ? null
          : () => unawaited(
              _markAsPost(
                selected,
                _labeled[selected.id]?.fields ?? _fieldsOf(matcher, selected),
              ),
            ),
      onNotPost: matcher == null ? null : () => _markAsNotPost(selected),
      onClearMark: () => _clearMark(selected),
      onRestore: excluded == null ? null : () => _restore(excluded),
    );
  }
}

/// How many posts the page has so far, with a button to preview them.
class const _StatusFooter({
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
                Expanded(child: CaptionText('$postCount post(s) found.')),
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

/// What the selected element holds and what can be done with it. The element
/// is a post if [isPost] is true, and one that the user marked if [isMarked]
/// is true. It is a post that the user marked as not one if [isExcluded] is
/// true, and cannot become a post for the reason in [rejection] if it is not
/// null. Otherwise it can be marked as a post.
class const _SelectionSheet({
  required final SemanticTreeNode selected,
  required final bool isPost,
  required final bool isMarked,
  required final bool isExcluded,
  required final String? rejection,
  required final List<SemanticTreeNode> levels,
  required final void Function(SemanticTreeNode node) onSelect,
  required final VoidCallback onCancel,
  required final VoidCallback onMarkAsPost,
  required final VoidCallback? onEditFields,
  required final VoidCallback? onNotPost,
  required final VoidCallback onClearMark,
  required final VoidCallback? onRestore,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
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
        if (!isPost && !isExcluded)
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
        if (isExcluded) ...[
          const CaptionText('Marked as not a post.'),
          const Gap(spacingSm),
        ],
        Row(
          children: [
            const Spacer(),
            if (isPost) ...[
              // Actions that are used rarely are kept in a menu, since the
              // row is too narrow for all of them.
              if (isMarked)
                PopupMenuButton<VoidCallback>(
                  icon: const Icon(Icons.more_vert),
                  tooltip: 'More',
                  onSelected: (action) => action(),
                  itemBuilder: (context) => [
                    PopupMenuItem(
                      value: onClearMark,
                      child: const Text('Clear mark'),
                    ),
                  ],
                ),
              TextButton(onPressed: onNotPost, child: const Text('Not a post')),
              const Gap(spacingSm),
              FilledButton(
                onPressed: onEditFields,
                child: const Text('Edit fields'),
              ),
            ] else if (isExcluded)
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

/// A post, shown as a card of the field [values] extracted from it, so that
/// the user can check them without reading the content of the post. The
/// fields without a value are left out. The card of a post that the user
/// marked has a thicker border than one found automatically.
///
/// A long press opens the link of the post in the in-app browser.
class const _PostCard({
  required final Map<PostField, String> values,
  final bool isMarked = false,
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
            : colorAccent.withValues(alpha: 0.06),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(spacingSm),
          side: BorderSide(
            color: isSelected ? colorText : colorAccent,
            width: isSelected || isMarked ? 2 : 1,
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
  required final SemanticTree tree,
  required final List<PostMatcher> matchers,
  required final ScrollController scrollController,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final posts = extractPosts(tree, matchers);
    return ListView(
      controller: scrollController,
      // The cards have their own side padding.
      padding: const EdgeInsets.symmetric(vertical: spacingMd),
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: spacingMd),
          child: HeadingText('${posts.length} post(s)'),
        ),
        const Gap(spacingSm),
        for (final post in posts) _PostCard(values: post.values),
      ],
    );
  }
}
