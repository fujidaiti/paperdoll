import 'package:freezed_annotation/freezed_annotation.dart';

part 'post_group.freezed.dart';

/// A list of post-like elements the server found in a plain HTML page that
/// has no feed. The user picks which of its candidates hold the post
/// attributes.
@freezed
abstract class PostGroup with _$PostGroup {
  const factory({
    /// Unique inside one search response only. Never sent back.
    required int id,

    /// CSS selector matching the items. Sent back as `PostSelectors.root`.
    required String selector,

    /// How many items the group holds in the page.
    required int count,

    /// How many items each [AttributeCandidate.values] describes. The sample
    /// always covers every candidate of the group, so a row is never empty in
    /// all of the sampled items.
    required int sampled,
    required List<AttributeCandidate> links,
    required List<AttributeCandidate> texts,
    required List<AttributeCandidate> images,
  }) = _PostGroup;
}

/// One relative selector inside a [PostGroup] item, with the values it
/// produces in the sampled items.
@freezed
abstract class AttributeCandidate with _$AttributeCandidate {
  const factory({
    /// Relative to the item. `:scope` means the item itself.
    required String selector,

    /// Exactly `PostGroup.sampled` entries, in item order. An entry with a
    /// null value means the selector reaches nothing in that item, which
    /// happens for a selector only some items of the group carry.
    required List<AttributeValue> values,
  }) = _AttributeCandidate;
}

@freezed
abstract class AttributeValue with _$AttributeValue {
  const factory({
    /// Text, resolved href, or resolved image URL. Null when the selector
    /// reaches nothing in that item.
    String? value,

    /// Images only.
    String? alt,
  }) = _AttributeValue;
}
