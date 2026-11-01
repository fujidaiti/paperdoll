//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class SaveFeedSelectorsRequest {
  /// Returns a new [SaveFeedSelectorsRequest] instance.
  SaveFeedSelectorsRequest({
    this.selectors = const [],
  });

  List<PostSelector> selectors;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveFeedSelectorsRequest &&
          _deepEquality.equals(other.selectors, selectors);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (selectors.hashCode);

  @override
  String toString() => 'SaveFeedSelectorsRequest[selectors=$selectors]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'selectors'] = this.selectors;
    return json;
  }

  /// Returns a new [SaveFeedSelectorsRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveFeedSelectorsRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'selectors'),
            'Required key "SaveFeedSelectorsRequest[selectors]" is missing from JSON.');
        assert(json[r'selectors'] != null,
            'Required key "SaveFeedSelectorsRequest[selectors]" has a null value in JSON.');
        return true;
      }());

      return SaveFeedSelectorsRequest(
        selectors: PostSelector.listFromJson(json[r'selectors']),
      );
    }
    return null;
  }

  static List<SaveFeedSelectorsRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveFeedSelectorsRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveFeedSelectorsRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveFeedSelectorsRequest> mapFromJson(dynamic json) {
    final map = <String, SaveFeedSelectorsRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveFeedSelectorsRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveFeedSelectorsRequest-objects as value to a dart map
  static Map<String, List<SaveFeedSelectorsRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveFeedSelectorsRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveFeedSelectorsRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'selectors',
  };
}
