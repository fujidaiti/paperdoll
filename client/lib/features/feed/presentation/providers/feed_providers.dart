import 'dart:async';

import 'package:openapi/api.dart' as api;
import 'package:paperdoll/core/network/dio_provider.dart';
import 'package:paperdoll/core/pagination/paged_state.dart';
import 'package:paperdoll/features/feed/data/feed_repository_impl.dart';
import 'package:paperdoll/features/feed/domain/feed.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed/domain/feed_repository.dart';
import 'package:paperdoll/features/feed/domain/post_selection.dart';
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

@riverpod
Future<Feed> feedDetail(Ref ref, {required int id}) =>
    ref.watch(feedRepositoryProvider).getFeed(id);

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

/// The user's answers while subscribing to a plain HTML page: which post
/// groups are ticked, and a [PostSelection] per group.
typedef SubscriptionDraftState = ({
  Set<int> ticked,
  Map<int, PostSelection> selections,
});

/// Holds the choices of the web page subscription flow for as long as its
/// screens are open. Ticking and unticking a group never touches its
/// selection, so the choices survive both.
@riverpod
class SubscriptionDraft extends _$SubscriptionDraft {
  @override
  SubscriptionDraftState build(FeedCandidate candidate) => (
    ticked: const {},
    selections: {
      for (final group in candidate.postGroups)
        // A group rarely offers more than one link candidate, so the first
        // one is selected up front and the user only confirms it when there
        // is a choice to make.
        group.id: PostSelection(
          groupId: group.id,
          link: group.links.firstOrNull?.selector,
        ),
    },
  );

  /// Ticks or unticks a group. A group with no link candidate cannot be
  /// ticked, because a post with no URL cannot be stored.
  void toggle(int groupId) {
    final group = candidate.postGroups.firstWhere((g) => g.id == groupId);
    if (group.links.isEmpty) {
      return;
    }
    final ticked = {...state.ticked};
    if (!ticked.remove(groupId)) {
      ticked.add(groupId);
    }
    state = (ticked: ticked, selections: state.selections);
  }

  /// Chooses [selector] for [attribute] of a group, or clears it when null.
  void select(int groupId, PostAttribute attribute, String? selector) {
    state = (
      ticked: state.ticked,
      selections: {
        ...state.selections,
        groupId: state.selections[groupId]!.withSelector(attribute, selector),
      },
    );
  }

  /// The `selectors` of the `PUT /feeds` request: one entry per ticked
  /// group, in group order.
  List<api.PostSelectors> request() => [
    for (final group in candidate.postGroups)
      if (state.ticked.contains(group.id))
        state.selections[group.id]!.toApi(group.selector),
  ];

  /// Subscribes to the page with the current choices. Throws a domain error
  /// on failure, leaving the draft untouched; on success the Feeds list is
  /// invalidated.
  Future<void> subscribe() async {
    await ref
        .read(feedRepositoryProvider)
        .subscribe(candidate.url, selectors: request());
    ref.invalidate(feedsProvider);
  }
}
