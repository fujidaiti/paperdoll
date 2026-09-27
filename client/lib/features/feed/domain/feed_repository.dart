import 'package:paperdoll/core/pagination/page_result.dart';
import 'package:paperdoll/features/feed/domain/feed.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed_entry/domain/feed_entry.dart';

/// Manages feed subscriptions and reads feed timelines.
///
/// The list methods take an optional `cursor` and return a [PageResult]
/// carrying the next cursor, so callers can page through the server's keyset
/// pagination.
abstract interface class FeedRepository {
  /// `GET /feeds` → a page of the subscribed feeds.
  Future<PageResult<Feed>> listFeeds({String? cursor});

  /// `GET /feeds/search?q=` → subscribable candidates.
  Future<List<FeedCandidate>> search(String query);

  /// `PUT /feeds` with `{ url }` → the subscribed feed (idempotent).
  Future<Feed> subscribe(String url);

  /// `DELETE /feeds/{id}` → drops the current user's subscription. The feed
  /// is shared across all users, so it is not deleted and its timeline stays
  /// readable.
  Future<void> unsubscribe(int id);

  /// `GET /feeds/{id}` → the feed header.
  Future<Feed> getFeed(int id);

  /// `GET /feeds/{id}/timeline` → a page of the feed's stream of entries.
  Future<PageResult<FeedEntry>> timeline(int id, {String? cursor});
}
