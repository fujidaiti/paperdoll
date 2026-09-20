import 'package:flutter/widgets.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:patrol/patrol.dart';

import 'helper.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  patrolTest('Sign up for a new account', tags: 'signup', (t) async {
    await setUpServer(seederId: 'auth_no_users');
    await pumpApp(t);

    const newUserEmail = 'newuser@example.com';
    await t(AppDebugKey.signInGoToSignUpButton).tap();
    await t(AppDebugKey.signUpEmailField).enterText(newUserEmail);
    await t(AppDebugKey.signUpPasswordField)
        .enterText('New-User-Account-Password');
    await t(AppDebugKey.signUpSubmitButton).tap();
    await t(AppDebugKey.verifyEmailScreen).waitUntilVisible();

    final code = extractVerificationCode(
      await readLastEmailViaRunner(newUserEmail),
    );
    await t(AppDebugKey.verifyEmailCodeField).enterText(code);
    await t(AppDebugKey.verifyEmailSubmitButton).tap();
    await t(AppDebugKey.todayScreen).waitUntilVisible();
  });

  patrolTest('Send another verification code', tags: 'signup', (t) async {
    await setUpServer(seederId: 'auth_no_users');
    await pumpApp(t);

    const newUserEmail = 'newuser@example.com';
    await t(AppDebugKey.signInGoToSignUpButton).tap();
    await t(AppDebugKey.signUpEmailField).enterText(newUserEmail);
    await t(AppDebugKey.signUpPasswordField)
        .enterText('New-User-Account-Password');
    await t(AppDebugKey.signUpSubmitButton).tap();
    await t(AppDebugKey.verifyEmailScreen).waitUntilVisible();
    await t(AppDebugKey.verifyEmailResendButton).tap();
    await t(AppDebugKey.verifyEmailCodeSentSnackBar).waitUntilVisible();
    final code = extractVerificationCode(
      await readLastEmailViaRunner(newUserEmail),
    );
    await t(AppDebugKey.verifyEmailCodeField).enterText(code);
    await t(AppDebugKey.verifyEmailSubmitButton).tap();
    await t(AppDebugKey.todayScreen).waitUntilVisible();
  });

  patrolTest('Sign in to an existing account', tags: 'signin', (t) async {
    await setUpServer(seederId: 'auth_existing_user');
    await pumpApp(t);

    await t(AppDebugKey.signInScreen).waitUntilVisible();
    await t(AppDebugKey.signInEmailField).enterText('alice@example.com');
    await t(AppDebugKey.signInPasswordField)
        .enterText('Police-Repurpose-Atypical-Gravel');
    await t(AppDebugKey.signInSubmitButton).tap();
    await t(AppDebugKey.todayScreen).waitUntilVisible();
  });

  patrolTest('Sign out ends the session', tags: 'signout', (t) async {
    await setUpServer(seederId: 'auth_signed_in');
    await pumpAppWithAuth(t);

    await t(AppDebugKey.todayScreen).waitUntilVisible();
    await t(AppDebugKey.settingsButton).tap();
    await t(AppDebugKey.settingsScreen).waitUntilVisible();
    await t(AppDebugKey.signOutButton).tap();
    await t(AppDebugKey.signInScreen).waitUntilVisible();
  });
}
