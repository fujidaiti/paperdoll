//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class SelectorEditor {
  /// Returns a new [SelectorEditor] instance.
  SelectorEditor({
    required this.id,
    required this.expiresAt,
    required this.page,
    this.markedPosts = const [],
    this.excludedLinks = const [],
    this.excludedNodes = const [],
    this.foundPosts = const [],
    this.excludedPosts = const [],
    this.preview = const [],
  });

  int id;

  DateTime expiresAt;

  /// The page simplified into an HTML document, with the node ID of each element in its `data-id` attribute. The node ID of a node is its position among the nodes of the page in document order, and a text takes the ID of its position too. See SimplifyPage in `server/feature/feed/simplify.go` for the format.
  String page;

  /// The posts marked so far. A new editor starts with the posts that the saved selectors find.
  List<MarkedPost> markedPosts;

  /// The host and path of each excluded link. A new editor starts with the ones of the saved selectors.
  List<String> excludedLinks;

  /// The node ID of each node that is excluded with its subtree. No post and no field is found in an excluded node, even when it is marked. A new editor starts with the nodes of the saved selectors that the page still has.
  List<int> excludedNodes;

  /// The posts that the selectors find, in document order.
  List<FoundPost> foundPosts;

  /// The nodes that would be posts if their links were not excluded, in document order.
  List<ExcludedPost> excludedPosts;

  /// The posts extracted from the page, merged by link, as the polling job will save them.
  List<ExtractedPost> preview;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SelectorEditor &&
          other.id == id &&
          other.expiresAt == expiresAt &&
          other.page == page &&
          _deepEquality.equals(other.markedPosts, markedPosts) &&
          _deepEquality.equals(other.excludedLinks, excludedLinks) &&
          _deepEquality.equals(other.excludedNodes, excludedNodes) &&
          _deepEquality.equals(other.foundPosts, foundPosts) &&
          _deepEquality.equals(other.excludedPosts, excludedPosts) &&
          _deepEquality.equals(other.preview, preview);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (expiresAt.hashCode) +
      (page.hashCode) +
      (markedPosts.hashCode) +
      (excludedLinks.hashCode) +
      (excludedNodes.hashCode) +
      (foundPosts.hashCode) +
      (excludedPosts.hashCode) +
      (preview.hashCode);

  @override
  String toString() =>
      'SelectorEditor[id=$id, expiresAt=$expiresAt, page=$page, markedPosts=$markedPosts, excludedLinks=$excludedLinks, excludedNodes=$excludedNodes, foundPosts=$foundPosts, excludedPosts=$excludedPosts, preview=$preview]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'expires_at'] = this.expiresAt.toUtc().toIso8601String();
    json[r'page'] = this.page;
    json[r'marked_posts'] = this.markedPosts;
    json[r'excluded_links'] = this.excludedLinks;
    json[r'excluded_nodes'] = this.excludedNodes;
    json[r'found_posts'] = this.foundPosts;
    json[r'excluded_posts'] = this.excludedPosts;
    json[r'preview'] = this.preview;
    return json;
  }

  /// Returns a new [SelectorEditor] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SelectorEditor? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "SelectorEditor[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "SelectorEditor[id]" has a null value in JSON.');
        assert(json.containsKey(r'expires_at'),
            'Required key "SelectorEditor[expires_at]" is missing from JSON.');
        assert(json[r'expires_at'] != null,
            'Required key "SelectorEditor[expires_at]" has a null value in JSON.');
        assert(json.containsKey(r'page'),
            'Required key "SelectorEditor[page]" is missing from JSON.');
        assert(json[r'page'] != null,
            'Required key "SelectorEditor[page]" has a null value in JSON.');
        assert(json.containsKey(r'marked_posts'),
            'Required key "SelectorEditor[marked_posts]" is missing from JSON.');
        assert(json[r'marked_posts'] != null,
            'Required key "SelectorEditor[marked_posts]" has a null value in JSON.');
        assert(json.containsKey(r'excluded_links'),
            'Required key "SelectorEditor[excluded_links]" is missing from JSON.');
        assert(json[r'excluded_links'] != null,
            'Required key "SelectorEditor[excluded_links]" has a null value in JSON.');
        assert(json.containsKey(r'excluded_nodes'),
            'Required key "SelectorEditor[excluded_nodes]" is missing from JSON.');
        assert(json[r'excluded_nodes'] != null,
            'Required key "SelectorEditor[excluded_nodes]" has a null value in JSON.');
        assert(json.containsKey(r'found_posts'),
            'Required key "SelectorEditor[found_posts]" is missing from JSON.');
        assert(json[r'found_posts'] != null,
            'Required key "SelectorEditor[found_posts]" has a null value in JSON.');
        assert(json.containsKey(r'excluded_posts'),
            'Required key "SelectorEditor[excluded_posts]" is missing from JSON.');
        assert(json[r'excluded_posts'] != null,
            'Required key "SelectorEditor[excluded_posts]" has a null value in JSON.');
        assert(json.containsKey(r'preview'),
            'Required key "SelectorEditor[preview]" is missing from JSON.');
        assert(json[r'preview'] != null,
            'Required key "SelectorEditor[preview]" has a null value in JSON.');
        return true;
      }());

      return SelectorEditor(
        id: mapValueOfType<int>(json, r'id')!,
        expiresAt: mapDateTime(json, r'expires_at', r'')!,
        page: mapValueOfType<String>(json, r'page')!,
        markedPosts: MarkedPost.listFromJson(json[r'marked_posts']),
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
        foundPosts: FoundPost.listFromJson(json[r'found_posts']),
        excludedPosts: ExcludedPost.listFromJson(json[r'excluded_posts']),
        preview: ExtractedPost.listFromJson(json[r'preview']),
      );
    }
    return null;
  }

  static List<SelectorEditor> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SelectorEditor>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SelectorEditor.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SelectorEditor> mapFromJson(dynamic json) {
    final map = <String, SelectorEditor>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SelectorEditor.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SelectorEditor-objects as value to a dart map
  static Map<String, List<SelectorEditor>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SelectorEditor>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SelectorEditor.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'expires_at',
    'page',
    'marked_posts',
    'excluded_links',
    'excluded_nodes',
    'found_posts',
    'excluded_posts',
    'preview',
  };
}
