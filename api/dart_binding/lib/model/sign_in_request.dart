//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class SignInRequest {
  /// Returns a new [SignInRequest] instance.
  SignInRequest({
    required this.email,
    required this.password,
    required this.device,
  });

  String email;

  String password;

  /// Human-readable client label stored alongside the issued token. There is no specific format, but empty string is rejected.
  String device;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SignInRequest &&
          other.email == email &&
          other.password == password &&
          other.device == device;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (email.hashCode) + (password.hashCode) + (device.hashCode);

  @override
  String toString() =>
      'SignInRequest[email=$email, password=$password, device=$device]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'email'] = this.email;
    json[r'password'] = this.password;
    json[r'device'] = this.device;
    return json;
  }

  /// Returns a new [SignInRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SignInRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'email'),
            'Required key "SignInRequest[email]" is missing from JSON.');
        assert(json[r'email'] != null,
            'Required key "SignInRequest[email]" has a null value in JSON.');
        assert(json.containsKey(r'password'),
            'Required key "SignInRequest[password]" is missing from JSON.');
        assert(json[r'password'] != null,
            'Required key "SignInRequest[password]" has a null value in JSON.');
        assert(json.containsKey(r'device'),
            'Required key "SignInRequest[device]" is missing from JSON.');
        assert(json[r'device'] != null,
            'Required key "SignInRequest[device]" has a null value in JSON.');
        return true;
      }());

      return SignInRequest(
        email: mapValueOfType<String>(json, r'email')!,
        password: mapValueOfType<String>(json, r'password')!,
        device: mapValueOfType<String>(json, r'device')!,
      );
    }
    return null;
  }

  static List<SignInRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SignInRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SignInRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SignInRequest> mapFromJson(dynamic json) {
    final map = <String, SignInRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SignInRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SignInRequest-objects as value to a dart map
  static Map<String, List<SignInRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SignInRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SignInRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'email',
    'password',
    'device',
  };
}
