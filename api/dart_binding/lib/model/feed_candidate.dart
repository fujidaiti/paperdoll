//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class FeedCandidate {
  /// Returns a new [FeedCandidate] instance.
  FeedCandidate({
    required this.url,
    this.siteUrl,
    this.iconUrl,
    required this.title,
    this.description,
    this.postGroups = const [],
  });

  String url;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? siteUrl;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? iconUrl;

  String title;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? description;

  /// Lists of post-like elements found in an HTML page that carries no RSS/Atom feed. The user picks which of them are post lists, and which element of an item is the title, the image and so on, then sends the resulting selectors back in `PUT /feeds`. Absent or empty when `url` points at a real feed, in which case the client subscribes with the URL alone. The groups are listed in the order they appear in the page, and the list is not cut. A page may produce a hundred of them, so the client renders it lazily.
  List<PostGroup> postGroups;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is FeedCandidate &&
          other.url == url &&
          other.siteUrl == siteUrl &&
          other.iconUrl == iconUrl &&
          other.title == title &&
          other.description == description &&
          _deepEquality.equals(other.postGroups, postGroups);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (url.hashCode) +
      (siteUrl == null ? 0 : siteUrl!.hashCode) +
      (iconUrl == null ? 0 : iconUrl!.hashCode) +
      (title.hashCode) +
      (description == null ? 0 : description!.hashCode) +
      (postGroups.hashCode);

  @override
  String toString() =>
      'FeedCandidate[url=$url, siteUrl=$siteUrl, iconUrl=$iconUrl, title=$title, description=$description, postGroups=$postGroups]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'url'] = this.url;
    if (this.siteUrl != null) {
      json[r'site_url'] = this.siteUrl;
    } else {
      json[r'site_url'] = null;
    }
    if (this.iconUrl != null) {
      json[r'icon_url'] = this.iconUrl;
    } else {
      json[r'icon_url'] = null;
    }
    json[r'title'] = this.title;
    if (this.description != null) {
      json[r'description'] = this.description;
    } else {
      json[r'description'] = null;
    }
    json[r'post_groups'] = this.postGroups;
    return json;
  }

  /// Returns a new [FeedCandidate] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static FeedCandidate? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'url'),
            'Required key "FeedCandidate[url]" is missing from JSON.');
        assert(json[r'url'] != null,
            'Required key "FeedCandidate[url]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "FeedCandidate[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "FeedCandidate[title]" has a null value in JSON.');
        return true;
      }());

      return FeedCandidate(
        url: mapValueOfType<String>(json, r'url')!,
        siteUrl: mapValueOfType<String>(json, r'site_url'),
        iconUrl: mapValueOfType<String>(json, r'icon_url'),
        title: mapValueOfType<String>(json, r'title')!,
        description: mapValueOfType<String>(json, r'description'),
        postGroups: PostGroup.listFromJson(json[r'post_groups']),
      );
    }
    return null;
  }

  static List<FeedCandidate> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <FeedCandidate>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = FeedCandidate.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, FeedCandidate> mapFromJson(dynamic json) {
    final map = <String, FeedCandidate>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = FeedCandidate.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of FeedCandidate-objects as value to a dart map
  static Map<String, List<FeedCandidate>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<FeedCandidate>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = FeedCandidate.listFromJson(
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
  };
}
