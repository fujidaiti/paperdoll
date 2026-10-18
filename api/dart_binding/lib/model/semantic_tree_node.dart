//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class SemanticTreeNode {
  /// Returns a new [SemanticTreeNode] instance.
  SemanticTreeNode({
    this.kind,
    this.val,
    this.link,
    this.ch = const [],
  });

  /// `h1`..`h6` for a heading, `img` for an image, `time` for a time.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? kind;

  /// Plain text, the alt text of an image, or the RFC 3339 timestamp of a time.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? val;

  /// The link target, or the URL of an image.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? link;

  List<SemanticTreeNode> ch;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SemanticTreeNode &&
          other.kind == kind &&
          other.val == val &&
          other.link == link &&
          _deepEquality.equals(other.ch, ch);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (kind == null ? 0 : kind!.hashCode) +
      (val == null ? 0 : val!.hashCode) +
      (link == null ? 0 : link!.hashCode) +
      (ch.hashCode);

  @override
  String toString() =>
      'SemanticTreeNode[kind=$kind, val=$val, link=$link, ch=$ch]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.kind != null) {
      json[r'kind'] = this.kind;
    } else {
      json[r'kind'] = null;
    }
    if (this.val != null) {
      json[r'val'] = this.val;
    } else {
      json[r'val'] = null;
    }
    if (this.link != null) {
      json[r'link'] = this.link;
    } else {
      json[r'link'] = null;
    }
    json[r'ch'] = this.ch;
    return json;
  }

  /// Returns a new [SemanticTreeNode] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SemanticTreeNode? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return SemanticTreeNode(
        kind: mapValueOfType<String>(json, r'kind'),
        val: mapValueOfType<String>(json, r'val'),
        link: mapValueOfType<String>(json, r'link'),
        ch: SemanticTreeNode.listFromJson(json[r'ch']),
      );
    }
    return null;
  }

  static List<SemanticTreeNode> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SemanticTreeNode>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SemanticTreeNode.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SemanticTreeNode> mapFromJson(dynamic json) {
    final map = <String, SemanticTreeNode>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SemanticTreeNode.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SemanticTreeNode-objects as value to a dart map
  static Map<String, List<SemanticTreeNode>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SemanticTreeNode>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SemanticTreeNode.listFromJson(
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
