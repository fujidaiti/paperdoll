import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/error/domain_error.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/auth/presentation/providers/auth_providers.dart';
import 'package:paperdoll/features/auth/presentation/verify_email_screen.dart';

/// Start creating an account with email + password. Submitting no longer signs
/// the user in: it registers a pending attempt, has a verification code mailed
/// to the address, and moves on to [VerifyEmailScreen], which finishes the
/// flow.
class const SignUpScreen({super.key}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<SignUpScreen> createState() => _SignUpScreenState();
}

class _SignUpScreenState extends ConsumerState<SignUpScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  var _submitting = false;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final email = _emailController.text.trim();
    final password = _passwordController.text;
    if (email.isEmpty || password.isEmpty) {
      return;
    }
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    setState(() => _submitting = true);
    try {
      await ref
          .read(signUpFlowProvider.notifier)
          .start(email: email, password: password);
      router.goNamed(routeVerifyEmailName);
    } on ConflictError catch (error) {
      // The address is taken, so the user's way forward is signing in. The
      // form stays open, and the address is not carried over — the user may
      // well have mistyped it.
      messenger.showSnackBar(
        SnackBar(
          content: Text(describeError(error)),
          action: SnackBarAction(
            label: 'Sign in',
            onPressed: () => router.goNamed(routeSignInName),
          ),
        ),
      );
    } on DomainError catch (error) {
      messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
    } finally {
      if (mounted) {
        setState(() => _submitting = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      key: AppDebugKey.signUpScreen,
      appBar: AppBar(title: const Text('Sign up')),
      body: Padding(
        padding: const EdgeInsets.all(spacingMd),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            TextField(
              key: AppDebugKey.signUpEmailField,
              controller: _emailController,
              keyboardType: TextInputType.emailAddress,
              decoration: const InputDecoration(
                labelText: 'Email',
                border: OutlineInputBorder(),
              ),
            ),
            const Gap(spacingSm),
            TextField(
              key: AppDebugKey.signUpPasswordField,
              controller: _passwordController,
              obscureText: true,
              decoration: const InputDecoration(
                labelText: 'Password',
                border: OutlineInputBorder(),
              ),
              onSubmitted: (_) => unawaited(_submit()),
            ),
            const Gap(spacingMd),
            FilledButton(
              key: AppDebugKey.signUpSubmitButton,
              onPressed: _submitting ? null : () => unawaited(_submit()),
              child: const Text('Sign up'),
            ),
            const Gap(spacingSm),
            TextButton(
              key: AppDebugKey.signUpGoToSignInButton,
              onPressed: () => context.goNamed(routeSignInName),
              child: const Text('Already have an account? Sign in'),
            ),
          ],
        ),
      ),
    );
  }
}
