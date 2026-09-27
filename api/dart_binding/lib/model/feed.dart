//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class Feed {
  /// Returns a new [Feed] instance.
  Feed({
    required this.id,
    required this.url,
    this.siteUrl,
    this.iconUrl,
    required this.title,
    this.description,
    required this.subscribed,
  });

  int id;

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

  /// Whether the calling user is subscribed to this feed.
  bool subscribed;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Feed &&
          other.id == id &&
          other.url == url &&
          other.siteUrl == siteUrl &&
          other.iconUrl == iconUrl &&
          other.title == title &&
          other.description == description &&
          other.subscribed == subscribed;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (url.hashCode) +
      (siteUrl == null ? 0 : siteUrl!.hashCode) +
      (iconUrl == null ? 0 : iconUrl!.hashCode) +
      (title.hashCode) +
      (description == null ? 0 : description!.hashCode) +
      (subscribed.hashCode);

  @override
  String toString() =>
      'Feed[id=$id, url=$url, siteUrl=$siteUrl, iconUrl=$iconUrl, title=$title, description=$description, subscribed=$subscribed]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
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
    json[r'subscribed'] = this.subscribed;
    return json;
  }

  /// Returns a new [Feed] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Feed? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Feed[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Feed[id]" has a null value in JSON.');
        assert(json.containsKey(r'url'),
            'Required key "Feed[url]" is missing from JSON.');
        assert(json[r'url'] != null,
            'Required key "Feed[url]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "Feed[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "Feed[title]" has a null value in JSON.');
        assert(json.containsKey(r'subscribed'),
            'Required key "Feed[subscribed]" is missing from JSON.');
        assert(json[r'subscribed'] != null,
            'Required key "Feed[subscribed]" has a null value in JSON.');
        return true;
      }());

      return Feed(
        id: mapValueOfType<int>(json, r'id')!,
        url: mapValueOfType<String>(json, r'url')!,
        siteUrl: mapValueOfType<String>(json, r'site_url'),
        iconUrl: mapValueOfType<String>(json, r'icon_url'),
        title: mapValueOfType<String>(json, r'title')!,
        description: mapValueOfType<String>(json, r'description'),
        subscribed: mapValueOfType<bool>(json, r'subscribed')!,
      );
    }
    return null;
  }

  static List<Feed> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Feed>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Feed.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Feed> mapFromJson(dynamic json) {
    final map = <String, Feed>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Feed.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Feed-objects as value to a dart map
  static Map<String, List<Feed>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Feed>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Feed.listFromJson(
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
    'url',
    'title',
    'subscribed',
  };
}
