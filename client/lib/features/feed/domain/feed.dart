import 'package:freezed_annotation/freezed_annotation.dart';

part 'feed.freezed.dart';

/// An RSS source. The source itself is shared across all users; [subscribed]
/// is the current user's own relation to it.
@freezed
abstract class Feed with _$Feed {
  const factory({
    required int id,
    required String url,
    required String title,
    required bool subscribed,
    String? siteUrl,
    String? iconUrl,
    String? description,
  }) = _Feed;
}
