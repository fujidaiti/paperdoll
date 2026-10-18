import 'package:dio/dio.dart';
import 'package:paperdoll/core/network/request_runner.dart';
import 'package:paperdoll/features/page_watch/domain/page_document.dart';
import 'package:paperdoll/features/page_watch/domain/post_matcher.dart';

/// The page of an unstructured feed and the post matchers saved for it, which
/// are empty when none are saved.
typedef FeedMatchers = ({PageDocument page, List<PostMatcher> matchers});

/// Reads and saves the post matchers of unstructured feeds. Matchers are
/// shared by all users, as feeds are.
class const MatcherRepository(final Dio _dio) {
  /// `GET /feeds/{id}/matcher` → the page of the feed, simplified by the
  /// server, and its saved matchers.
  Future<FeedMatchers> get(int feedId) {
    return runRequest(() async {
      final res = await _dio.get<Map<String, dynamic>>(
        '/feeds/$feedId/matcher',
      );
      final body = res.data!;
      return (
        page: PageDocument.fromJson(body['page'] as Map<String, Object?>),
        matchers: [
          for (final m in body['matchers'] as List<Object?>? ?? const [])
            PostMatcher.fromJson(m! as Map<String, Object?>),
        ],
      );
    });
  }

  /// `PUT /feeds/{id}/matcher` → replaces the matchers of the feed.
  Future<void> save(int feedId, List<PostMatcher> matchers) {
    return runRequest(
      () => _dio.put<void>(
        '/feeds/$feedId/matcher',
        data: {
          'matchers': [for (final m in matchers) m.toJson()],
        },
      ),
    );
  }
}
