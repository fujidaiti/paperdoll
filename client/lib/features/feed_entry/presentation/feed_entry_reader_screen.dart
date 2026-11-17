import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/widgets/async_value_view.dart';
import 'package:paperdoll/core/ui/widgets/heading_text.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed_entry/domain/feed_entry.dart';
import 'package:paperdoll/features/feed_entry/presentation/providers/feed_entry_providers.dart';
import 'package:paperdoll/features/feed_entry/presentation/widgets/feed_entry_reader_view.dart';

class const FeedEntryReaderScreen({required final int id, super.key})
    extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final entryAsync = ref.watch(feedEntryControllerProvider(id: id));
    final entry = entryAsync.asData?.value;
    return Scaffold(
      key: AppDebugKey.feedEntryReaderScreen,
      appBar: AppBar(
        title: HeadingText(
          entry?.title ?? '',
          key: entry != null ? AppDebugKey.readerTitle(entry.title) : null,
        ),
        actions: [
          if (entry != null) _ReadingListActions(id: id, entry: entry),
          if (entry != null)
            IconButton(
              key: AppDebugKey.feedEntryReaderOpenOriginalButton,
              tooltip: 'Open original',
              icon: const Icon(Icons.open_in_new),
              onPressed: () => unawaited(openExternalLink(context, entry.url)),
            ),
        ],
      ),
      body: AsyncValueView<FeedEntry>(
        value: entryAsync,
        onRetry: () => ref.invalidate(feedEntryControllerProvider(id: id)),
        data: (entry) => FeedEntryReaderView(entry: entry),
      ),
    );
  }
}

/// Reading-list controls in the reader appbar: a bookmark toggle that saves the
/// entry to the reading list (`POST /reading-list`) or removes it
/// (`DELETE /reading-list/{id}`), and — once saved — an archive toggle that
/// hides the item from the list (`PATCH /reading-list/{id}`) or unarchives it.
///
/// The entry's state and every mutation live in [feedEntryControllerProvider];
/// this widget just reflects [entry], shows snackbars, and disables both
/// toggles while a request it started is in flight so a second tap can't race
/// it. The archive button appears the instant a save is tapped (gated on
/// `entry.saved`) but stays disabled until the created item id arrives, since
/// archiving needs it.
class const _ReadingListActions({
  required final int id,
  required final FeedEntry entry,
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

  FeedEntryController get _controller =>
      ref.read(feedEntryControllerProvider(id: widget.id).notifier);

  @override
  Widget build(BuildContext context) {
    final entry = widget.entry;
    final archived = entry.archived ?? false;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        IconButton(
          key: AppDebugKey.feedEntryReaderBookmarkButton,
          tooltip: entry.saved
              ? 'Remove from reading list'
              : 'Save to reading list',
          icon: Icon(entry.saved ? Icons.bookmark : Icons.bookmark_border),
          onPressed: _busy
              ? null
              : () => unawaited(entry.saved ? _unsave() : _save()),
        ),
        // The archive toggle only applies to an item in the list, so it shows
        // together with the saved state — but it needs the server-assigned item
        // id, so it stays disabled until that arrives (and while busy).
        if (entry.saved)
          IconButton(
            key: AppDebugKey.feedEntryReaderArchiveButton,
            tooltip: archived ? 'Unarchive' : 'Archive',
            icon: Icon(
              archived ? Icons.unarchive_outlined : Icons.archive_outlined,
            ),
            onPressed: _busy || entry.readingListItemId == null
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
