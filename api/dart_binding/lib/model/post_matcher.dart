//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class PostMatcher {
  /// Returns a new [PostMatcher] instance.
  PostMatcher({
    this.post = const [],
    required this.fields,
    this.exclude = const [],
  });

  /// The chain of steps from the root to the post nodes.
  List<Step> post;

  PostMatcherFields fields;

  /// Glob patterns of the links of nodes that are not posts.
  List<String> exclude;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PostMatcher &&
          _deepEquality.equals(other.post, post) &&
          other.fields == fields &&
          _deepEquality.equals(other.exclude, exclude);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (post.hashCode) + (fields.hashCode) + (exclude.hashCode);

  @override
  String toString() =>
      'PostMatcher[post=$post, fields=$fields, exclude=$exclude]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'post'] = this.post;
    json[r'fields'] = this.fields;
    json[r'exclude'] = this.exclude;
    return json;
  }

  /// Returns a new [PostMatcher] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PostMatcher? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'post'),
            'Required key "PostMatcher[post]" is missing from JSON.');
        assert(json[r'post'] != null,
            'Required key "PostMatcher[post]" has a null value in JSON.');
        assert(json.containsKey(r'fields'),
            'Required key "PostMatcher[fields]" is missing from JSON.');
        assert(json[r'fields'] != null,
            'Required key "PostMatcher[fields]" has a null value in JSON.');
        assert(json.containsKey(r'exclude'),
            'Required key "PostMatcher[exclude]" is missing from JSON.');
        assert(json[r'exclude'] != null,
            'Required key "PostMatcher[exclude]" has a null value in JSON.');
        return true;
      }());

      return PostMatcher(
        post: Step.listFromJson(json[r'post']),
        fields: PostMatcherFields.fromJson(json[r'fields'])!,
        exclude: json[r'exclude'] is Iterable
            ? (json[r'exclude'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<PostMatcher> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PostMatcher>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PostMatcher.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PostMatcher> mapFromJson(dynamic json) {
    final map = <String, PostMatcher>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PostMatcher.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PostMatcher-objects as value to a dart map
  static Map<String, List<PostMatcher>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PostMatcher>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PostMatcher.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'post',
    'fields',
    'exclude',
  };
}
