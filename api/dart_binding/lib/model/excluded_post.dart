//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class ExcludedPost {
  /// Returns a new [ExcludedPost] instance.
  ExcludedPost({
    required this.box,
    required this.link,
  });

  int box;

  /// The excluded link that excludes the post.
  String link;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ExcludedPost && other.box == box && other.link == link;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (box.hashCode) + (link.hashCode);

  @override
  String toString() => 'ExcludedPost[box=$box, link=$link]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'box'] = this.box;
    json[r'link'] = this.link;
    return json;
  }

  /// Returns a new [ExcludedPost] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ExcludedPost? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'box'),
            'Required key "ExcludedPost[box]" is missing from JSON.');
        assert(json[r'box'] != null,
            'Required key "ExcludedPost[box]" has a null value in JSON.');
        assert(json.containsKey(r'link'),
            'Required key "ExcludedPost[link]" is missing from JSON.');
        assert(json[r'link'] != null,
            'Required key "ExcludedPost[link]" has a null value in JSON.');
        return true;
      }());

      return ExcludedPost(
        box: mapValueOfType<int>(json, r'box')!,
        link: mapValueOfType<String>(json, r'link')!,
      );
    }
    return null;
  }

  static List<ExcludedPost> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ExcludedPost>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ExcludedPost.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ExcludedPost> mapFromJson(dynamic json) {
    final map = <String, ExcludedPost>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ExcludedPost.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ExcludedPost-objects as value to a dart map
  static Map<String, List<ExcludedPost>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ExcludedPost>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ExcludedPost.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'box',
    'link',
  };
}
