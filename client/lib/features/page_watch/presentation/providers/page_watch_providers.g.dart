// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'page_watch_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(matcherRepository)
final matcherRepositoryProvider = MatcherRepositoryProvider._();

final class MatcherRepositoryProvider
    extends
        $FunctionalProvider<
          MatcherRepository,
          MatcherRepository,
          MatcherRepository
        >
    with $Provider<MatcherRepository> {
  MatcherRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'matcherRepositoryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$matcherRepositoryHash();

  @$internal
  @override
  $ProviderElement<MatcherRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  MatcherRepository create(Ref ref) {
    return matcherRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(MatcherRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<MatcherRepository>(value),
    );
  }
}

String _$matcherRepositoryHash() => r'7217383836b0f5afd6183b33ff93b5e0f9ac3ee8';

@ProviderFor(feedMatchers)
final feedMatchersProvider = FeedMatchersFamily._();

final class FeedMatchersProvider
    extends
        $FunctionalProvider<
          AsyncValue<FeedMatchers>,
          FeedMatchers,
          FutureOr<FeedMatchers>
        >
    with $FutureModifier<FeedMatchers>, $FutureProvider<FeedMatchers> {
  FeedMatchersProvider._({
    required FeedMatchersFamily super.from,
    required int super.argument,
  }) : super(
         retry: null,
         name: r'feedMatchersProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$feedMatchersHash();

  @override
  String toString() {
    return r'feedMatchersProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<FeedMatchers> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<FeedMatchers> create(Ref ref) {
    final argument = this.argument as int;
    return feedMatchers(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is FeedMatchersProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$feedMatchersHash() => r'3b0222b87b2ba3c6b22fb85b5cbeca85ae6412d8';

final class FeedMatchersFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<FeedMatchers>, int> {
  FeedMatchersFamily._()
    : super(
        retry: null,
        name: r'feedMatchersProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  FeedMatchersProvider call(int feedId) =>
      FeedMatchersProvider._(argument: feedId, from: this);

  @override
  String toString() => r'feedMatchersProvider';
}
