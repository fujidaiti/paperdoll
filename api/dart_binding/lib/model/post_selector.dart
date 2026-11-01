//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class PostSelector {
  /// Returns a new [PostSelector] instance.
  PostSelector({
    this.post = const [],
    required this.fields,
    this.exclude = const [],
  });

  /// The path of steps from the root to the post nodes.
  List<Step> post;

  PostSelectorFields fields;

  /// Glob patterns of the links of nodes that are not posts. It is left out when there are none.
  List<String> exclude;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PostSelector &&
          _deepEquality.equals(other.post, post) &&
          other.fields == fields &&
          _deepEquality.equals(other.exclude, exclude);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (post.hashCode) + (fields.hashCode) + (exclude.hashCode);

  @override
  String toString() =>
      'PostSelector[post=$post, fields=$fields, exclude=$exclude]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'post'] = this.post;
    json[r'fields'] = this.fields;
    json[r'exclude'] = this.exclude;
    return json;
  }

  /// Returns a new [PostSelector] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PostSelector? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'post'),
            'Required key "PostSelector[post]" is missing from JSON.');
        assert(json[r'post'] != null,
            'Required key "PostSelector[post]" has a null value in JSON.');
        assert(json.containsKey(r'fields'),
            'Required key "PostSelector[fields]" is missing from JSON.');
        assert(json[r'fields'] != null,
            'Required key "PostSelector[fields]" has a null value in JSON.');
        return true;
      }());

      return PostSelector(
        post: Step.listFromJson(json[r'post']),
        fields: PostSelectorFields.fromJson(json[r'fields'])!,
        exclude: json[r'exclude'] is Iterable
            ? (json[r'exclude'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<PostSelector> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PostSelector>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PostSelector.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PostSelector> mapFromJson(dynamic json) {
    final map = <String, PostSelector>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PostSelector.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PostSelector-objects as value to a dart map
  static Map<String, List<PostSelector>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PostSelector>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PostSelector.listFromJson(
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
  };
}
