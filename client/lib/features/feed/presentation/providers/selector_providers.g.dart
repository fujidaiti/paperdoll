// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'selector_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(selectorRepository)
final selectorRepositoryProvider = SelectorRepositoryProvider._();

final class SelectorRepositoryProvider
    extends
        $FunctionalProvider<
          SelectorRepository,
          SelectorRepository,
          SelectorRepository
        >
    with $Provider<SelectorRepository> {
  SelectorRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'selectorRepositoryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$selectorRepositoryHash();

  @$internal
  @override
  $ProviderElement<SelectorRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  SelectorRepository create(Ref ref) {
    return selectorRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SelectorRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SelectorRepository>(value),
    );
  }
}

String _$selectorRepositoryHash() =>
    r'b80034945653c156c0c62e8038e1531c01f4916a';

@ProviderFor(feedSelectors)
final feedSelectorsProvider = FeedSelectorsFamily._();

final class FeedSelectorsProvider
    extends
        $FunctionalProvider<
          AsyncValue<FeedSelectors>,
          FeedSelectors,
          FutureOr<FeedSelectors>
        >
    with $FutureModifier<FeedSelectors>, $FutureProvider<FeedSelectors> {
  FeedSelectorsProvider._({
    required FeedSelectorsFamily super.from,
    required int super.argument,
  }) : super(
         retry: null,
         name: r'feedSelectorsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$feedSelectorsHash();

  @override
  String toString() {
    return r'feedSelectorsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<FeedSelectors> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<FeedSelectors> create(Ref ref) {
    final argument = this.argument as int;
    return feedSelectors(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is FeedSelectorsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$feedSelectorsHash() => r'51366322472e2d07caac938a31972b1dde61d1cc';

final class FeedSelectorsFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<FeedSelectors>, int> {
  FeedSelectorsFamily._()
    : super(
        retry: null,
        name: r'feedSelectorsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  FeedSelectorsProvider call(int feedId) =>
      FeedSelectorsProvider._(argument: feedId, from: this);

  @override
  String toString() => r'feedSelectorsProvider';
}
