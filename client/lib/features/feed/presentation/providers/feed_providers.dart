import 'dart:async';

import 'package:paperdoll/core/network/dio_provider.dart';
import 'package:paperdoll/core/pagination/paged_state.dart';
import 'package:paperdoll/features/feed/data/feed_repository_impl.dart';
import 'package:paperdoll/features/feed/domain/feed.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed/domain/feed_repository.dart';
import 'package:paperdoll/features/feed_entry/domain/feed_entry.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'feed_providers.g.dart';

@riverpod
FeedRepository feedRepository(Ref ref) =>
    FeedRepositoryImpl(ref.watch(dioProvider));

/// The subscribed feeds, paginated. [build] loads the first page; [loadMore]
/// appends the next.
@riverpod
class Feeds extends _$Feeds {
  @override
  Future<PagedState<Feed>> build() async =>
      PagedState.first(await ref.watch(feedRepositoryProvider).listFeeds());

  Future<void> loadMore() => appendNextPage(
    read: () => state,
    write: (next) => state = next,
    fetch: (cursor) =>
        ref.read(feedRepositoryProvider).listFeeds(cursor: cursor),
  );
}

/// Single source of truth for one feed in the timeline screen. It loads the
/// feed header and owns both subscription mutations: each one flips
/// [Feed.subscribed] on the cached feed optimistically so the app bar menu
/// updates instantly, fires the request, and rolls the state back before
/// rethrowing if it fails.
///
/// Neither mutation touches [feedsProvider]. The feed list keeps its cached
/// page, so an unsubscribed feed disappears from it only once the user reloads
/// that screen.
@riverpod
class FeedDetailController extends _$FeedDetailController {
  late int _id;

  @override
  Future<Feed> build({required int id}) {
    _id = id;
    return ref.watch(feedRepositoryProvider).getFeed(id);
  }

  /// Subscribes to the feed. `PUT /feeds` is idempotent and keyed by URL, so
  /// this is also how a previously unsubscribed feed is subscribed to again.
  Future<void> subscribe() async {
    final feed = state.asData?.value;
    if (feed == null || feed.subscribed) {
      return;
    }
    state = AsyncData(feed.copyWith(subscribed: true));
    try {
      await ref.read(feedRepositoryProvider).subscribe(feed.url);
    } on Exception {
      final current = state.asData?.value;
      if (current != null) {
        state = AsyncData(current.copyWith(subscribed: false));
      }
      rethrow;
    }
  }

  /// Drops the subscription. The feed itself is shared across all users and
  /// survives, so the timeline stays readable afterwards.
  Future<void> unsubscribe() async {
    final feed = state.asData?.value;
    if (feed == null || !feed.subscribed) {
      return;
    }
    state = AsyncData(feed.copyWith(subscribed: false));
    try {
      await ref.read(feedRepositoryProvider).unsubscribe(_id);
    } on Exception {
      final current = state.asData?.value;
      if (current != null) {
        state = AsyncData(current.copyWith(subscribed: true));
      }
      rethrow;
    }
  }
}

/// A feed's timeline entries, paginated. [build] loads the first page;
/// [loadMore] appends the next.
@riverpod
class FeedTimeline extends _$FeedTimeline {
  @override
  Future<PagedState<FeedEntry>> build({required int id}) async =>
      PagedState.first(await ref.watch(feedRepositoryProvider).timeline(id));

  Future<void> loadMore() => appendNextPage(
    read: () => state,
    write: (next) => state = next,
    fetch: (cursor) =>
        ref.read(feedRepositoryProvider).timeline(id, cursor: cursor),
  );
}

/// Drives the Feed Search / Subscribe screen. [search] populates the candidate
/// list; [subscribe] adds a feed and refreshes the Feeds list.
@riverpod
class FeedSearchController extends _$FeedSearchController {
  @override
  FutureOr<List<FeedCandidate>> build() => const [];

  Future<void> search(String query) async {
    state = const AsyncLoading<List<FeedCandidate>>();
    state = await AsyncValue.guard(
      () => ref.read(feedRepositoryProvider).search(query),
    );
  }

  /// Subscribes to the given url. Throws a domain error on failure so the
  /// caller can surface a snackbar; on success the Feeds list is invalidated.
  Future<void> subscribe(String url) async {
    await ref.read(feedRepositoryProvider).subscribe(url);
    ref.invalidate(feedsProvider);
  }
}
