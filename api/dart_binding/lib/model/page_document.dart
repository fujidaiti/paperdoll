//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class PageDocument {
  /// Returns a new [PageDocument] instance.
  PageDocument({
    required this.url,
    required this.title,
    required this.root,
  });

  String url;

  String title;

  PageNode root;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PageDocument &&
          other.url == url &&
          other.title == title &&
          other.root == root;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (url.hashCode) + (title.hashCode) + (root.hashCode);

  @override
  String toString() => 'PageDocument[url=$url, title=$title, root=$root]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'url'] = this.url;
    json[r'title'] = this.title;
    json[r'root'] = this.root;
    return json;
  }

  /// Returns a new [PageDocument] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PageDocument? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'url'),
            'Required key "PageDocument[url]" is missing from JSON.');
        assert(json[r'url'] != null,
            'Required key "PageDocument[url]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "PageDocument[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "PageDocument[title]" has a null value in JSON.');
        assert(json.containsKey(r'root'),
            'Required key "PageDocument[root]" is missing from JSON.');
        assert(json[r'root'] != null,
            'Required key "PageDocument[root]" has a null value in JSON.');
        return true;
      }());

      return PageDocument(
        url: mapValueOfType<String>(json, r'url')!,
        title: mapValueOfType<String>(json, r'title')!,
        root: PageNode.fromJson(json[r'root'])!,
      );
    }
    return null;
  }

  static List<PageDocument> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PageDocument>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PageDocument.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PageDocument> mapFromJson(dynamic json) {
    final map = <String, PageDocument>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PageDocument.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PageDocument-objects as value to a dart map
  static Map<String, List<PageDocument>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PageDocument>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PageDocument.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'url',
    'title',
    'root',
  };
}
