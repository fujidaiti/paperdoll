//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class VerifySignUpEmailRequest {
  /// Returns a new [VerifySignUpEmailRequest] instance.
  VerifySignUpEmailRequest({
    required this.ticket,
    required this.verificationCode,
    required this.device,
  });

  /// The ticket returned by `POST /signup`.
  String ticket;

  /// The code mailed to the address, exactly 6 digits. Leading zeros are significant, so send it as a string.
  String verificationCode;

  /// Human-readable client label stored alongside the issued token. There is no specific format, but empty string is rejected.
  String device;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is VerifySignUpEmailRequest &&
          other.ticket == ticket &&
          other.verificationCode == verificationCode &&
          other.device == device;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (ticket.hashCode) + (verificationCode.hashCode) + (device.hashCode);

  @override
  String toString() =>
      'VerifySignUpEmailRequest[ticket=$ticket, verificationCode=$verificationCode, device=$device]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'ticket'] = this.ticket;
    json[r'verification_code'] = this.verificationCode;
    json[r'device'] = this.device;
    return json;
  }

  /// Returns a new [VerifySignUpEmailRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static VerifySignUpEmailRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'ticket'),
            'Required key "VerifySignUpEmailRequest[ticket]" is missing from JSON.');
        assert(json[r'ticket'] != null,
            'Required key "VerifySignUpEmailRequest[ticket]" has a null value in JSON.');
        assert(json.containsKey(r'verification_code'),
            'Required key "VerifySignUpEmailRequest[verification_code]" is missing from JSON.');
        assert(json[r'verification_code'] != null,
            'Required key "VerifySignUpEmailRequest[verification_code]" has a null value in JSON.');
        assert(json.containsKey(r'device'),
            'Required key "VerifySignUpEmailRequest[device]" is missing from JSON.');
        assert(json[r'device'] != null,
            'Required key "VerifySignUpEmailRequest[device]" has a null value in JSON.');
        return true;
      }());

      return VerifySignUpEmailRequest(
        ticket: mapValueOfType<String>(json, r'ticket')!,
        verificationCode: mapValueOfType<String>(json, r'verification_code')!,
        device: mapValueOfType<String>(json, r'device')!,
      );
    }
    return null;
  }

  static List<VerifySignUpEmailRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <VerifySignUpEmailRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = VerifySignUpEmailRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, VerifySignUpEmailRequest> mapFromJson(dynamic json) {
    final map = <String, VerifySignUpEmailRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = VerifySignUpEmailRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of VerifySignUpEmailRequest-objects as value to a dart map
  static Map<String, List<VerifySignUpEmailRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<VerifySignUpEmailRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = VerifySignUpEmailRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'ticket',
    'verification_code',
    'device',
  };
}
