//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class SignUpTicket {
  /// Returns a new [SignUpTicket] instance.
  SignUpTicket({
    required this.ticket,
  });

  /// Opaque secret identifying the sign-up attempt the verification code was mailed for. Send it back to verify the address or to request another code. Valid for 10 minutes.
  String ticket;

  @override
  bool operator ==(Object other) =>
      identical(this, other) || other is SignUpTicket && other.ticket == ticket;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (ticket.hashCode);

  @override
  String toString() => 'SignUpTicket[ticket=$ticket]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'ticket'] = this.ticket;
    return json;
  }

  /// Returns a new [SignUpTicket] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SignUpTicket? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'ticket'),
            'Required key "SignUpTicket[ticket]" is missing from JSON.');
        assert(json[r'ticket'] != null,
            'Required key "SignUpTicket[ticket]" has a null value in JSON.');
        return true;
      }());

      return SignUpTicket(
        ticket: mapValueOfType<String>(json, r'ticket')!,
      );
    }
    return null;
  }

  static List<SignUpTicket> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SignUpTicket>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SignUpTicket.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SignUpTicket> mapFromJson(dynamic json) {
    final map = <String, SignUpTicket>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SignUpTicket.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SignUpTicket-objects as value to a dart map
  static Map<String, List<SignUpTicket>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SignUpTicket>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SignUpTicket.listFromJson(
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
  };
}
