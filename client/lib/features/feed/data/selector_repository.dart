import 'package:dio/dio.dart';
import 'package:paperdoll/core/network/request_runner.dart';
import 'package:paperdoll/features/feed/domain/post_selector.dart';
import 'package:paperdoll/features/feed/domain/semantic_tree.dart';

/// The semantic tree of the page of an unstructured feed and the post
/// selectors saved for it, which are empty when none are saved.
typedef FeedSelectors = ({SemanticTree tree, List<PostSelector> selectors});

/// Reads and saves the post selectors of unstructured feeds. Selectors are
/// shared by all users, as feeds are.
class const SelectorRepository(final Dio _dio) {
  /// `GET /feeds/{id}/selector` → the semantic tree of the page of the feed,
  /// built by the server, and its saved selectors.
  Future<FeedSelectors> get(int feedId) {
    return runRequest(() async {
      final res = await _dio.get<Map<String, dynamic>>(
        '/feeds/$feedId/selector',
      );
      final body = res.data!;
      return (
        tree: SemanticTree.fromJson(
          body['semantic_tree'] as Map<String, Object?>,
        ),
        selectors: [
          for (final m in body['selectors'] as List<Object?>? ?? const [])
            PostSelector.fromJson(m! as Map<String, Object?>),
        ],
      );
    });
  }

  /// `PUT /feeds/{id}/selector` → replaces the selectors of the feed.
  Future<void> save(int feedId, List<PostSelector> selectors) {
    return runRequest(
      () => _dio.put<void>(
        '/feeds/$feedId/selector',
        data: {
          'selectors': [for (final m in selectors) m.toJson()],
        },
      ),
    );
  }
}
