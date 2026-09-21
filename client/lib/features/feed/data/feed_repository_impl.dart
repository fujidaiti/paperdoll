import 'package:dio/dio.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/core/network/request_runner.dart';
import 'package:paperdoll/core/pagination/page_result.dart';
import 'package:paperdoll/features/feed/domain/feed.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed/domain/feed_repository.dart';
import 'package:paperdoll/features/feed_entry/domain/feed_entry.dart';

class const FeedRepositoryImpl(final Dio _dio) implements FeedRepository {
  @override
  Future<PageResult<Feed>> listFeeds({String? cursor}) {
    return runRequest(() async {
      final res = await _dio.get<Map<String, dynamic>>(
        '/feeds',
        queryParameters: cursor != null ? {'after': cursor} : null,
      );
      final body = api.GetFeeds200Response.fromJson(res.data)!;
      return PageResult(
        items: body.feeds.map(_toFeed).toList(),
        nextCursor: body.nextCursor,
      );
    });
  }

  @override
  Future<List<FeedCandidate>> search(String query) {
    return runRequest(() async {
      final res = await _dio.get<Map<String, dynamic>>(
        '/feeds/search',
        queryParameters: {'q': query},
      );
      final body = api.SearchFeeds200Response.fromJson(res.data)!;
      return body.feeds
          .map(
            (c) => FeedCandidate(
              url: c.url,
              title: c.title,
              siteUrl: c.siteUrl,
              iconUrl: c.iconUrl,
              description: c.description,
            ),
          )
          .toList();
    });
  }

  @override
  Future<Feed> subscribe(String url) {
    return runRequest(() async {
      final res = await _dio.put<Map<String, dynamic>>(
        '/feeds',
        data: {'url': url},
      );
      return _toFeed(api.Feed.fromJson(res.data)!);
    });
  }

  @override
  Future<void> unsubscribe(int id) {
    return runRequest(() => _dio.delete<void>('/feeds/$id'));
  }

  @override
  Future<Feed> getFeed(int id) {
    return runRequest(() async {
      final res = await _dio.get<Map<String, dynamic>>('/feeds/$id');
      return _toFeed(api.Feed.fromJson(res.data)!);
    });
  }

  @override
  Future<PageResult<FeedEntry>> timeline(int id, {String? cursor}) {
    return runRequest(() async {
      final res = await _dio.get<Map<String, dynamic>>(
        '/feeds/$id/timeline',
        queryParameters: cursor != null ? {'after': cursor} : null,
      );
      final body = api.GetFeedTimeline200Response.fromJson(res.data)!;
      return PageResult(
        items: body.entries
            .map(
              (e) => FeedEntry(
                id: e.id,
                feedId: e.feedId,
                url: e.url,
                title: e.title,
                description: e.description,
                content: e.content,
                publishedAt: e.publishedAt,
                snapshotAt: e.snapshotAt,
                readingListItemId: e.readLater?.id,
                archived: e.readLater?.archived,
                saved: e.readLater != null,
              ),
            )
            .toList(),
        nextCursor: body.nextCursor,
      );
    });
  }

  Feed _toFeed(api.Feed f) => Feed(
    id: f.id,
    url: f.url,
    title: f.title,
    siteUrl: f.siteUrl,
    iconUrl: f.iconUrl,
    description: f.description,
    subscribed: f.subscribed,
  );
}
