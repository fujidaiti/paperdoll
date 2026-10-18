//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class FieldStep {
  /// Returns a new [FieldStep] instance.
  FieldStep({
    required this.shape,
    required this.nth,
  });

  String shape;

  int nth;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is FieldStep && other.shape == shape && other.nth == nth;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (shape.hashCode) + (nth.hashCode);

  @override
  String toString() => 'FieldStep[shape=$shape, nth=$nth]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'shape'] = this.shape;
    json[r'nth'] = this.nth;
    return json;
  }

  /// Returns a new [FieldStep] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static FieldStep? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'shape'),
            'Required key "FieldStep[shape]" is missing from JSON.');
        assert(json[r'shape'] != null,
            'Required key "FieldStep[shape]" has a null value in JSON.');
        assert(json.containsKey(r'nth'),
            'Required key "FieldStep[nth]" is missing from JSON.');
        assert(json[r'nth'] != null,
            'Required key "FieldStep[nth]" has a null value in JSON.');
        return true;
      }());

      return FieldStep(
        shape: mapValueOfType<String>(json, r'shape')!,
        nth: mapValueOfType<int>(json, r'nth')!,
      );
    }
    return null;
  }

  static List<FieldStep> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <FieldStep>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = FieldStep.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, FieldStep> mapFromJson(dynamic json) {
    final map = <String, FieldStep>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = FieldStep.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of FieldStep-objects as value to a dart map
  static Map<String, List<FieldStep>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<FieldStep>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = FieldStep.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'shape',
    'nth',
  };
}
