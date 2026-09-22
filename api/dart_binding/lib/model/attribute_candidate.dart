//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class AttributeCandidate {
  /// Returns a new [AttributeCandidate] instance.
  AttributeCandidate({
    required this.selector,
    required this.matched,
    this.values = const [],
  });

  /// Selector relative to the group item. `:scope` means the item element itself, which happens when the whole card is a link.
  String selector;

  /// How many of the group's `count` items this selector reaches. A value lower than `count` means the choice leaves some posts without this attribute, which the client shows as a warning on the row.
  int matched;

  /// What the selector produces in each sampled item, in the same order as the items. The array holds exactly `sampled` entries, so the client can build the preview of item _i_ by reading index _i_ of every row. An entry with no `value` means the selector reaches nothing in that item.
  List<AttributeValue> values;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AttributeCandidate &&
          other.selector == selector &&
          other.matched == matched &&
          _deepEquality.equals(other.values, values);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (selector.hashCode) + (matched.hashCode) + (values.hashCode);

  @override
  String toString() =>
      'AttributeCandidate[selector=$selector, matched=$matched, values=$values]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'selector'] = this.selector;
    json[r'matched'] = this.matched;
    json[r'values'] = this.values;
    return json;
  }

  /// Returns a new [AttributeCandidate] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AttributeCandidate? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'selector'),
            'Required key "AttributeCandidate[selector]" is missing from JSON.');
        assert(json[r'selector'] != null,
            'Required key "AttributeCandidate[selector]" has a null value in JSON.');
        assert(json.containsKey(r'matched'),
            'Required key "AttributeCandidate[matched]" is missing from JSON.');
        assert(json[r'matched'] != null,
            'Required key "AttributeCandidate[matched]" has a null value in JSON.');
        assert(json.containsKey(r'values'),
            'Required key "AttributeCandidate[values]" is missing from JSON.');
        assert(json[r'values'] != null,
            'Required key "AttributeCandidate[values]" has a null value in JSON.');
        return true;
      }());

      return AttributeCandidate(
        selector: mapValueOfType<String>(json, r'selector')!,
        matched: mapValueOfType<int>(json, r'matched')!,
        values: AttributeValue.listFromJson(json[r'values']),
      );
    }
    return null;
  }

  static List<AttributeCandidate> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AttributeCandidate>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AttributeCandidate.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AttributeCandidate> mapFromJson(dynamic json) {
    final map = <String, AttributeCandidate>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AttributeCandidate.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AttributeCandidate-objects as value to a dart map
  static Map<String, List<AttributeCandidate>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AttributeCandidate>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AttributeCandidate.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'selector',
    'matched',
    'values',
  };
}
