import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:paperdoll/app.dart';
import 'package:paperdoll/features/auth/presentation/providers/auth_providers.dart';
import 'package:patrol/patrol.dart';

Future<void> pumpApp(PatrolIntegrationTester $) async {
  final container = createPaperdollContainer();
  addTearDown(container.dispose);
  await $.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: const PaperdollApp(),
    ),
  );
}

/// Boots the app already authenticated, skipping the sign-in/up UI: fetches a
/// real token from the runner's `/signin` endpoint, persists it to secure
/// storage, and pumps the app so the real `AuthSession.build()` reads it and
/// the router lands straight on Today. Use this for tests behind the auth gate
/// that aren't about the auth flow itself (e.g. the newspaper suite).
Future<void> pumpAppWithAuth(PatrolIntegrationTester $) async {
  final token = await signInViaRunner();
  // Persist the token through the real repository path, then run the app on the
  // same container so its AuthSession reads the token back at startup.
  final container = createPaperdollContainer();
  addTearDown(container.dispose);
  await container.read(authRepositoryProvider).writeAuthToken(token);
  await $.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: const PaperdollApp(),
    ),
  );
}

/// Builds a URL to the runner's message server. It listens on the host, reached
/// from the Android emulator via 10.0.2.2.
// TODO: make the host and port number configurable (assumes an Android
// emulator).
Uri _runnerUri(String path) =>
    Uri(scheme: 'http', host: '10.0.2.2', port: 9000, path: path);

const _runnerTimeout = Duration(seconds: 60);

/// Provisions the pre-defined test account via the runner's `/signin` endpoint
/// and returns its bearer token.
Future<String> signInViaRunner() async {
  final response = await http
      .post(
        _runnerUri('/signin'),
        headers: {'Content-Type': 'application/json'},
      )
      .timeout(_runnerTimeout);
  if (response.statusCode != 200) {
    throw Exception(
      'signin request failed (${response.statusCode}): ${response.body}',
    );
  }
  final body = jsonDecode(response.body) as Map<String, dynamic>;
  return body['token'] as String;
}

/// Reads back the body of the last email the running session sent to
/// [address], so a sign-up test can read the verification code out of it.
///
/// The email is typically still in flight when this is called, so a missing
/// email is retried rather than treated as a failure.
Future<String> readLastEmailViaRunner(String address) async {
  const timeout = Duration(seconds: 30);
  const interval = Duration(milliseconds: 250);

  final deadline = DateTime.now().add(timeout);
  final uri = _runnerUri('/mailbox/last')
      .replace(queryParameters: {'addr': address});
  while (true) {
    final response = await http.get(uri).timeout(_runnerTimeout);
    if (response.statusCode == 200) {
      return response.body;
    }
    if (response.statusCode != 404) {
      throw Exception(
        'last email request failed '
        '(${response.statusCode}): ${response.body}',
      );
    }
    if (DateTime.now().isAfter(deadline)) {
      throw Exception(
        'no email was sent to $address within ${timeout.inSeconds}s: '
        '${response.body}',
      );
    }
    await Future<void>.delayed(interval);
  }
}

final _verificationCodeInEmail = RegExp(r'\b(\d{6})\b');

/// Extracts the verification code from an email body read via
/// [readLastEmailViaRunner]. Exactly one 6-digit number must appear, so a
/// change to the email template fails loudly instead of yielding a wrong code.
String extractVerificationCode(String emailBody) {
  final found = _verificationCodeInEmail.allMatches(emailBody).toList();
  if (found.length != 1) {
    throw Exception(
      'expected exactly one 6-digit number in the email, '
      'found ${found.length}: $emailBody',
    );
  }
  return found.single.group(1)!;
}

Future<void> setUpServer({required String seederId}) async {
  final response = await http
      .post(
        _runnerUri('/setup'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'seeder_id': seederId}),
      )
      .timeout(_runnerTimeout);
  if (response.body != 'ready') {
    throw Exception(
      'setup request has finished with an error: ${response.body}',
    );
  }
}
