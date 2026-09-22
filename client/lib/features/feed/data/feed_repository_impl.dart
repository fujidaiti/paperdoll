import 'package:dio/dio.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/core/network/request_runner.dart';
import 'package:paperdoll/core/pagination/page_result.dart';
import 'package:paperdoll/features/feed/domain/feed.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed/domain/feed_repository.dart';
import 'package:paperdoll/features/feed/domain/post_group.dart';
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
              postGroups: c.postGroups
                  .map(
                    (g) => PostGroup(
                      id: g.id,
                      selector: g.selector,
                      count: g.count,
                      sampled: g.sampled,
                      links: g.links.map(_toAttributeCandidate).toList(),
                      texts: g.texts.map(_toAttributeCandidate).toList(),
                      images: g.images.map(_toAttributeCandidate).toList(),
                    ),
                  )
                  .toList(),
            ),
          )
          .toList();
    });
  }

  @override
  Future<Feed> subscribe(
    String url, {
    List<api.PostSelectors> selectors = const [],
  }) {
    return runRequest(() async {
      final res = await _dio.put<Map<String, dynamic>>(
        '/feeds',
        data: {
          'url': url,
          if (selectors.isNotEmpty)
            // The generated toJson writes unset attributes as null, but the
            // API declares them as non-nullable strings, so omit them.
            'selectors': [
              for (final s in selectors)
                s.toJson()..removeWhere((_, value) => value == null),
            ],
        },
      );
      return _toFeed(api.Feed.fromJson(res.data)!);
    });
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

  AttributeCandidate _toAttributeCandidate(api.AttributeCandidate c) =>
      AttributeCandidate(
        selector: c.selector,
        matched: c.matched,
        values: c.values
            .map((v) => AttributeValue(value: v.value, alt: v.alt))
            .toList(),
      );

  Feed _toFeed(api.Feed f) => Feed(
    id: f.id,
    url: f.url,
    title: f.title,
    siteUrl: f.siteUrl,
    iconUrl: f.iconUrl,
    description: f.description,
  );
}
