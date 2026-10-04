import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/widgets/async_value_view.dart';
import 'package:paperdoll/core/ui/widgets/heading_text.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/web_clip/domain/web_clip.dart';
import 'package:paperdoll/features/web_clip/presentation/providers/web_clip_providers.dart';
import 'package:paperdoll/features/web_clip/presentation/widgets/web_clip_reader_view.dart';

/// Reader for a web clip: fetches the clip's details by its id and
/// renders its content, falling back to a placeholder when there is none.
class const WebClipReaderScreen({
  required final int id,

  /// The title already known from the reading list row, shown while the
  /// clip details are still loading so the app bar isn't blank.
  final String? initialTitle,
  super.key,
}) extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final clipAsync = ref.watch(webClipControllerProvider(id: id));
    final clip = clipAsync.asData?.value;
    final title = switch ((clip?.title, initialTitle)) {
      (final title?, _) when title.isNotEmpty => title,
      (_, final title?) when title.isNotEmpty => title,
      _ => 'Fetching…',
    };
    return Scaffold(
      key: AppDebugKey.webClipReaderScreen,
      appBar: AppBar(
        title: HeadingText(
          title,
          key: clip != null ? AppDebugKey.readerTitle(clip.title ?? '') : null,
        ),
        actions: [
          if (clip != null) _ReadingListActions(id: id, clip: clip),
          if (clip != null)
            IconButton(
              key: AppDebugKey.webClipReaderOpenOriginalButton,
              tooltip: 'Open original',
              icon: const Icon(Icons.open_in_new),
              onPressed: () => unawaited(openExternalLink(context, clip.url)),
            ),
        ],
      ),
      body: AsyncValueView<WebClip>(
        value: clipAsync,
        onRetry: () => ref.invalidate(webClipControllerProvider(id: id)),
        data: (clip) => WebClipReaderView(clip: clip),
      ),
    );
  }
}

/// Reading-list controls in the reader appbar: a bookmark toggle that removes
/// the clip from the reading list (`DELETE /reading-list/{id}`) or re-saves it
/// (`POST /reading-list {web_clip_id}`), and — once saved — an archive toggle
/// that hides the item from the list (`PATCH /reading-list/{id}`) or unarchives
/// it. A web clip opened from the reading list starts saved, so the common
/// action here is removal.
///
/// The clip's state and every mutation live in [webClipControllerProvider];
/// this widget just reflects [clip], shows snackbars, and disables both toggles
/// while a request it started is in flight so a second tap can't race it. The
/// archive button appears the instant a save is tapped (gated on `clip.saved`)
/// but stays disabled until the created item id arrives, since archiving needs
/// it.
class const _ReadingListActions({
  required final int id,
  required final WebClip clip,
}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<_ReadingListActions> createState() =>
      _ReadingListActionsState();
}

class _ReadingListActionsState extends ConsumerState<_ReadingListActions> {
  // Whether a reading-list request kicked off from these buttons is still in
  // flight. While it is, both toggles are disabled so a second tap can't race
  // the first request (which would fire the opposite mutation on stale state).
  var _busy = false;

  WebClipController get _controller =>
      ref.read(webClipControllerProvider(id: widget.id).notifier);

  @override
  Widget build(BuildContext context) {
    final clip = widget.clip;
    final archived = clip.archived ?? false;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        IconButton(
          key: AppDebugKey.webClipReaderBookmarkButton,
          tooltip: clip.saved
              ? 'Remove from reading list'
              : 'Save to reading list',
          icon: Icon(clip.saved ? Icons.bookmark : Icons.bookmark_border),
          onPressed: _busy
              ? null
              : () => unawaited(clip.saved ? _unsave() : _save()),
        ),
        // The archive toggle only applies to an item in the list, so it shows
        // together with the saved state — but it needs the server-assigned item
        // id, so it stays disabled until that arrives (and while busy).
        if (clip.saved)
          IconButton(
            key: AppDebugKey.webClipReaderArchiveButton,
            tooltip: archived ? 'Unarchive' : 'Archive',
            icon: Icon(
              archived ? Icons.unarchive_outlined : Icons.archive_outlined,
            ),
            onPressed: _busy || clip.readingListItemId == null
                ? null
                : () => unawaited(archived ? _unarchive() : _archive()),
          ),
      ],
    );
  }

  /// Confirms the action optimistically, runs the controller mutation, and
  /// keeps the buttons disabled until it settles. The controller owns the
  /// optimistic state change and its rollback; this only surfaces snackbars.
  Future<void> _run(SnackBar success, Future<void> Function() action) async {
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    messenger.showSnackBar(success);
    try {
      await action();
    } on Exception {
      messenger.showSnackBar(
        const SnackBar(content: Text('Something went wrong', maxLines: 1)),
      );
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _save() => _run(
    const SnackBar(
      key: AppDebugKey.saveToReadingListSuccessSnackBar,
      content: Text('Saved to reading list'),
    ),
    _controller.save,
  );

  Future<void> _unsave() => _run(
    const SnackBar(
      key: AppDebugKey.removeFromReadingListSuccessSnackBar,
      content: Text('Removed from reading list'),
    ),
    _controller.remove,
  );

  Future<void> _archive() => _run(
    const SnackBar(
      key: AppDebugKey.archiveSuccessSnackBar,
      content: Text('Archived'),
    ),
    _controller.archive,
  );

  Future<void> _unarchive() => _run(
    const SnackBar(
      key: AppDebugKey.unarchiveSuccessSnackBar,
      content: Text('Unarchived'),
    ),
    _controller.unarchive,
  );
}
