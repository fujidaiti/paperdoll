//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class UpdateSelectorEditorMarksRequest {
  /// Returns a new [UpdateSelectorEditorMarksRequest] instance.
  UpdateSelectorEditorMarksRequest({
    this.labeledPosts = const [],
    this.excludedLinks = const [],
    this.excludedNodes = const [],
  });

  List<LabeledPost> labeledPosts;

  /// The host and path of each excluded link.
  List<String> excludedLinks;

  /// The node ID of each node that is excluded with its subtree. An excluded node wins over the other marks, so a labeled post or a field in an excluded node is ignored.
  List<int> excludedNodes;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is UpdateSelectorEditorMarksRequest &&
          _deepEquality.equals(other.labeledPosts, labeledPosts) &&
          _deepEquality.equals(other.excludedLinks, excludedLinks) &&
          _deepEquality.equals(other.excludedNodes, excludedNodes);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (labeledPosts.hashCode) +
      (excludedLinks.hashCode) +
      (excludedNodes.hashCode);

  @override
  String toString() =>
      'UpdateSelectorEditorMarksRequest[labeledPosts=$labeledPosts, excludedLinks=$excludedLinks, excludedNodes=$excludedNodes]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'labeled_posts'] = this.labeledPosts;
    json[r'excluded_links'] = this.excludedLinks;
    json[r'excluded_nodes'] = this.excludedNodes;
    return json;
  }

  /// Returns a new [UpdateSelectorEditorMarksRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static UpdateSelectorEditorMarksRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'labeled_posts'),
            'Required key "UpdateSelectorEditorMarksRequest[labeled_posts]" is missing from JSON.');
        assert(json[r'labeled_posts'] != null,
            'Required key "UpdateSelectorEditorMarksRequest[labeled_posts]" has a null value in JSON.');
        assert(json.containsKey(r'excluded_links'),
            'Required key "UpdateSelectorEditorMarksRequest[excluded_links]" is missing from JSON.');
        assert(json[r'excluded_links'] != null,
            'Required key "UpdateSelectorEditorMarksRequest[excluded_links]" has a null value in JSON.');
        assert(json.containsKey(r'excluded_nodes'),
            'Required key "UpdateSelectorEditorMarksRequest[excluded_nodes]" is missing from JSON.');
        assert(json[r'excluded_nodes'] != null,
            'Required key "UpdateSelectorEditorMarksRequest[excluded_nodes]" has a null value in JSON.');
        return true;
      }());

      return UpdateSelectorEditorMarksRequest(
        labeledPosts: LabeledPost.listFromJson(json[r'labeled_posts']),
        excludedLinks: json[r'excluded_links'] is Iterable
            ? (json[r'excluded_links'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        excludedNodes: json[r'excluded_nodes'] is Iterable
            ? (json[r'excluded_nodes'] as Iterable)
                .cast<int>()
                .toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<UpdateSelectorEditorMarksRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <UpdateSelectorEditorMarksRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = UpdateSelectorEditorMarksRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, UpdateSelectorEditorMarksRequest> mapFromJson(
      dynamic json) {
    final map = <String, UpdateSelectorEditorMarksRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = UpdateSelectorEditorMarksRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of UpdateSelectorEditorMarksRequest-objects as value to a dart map
  static Map<String, List<UpdateSelectorEditorMarksRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<UpdateSelectorEditorMarksRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = UpdateSelectorEditorMarksRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'labeled_posts',
    'excluded_links',
    'excluded_nodes',
  };
}
