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

/// Why the pending attempt can no longer be verified. Both ends stop the
/// screen from taking further codes; they differ only in where the user goes
/// next.
enum _DeadEnd {
  /// `404` — the attempt is unknown, expired, or out of guesses. The user
  /// starts over from the sign-up form.
  attemptGone,

  /// `409` — someone else registered the address in the meantime. The user's
  /// way forward is signing in.
  addressTaken,
}

/// Finishes the sign-up attempt held by [signUpFlowProvider]: takes the
/// 6-digit code that was mailed and exchanges it for a session. On success the
/// router reacts to the updated [authSessionProvider] state and navigates to
/// Today on its own.
class const VerifyEmailScreen({super.key}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<VerifyEmailScreen> createState() => _VerifyEmailScreenState();
}

class _VerifyEmailScreenState extends ConsumerState<VerifyEmailScreen> {
  final _codeController = TextEditingController();
  var _submitting = false;
  String? _inlineError;
  _DeadEnd? _deadEnd;

  static const _codeLength = 6;

  @override
  void initState() {
    super.initState();
    // The submit button enables on the 6th digit, so every keystroke matters.
    _codeController.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _codeController.dispose();
    super.dispose();
  }

  /// Handles the statuses that end the attempt. Returns false when [error] is
  /// not one of them, so the caller can fall back to its own surface.
  bool _handleDeadEnd(Object error) {
    final deadEnd = switch (error) {
      NotFoundError() => _DeadEnd.attemptGone,
      ConflictError() => _DeadEnd.addressTaken,
      _ => null,
    };
    if (deadEnd == null) {
      return false;
    }
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(_deadEndMessage(deadEnd, describeError(error)))),
    );
    setState(() {
      _deadEnd = deadEnd;
      _inlineError = null;
    });
    return true;
  }

  String _deadEndMessage(_DeadEnd deadEnd, String serverMessage) {
    return switch (deadEnd) {
      _DeadEnd.attemptGone =>
        '$serverMessage Please sign up again to get a new code.',
      _DeadEnd.addressTaken => '$serverMessage Please sign in instead.',
    };
  }

  /// Drops the attempt and leaves the flow. The abandoned attempt is not
  /// cancelled server-side; it expires on its own.
  void _leaveFor(String routeName) {
    ref.read(signUpFlowProvider.notifier).clear();
    context.goNamed(routeName);
  }

  Future<void> _submit(PendingSignUpAttempt attempt) async {
    final code = _codeController.text;
    if (code.length != _codeLength || _deadEnd != null) {
      return;
    }
    final messenger = ScaffoldMessenger.of(context);
    setState(() {
      _submitting = true;
      _inlineError = null;
    });
    try {
      await ref
          .read(authSessionProvider.notifier)
          .verifySignUpEmail(ticket: attempt.ticket, code: code);
      // The token now exists, so the router moves to Today by itself. The
      // attempt has served its purpose.
      ref.read(signUpFlowProvider.notifier).clear();
    } on DomainError catch (error) {
      if (!mounted || _handleDeadEnd(error)) {
        return;
      }
      switch (error) {
        // A wrong or malformed code leaves the attempt alive, so the user
        // corrects it in place rather than being sent elsewhere.
        case UnauthorizedError() || BadRequestError():
          setState(() => _inlineError = describeError(error));
        case _:
          messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
      }
    } finally {
      if (mounted) {
        setState(() => _submitting = false);
      }
    }
  }

  Future<void> _resend(PendingSignUpAttempt attempt) async {
    if (_deadEnd != null) {
      return;
    }
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _submitting = true);
    try {
      await ref.read(signUpFlowProvider.notifier).resend();
      _codeController.clear();
      setState(() => _inlineError = null);
      messenger.showSnackBar(
        SnackBar(
          key: AppDebugKey.verifyEmailCodeSentSnackBar,
          content: Text('A new code was sent to ${attempt.email}'),
        ),
      );
    } on DomainError catch (error) {
      if (!mounted || _handleDeadEnd(error)) {
        return;
      }
      messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
    } finally {
      if (mounted) {
        setState(() => _submitting = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final attempt = ref.watch(signUpFlowProvider);
    if (attempt == null) {
      // Verification succeeded, or the screen was reached without an attempt.
      // Either way the router is already moving the user away.
      return const SizedBox.shrink();
    }
    return Scaffold(
      key: AppDebugKey.verifyEmailScreen,
      appBar: AppBar(
        title: const Text('Verify your email'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => _leaveFor(routeSignUpName),
        ),
      ),
      body: Padding(
        padding: const EdgeInsets.all(spacingMd),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Text('We sent a 6-digit code to'),
            Text(attempt.email),
            const Gap(spacingMd),
            if (_deadEnd case final deadEnd?)
              ..._deadEndActions(deadEnd)
            else
              ..._codeEntry(attempt),
          ],
        ),
      ),
    );
  }

  List<Widget> _codeEntry(PendingSignUpAttempt attempt) {
    final canSubmit =
        !_submitting && _codeController.text.length == _codeLength;
    return [
      TextField(
        key: AppDebugKey.verifyEmailCodeField,
        controller: _codeController,
        keyboardType: TextInputType.number,
        maxLength: _codeLength,
        decoration: InputDecoration(
          labelText: 'Verification code',
          border: const OutlineInputBorder(),
          errorText: _inlineError,
        ),
      ),
      const Gap(spacingMd),
      FilledButton(
        key: AppDebugKey.verifyEmailSubmitButton,
        onPressed: canSubmit ? () => unawaited(_submit(attempt)) : null,
        child: const Text('Continue'),
      ),
      const Gap(spacingMd),
      const Text('The code expires in 10 minutes.'),
      const Gap(spacingSm),
      TextButton(
        key: AppDebugKey.verifyEmailResendButton,
        onPressed: _submitting ? null : () => unawaited(_resend(attempt)),
        child: const Text('Send another code'),
      ),
    ];
  }

  List<Widget> _deadEndActions(_DeadEnd deadEnd) {
    return switch (deadEnd) {
      _DeadEnd.attemptGone => [
        const Text('This code is no longer valid.'),
        const Gap(spacingMd),
        FilledButton(
          key: AppDebugKey.verifyEmailStartOverButton,
          onPressed: () => _leaveFor(routeSignUpName),
          child: const Text('Back to sign up'),
        ),
      ],
      _DeadEnd.addressTaken => [
        const Text('This email address is already registered.'),
        const Gap(spacingMd),
        FilledButton(
          key: AppDebugKey.verifyEmailGoToSignInButton,
          onPressed: () => _leaveFor(routeSignInName),
          child: const Text('Go to sign in'),
        ),
      ],
    };
  }
}
