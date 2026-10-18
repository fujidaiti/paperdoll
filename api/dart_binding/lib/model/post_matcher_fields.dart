//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class PostMatcherFields {
  /// Returns a new [PostMatcherFields] instance.
  PostMatcherFields({
    this.title = const [],
    this.body = const [],
    this.link = const [],
    this.thumbnail = const [],
    this.pubDate = const [],
  });

  List<FieldStep> title;

  List<FieldStep> body;

  List<FieldStep> link;

  List<FieldStep> thumbnail;

  List<FieldStep> pubDate;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PostMatcherFields &&
          _deepEquality.equals(other.title, title) &&
          _deepEquality.equals(other.body, body) &&
          _deepEquality.equals(other.link, link) &&
          _deepEquality.equals(other.thumbnail, thumbnail) &&
          _deepEquality.equals(other.pubDate, pubDate);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (title.hashCode) +
      (body.hashCode) +
      (link.hashCode) +
      (thumbnail.hashCode) +
      (pubDate.hashCode);

  @override
  String toString() =>
      'PostMatcherFields[title=$title, body=$body, link=$link, thumbnail=$thumbnail, pubDate=$pubDate]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'title'] = this.title;
    json[r'body'] = this.body;
    json[r'link'] = this.link;
    json[r'thumbnail'] = this.thumbnail;
    json[r'pubDate'] = this.pubDate;
    return json;
  }

  /// Returns a new [PostMatcherFields] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PostMatcherFields? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return PostMatcherFields(
        title: FieldStep.listFromJson(json[r'title']),
        body: FieldStep.listFromJson(json[r'body']),
        link: FieldStep.listFromJson(json[r'link']),
        thumbnail: FieldStep.listFromJson(json[r'thumbnail']),
        pubDate: FieldStep.listFromJson(json[r'pubDate']),
      );
    }
    return null;
  }

  static List<PostMatcherFields> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PostMatcherFields>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PostMatcherFields.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PostMatcherFields> mapFromJson(dynamic json) {
    final map = <String, PostMatcherFields>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PostMatcherFields.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PostMatcherFields-objects as value to a dart map
  static Map<String, List<PostMatcherFields>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PostMatcherFields>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PostMatcherFields.listFromJson(
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
