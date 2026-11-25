// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'selector_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(selectorEditorRepository)
final selectorEditorRepositoryProvider = SelectorEditorRepositoryProvider._();

final class SelectorEditorRepositoryProvider
    extends
        $FunctionalProvider<
          SelectorEditorRepository,
          SelectorEditorRepository,
          SelectorEditorRepository
        >
    with $Provider<SelectorEditorRepository> {
  SelectorEditorRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'selectorEditorRepositoryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$selectorEditorRepositoryHash();

  @$internal
  @override
  $ProviderElement<SelectorEditorRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  SelectorEditorRepository create(Ref ref) {
    return selectorEditorRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SelectorEditorRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SelectorEditorRepository>(value),
    );
  }
}

String _$selectorEditorRepositoryHash() =>
    r'79a1f15c4f30866ead0c095eb72f9e4aeffd5bea';
