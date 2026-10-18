//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class SemanticTree {
  /// Returns a new [SemanticTree] instance.
  SemanticTree({
    required this.url,
    required this.title,
    required this.root,
  });

  String url;

  String title;

  SemanticTreeNode root;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SemanticTree &&
          other.url == url &&
          other.title == title &&
          other.root == root;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (url.hashCode) + (title.hashCode) + (root.hashCode);

  @override
  String toString() => 'SemanticTree[url=$url, title=$title, root=$root]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'url'] = this.url;
    json[r'title'] = this.title;
    json[r'root'] = this.root;
    return json;
  }

  /// Returns a new [SemanticTree] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SemanticTree? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'url'),
            'Required key "SemanticTree[url]" is missing from JSON.');
        assert(json[r'url'] != null,
            'Required key "SemanticTree[url]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "SemanticTree[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "SemanticTree[title]" has a null value in JSON.');
        assert(json.containsKey(r'root'),
            'Required key "SemanticTree[root]" is missing from JSON.');
        assert(json[r'root'] != null,
            'Required key "SemanticTree[root]" has a null value in JSON.');
        return true;
      }());

      return SemanticTree(
        url: mapValueOfType<String>(json, r'url')!,
        title: mapValueOfType<String>(json, r'title')!,
        root: SemanticTreeNode.fromJson(json[r'root'])!,
      );
    }
    return null;
  }

  static List<SemanticTree> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SemanticTree>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SemanticTree.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SemanticTree> mapFromJson(dynamic json) {
    final map = <String, SemanticTree>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SemanticTree.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SemanticTree-objects as value to a dart map
  static Map<String, List<SemanticTree>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SemanticTree>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SemanticTree.listFromJson(
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
