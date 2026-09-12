// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'app_router.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The app's navigation graph: a bottom-nav shell over Today and Feeds, with
/// detail/discovery screens nested under each branch. The feed entry and web
/// article readers push onto the root navigator so they cover the bottom nav
/// bar. Sign-in/up and a splash screen sit outside the shell, gated by
/// [authSessionProvider] via [_authRedirect].

@ProviderFor(goRouter)
final goRouterProvider = GoRouterProvider._();

/// The app's navigation graph: a bottom-nav shell over Today and Feeds, with
/// detail/discovery screens nested under each branch. The feed entry and web
/// article readers push onto the root navigator so they cover the bottom nav
/// bar. Sign-in/up and a splash screen sit outside the shell, gated by
/// [authSessionProvider] via [_authRedirect].

final class GoRouterProvider
    extends $FunctionalProvider<GoRouter, GoRouter, GoRouter>
    with $Provider<GoRouter> {
  /// The app's navigation graph: a bottom-nav shell over Today and Feeds, with
  /// detail/discovery screens nested under each branch. The feed entry and web
  /// article readers push onto the root navigator so they cover the bottom nav
  /// bar. Sign-in/up and a splash screen sit outside the shell, gated by
  /// [authSessionProvider] via [_authRedirect].
  GoRouterProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'goRouterProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$goRouterHash();

  @$internal
  @override
  $ProviderElement<GoRouter> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  GoRouter create(Ref ref) {
    return goRouter(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(GoRouter value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<GoRouter>(value),
    );
  }
}

String _$goRouterHash() => r'46bb4e2c5a676aa047731c4b2fa6757096e5a6c6';
