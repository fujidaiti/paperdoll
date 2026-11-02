//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class MatchedPost {
  /// Returns a new [MatchedPost] instance.
  MatchedPost({
    required this.box,
    required this.title,
    this.body,
    required this.link,
    this.thumbnail,
    this.pubDate,
    required this.values,
  });

  int box;

  int title;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? body;

  int link;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? thumbnail;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? pubDate;

  ExtractedPost values;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is MatchedPost &&
          other.box == box &&
          other.title == title &&
          other.body == body &&
          other.link == link &&
          other.thumbnail == thumbnail &&
          other.pubDate == pubDate &&
          other.values == values;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (box.hashCode) +
      (title.hashCode) +
      (body == null ? 0 : body!.hashCode) +
      (link.hashCode) +
      (thumbnail == null ? 0 : thumbnail!.hashCode) +
      (pubDate == null ? 0 : pubDate!.hashCode) +
      (values.hashCode);

  @override
  String toString() =>
      'MatchedPost[box=$box, title=$title, body=$body, link=$link, thumbnail=$thumbnail, pubDate=$pubDate, values=$values]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'box'] = this.box;
    json[r'title'] = this.title;
    if (this.body != null) {
      json[r'body'] = this.body;
    } else {
      json[r'body'] = null;
    }
    json[r'link'] = this.link;
    if (this.thumbnail != null) {
      json[r'thumbnail'] = this.thumbnail;
    } else {
      json[r'thumbnail'] = null;
    }
    if (this.pubDate != null) {
      json[r'pubDate'] = this.pubDate;
    } else {
      json[r'pubDate'] = null;
    }
    json[r'values'] = this.values;
    return json;
  }

  /// Returns a new [MatchedPost] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static MatchedPost? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'box'),
            'Required key "MatchedPost[box]" is missing from JSON.');
        assert(json[r'box'] != null,
            'Required key "MatchedPost[box]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "MatchedPost[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "MatchedPost[title]" has a null value in JSON.');
        assert(json.containsKey(r'link'),
            'Required key "MatchedPost[link]" is missing from JSON.');
        assert(json[r'link'] != null,
            'Required key "MatchedPost[link]" has a null value in JSON.');
        assert(json.containsKey(r'values'),
            'Required key "MatchedPost[values]" is missing from JSON.');
        assert(json[r'values'] != null,
            'Required key "MatchedPost[values]" has a null value in JSON.');
        return true;
      }());

      return MatchedPost(
        box: mapValueOfType<int>(json, r'box')!,
        title: mapValueOfType<int>(json, r'title')!,
        body: mapValueOfType<int>(json, r'body'),
        link: mapValueOfType<int>(json, r'link')!,
        thumbnail: mapValueOfType<int>(json, r'thumbnail'),
        pubDate: mapValueOfType<int>(json, r'pubDate'),
        values: ExtractedPost.fromJson(json[r'values'])!,
      );
    }
    return null;
  }

  static List<MatchedPost> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <MatchedPost>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = MatchedPost.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, MatchedPost> mapFromJson(dynamic json) {
    final map = <String, MatchedPost>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = MatchedPost.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of MatchedPost-objects as value to a dart map
  static Map<String, List<MatchedPost>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<MatchedPost>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = MatchedPost.listFromJson(
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
    'title',
    'link',
    'values',
  };
}
