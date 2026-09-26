import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:paperdoll/features/feed/domain/post_group.dart';

part 'feed_candidate.freezed.dart';

/// A feed discovered via search that has no local id yet and can be
/// subscribed to.
///
/// When [postGroups] is non-empty, [url] is a plain HTML page with no feed,
/// and subscribing to it needs the selectors the user builds from the groups.
@freezed
abstract class FeedCandidate with _$FeedCandidate {
  const factory({
    required String url,
    required String title,
    String? siteUrl,
    String? iconUrl,
    String? description,
    @Default([]) List<PostGroup> postGroups,
  }) = _FeedCandidate;
}
