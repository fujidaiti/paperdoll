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
import 'package:paperdoll/features/feed/data/selector_editor_repository.dart';
import 'package:paperdoll/features/feed/domain/post_field.dart';
import 'package:paperdoll/features/feed/domain/selector_editor.dart';
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
/// The page is the one of the unstructured feed [feedId], and is edited in a
/// selector editor on the server (see [SelectorEditor]). The server generates
/// the selectors from the posts that the user marked and the links that the
/// user marked as not posts, so the posts that look like a marked one are
/// found automatically. The user never sees the selectors: a card is either a
/// post that is marked or one that was found automatically, and a post that
/// is not one can be marked as such. The posts that the saved selectors find
/// start as marked. Saving replaces the saved selectors.
///
/// The editor of the feed is remembered on the device until it is saved or
/// left, so that the user can continue it after the app was killed.
class const PageAnnotationScreen({required final int feedId, super.key})
    extends ConsumerStatefulWidget {
  @override
  ConsumerState<PageAnnotationScreen> createState() =>
      _PageAnnotationScreenState();
}

class _PageAnnotationScreenState extends ConsumerState<PageAnnotationScreen> {
  AsyncValue<SelectorEditor> _editor = const AsyncLoading();

  @override
  void initState() {
    super.initState();
    unawaited(_open());
  }

  /// Resumes the editor of the feed that has not ended on this device, if the
  /// server still has it and the user wants to continue it, or else starts a
  /// new editor and remembers it. An editor that expired, or that the server
  /// deleted after it expired, cannot be saved anymore, so a new one is
  /// started without asking.
  Future<void> _open() async {
    final repository = ref.read(selectorEditorRepositoryProvider);
    final feedId = widget.feedId;
    try {
      if (await repository.storedEditorId(feedId) case final storedId?) {
        try {
          final editor = await repository.resume(feedId, storedId);
          if (!mounted) {
            return;
          }
          if (await _askToContinue()) {
            setState(() => _editor = AsyncData(editor));
            return;
          }
          // The server deletes the editor once it expires, if this request
          // does not reach it.
          unawaited(repository.close(feedId, storedId).catchError((_) {}));
        } on DomainError catch (error) {
          if (error is! GoneError && error is! NotFoundError) {
            rethrow;
          }
          await repository.storeEditorId(feedId, null);
        }
      }
      final editor = await repository.start(feedId);
      if (!mounted) {
        // The user left before the editor started.
        unawaited(repository.close(feedId, editor.id).catchError((_) {}));
        return;
      }
      await repository.storeEditorId(feedId, editor.id);
      if (mounted) {
        setState(() => _editor = AsyncData(editor));
      }
    } on DomainError catch (error, stackTrace) {
      if (mounted) {
        setState(() => _editor = AsyncError(error, stackTrace));
      }
    }
  }

  /// Asks whether to continue the editor that the user left without saving
  /// or leaving it, such as when the app was killed. Going back from the
  /// question continues it, so that the edits are not lost by accident.
  Future<bool> _askToContinue() async {
    final keep = await showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (context) => AlertDialog(
        title: const Text('Continue your previous edits?'),
        content: const Text(
          'The posts you marked on this page last time were not saved.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Start over'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Continue'),
          ),
        ],
      ),
    );
    return keep ?? true;
  }

  @override
  Widget build(BuildContext context) {
    if (_editor case AsyncData(:final value)) {
      return _Annotator(feedId: widget.feedId, editor: value);
    }
    return Scaffold(
      appBar: AppBar(),
      body: AsyncValueView<SelectorEditor>(
        value: _editor,
        onRetry: () {
          setState(() => _editor = const AsyncLoading());
          unawaited(_open());
        },
        data: (_) => const SizedBox.shrink(),
      ),
    );
  }
}

/// Whether [node] or the content inside it has a link to a page.
bool _hasLink(Node node) =>
    node.pageLink != null || node.pageChildren.any(_hasLink);

class const _Annotator({
  required final int feedId,
  required final SelectorEditor editor,
}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<_Annotator> createState() => _AnnotatorState();
}

class _AnnotatorState extends ConsumerState<_Annotator> {
  /// The marked posts, by node.
  late var _labeled = {for (final p in widget.editor.labeledPosts) p.node: p};

  /// The host and path of the links that the user marked as not posts.
  late var _excluded = widget.editor.excludedLinks;

  /// What the selectors generated from the marks find.
  late var _state = widget.editor.state;

  /// Whether a request to the editor is in flight. The marks cannot change
  /// meanwhile, so that the responses cannot arrive out of order and saving
  /// saves the selectors of the last marks.
  var _pending = false;

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

  SimplifiedPage get _page => widget.editor.page;

  SelectorEditorRepository get _repository =>
      ref.read(selectorEditorRepositoryProvider);

  /// The lines of the whole page. They are built lazily by a list, since a
  /// page can have thousands of them.
  late final _lines = layOutLines([_page.body]);

  /// The post that contains [node], if any.
  Node? _postOf(Node node) {
    for (Node? n = node; n != null; n = n.parentNode) {
      if (_state.matchedPosts.containsKey(n)) {
        return n;
      }
    }
    return null;
  }

  /// The post or excluded post that contains [node], if any.
  Node? _enclosingPost(Node node) {
    for (Node? n = node; n != null; n = n.parentNode) {
      if (_state.matchedPosts.containsKey(n) ||
          _state.excludedPosts.containsKey(n)) {
        return n;
      }
    }
    return null;
  }

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
    for (final post in _state.matchedPosts.keys) {
      if (post.overlaps(node)) {
        return 'This overlaps with a post.';
      }
    }
    return null;
  }

  /// Sends [labeled] and [excluded] as all marks of the editor, and applies
  /// them with what the selectors generated from them find once the server
  /// accepts them. If the request fails, the previous marks stay.
  Future<void> _updateMarks(
    Map<Node, LabeledPost> labeled,
    Set<String> excluded,
  ) async {
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _pending = true);
    try {
      final state = await _repository.updateMarks(
        widget.feedId,
        widget.editor,
        labeled.values,
        excluded,
      );
      if (mounted) {
        setState(() {
          _labeled = labeled;
          _excluded = excluded;
          _state = state;
          _selected = null;
        });
      }
    } on DomainError catch (error) {
      if (error is GoneError || error is NotFoundError) {
        await _showExpired();
      } else {
        messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
      }
    } finally {
      if (mounted) {
        setState(() => _pending = false);
      }
    }
  }

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
    await _updateMarks({
      ..._labeled,
      post: (node: post, fields: labeled),
    }, {..._excluded}..remove(hostPathOf(labeled[PostField.link]!.link!)));
  }

  /// Marks the link of [post] as not a post, which also unmarks the marked
  /// posts with the same host and path.
  void _markAsNotPost(Node post) {
    final hostPath = hostPathOf(
      _state.matchedPosts[post]!.fields[PostField.link]!.link!,
    );
    unawaited(
      _updateMarks(
        {
          for (final MapEntry(:key, :value) in _labeled.entries)
            if (hostPathOf(value.fields[PostField.link]!.link!) != hostPath)
              key: value,
        },
        {..._excluded, hostPath},
      ),
    );
  }

  /// Restores the posts that the excluded link [hostPath] excludes.
  void _restore(String hostPath) {
    unawaited(_updateMarks(_labeled, {..._excluded}..remove(hostPath)));
  }

  void _clearMark(Node post) {
    unawaited(_updateMarks({..._labeled}..remove(post), _excluded));
  }

  Future<void> _save() async {
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    setState(() => _pending = true);
    try {
      await _repository.commit(widget.feedId, widget.editor.id);
      await _repository.storeEditorId(widget.feedId, null);
      messenger.showSnackBar(
        const SnackBar(
          key: AppDebugKey.saveSelectorsSuccessSnackBar,
          content: Text('Saved the posts of the page.'),
        ),
      );
      router.pop();
    } on DomainError catch (error) {
      if (error is GoneError || error is NotFoundError) {
        await _showExpired();
      } else {
        messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
      }
    } finally {
      if (mounted) {
        setState(() => _pending = false);
      }
    }
  }

  /// Tells the user that the editor expired, and leaves the screen. The server
  /// keeps an expired editor until a periodic job deletes it, so a missing
  /// editor is one that expired too.
  Future<void> _showExpired() async {
    await _repository.storeEditorId(widget.feedId, null);
    if (!mounted) {
      return;
    }
    await showDialog<void>(
      context: context,
      builder: (context) => AlertDialog(
        key: AppDebugKey.selectorEditorExpiredDialog,
        title: const Text('The editor expired'),
        content: const Text(
          'The page may have changed since you opened the editor. Open it '
          'again to mark the posts.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('OK'),
          ),
        ],
      ),
    );
    if (mounted) {
      context.pop();
    }
  }

  /// Closes the sheet on back, and asks before leaving the editor without
  /// saving.
  Future<void> _onBack() async {
    if (_selected != null) {
      _select(null);
      return;
    }
    final leave = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Leave without saving?'),
        content: const Text(
          'The changes to the posts of this page will be lost.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Leave'),
          ),
        ],
      ),
    );
    if (!(leave ?? false) || !mounted) {
      return;
    }
    // The server deletes the editor once it expires, if this request does
    // not reach it.
    unawaited(
      _repository.close(widget.feedId, widget.editor.id).catchError((_) {}),
    );
    await _repository.storeEditorId(widget.feedId, null);
    if (mounted) {
      context.pop();
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
          builder: (context, controller) =>
              _Preview(posts: _state.preview, scrollController: controller),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final matched = _state.matchedPosts;
    final selected = _selected;
    // The lines of a post are replaced by its card. They are next to each
    // other, since the lines are in document order.
    final items = <Node>[];
    for (final line in _lines) {
      final post = _postOf(line);
      if (post == null) {
        items.add(line);
      } else if (items.isEmpty || !identical(items.last, post)) {
        items.add(post);
      }
    }
    final areas = <PageArea>[
      for (final node in _state.excludedPosts.keys)
        (node: node, color: colorTextMuted, linkOnly: false),
      if (selected != null) selectionArea(selected),
    ];

    // Going back asks first, since it ends the editor without saving.
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) {
          unawaited(_onBack());
        }
      },
      child: Scaffold(
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
              onPressed: matched.isEmpty || _pending
                  ? null
                  : () => unawaited(_save()),
              child: const Text('Save'),
            ),
          ],
          bottom: PreferredSize(
            preferredSize: const Size.fromHeight(4),
            child: _pending
                ? const LinearProgressIndicator()
                : const SizedBox.shrink(),
          ),
        ),
        body: Column(
          children: [
            Expanded(
              child: ListView.builder(
                padding: const EdgeInsets.symmetric(vertical: spacingSm),
                itemCount: items.length,
                itemBuilder: (context, index) {
                  final line = items[index];
                  if (matched[line] case final post?) {
                    return _PostCard(
                      key: ObjectKey(line),
                      values: post.values,
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
                      // A tap on a line selects the narrowest element around
                      // it that can be a post, or the post that contains it.
                      final element = _selectableElement(line);
                      if (element != null) {
                        final target = _enclosingPost(element) ?? element;
                        _select(target, tapped: target);
                      }
                    },
                  );
                },
              ),
            ),
            if (selected != null)
              SelectionSheet(child: _buildSheet(selected))
            else if (matched.isNotEmpty)
              _StatusFooter(postCount: matched.length, onPreview: _showPreview),
          ],
        ),
      ),
    );
  }

  Widget _buildSheet(Node selected) {
    final post = _state.matchedPosts[selected];
    final excluded = _state.excludedPosts[selected];
    final rejection = post == null && excluded == null
        ? _postRejection(selected)
        : null;
    // The marks cannot change while a request is in flight.
    VoidCallback? unlessPending(VoidCallback? action) =>
        _pending ? null : action;
    return _SelectionSheet(
      selected: selected,
      levels: selectionLevels(_tapped ?? selected),
      onSelect: _select,
      isPost: post != null,
      isMarked: _labeled.containsKey(selected),
      isExcluded: excluded != null,
      rejection: rejection,
      onCancel: () => _select(null),
      onMarkAsPost: unlessPending(
        () => unawaited(_markAsPost(selected, evidentFields(selected))),
      ),
      onEditFields: unlessPending(
        post == null
            ? null
            : () => unawaited(
                _markAsPost(
                  selected,
                  _labeled[selected]?.fields ?? post.fields,
                ),
              ),
      ),
      onNotPost: unlessPending(
        post == null ? null : () => _markAsNotPost(selected),
      ),
      onClearMark: unlessPending(() => _clearMark(selected)),
      onRestore: unlessPending(
        excluded == null ? null : () => _restore(excluded),
      ),
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
  required final VoidCallback? onMarkAsPost,
  required final VoidCallback? onEditFields,
  required final VoidCallback? onNotPost,
  required final VoidCallback? onClearMark,
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
                      enabled: onClearMark != null,
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
  required final List<Map<PostField, String>> posts,
  required final ScrollController scrollController,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
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
        for (final post in posts) _PostCard(values: post),
      ],
    );
  }
}
