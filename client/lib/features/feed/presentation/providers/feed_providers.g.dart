// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'feed_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(feedRepository)
final feedRepositoryProvider = FeedRepositoryProvider._();

final class FeedRepositoryProvider
    extends $FunctionalProvider<FeedRepository, FeedRepository, FeedRepository>
    with $Provider<FeedRepository> {
  FeedRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'feedRepositoryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$feedRepositoryHash();

  @$internal
  @override
  $ProviderElement<FeedRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  FeedRepository create(Ref ref) {
    return feedRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(FeedRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<FeedRepository>(value),
    );
  }
}

String _$feedRepositoryHash() => r'd4a3492cc9d9fd60215d0f6583f52b2b9b89c7e8';

/// The subscribed feeds, paginated. [build] loads the first page; [loadMore]
/// appends the next.

@ProviderFor(Feeds)
final feedsProvider = FeedsProvider._();

/// The subscribed feeds, paginated. [build] loads the first page; [loadMore]
/// appends the next.
final class FeedsProvider
    extends $AsyncNotifierProvider<Feeds, PagedState<Feed>> {
  /// The subscribed feeds, paginated. [build] loads the first page; [loadMore]
  /// appends the next.
  FeedsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'feedsProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$feedsHash();

  @$internal
  @override
  Feeds create() => Feeds();
}

String _$feedsHash() => r'9cbd3693242f4933838309e873cc2cc300754b62';

/// The subscribed feeds, paginated. [build] loads the first page; [loadMore]
/// appends the next.

abstract class _$Feeds extends $AsyncNotifier<PagedState<Feed>> {
  FutureOr<PagedState<Feed>> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<PagedState<Feed>>, PagedState<Feed>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<PagedState<Feed>>, PagedState<Feed>>,
              AsyncValue<PagedState<Feed>>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

@ProviderFor(feedDetail)
final feedDetailProvider = FeedDetailFamily._();

final class FeedDetailProvider
    extends $FunctionalProvider<AsyncValue<Feed>, Feed, FutureOr<Feed>>
    with $FutureModifier<Feed>, $FutureProvider<Feed> {
  FeedDetailProvider._({
    required FeedDetailFamily super.from,
    required int super.argument,
  }) : super(
         retry: null,
         name: r'feedDetailProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$feedDetailHash();

  @override
  String toString() {
    return r'feedDetailProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<Feed> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<Feed> create(Ref ref) {
    final argument = this.argument as int;
    return feedDetail(ref, id: argument);
  }

  @override
  bool operator ==(Object other) {
    return other is FeedDetailProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$feedDetailHash() => r'df5e5de4ca1311fa7cb2ede86ef1fd18a6df27d6';

final class FeedDetailFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<Feed>, int> {
  FeedDetailFamily._()
    : super(
        retry: null,
        name: r'feedDetailProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  FeedDetailProvider call({required int id}) =>
      FeedDetailProvider._(argument: id, from: this);

  @override
  String toString() => r'feedDetailProvider';
}

/// A feed's timeline entries, paginated. [build] loads the first page;
/// [loadMore] appends the next.

@ProviderFor(FeedTimeline)
final feedTimelineProvider = FeedTimelineFamily._();

/// A feed's timeline entries, paginated. [build] loads the first page;
/// [loadMore] appends the next.
final class FeedTimelineProvider
    extends $AsyncNotifierProvider<FeedTimeline, PagedState<FeedEntry>> {
  /// A feed's timeline entries, paginated. [build] loads the first page;
  /// [loadMore] appends the next.
  FeedTimelineProvider._({
    required FeedTimelineFamily super.from,
    required int super.argument,
  }) : super(
         retry: null,
         name: r'feedTimelineProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$feedTimelineHash();

  @override
  String toString() {
    return r'feedTimelineProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  FeedTimeline create() => FeedTimeline();

  @override
  bool operator ==(Object other) {
    return other is FeedTimelineProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$feedTimelineHash() => r'c9f7eeccd2d901ed7529c21c244ca3d74c37a8b8';

/// A feed's timeline entries, paginated. [build] loads the first page;
/// [loadMore] appends the next.

final class FeedTimelineFamily extends $Family
    with
        $ClassFamilyOverride<
          FeedTimeline,
          AsyncValue<PagedState<FeedEntry>>,
          PagedState<FeedEntry>,
          FutureOr<PagedState<FeedEntry>>,
          int
        > {
  FeedTimelineFamily._()
    : super(
        retry: null,
        name: r'feedTimelineProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// A feed's timeline entries, paginated. [build] loads the first page;
  /// [loadMore] appends the next.

  FeedTimelineProvider call({required int id}) =>
      FeedTimelineProvider._(argument: id, from: this);

  @override
  String toString() => r'feedTimelineProvider';
}

/// A feed's timeline entries, paginated. [build] loads the first page;
/// [loadMore] appends the next.

abstract class _$FeedTimeline extends $AsyncNotifier<PagedState<FeedEntry>> {
  late final _$args = ref.$arg as int;
  int get id => _$args;

  FutureOr<PagedState<FeedEntry>> build({required int id});
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<PagedState<FeedEntry>>, PagedState<FeedEntry>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<PagedState<FeedEntry>>,
                PagedState<FeedEntry>
              >,
              AsyncValue<PagedState<FeedEntry>>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(id: _$args));
  }
}

/// Drives the Feed Search / Subscribe screen. [search] populates the candidate
/// list; [subscribe] adds a feed and refreshes the Feeds list.

@ProviderFor(FeedSearchController)
final feedSearchControllerProvider = FeedSearchControllerProvider._();

/// Drives the Feed Search / Subscribe screen. [search] populates the candidate
/// list; [subscribe] adds a feed and refreshes the Feeds list.
final class FeedSearchControllerProvider
    extends $AsyncNotifierProvider<FeedSearchController, List<FeedCandidate>> {
  /// Drives the Feed Search / Subscribe screen. [search] populates the candidate
  /// list; [subscribe] adds a feed and refreshes the Feeds list.
  FeedSearchControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'feedSearchControllerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$feedSearchControllerHash();

  @$internal
  @override
  FeedSearchController create() => FeedSearchController();
}

String _$feedSearchControllerHash() =>
    r'ca1049cf6563a5ac646514f9f3a3b26739c025c5';

/// Drives the Feed Search / Subscribe screen. [search] populates the candidate
/// list; [subscribe] adds a feed and refreshes the Feeds list.

abstract class _$FeedSearchController
    extends $AsyncNotifier<List<FeedCandidate>> {
  FutureOr<List<FeedCandidate>> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<List<FeedCandidate>>, List<FeedCandidate>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<FeedCandidate>>, List<FeedCandidate>>,
              AsyncValue<List<FeedCandidate>>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

/// Holds the choices of the web page subscription flow for as long as its
/// screens are open. Ticking and unticking a group never touches its
/// selection, so the choices survive both.

@ProviderFor(SubscriptionDraft)
final subscriptionDraftProvider = SubscriptionDraftFamily._();

/// Holds the choices of the web page subscription flow for as long as its
/// screens are open. Ticking and unticking a group never touches its
/// selection, so the choices survive both.
final class SubscriptionDraftProvider
    extends $NotifierProvider<SubscriptionDraft, SubscriptionDraftState> {
  /// Holds the choices of the web page subscription flow for as long as its
  /// screens are open. Ticking and unticking a group never touches its
  /// selection, so the choices survive both.
  SubscriptionDraftProvider._({
    required SubscriptionDraftFamily super.from,
    required FeedCandidate super.argument,
  }) : super(
         retry: null,
         name: r'subscriptionDraftProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$subscriptionDraftHash();

  @override
  String toString() {
    return r'subscriptionDraftProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  SubscriptionDraft create() => SubscriptionDraft();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SubscriptionDraftState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SubscriptionDraftState>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is SubscriptionDraftProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$subscriptionDraftHash() => r'0d391931cd9d32fe77dfb24de89d81d94d9a478e';

/// Holds the choices of the web page subscription flow for as long as its
/// screens are open. Ticking and unticking a group never touches its
/// selection, so the choices survive both.

final class SubscriptionDraftFamily extends $Family
    with
        $ClassFamilyOverride<
          SubscriptionDraft,
          SubscriptionDraftState,
          SubscriptionDraftState,
          SubscriptionDraftState,
          FeedCandidate
        > {
  SubscriptionDraftFamily._()
    : super(
        retry: null,
        name: r'subscriptionDraftProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Holds the choices of the web page subscription flow for as long as its
  /// screens are open. Ticking and unticking a group never touches its
  /// selection, so the choices survive both.

  SubscriptionDraftProvider call(FeedCandidate candidate) =>
      SubscriptionDraftProvider._(argument: candidate, from: this);

  @override
  String toString() => r'subscriptionDraftProvider';
}

/// Holds the choices of the web page subscription flow for as long as its
/// screens are open. Ticking and unticking a group never touches its
/// selection, so the choices survive both.

abstract class _$SubscriptionDraft extends $Notifier<SubscriptionDraftState> {
  late final _$args = ref.$arg as FeedCandidate;
  FeedCandidate get candidate => _$args;

  SubscriptionDraftState build(FeedCandidate candidate);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref as $Ref<SubscriptionDraftState, SubscriptionDraftState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<SubscriptionDraftState, SubscriptionDraftState>,
              SubscriptionDraftState,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}
