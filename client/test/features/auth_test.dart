import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/core/router/app_router.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/auth/presentation/providers/auth_providers.dart';
import 'package:patrol_finders/patrol_finders.dart';

import '../src/boilerplate.dart';
import '../src/stub_server.dart';

const _email = 'newuser@example.com';
const _password = 'a-strong-password';
const _ticket = 'ticket-from-signup';
const _code = '123456';
const _token = 'issued-token';

void main() {
  group('Sign up', () {
    patrolWidgetTest('Sign up to get a verification code', (t) async {
      Object? sentBody;
      final server = StubServer.withDefaultResponses()
        ..onPost(
          '/signup',
          respond: (body) {
            sentBody = body;
            return (202, api.SignUpTicket(ticket: _ticket).toJson());
          },
        );
      await pumpApp(t, server);

      await startSignUp(t);
      expect(t(AppDebugKey.verifyEmailScreen), findsOneWidget);
      expect(t(_email), findsOneWidget);
      // The device label moved to the verify call, so it must not be sent here.
      expect(sentBody, {'email': _email, 'password': _password});
    });

    patrolWidgetTest('Sign up with an invalid address keeps the form open', (
      t,
    ) async {
      await pumpApp(t, serverStubbingSignUp(status: 400, message: 'Bad email'));
      await startSignUp(t);
      expect(t('Bad email'), findsOneWidget);
      expect(t(AppDebugKey.signUpScreen), findsOneWidget);
    });

    patrolWidgetTest('Sign up with a taken address offers signing in', (
      t,
    ) async {
      await pumpApp(
        t,
        serverStubbingSignUp(status: 409, message: 'Email already registered'),
      );
      await startSignUp(t);
      expect(t('Email already registered'), findsOneWidget);
      expect(t(AppDebugKey.signUpScreen), findsOneWidget);

      await t('Sign in').tap();
      expect(t(AppDebugKey.signInScreen), findsOneWidget);
    });

    patrolWidgetTest(
      'Sign up refused by the send throttle keeps the form open',
      (t) async {
        await pumpApp(
          t,
          serverStubbingSignUp(status: 429, message: 'Too many codes sent'),
        );
        await startSignUp(t);
        expect(t('Too many codes sent'), findsOneWidget);
        expect(t(AppDebugKey.signUpScreen), findsOneWidget);
      },
    );

    patrolWidgetTest('Sign up failing on the server keeps the form open', (
      t,
    ) async {
      await pumpApp(
        t,
        serverStubbingSignUp(status: 500, message: 'Server down'),
      );
      await startSignUp(t);
      expect(t('Server down'), findsOneWidget);
      expect(t(AppDebugKey.signUpScreen), findsOneWidget);
    });
  });

  group('Verify email', () {
    patrolWidgetTest('Verify the email address to finish signing up', (
      t,
    ) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/verify-email',
          body: api.AuthToken(token: _token).toJson(),
          bodyMatcher: api.VerifySignUpEmailRequest(
            ticket: _ticket,
            verificationCode: _code,
            device: 'TestDevice',
          ).toJson(),
        );
      final container = await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, _code);
      await t(AppDebugKey.verifyEmailSubmitButton).tap();

      expect(t(AppDebugKey.todayScreen), findsOneWidget);
      final storage = container.read(authRepositoryProvider);
      expect(await storage.readAuthToken(), _token);
      // The attempt has served its purpose; nothing keeps the ticket around.
      expect(container.read(signUpFlowProvider), isNull);
    });

    patrolWidgetTest('Verify a code that starts with a zero', (t) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/verify-email',
          body: api.AuthToken(token: _token).toJson(),
          // A String, not a number: the leading zero is part of the code.
          bodyMatcher: {'verification_code': '042931'},
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, '042931');
      await t(AppDebugKey.verifyEmailSubmitButton).tap();
      expect(t(AppDebugKey.todayScreen), findsOneWidget);
    });

    patrolWidgetTest('Verify with a wrong code to correct it in place', (
      t,
    ) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/verify-email',
          status: 401,
          body: api.Error(message: 'The code is wrong').toJson(),
        );
      final container = await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, '000000');
      await t(AppDebugKey.verifyEmailSubmitButton).tap();

      expect(t('The code is wrong'), findsOneWidget);
      expect(t(AppDebugKey.verifyEmailScreen), findsOneWidget);
      // The code stays put so the user edits it rather than retyping it.
      expect(codeFieldText(t), '000000');
      final storage = container.read(authRepositoryProvider);
      expect(await storage.readAuthToken(), isNull);
    });

    patrolWidgetTest('Retry after a wrong code finishes signing up', (t) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/verify-email',
          status: 401,
          body: api.Error(message: 'The code is wrong').toJson(),
          bodyMatcher: {'verification_code': '000000'},
        )
        ..stubPost(
          '/signup/verify-email',
          body: api.AuthToken(token: _token).toJson(),
          bodyMatcher: {'verification_code': _code},
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, '000000');
      await t(AppDebugKey.verifyEmailSubmitButton).tap();
      await enterCode(t, _code);
      await t(AppDebugKey.verifyEmailSubmitButton).tap();
      expect(t(AppDebugKey.todayScreen), findsOneWidget);
    });

    patrolWidgetTest('Verify with a malformed code keeps the screen open', (
      t,
    ) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/verify-email',
          status: 400,
          body: api.Error(message: 'Malformed code').toJson(),
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, _code);
      await t(AppDebugKey.verifyEmailSubmitButton).tap();
      expect(t('Malformed code'), findsOneWidget);
      expect(t(AppDebugKey.verifyEmailScreen), findsOneWidget);
    });

    patrolWidgetTest('Verify a dead attempt to start over', (t) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/verify-email',
          status: 404,
          body: api.Error(message: 'No live attempt').toJson(),
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, _code);
      await t(AppDebugKey.verifyEmailSubmitButton).tap();

      // Further codes are refused, so the screen stops taking them.
      expect(t(AppDebugKey.verifyEmailCodeField), findsNothing);
      expect(t(AppDebugKey.verifyEmailSubmitButton), findsNothing);
      await t(AppDebugKey.verifyEmailStartOverButton).tap();
      expect(t(AppDebugKey.signUpScreen), findsOneWidget);
    });

    patrolWidgetTest('Verify an address taken meanwhile to sign in instead', (
      t,
    ) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/verify-email',
          status: 409,
          body: api.Error(message: 'Email already registered').toJson(),
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, _code);
      await t(AppDebugKey.verifyEmailSubmitButton).tap();

      // The user is told, not moved: leaving the screen is their choice.
      expect(t(AppDebugKey.verifyEmailScreen), findsOneWidget);
      await t(AppDebugKey.verifyEmailGoToSignInButton).tap();
      expect(t(AppDebugKey.signInScreen), findsOneWidget);
    });

    patrolWidgetTest('Verify again after the server fails', (t) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/verify-email',
          status: 500,
          body: api.Error(message: 'Server down').toJson(),
          bodyMatcher: {'verification_code': '111111'},
        )
        ..stubPost(
          '/signup/verify-email',
          body: api.AuthToken(token: _token).toJson(),
          bodyMatcher: {'verification_code': _code},
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, '111111');
      await t(AppDebugKey.verifyEmailSubmitButton).tap();
      expect(t('Server down'), findsOneWidget);

      // The attempt is still live, so the screen stays usable.
      await enterCode(t, _code);
      await t(AppDebugKey.verifyEmailSubmitButton).tap();
      expect(t(AppDebugKey.todayScreen), findsOneWidget);
    });

    patrolWidgetTest('Submit stays disabled below six digits', (t) async {
      await pumpApp(t, serverStubbingSignUp());
      await startSignUp(t);

      expect(submitEnabled(t), isFalse);
      await enterCode(t, '12345');
      expect(submitEnabled(t), isFalse);
      await enterCode(t, _code);
      expect(submitEnabled(t), isTrue);
    });
  });

  group('Resend verification', () {
    patrolWidgetTest('Send another code and verify with it', (t) async {
      const newTicket = 'ticket-from-resend';
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/resend-verification',
          status: 202,
          body: api.SignUpTicket(ticket: newTicket).toJson(),
          bodyMatcher: api.ResendSignUpVerificationRequest(ticket: _ticket)
              .toJson(),
        )
        // Only the replacement ticket is answered: verifying with the original
        // one would go unstubbed and fail the test.
        ..stubPost(
          '/signup/verify-email',
          body: api.AuthToken(token: _token).toJson(),
          bodyMatcher: {'ticket': newTicket},
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await enterCode(t, '111111');
      await t(AppDebugKey.verifyEmailResendButton).tap();

      expect(t(AppDebugKey.verifyEmailCodeSentSnackBar), findsOneWidget);
      // The old code no longer verifies, so keeping it on screen would mislead.
      expect(codeFieldText(t), isEmpty);

      await enterCode(t, _code);
      await t(AppDebugKey.verifyEmailSubmitButton).tap();
      expect(t(AppDebugKey.todayScreen), findsOneWidget);
    });

    patrolWidgetTest('Send another code for a dead attempt to start over', (
      t,
    ) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/resend-verification',
          status: 404,
          body: api.Error(message: 'No live attempt').toJson(),
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await t(AppDebugKey.verifyEmailResendButton).tap();
      await t(AppDebugKey.verifyEmailStartOverButton).tap();
      expect(t(AppDebugKey.signUpScreen), findsOneWidget);
    });

    patrolWidgetTest('Send another code to an address taken meanwhile', (
      t,
    ) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/resend-verification',
          status: 409,
          body: api.Error(message: 'Email already registered').toJson(),
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await t(AppDebugKey.verifyEmailResendButton).tap();
      expect(t(AppDebugKey.verifyEmailScreen), findsOneWidget);
      await t(AppDebugKey.verifyEmailGoToSignInButton).tap();
      expect(t(AppDebugKey.signInScreen), findsOneWidget);
    });

    patrolWidgetTest('Send another code refused by the send throttle', (
      t,
    ) async {
      final server = serverStubbingSignUp()
        ..stubPost(
          '/signup/resend-verification',
          status: 429,
          body: api.Error(message: 'Too many codes sent').toJson(),
        )
        ..stubPost(
          '/signup/verify-email',
          body: api.AuthToken(token: _token).toJson(),
          bodyMatcher: {'ticket': _ticket},
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await t(AppDebugKey.verifyEmailResendButton).tap();
      expect(t('Too many codes sent'), findsOneWidget);

      // The earlier code may still be valid, so the screen stays usable.
      await enterCode(t, _code);
      await t(AppDebugKey.verifyEmailSubmitButton).tap();
      expect(t(AppDebugKey.todayScreen), findsOneWidget);
    });

    patrolWidgetTest('Send another code again after the server fails', (
      t,
    ) async {
      var resends = 0;
      final server = serverStubbingSignUp()
        ..onPost(
          '/signup/resend-verification',
          respond: (_) {
            resends++;
            return (500, api.Error(message: 'Server down').toJson());
          },
        );
      await pumpApp(t, server);

      await startSignUp(t);
      await t(AppDebugKey.verifyEmailResendButton).tap();
      expect(t('Server down'), findsOneWidget);
      await t(AppDebugKey.verifyEmailResendButton).tap();
      expect(resends, 2);
    });
  });

  group('Router', () {
    patrolWidgetTest(
      'Opening verification without an attempt goes to sign up',
      (t) async {
        final container = await pumpApp(t, StubServer.withDefaultResponses());
        container.read(goRouterProvider).goNamed(routeVerifyEmailName);
        await t.pumpAndSettle();
        expect(t(AppDebugKey.signUpScreen), findsOneWidget);
      },
    );
  });

  group('Sign in', () {
    patrolWidgetTest('Sign in to an existing account', (t) async {
      final server = StubServer.withDefaultResponses()
        ..stubPost(
          '/signin',
          body: api.AuthToken(token: _token).toJson(),
          bodyMatcher: api.SignInRequest(
            email: 'alice@example.com',
            password: 'correct-password',
            device: 'TestDevice',
          ).toJson(),
        );
      await pumpApp(t, server);

      expect(t(AppDebugKey.signInScreen), findsOneWidget);
      await signIn(t, 'alice@example.com', 'correct-password');
      expect(t(AppDebugKey.todayScreen), findsOneWidget);
    });

    patrolWidgetTest('Sign in with a padded email address', (t) async {
      final server = StubServer.withDefaultResponses()
        ..stubPost(
          '/signin',
          body: api.AuthToken(token: _token).toJson(),
          bodyMatcher: api.SignInRequest(
            email: 'user@example.com',
            password: 'a-password',
            device: 'TestDevice',
          ).toJson(),
        );
      await pumpApp(t, server);

      await signIn(t, '  user@example.com  ', 'a-password');
      expect(t(AppDebugKey.todayScreen), findsOneWidget);
    });

    patrolWidgetTest('Signing in with wrong credentials keeps the form open', (
      t,
    ) async {
      final server = StubServer.withDefaultResponses()
        ..stubPost(
          '/signin',
          status: 400,
          body: api.Error(message: 'Email or password is incorrect').toJson(),
          bodyMatcher: api.SignInRequest(
            email: 'alice@example.com',
            password: 'wrong-password',
            device: 'TestDevice',
          ).toJson(),
        );
      await pumpApp(t, server);

      await signIn(t, 'alice@example.com', 'wrong-password');
      expect(t('Email or password is incorrect'), findsOneWidget);
      expect(t(AppDebugKey.signInScreen), findsOneWidget);
    });

    patrolWidgetTest('Submitting empty credentials does nothing', (t) async {
      await pumpApp(t, StubServer.withDefaultResponses());
      await t(AppDebugKey.signInSubmitButton).tap();
      // The screen guards on empty input, so it stays put without asking the
      // server — `/signin` is left unstubbed, and a request would surface an
      // error snackbar.
      expect(find.byType(SnackBar), findsNothing);
      expect(t(AppDebugKey.signInScreen), findsOneWidget);
    });

    patrolWidgetTest('Switch between sign-in and sign-up', (t) async {
      await pumpApp(t, StubServer.withDefaultResponses());
      await t(AppDebugKey.signInGoToSignUpButton).tap();
      expect(t(AppDebugKey.signUpScreen), findsOneWidget);
      await t(AppDebugKey.signUpGoToSignInButton).tap();
      expect(t(AppDebugKey.signInScreen), findsOneWidget);
    });

    patrolWidgetTest('Open the app with a stored token', (t) async {
      await pumpApp(
        t,
        StubServer.withDefaultResponses(),
        token: 'stored-token',
      );
      expect(t(AppDebugKey.todayScreen), findsOneWidget);
      expect(t(AppDebugKey.signInScreen), findsNothing);
      expect(t(AppDebugKey.signUpScreen), findsNothing);
    });
  });
}

/// A server that answers `POST /signup` for the fixed credentials. Defaults to
/// the 202 + ticket of the happy path; pass [status] and [message] for the
/// failures.
StubServer serverStubbingSignUp({int status = 202, String? message}) {
  return StubServer.withDefaultResponses()..stubPost(
    '/signup',
    status: status,
    body: message == null
        ? api.SignUpTicket(ticket: _ticket).toJson()
        : api.Error(message: message).toJson(),
    bodyMatcher: api.SignUpRequest(email: _email, password: _password).toJson(),
  );
}

/// Walks from the sign-in screen through the sign-up form with the fixed
/// credentials, landing on the verification screen when the server answered
/// 202.
Future<void> startSignUp(PatrolTester t) async {
  await t(AppDebugKey.signInGoToSignUpButton).tap();
  await t(AppDebugKey.signUpEmailField).enterText(_email);
  await t(AppDebugKey.signUpPasswordField).enterText(_password);
  await t(AppDebugKey.signUpSubmitButton).tap();
}

Future<void> enterCode(PatrolTester t, String code) =>
    t(AppDebugKey.verifyEmailCodeField).enterText(code);

String codeFieldText(PatrolTester t) => t.tester
    .widget<TextField>(find.byKey(AppDebugKey.verifyEmailCodeField))
    .controller!
    .text;

bool submitEnabled(PatrolTester t) =>
    t.tester
        .widget<FilledButton>(find.byKey(AppDebugKey.verifyEmailSubmitButton))
        .onPressed !=
    null;

Future<void> signIn(PatrolTester t, String email, String password) async {
  await t(AppDebugKey.signInEmailField).enterText(email);
  await t(AppDebugKey.signInPasswordField).enterText(password);
  await t(AppDebugKey.signInSubmitButton).tap();
}
