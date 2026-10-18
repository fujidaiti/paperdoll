import 'package:freezed_annotation/freezed_annotation.dart';

part 'feed.freezed.dart';

/// An RSS source, or an unstructured feed: a web page without an RSS feed,
/// whose posts are extracted with post matchers, and whose [url] and [siteUrl]
/// are both the page's URL. The source itself is shared across all users;
/// [subscribed] is the current user's own relation to it.
@freezed
abstract class Feed with _$Feed {
  const factory({
    required int id,
    required String url,
    required String title,
    required bool subscribed,

    /// Whether post matchers are saved for this feed, which only an
    /// unstructured feed can have.
    required bool hasMatchers,
    String? siteUrl,
    String? iconUrl,
    String? description,
  }) = _Feed;
}
