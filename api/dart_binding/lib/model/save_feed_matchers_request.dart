//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class SaveFeedMatchersRequest {
  /// Returns a new [SaveFeedMatchersRequest] instance.
  SaveFeedMatchersRequest({
    this.matchers = const [],
  });

  List<PostMatcher> matchers;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveFeedMatchersRequest &&
          _deepEquality.equals(other.matchers, matchers);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (matchers.hashCode);

  @override
  String toString() => 'SaveFeedMatchersRequest[matchers=$matchers]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'matchers'] = this.matchers;
    return json;
  }

  /// Returns a new [SaveFeedMatchersRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveFeedMatchersRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'matchers'),
            'Required key "SaveFeedMatchersRequest[matchers]" is missing from JSON.');
        assert(json[r'matchers'] != null,
            'Required key "SaveFeedMatchersRequest[matchers]" has a null value in JSON.');
        return true;
      }());

      return SaveFeedMatchersRequest(
        matchers: PostMatcher.listFromJson(json[r'matchers']),
      );
    }
    return null;
  }

  static List<SaveFeedMatchersRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveFeedMatchersRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveFeedMatchersRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveFeedMatchersRequest> mapFromJson(dynamic json) {
    final map = <String, SaveFeedMatchersRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveFeedMatchersRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveFeedMatchersRequest-objects as value to a dart map
  static Map<String, List<SaveFeedMatchersRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveFeedMatchersRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveFeedMatchersRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'matchers',
  };
}
