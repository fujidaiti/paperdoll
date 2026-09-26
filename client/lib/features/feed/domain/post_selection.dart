import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/features/feed/domain/post_group.dart';

part 'post_selection.freezed.dart';

/// A post attribute the user assigns to one [AttributeCandidate] of a group.
enum PostAttribute {
  link,
  title,
  description,
  timestamp,
  image;

  /// The candidates of [group] this attribute is chosen from.
  List<AttributeCandidate> candidatesOf(PostGroup group) => switch (this) {
    link => group.links,
    title || description || timestamp => group.texts,
    image => group.images,
  };
}

/// The user's answer for one [PostGroup]: the selector chosen for each
/// attribute, or null when none is chosen.
@freezed
abstract class PostSelection with _$PostSelection {
  const factory({
    required int groupId,
    String? link,
    String? title,
    String? description,
    String? image,
    String? timestamp,

    /// Whether the user has answered any attribute other than the link, even
    /// with "None of them". Until then the group preview shows a guess.
    @Default(false) bool answered,
  }) = _PostSelection;
}

extension PostSelectionAttributes on PostSelection {
  String? selectorOf(PostAttribute attribute) => switch (attribute) {
    PostAttribute.link => link,
    PostAttribute.title => title,
    PostAttribute.description => description,
    PostAttribute.timestamp => timestamp,
    PostAttribute.image => image,
  };

  PostSelection withSelector(PostAttribute attribute, String? selector) =>
      switch (attribute) {
        PostAttribute.link => copyWith(link: selector),
        PostAttribute.title => copyWith(title: selector, answered: true),
        PostAttribute.description => copyWith(
          description: selector,
          answered: true,
        ),
        PostAttribute.timestamp => copyWith(
          timestamp: selector,
          answered: true,
        ),
        PostAttribute.image => copyWith(image: selector, answered: true),
      };

  /// The request entry for this group. [root] is `PostGroup.selector`.
  api.PostSelectors toApi(String root) => api.PostSelectors(
    root: root,
    link: link!,
    title: title,
    description: description,
    image: image,
    timestamp: timestamp,
  );
}
