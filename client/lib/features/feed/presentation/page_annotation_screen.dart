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
import 'package:paperdoll/features/feed/data/selector_repository.dart';
import 'package:paperdoll/features/feed/domain/post_selector.dart';
import 'package:paperdoll/features/feed/domain/simplified_page.dart';
import 'package:paperdoll/features/feed/presentation/post_fields_screen.dart';
import 'package:paperdoll/features/feed/presentation/providers/selector_providers.dart';
import 'package:paperdoll/features/feed/presentation/widgets/page_lines.dart';

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
/// The selectors are generated from the posts that the user marked and the
/// links that the user marked as not posts (see [generateSelectors]), so the
/// posts that look like a marked one are found automatically. The user never
/// sees the selectors: a card is either a post that the user marked or one that
/// was found automatically, and a post that is not one can be marked as such.
///
/// The page is the one of the unstructured feed [feedId]. The selectors saved
/// for the feed are shown as one post for each of them. Saving replaces them.
class const PageAnnotationScreen({required final int feedId, super.key})
    extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectors = ref.watch(feedSelectorsProvider(feedId));
    if (selectors case AsyncData(:final value)) {
      return _Annotator(
        feedId: feedId,
        page: value.page,
        selectors: value.selectors,
      );
    }
    return Scaffold(
      appBar: AppBar(),
      body: AsyncValueView<FeedSelectors>(
        value: selectors,
        onRetry: () => ref.invalidate(feedSelectorsProvider(feedId)),
        data: (_) => const SizedBox.shrink(),
      ),
    );
  }
}

/// Whether [node] or the content inside it has a link to a page.
bool _hasLink(Node node) =>
    node.pageLink != null || node.pageChildren.any(_hasLink);

/// The posts that the selectors find in the page, with the selector that
/// finds each, and the excluded ones, with the host and path that excludes
/// them.
class _Marks {
  final posts = <Node, PostSelector>{};
  final excluded = <Node, String>{};

  /// The post that contains [node], if any.
  Node? postOf(Node node) {
    for (Node? n = node; n != null; n = n.parentNode) {
      if (posts.containsKey(n)) {
        return n;
      }
    }
    return null;
  }

  /// The post or excluded post that contains [node], if any.
  Node? enclosingPost(Node node) {
    for (Node? n = node; n != null; n = n.parentNode) {
      if (posts.containsKey(n) || excluded.containsKey(n)) {
        return n;
      }
    }
    return null;
  }
}

class const _Annotator({
  required final int feedId,
  required final SimplifiedPage page,
  required final List<PostSelector> selectors,
}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<_Annotator> createState() => _AnnotatorState();
}

class _AnnotatorState extends ConsumerState<_Annotator> {
  /// The posts that the saved selectors find, by node. The selectors are
  /// generated from them too, so that they keep finding the same posts, but
  /// they are not shown as marked, since the user did not mark them here.
  late final _found = {
    for (final p in labelFoundPosts(widget.page, widget.selectors)) p.node: p,
  };

  /// The posts that the user marked, by node.
  final _labeled = <Node, LabeledPost>{};

  /// The host and path of the links that the user marked as not posts.
  late final _excluded = {for (final m in widget.selectors) ...m.exclude};

  late List<PostSelector> _selectors;
  late _Marks _marks;
  var _saving = false;

  @override
  void initState() {
    super.initState();
    _update();
  }

  /// Generates the selectors again after the marks changed.
  void _update() {
    _selectors = generateSelectors(
      _page,
      {..._found, ..._labeled}.values,
      _excluded,
    );
    _marks = _Marks();
    for (final selector in _selectors) {
      for (final post in selector.selectPosts(_page)) {
        _marks.posts.putIfAbsent(post, () => selector);
      }
      for (final post in selector.matchExcluded(_page)) {
        _marks.excluded[post] = selector.excludedBy(post)!;
      }
    }
  }

  /// The element shown on the selection sheet, while the sheet is open.
  Node? _selected;

  /// The content the user tapped last. The selection can be narrowed back
  /// down to it after being widened.
  Node? _tapped;

  void _select(Node? node, {Node? tapped}) {
    setState(() {
      _selected = node;
      _tapped = tapped ?? _tapped;
    });
  }

  SimplifiedPage get _page => widget.page;

  /// The lines of the whole page. They are built lazily by a list, since a
  /// page can have thousands of them.
  late final _lines = layOutLines([_page.body]);

  /// [line] or the narrowest element around it that can be a post, if any.
  /// A post needs a link and a text for its title, and the whole page cannot
  /// be a post.
  Node? _selectableElement(Node line) {
    for (var n = line; !identical(n, _page.body); n = n.parentNode!) {
      if (_hasLink(n) && n.allText.trim().isNotEmpty) {
        return n;
      }
    }
    return null;
  }

  /// Why [node] cannot become a post, or null if it can.
  ///
  /// Only [node] itself is checked. Other nodes that its selector would select
  /// can be posts of another selector, since the labeled fields decide which of
  /// them are posts, such as cards whose title is in a different element.
  String? _postRejection(Node node) {
    for (final post in _marks.posts.keys) {
      if (post.overlaps(node)) {
        return 'This overlaps with a post.';
      }
    }
    return null;
  }

  /// The nodes that [selector] labels as the fields of [post].
  PostFields _fieldsOf(PostSelector selector, Node post) => {
    for (final field in PostField.values)
      field: ?selector.matchField(post, field),
  };

  /// Opens the screen that labels the fields of [post], and marks [post] as a
  /// post with them, or does nothing if the user left without finishing.
  Future<void> _markAsPost(Node post, PostFields fields) async {
    final labeled = await context.pushNamed<PostFields>(
      routePostFieldsName,
      pathParameters: GoRouterState.of(context).pathParameters,
      extra: (post: post, fields: fields),
    );
    if (labeled == null || !mounted) {
      return;
    }
    setState(() {
      _labeled[post] = (node: post, fields: labeled);
      _excluded.remove(hostPathOf(labeled[PostField.link]!.link!));
      _update();
      _selected = null;
    });
  }

  /// Marks the link of [post] as not a post, which also unmarks the posts that
  /// the user marked, or that the saved selectors found, with the same host and
  /// path.
  void _markAsNotPost(Node post) {
    final hostPath = hostPathOf(
      _marks.posts[post]!.matchField(post, PostField.link)!.link!,
    );
    bool hasHostPath(Node _, LabeledPost p) =>
        hostPathOf(p.fields[PostField.link]!.link!) == hostPath;
    setState(() {
      _excluded.add(hostPath);
      _labeled.removeWhere(hasHostPath);
      _found.removeWhere(hasHostPath);
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

  void _clearMark(Node post) {
    setState(() {
      _labeled.remove(post);
      _update();
      _selected = null;
    });
  }

  Future<void> _save() async {
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    setState(() => _saving = true);
    try {
      await ref
          .read(selectorRepositoryProvider)
          .save(widget.feedId, _selectors);
      ref.invalidate(feedSelectorsProvider(widget.feedId));
      messenger.showSnackBar(
        const SnackBar(
          key: AppDebugKey.saveSelectorsSuccessSnackBar,
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
            page: _page,
            selectors: _selectors,
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
    final items = <Node>[];
    for (final line in _lines) {
      final post = marks.postOf(line);
      if (post == null) {
        items.add(line);
      } else if (items.isEmpty || !identical(items.last, post)) {
        items.add(post);
      }
    }
    final areas = <PageArea>[
      for (final node in marks.excluded.keys)
        (node: node, color: colorTextMuted, linkOnly: false),
      if (selected != null) selectionArea(selected),
    ];

    return Scaffold(
      key: AppDebugKey.pageAnnotationScreen,
      appBar: AppBar(
        title: Text(
          _page.title.isEmpty ? _page.url : _page.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          TextButton(
            key: AppDebugKey.saveSelectorsButton,
            onPressed: _selectors.isEmpty || _saving
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
                if (marks.posts[line] case final selector?) {
                  return _PostCard(
                    key: ObjectKey(line),
                    values: {
                      for (final field in PostField.values)
                        if (selector.matchField(line, field) case final node?)
                          field: ?field.valueOf(node),
                    },
                    isMarked: _labeled.containsKey(line),
                    isSelected: selected?.includes(line) ?? false,
                    onTap: () => _select(line, tapped: line),
                  );
                }
                return PageLine(
                  key: ObjectKey(line),
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
          else if (_selectors.isNotEmpty)
            _StatusFooter(
              postCount: marks.posts.length,
              onPreview: _showPreview,
            ),
        ],
      ),
    );
  }

  Widget _buildSheet(Node selected) {
    final selector = _marks.posts[selected];
    final excluded = _marks.excluded[selected];
    final rejection = selector == null && excluded == null
        ? _postRejection(selected)
        : null;
    return _SelectionSheet(
      selected: selected,
      levels: selectionLevels(_tapped ?? selected),
      onSelect: _select,
      isPost: selector != null,
      isMarked: _labeled.containsKey(selected),
      isExcluded: excluded != null,
      rejection: rejection,
      onCancel: () => _select(null),
      onMarkAsPost: () =>
          unawaited(_markAsPost(selected, evidentFields(selected))),
      onEditFields: selector == null
          ? null
          : () => unawaited(
              _markAsPost(
                selected,
                _labeled[selected]?.fields ?? _fieldsOf(selector, selected),
              ),
            ),
      onNotPost: selector == null ? null : () => _markAsNotPost(selected),
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
  required final Node selected,
  required final bool isPost,
  required final bool isMarked,
  required final bool isExcluded,
  required final String? rejection,
  required final List<Node> levels,
  required final void Function(Node node) onSelect,
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
                  BodyText(body, maxLines: 3, overflow: TextOverflow.ellipsis),
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

/// The posts extracted with the current selectors, merged by link.
class const _Preview({
  required final SimplifiedPage page,
  required final List<PostSelector> selectors,
  required final ScrollController scrollController,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final posts = extractPosts(page, selectors);
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
