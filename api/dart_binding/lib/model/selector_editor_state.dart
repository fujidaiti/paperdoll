//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class SelectorEditorState {
  /// Returns a new [SelectorEditorState] instance.
  SelectorEditorState({
    this.matchedPosts = const [],
    this.excludedPosts = const [],
    this.preview = const [],
  });

  /// The posts that the selectors find, in document order.
  List<MatchedPost> matchedPosts;

  /// The nodes that would be posts if their links were not excluded, in document order.
  List<ExcludedPost> excludedPosts;

  /// The posts extracted from the page, merged by link, as the polling job will save them.
  List<ExtractedPost> preview;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SelectorEditorState &&
          _deepEquality.equals(other.matchedPosts, matchedPosts) &&
          _deepEquality.equals(other.excludedPosts, excludedPosts) &&
          _deepEquality.equals(other.preview, preview);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (matchedPosts.hashCode) + (excludedPosts.hashCode) + (preview.hashCode);

  @override
  String toString() =>
      'SelectorEditorState[matchedPosts=$matchedPosts, excludedPosts=$excludedPosts, preview=$preview]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'matched_posts'] = this.matchedPosts;
    json[r'excluded_posts'] = this.excludedPosts;
    json[r'preview'] = this.preview;
    return json;
  }

  /// Returns a new [SelectorEditorState] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SelectorEditorState? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'matched_posts'),
            'Required key "SelectorEditorState[matched_posts]" is missing from JSON.');
        assert(json[r'matched_posts'] != null,
            'Required key "SelectorEditorState[matched_posts]" has a null value in JSON.');
        assert(json.containsKey(r'excluded_posts'),
            'Required key "SelectorEditorState[excluded_posts]" is missing from JSON.');
        assert(json[r'excluded_posts'] != null,
            'Required key "SelectorEditorState[excluded_posts]" has a null value in JSON.');
        assert(json.containsKey(r'preview'),
            'Required key "SelectorEditorState[preview]" is missing from JSON.');
        assert(json[r'preview'] != null,
            'Required key "SelectorEditorState[preview]" has a null value in JSON.');
        return true;
      }());

      return SelectorEditorState(
        matchedPosts: MatchedPost.listFromJson(json[r'matched_posts']),
        excludedPosts: ExcludedPost.listFromJson(json[r'excluded_posts']),
        preview: ExtractedPost.listFromJson(json[r'preview']),
      );
    }
    return null;
  }

  static List<SelectorEditorState> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SelectorEditorState>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SelectorEditorState.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SelectorEditorState> mapFromJson(dynamic json) {
    final map = <String, SelectorEditorState>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SelectorEditorState.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SelectorEditorState-objects as value to a dart map
  static Map<String, List<SelectorEditorState>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SelectorEditorState>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SelectorEditorState.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'matched_posts',
    'excluded_posts',
    'preview',
  };
}
