//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class PostGroup {
  /// Returns a new [PostGroup] instance.
  PostGroup({
    required this.id,
    required this.selector,
    required this.count,
    required this.sampled,
    this.links = const [],
    this.texts = const [],
    this.images = const [],
  });

  /// Identifies the group inside this `FeedCandidate`. It is assigned per response and is not stored, so it must not be sent back to the server.
  int id;

  /// CSS selector matching the items of the group in the fetched page. Send it back as `root` in `PUT /feeds`.
  String selector;

  /// How many items the group holds in the fetched page.
  int count;

  /// How many of those items the `values` arrays below describe. The server samples the first few items of the group, so `sampled` is at most `count`.
  int sampled;

  /// The candidate selectors for the post link, that is, elements carrying an `href`. A group with no link candidate cannot be subscribed to, because a post without a URL cannot be stored.
  List<AttributeCandidate> links;

  /// The candidate selectors for the post title, description and timestamp. The three are chosen from the same list, because the server cannot tell them apart.
  List<AttributeCandidate> texts;

  /// The candidate selectors for the post thumbnail.
  List<AttributeCandidate> images;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PostGroup &&
          other.id == id &&
          other.selector == selector &&
          other.count == count &&
          other.sampled == sampled &&
          _deepEquality.equals(other.links, links) &&
          _deepEquality.equals(other.texts, texts) &&
          _deepEquality.equals(other.images, images);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (selector.hashCode) +
      (count.hashCode) +
      (sampled.hashCode) +
      (links.hashCode) +
      (texts.hashCode) +
      (images.hashCode);

  @override
  String toString() =>
      'PostGroup[id=$id, selector=$selector, count=$count, sampled=$sampled, links=$links, texts=$texts, images=$images]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'selector'] = this.selector;
    json[r'count'] = this.count;
    json[r'sampled'] = this.sampled;
    json[r'links'] = this.links;
    json[r'texts'] = this.texts;
    json[r'images'] = this.images;
    return json;
  }

  /// Returns a new [PostGroup] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PostGroup? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "PostGroup[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "PostGroup[id]" has a null value in JSON.');
        assert(json.containsKey(r'selector'),
            'Required key "PostGroup[selector]" is missing from JSON.');
        assert(json[r'selector'] != null,
            'Required key "PostGroup[selector]" has a null value in JSON.');
        assert(json.containsKey(r'count'),
            'Required key "PostGroup[count]" is missing from JSON.');
        assert(json[r'count'] != null,
            'Required key "PostGroup[count]" has a null value in JSON.');
        assert(json.containsKey(r'sampled'),
            'Required key "PostGroup[sampled]" is missing from JSON.');
        assert(json[r'sampled'] != null,
            'Required key "PostGroup[sampled]" has a null value in JSON.');
        assert(json.containsKey(r'links'),
            'Required key "PostGroup[links]" is missing from JSON.');
        assert(json[r'links'] != null,
            'Required key "PostGroup[links]" has a null value in JSON.');
        assert(json.containsKey(r'texts'),
            'Required key "PostGroup[texts]" is missing from JSON.');
        assert(json[r'texts'] != null,
            'Required key "PostGroup[texts]" has a null value in JSON.');
        assert(json.containsKey(r'images'),
            'Required key "PostGroup[images]" is missing from JSON.');
        assert(json[r'images'] != null,
            'Required key "PostGroup[images]" has a null value in JSON.');
        return true;
      }());

      return PostGroup(
        id: mapValueOfType<int>(json, r'id')!,
        selector: mapValueOfType<String>(json, r'selector')!,
        count: mapValueOfType<int>(json, r'count')!,
        sampled: mapValueOfType<int>(json, r'sampled')!,
        links: AttributeCandidate.listFromJson(json[r'links']),
        texts: AttributeCandidate.listFromJson(json[r'texts']),
        images: AttributeCandidate.listFromJson(json[r'images']),
      );
    }
    return null;
  }

  static List<PostGroup> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PostGroup>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PostGroup.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PostGroup> mapFromJson(dynamic json) {
    final map = <String, PostGroup>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PostGroup.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PostGroup-objects as value to a dart map
  static Map<String, List<PostGroup>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PostGroup>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PostGroup.listFromJson(
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
    'selector',
    'count',
    'sampled',
    'links',
    'texts',
    'images',
  };
}
