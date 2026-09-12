// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(authRepository)
final authRepositoryProvider = AuthRepositoryProvider._();

final class AuthRepositoryProvider
    extends $FunctionalProvider<AuthRepository, AuthRepository, AuthRepository>
    with $Provider<AuthRepository> {
  AuthRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'authRepositoryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$authRepositoryHash();

  @$internal
  @override
  $ProviderElement<AuthRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  AuthRepository create(Ref ref) {
    return authRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(AuthRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<AuthRepository>(value),
    );
  }
}

String _$authRepositoryHash() => r'5dfc4a721997a279f4ee58ceed1d946abe36dd9c';

/// Bumped when a request comes back 401, so [AuthSession] can end the session
/// reactively. The Dio auth interceptor lives on [dioProvider], which
/// [authSessionProvider] itself depends on (transitively, via
/// [authRepositoryProvider]) — the interceptor reading or invalidating
/// `authSessionProvider` directly would be a genuine dependency cycle
/// (Riverpod's `Ref.read`/`invalidate` reject reading a provider that
/// depends back on the reader). This signal has no dependencies of its own,
/// so the interceptor can safely bump it and [AuthSession] can safely listen
/// to it.

@ProviderFor(SessionInvalidationSignal)
final sessionInvalidationSignalProvider = SessionInvalidationSignalProvider._();

/// Bumped when a request comes back 401, so [AuthSession] can end the session
/// reactively. The Dio auth interceptor lives on [dioProvider], which
/// [authSessionProvider] itself depends on (transitively, via
/// [authRepositoryProvider]) — the interceptor reading or invalidating
/// `authSessionProvider` directly would be a genuine dependency cycle
/// (Riverpod's `Ref.read`/`invalidate` reject reading a provider that
/// depends back on the reader). This signal has no dependencies of its own,
/// so the interceptor can safely bump it and [AuthSession] can safely listen
/// to it.
final class SessionInvalidationSignalProvider
    extends $NotifierProvider<SessionInvalidationSignal, int> {
  /// Bumped when a request comes back 401, so [AuthSession] can end the session
  /// reactively. The Dio auth interceptor lives on [dioProvider], which
  /// [authSessionProvider] itself depends on (transitively, via
  /// [authRepositoryProvider]) — the interceptor reading or invalidating
  /// `authSessionProvider` directly would be a genuine dependency cycle
  /// (Riverpod's `Ref.read`/`invalidate` reject reading a provider that
  /// depends back on the reader). This signal has no dependencies of its own,
  /// so the interceptor can safely bump it and [AuthSession] can safely listen
  /// to it.
  SessionInvalidationSignalProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'sessionInvalidationSignalProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$sessionInvalidationSignalHash();

  @$internal
  @override
  SessionInvalidationSignal create() => SessionInvalidationSignal();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(int value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<int>(value),
    );
  }
}

String _$sessionInvalidationSignalHash() =>
    r'6fec66d36fcf9052bc4f01dd7a4b6775971641cb';

/// Bumped when a request comes back 401, so [AuthSession] can end the session
/// reactively. The Dio auth interceptor lives on [dioProvider], which
/// [authSessionProvider] itself depends on (transitively, via
/// [authRepositoryProvider]) — the interceptor reading or invalidating
/// `authSessionProvider` directly would be a genuine dependency cycle
/// (Riverpod's `Ref.read`/`invalidate` reject reading a provider that
/// depends back on the reader). This signal has no dependencies of its own,
/// so the interceptor can safely bump it and [AuthSession] can safely listen
/// to it.

abstract class _$SessionInvalidationSignal extends $Notifier<int> {
  int build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<int, int>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<int, int>,
              int,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

/// Owns the pending sign-up attempt, from `POST /signup` until the code is
/// accepted. The attempt is held in memory only: the ticket is a bearer secret
/// and its code expires in ten minutes, so persisting it would buy nothing.
/// Kept alive because no widget listens to it while the app navigates from the
/// sign-up screen to the verification screen.

@ProviderFor(SignUpFlow)
final signUpFlowProvider = SignUpFlowProvider._();

/// Owns the pending sign-up attempt, from `POST /signup` until the code is
/// accepted. The attempt is held in memory only: the ticket is a bearer secret
/// and its code expires in ten minutes, so persisting it would buy nothing.
/// Kept alive because no widget listens to it while the app navigates from the
/// sign-up screen to the verification screen.
final class SignUpFlowProvider
    extends $NotifierProvider<SignUpFlow, PendingSignUpAttempt?> {
  /// Owns the pending sign-up attempt, from `POST /signup` until the code is
  /// accepted. The attempt is held in memory only: the ticket is a bearer secret
  /// and its code expires in ten minutes, so persisting it would buy nothing.
  /// Kept alive because no widget listens to it while the app navigates from the
  /// sign-up screen to the verification screen.
  SignUpFlowProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'signUpFlowProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$signUpFlowHash();

  @$internal
  @override
  SignUpFlow create() => SignUpFlow();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(PendingSignUpAttempt? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<PendingSignUpAttempt?>(value),
    );
  }
}

String _$signUpFlowHash() => r'3afb248b4ceb4889dd74c033adc45469f8db285c';

/// Owns the pending sign-up attempt, from `POST /signup` until the code is
/// accepted. The attempt is held in memory only: the ticket is a bearer secret
/// and its code expires in ten minutes, so persisting it would buy nothing.
/// Kept alive because no widget listens to it while the app navigates from the
/// sign-up screen to the verification screen.

abstract class _$SignUpFlow extends $Notifier<PendingSignUpAttempt?> {
  PendingSignUpAttempt? build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<PendingSignUpAttempt?, PendingSignUpAttempt?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<PendingSignUpAttempt?, PendingSignUpAttempt?>,
              PendingSignUpAttempt?,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

/// The signed-in session: `null` when signed out, the bearer token when
/// signed in. [build] resolves the persisted token at startup; [signIn] and
/// [verifySignUpEmail] authenticate, persist the returned token, and update
/// the state so the router can react (see `goRouter`'s `redirect`).

@ProviderFor(AuthSession)
final authSessionProvider = AuthSessionProvider._();

/// The signed-in session: `null` when signed out, the bearer token when
/// signed in. [build] resolves the persisted token at startup; [signIn] and
/// [verifySignUpEmail] authenticate, persist the returned token, and update
/// the state so the router can react (see `goRouter`'s `redirect`).
final class AuthSessionProvider
    extends $AsyncNotifierProvider<AuthSession, String?> {
  /// The signed-in session: `null` when signed out, the bearer token when
  /// signed in. [build] resolves the persisted token at startup; [signIn] and
  /// [verifySignUpEmail] authenticate, persist the returned token, and update
  /// the state so the router can react (see `goRouter`'s `redirect`).
  AuthSessionProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'authSessionProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$authSessionHash();

  @$internal
  @override
  AuthSession create() => AuthSession();
}

String _$authSessionHash() => r'e9b3b12d4912c050db08c7ca75330002ab10c307';

/// The signed-in session: `null` when signed out, the bearer token when
/// signed in. [build] resolves the persisted token at startup; [signIn] and
/// [verifySignUpEmail] authenticate, persist the returned token, and update
/// the state so the router can react (see `goRouter`'s `redirect`).

abstract class _$AuthSession extends $AsyncNotifier<String?> {
  FutureOr<String?> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<String?>, String?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<String?>, String?>,
              AsyncValue<String?>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
