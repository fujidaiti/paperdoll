import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/error/domain_error.dart';
import 'package:paperdoll/core/pagination/infinite_scroll.dart';
import 'package:paperdoll/core/pagination/load_more_footer.dart';
import 'package:paperdoll/core/pagination/paged_state.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/app_divider.dart';
import 'package:paperdoll/core/ui/widgets/async_value_view.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/empty_placeholder.dart';
import 'package:paperdoll/core/ui/widgets/error_placeholder.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/ui/widgets/loading_indicator.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed/domain/feed.dart';
import 'package:paperdoll/features/feed/presentation/providers/feed_providers.dart';
import 'package:paperdoll/features/feed_entry/domain/feed_entry.dart';
import 'package:paperdoll/features/feed_entry/presentation/widgets/feed_entry_row.dart';

/// Feed Detail (Timeline): a feed's header and its full stream of entries.
class const FeedDetailScreen({required final int id, super.key})
    extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final feedAsync = ref.watch(feedDetailControllerProvider(id: id));
    final timelineAsync = ref.watch(feedTimelineProvider(id: id));
    return Scaffold(
      key: AppDebugKey.feedDetailScreen,
      body: RefreshIndicator(
        onRefresh: () {
          ref.invalidate(feedDetailControllerProvider(id: id));
          return ref.refresh(feedTimelineProvider(id: id).future);
        },
        child: AsyncValueView<Feed>(
          value: feedAsync,
          onRetry: () => ref.invalidate(feedDetailControllerProvider(id: id)),
          data: (feed) => _FeedDetailBody(
            feed: feed,
            timeline: timelineAsync,
            onRetryTimeline: () => ref.invalidate(feedTimelineProvider(id: id)),
            onLoadMore: () =>
                ref.read(feedTimelineProvider(id: id).notifier).loadMore(),
            onOpenEntry: (entryId) => context.pushNamed(
              routeFeedEntryReaderName,
              pathParameters: {
                'id': id.toString(),
                'feedEntryId': entryId.toString(),
              },
            ),
          ),
        ),
      ),
    );
  }
}

class const _FeedDetailBody({
  required final Feed feed,
  required final AsyncValue<PagedState<FeedEntry>> timeline,
  required final VoidCallback onRetryTimeline,
  required final VoidCallback onLoadMore,
  required final void Function(int entryId) onOpenEntry,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return InfiniteScrollList(
      onEndReached: onLoadMore,
      child: CustomScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        slivers: [
          SliverAppBar(
            pinned: true,
            centerTitle: false,
            title: BodyText(feed.title),
            actions: [_SubscriptionMenu(feed: feed)],
          ),
          SliverToBoxAdapter(child: _FeedDetails(feed: feed)),
          ...timeline.when(
            data: _timelineSlivers,
            loading: () => const [
              SliverFillRemaining(
                hasScrollBody: false,
                child: LoadingIndicator(),
              ),
            ],
            error: (error, _) => [
              SliverFillRemaining(
                hasScrollBody: false,
                child: ErrorPlaceholder(
                  message: describeError(error),
                  onRetry: onRetryTimeline,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  List<Widget> _timelineSlivers(PagedState<FeedEntry> page) {
    final entries = page.items;
    if (entries.isEmpty) {
      return const [
        SliverFillRemaining(
          hasScrollBody: false,
          child: EmptyPlaceholder(message: 'No entries yet.'),
        ),
      ];
    }
    final showFooter = page.isLoadingMore || page.loadMoreError != null;
    return [
      SliverList.separated(
        itemCount: entries.length,
        separatorBuilder: (context, index) => const AppDivider(),
        itemBuilder: (context, index) {
          final entry = entries[index];
          return FeedEntryRow(
            key: AppDebugKey.feedEntryRow(entry.title),
            entry: entry,
            onTap: () => onOpenEntry(entry.id),
          );
        },
      ),
      if (showFooter)
        SliverToBoxAdapter(
          child: LoadMoreFooter(
            isLoading: page.isLoadingMore,
            error: page.loadMoreError,
            onRetry: onLoadMore,
          ),
        ),
    ];
  }
}

/// The app bar's overflow menu, offering the single subscription action that
/// applies right now: "Unsubscribe" while subscribed, "Subscribe" once not.
///
/// The feed's state and both mutations live in [feedDetailControllerProvider];
/// this widget only reflects [feed] and shows snackbars. Unsubscribing does not
/// leave the screen — the feed is shared across all users and survives, so its
/// timeline stays readable. The menu is disabled while a request it started is
/// in flight so a second tap can't fire the opposite mutation on stale state.
class const _SubscriptionMenu({required final Feed feed})
    extends ConsumerStatefulWidget {
  @override
  ConsumerState<_SubscriptionMenu> createState() => _SubscriptionMenuState();
}

class _SubscriptionMenuState extends ConsumerState<_SubscriptionMenu> {
  var _busy = false;

  FeedDetailController get _controller =>
      ref.read(feedDetailControllerProvider(id: widget.feed.id).notifier);

  @override
  Widget build(BuildContext context) {
    final subscribed = widget.feed.subscribed;
    return PopupMenuButton<void>(
      key: AppDebugKey.feedDetailMenuButton,
      icon: const Icon(Icons.more_vert),
      itemBuilder: (context) => [
        PopupMenuItem<void>(
          key: subscribed
              ? AppDebugKey.unsubscribeMenuItem
              : AppDebugKey.subscribeMenuItem,
          enabled: !_busy,
          onTap: () => unawaited(subscribed ? _unsubscribe() : _subscribe()),
          child: Text(subscribed ? 'Unsubscribe' : 'Subscribe'),
        ),
      ],
    );
  }

  /// Shows [success] straight away, runs the controller mutation, and keeps the
  /// menu disabled until it settles. The controller owns the optimistic state
  /// change and its rollback; this only surfaces snackbars.
  ///
  /// The snackbar is hosted by the [ScaffoldMessenger], so its "Undo" action
  /// outlives this screen if the user navigates back while it is still up.
  /// Tapping it then does nothing rather than reading a disposed [ref].
  Future<void> _run(SnackBar success, Future<void> Function() action) async {
    if (!mounted) {
      return;
    }
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

  Future<void> _subscribe() => _run(
    const SnackBar(
      key: AppDebugKey.subscribeSuccessSnackBar,
      content: Text('Subscribed'),
    ),
    _controller.subscribe,
  );

  Future<void> _unsubscribe() => _run(
    SnackBar(
      key: AppDebugKey.unsubscribeSuccessSnackBar,
      persist: false,
      content: const Text('Unsubscribed'),
      action: SnackBarAction(
        label: 'Undo',
        onPressed: () => unawaited(_subscribe()),
      ),
    ),
    _controller.unsubscribe,
  );
}

/// The feed's description and a link to its site, shown below the app bar.
class const _FeedDetails({required final Feed feed}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final description = feed.description;
    final siteUrl = feed.siteUrl;
    if (description == null && siteUrl == null) {
      return const SizedBox.shrink();
    }
    return Padding(
      padding: const EdgeInsets.all(spacingMd),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (description != null) BodyText(description),
          if (siteUrl != null) ...[
            if (description != null) const Gap(spacingSm),
            Align(
              alignment: Alignment.centerRight,
              child: TextButton.icon(
                onPressed: () => unawaited(openExternalLink(context, siteUrl)),
                icon: const Icon(Icons.public),
                label: const Text('Visit site'),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
