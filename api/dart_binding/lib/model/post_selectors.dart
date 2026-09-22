//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class PostSelectors {
  /// Returns a new [PostSelectors] instance.
  PostSelectors({
    required this.root,
    required this.link,
    this.title,
    this.description,
    this.image,
    this.timestamp,
  });

  /// CSS selector matching the items, taken from `PostGroup.selector`.
  String root;

  /// Selector for the post URL, relative to an item. This one is required, and an item where it matches nothing is skipped at polling time.
  String link;

  /// Selector for the post title, relative to an item.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? title;

  /// Selector for the post description, relative to an item.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? description;

  /// Selector for the post thumbnail, relative to an item.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? image;

  /// Selector for the post publication date, relative to an item.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? timestamp;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PostSelectors &&
          other.root == root &&
          other.link == link &&
          other.title == title &&
          other.description == description &&
          other.image == image &&
          other.timestamp == timestamp;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (root.hashCode) +
      (link.hashCode) +
      (title == null ? 0 : title!.hashCode) +
      (description == null ? 0 : description!.hashCode) +
      (image == null ? 0 : image!.hashCode) +
      (timestamp == null ? 0 : timestamp!.hashCode);

  @override
  String toString() =>
      'PostSelectors[root=$root, link=$link, title=$title, description=$description, image=$image, timestamp=$timestamp]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'root'] = this.root;
    json[r'link'] = this.link;
    if (this.title != null) {
      json[r'title'] = this.title;
    } else {
      json[r'title'] = null;
    }
    if (this.description != null) {
      json[r'description'] = this.description;
    } else {
      json[r'description'] = null;
    }
    if (this.image != null) {
      json[r'image'] = this.image;
    } else {
      json[r'image'] = null;
    }
    if (this.timestamp != null) {
      json[r'timestamp'] = this.timestamp;
    } else {
      json[r'timestamp'] = null;
    }
    return json;
  }

  /// Returns a new [PostSelectors] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PostSelectors? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'root'),
            'Required key "PostSelectors[root]" is missing from JSON.');
        assert(json[r'root'] != null,
            'Required key "PostSelectors[root]" has a null value in JSON.');
        assert(json.containsKey(r'link'),
            'Required key "PostSelectors[link]" is missing from JSON.');
        assert(json[r'link'] != null,
            'Required key "PostSelectors[link]" has a null value in JSON.');
        return true;
      }());

      return PostSelectors(
        root: mapValueOfType<String>(json, r'root')!,
        link: mapValueOfType<String>(json, r'link')!,
        title: mapValueOfType<String>(json, r'title'),
        description: mapValueOfType<String>(json, r'description'),
        image: mapValueOfType<String>(json, r'image'),
        timestamp: mapValueOfType<String>(json, r'timestamp'),
      );
    }
    return null;
  }

  static List<PostSelectors> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PostSelectors>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PostSelectors.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PostSelectors> mapFromJson(dynamic json) {
    final map = <String, PostSelectors>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PostSelectors.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PostSelectors-objects as value to a dart map
  static Map<String, List<PostSelectors>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PostSelectors>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PostSelectors.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'root',
    'link',
  };
}
