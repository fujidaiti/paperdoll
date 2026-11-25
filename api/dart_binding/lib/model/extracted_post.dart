//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class ExtractedPost {
  /// Returns a new [ExtractedPost] instance.
  ExtractedPost({
    this.title,
    this.body,
    this.link,
    this.thumbnail,
    this.pubDate,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? title;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? body;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? link;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? thumbnail;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? pubDate;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ExtractedPost &&
          other.title == title &&
          other.body == body &&
          other.link == link &&
          other.thumbnail == thumbnail &&
          other.pubDate == pubDate;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (title == null ? 0 : title!.hashCode) +
      (body == null ? 0 : body!.hashCode) +
      (link == null ? 0 : link!.hashCode) +
      (thumbnail == null ? 0 : thumbnail!.hashCode) +
      (pubDate == null ? 0 : pubDate!.hashCode);

  @override
  String toString() =>
      'ExtractedPost[title=$title, body=$body, link=$link, thumbnail=$thumbnail, pubDate=$pubDate]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.title != null) {
      json[r'title'] = this.title;
    } else {
      json[r'title'] = null;
    }
    if (this.body != null) {
      json[r'body'] = this.body;
    } else {
      json[r'body'] = null;
    }
    if (this.link != null) {
      json[r'link'] = this.link;
    } else {
      json[r'link'] = null;
    }
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
    return json;
  }

  /// Returns a new [ExtractedPost] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ExtractedPost? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return ExtractedPost(
        title: mapValueOfType<String>(json, r'title'),
        body: mapValueOfType<String>(json, r'body'),
        link: mapValueOfType<String>(json, r'link'),
        thumbnail: mapValueOfType<String>(json, r'thumbnail'),
        pubDate: mapValueOfType<String>(json, r'pubDate'),
      );
    }
    return null;
  }

  static List<ExtractedPost> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ExtractedPost>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ExtractedPost.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ExtractedPost> mapFromJson(dynamic json) {
    final map = <String, ExtractedPost>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ExtractedPost.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ExtractedPost-objects as value to a dart map
  static Map<String, List<ExtractedPost>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ExtractedPost>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ExtractedPost.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{};
}
