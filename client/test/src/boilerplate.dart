import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:paperdoll/app.dart';
import 'package:paperdoll/core/config/app_config.dart';
import 'package:paperdoll/core/config/app_config_provider.dart';
import 'package:paperdoll/core/network/dio_provider.dart';
import 'package:paperdoll/core/platform/device.dart';
import 'package:paperdoll/core/platform/secure_storage.dart';
import 'package:paperdoll/features/auth/presentation/providers/auth_providers.dart';
import 'package:patrol_finders/patrol_finders.dart';

import 'fake_webview.dart';
import 'stub_server.dart';

/// Boots the whole app on the Flutter test framework, with in-memory
/// stand-ins for everything the app would otherwise get from a device: no
/// network, no secure storage plugin, no device info plugin.
///
/// Every HTTP call goes through [server]; build and stub it in the test
/// *before* calling this (start from [StubServer.withDefaultResponses] so the
/// pre-stubbed tabs let the app boot). The shell builds all three tabs on
/// startup, so
/// the data a screen shows on first load (Today's stories, the feeds list, the
/// reading list) must already be stubbed when the app boots — that's why the
/// caller owns the server. Any request no stub matches fails the test at
/// teardown, naming the endpoint.
///
/// Starts signed out unless [token] is given; [pumpAppWithAuth] is the
/// signed-in shortcut for feature tests that aren't about the auth flow.
Future<void> pumpApp(PatrolTester $, StubServer server, {String? token}) async {
  installFakeWebViewPlatform();

  final container = createPaperdollContainer(
    overrides: [
      appConfigProvider.overrideWithValue(const AppConfig('http://mock')),
      deviceProvider.overrideWithValue(_StubDevice()),
      secureStorageProvider.overrideWithValue(_InMemorySecureStorage()),
    ],
  );
  addTearDown(container.dispose);

  container.read(dioProvider).interceptors.add(server);
  addTearDown(
    () => expect(
      server.unmatched,
      isEmpty,
      reason: 'App made unstubbed request(s): ${server.unmatched}',
    ),
  );

  if (token != null) {
    await container.read(authRepositoryProvider).writeAuthToken(token);
  }

  await $.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: const PaperdollApp(),
    ),
  );
  // The splash screen outlives the first settle: reading the token and the
  // redirect that follows land a frame later.
  await $.pumpAndTrySettle();
}

/// [pumpApp] pre-seeded with a signed-in session, so feature tests land
/// straight on Today instead of the sign-in screen.
Future<void> pumpAppWithAuth(PatrolTester $, StubServer server) =>
    pumpApp($, server, token: 'test-token');

class _InMemorySecureStorage implements SecureStorage {
  String? _token;

  @override
  Future<String?> readAuthToken() async => _token;

  @override
  Future<void> writeAuthToken(String? value) async => _token = value;
}

class _StubDevice implements Device {
  @override
  Future<String> label() {
    return Future.value('TestDevice');
  }
}
