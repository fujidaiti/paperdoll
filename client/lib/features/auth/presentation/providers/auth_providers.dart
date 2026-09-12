import 'package:paperdoll/core/error/domain_error.dart';
import 'package:paperdoll/core/network/dio_provider.dart';
import 'package:paperdoll/core/platform/device.dart';
import 'package:paperdoll/core/platform/secure_storage.dart';
import 'package:paperdoll/features/auth/data/auth_repository_impl.dart';
import 'package:paperdoll/features/auth/domain/auth_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_providers.g.dart';

@riverpod
AuthRepository authRepository(Ref ref) => AuthRepositoryImpl(
  ref.watch(dioProvider),
  ref.watch(secureStorageProvider),
);

/// Bumped when a request comes back 401, so [AuthSession] can end the session
/// reactively. The Dio auth interceptor lives on [dioProvider], which
/// [authSessionProvider] itself depends on (transitively, via
/// [authRepositoryProvider]) — the interceptor reading or invalidating
/// `authSessionProvider` directly would be a genuine dependency cycle
/// (Riverpod's `Ref.read`/`invalidate` reject reading a provider that
/// depends back on the reader). This signal has no dependencies of its own,
/// so the interceptor can safely bump it and [AuthSession] can safely listen
/// to it.
@Riverpod(keepAlive: true)
class SessionInvalidationSignal extends _$SessionInvalidationSignal {
  @override
  int build() => 0;

  void fire() => state++;
}

/// The sign-up attempt that `POST /signup` started: the ticket identifying it
/// server-side, and the address the verification code was mailed to.
class const PendingSignUpAttempt({
  required final String ticket,
  required final String email,
});

/// Owns the pending sign-up attempt, from `POST /signup` until the code is
/// accepted. The attempt is held in memory only: the ticket is a bearer secret
/// and its code expires in ten minutes, so persisting it would buy nothing.
/// Kept alive because no widget listens to it while the app navigates from the
/// sign-up screen to the verification screen.
@Riverpod(keepAlive: true)
class SignUpFlow extends _$SignUpFlow {
  @override
  PendingSignUpAttempt? build() => null;

  /// `POST /signup` → registers the attempt and has a code mailed to [email].
  /// This no longer creates a session; [AuthSession.verifySignUpEmail] does.
  Future<void> start({required String email, required String password}) async {
    final ticket = await ref
        .read(authRepositoryProvider)
        .signUp(email: email, password: password);
    state = PendingSignUpAttempt(ticket: ticket, email: email);
  }

  /// `POST /signup/resend-verification` → has another code mailed, and adopts
  /// the replacement ticket. A no-op when no attempt is held.
  Future<void> resend() async {
    final attempt = state;
    if (attempt == null) {
      return;
    }
    final ticket = await ref
        .read(authRepositoryProvider)
        .resendSignUpVerification(ticket: attempt.ticket);
    state = PendingSignUpAttempt(ticket: ticket, email: attempt.email);
  }

  /// Drops the attempt, after it is verified or once it is dead. The server
  /// keeps the abandoned row until it expires; there is nothing to cancel.
  void clear() => state = null;
}

/// The signed-in session: `null` when signed out, the bearer token when
/// signed in. [build] resolves the persisted token at startup; [signIn] and
/// [verifySignUpEmail] authenticate, persist the returned token, and update
/// the state so the router can react (see `goRouter`'s `redirect`).
@riverpod
class AuthSession extends _$AuthSession {
  late AuthRepository _repo;
  late Device _device;

  @override
  Future<String?> build() async {
    _repo = ref.watch(authRepositoryProvider);
    _device = ref.watch(deviceProvider);
    ref.listen(sessionInvalidationSignalProvider, (_, _) => forceSignOut());
    try {
      return await _repo.readAuthToken();
    } on Exception {
      // Treat an unreadable token as signed-out rather than surfacing an
      // error screen at startup.
      return null;
    }
  }

  Future<void> signIn({required String email, required String password}) async {
    final device = await _device.label();
    final token = await _repo.signIn(
      email: email,
      password: password,
      device: device,
    );
    await _repo.writeAuthToken(token);
    state = AsyncData(token);
  }

  /// Finishes a sign-up attempt: exchanges [ticket] and the mailed [code] for
  /// a token, which is what creates the session. The device label is read here
  /// rather than at sign-up time, because this is the call that issues the
  /// token the label is stored with.
  Future<void> verifySignUpEmail({
    required String ticket,
    required String code,
  }) async {
    final device = await _device.label();
    final token = await _repo.verifySignUpEmail(
      ticket: ticket,
      code: code,
      device: device,
    );
    await _repo.writeAuthToken(token);
    state = AsyncData(token);
  }

  /// Revokes the session token and signs out locally regardless of whether
  /// the revoke call succeeds — `/signout` is best-effort from the client's
  /// perspective, the server already treats it as always-successful.
  Future<void> signOut() async {
    final token = state.value;
    if (token != null) {
      try {
        await _repo.signOut(token);
      } on DomainError {
        // Best-effort: still sign out locally below.
      }
    }
    await _repo.clearAuthToken();
    state = const AsyncData(null);
  }

  /// Ends the session locally without calling `/signout` — used when a
  /// request comes back 401, since the token is already invalid server-side
  /// and calling `/signout` would just 401 again. A no-op when already
  /// signed out.
  Future<void> forceSignOut() async {
    if (state.value == null) {
      return;
    }
    await _repo.clearAuthToken();
    state = const AsyncData(null);
  }
}
