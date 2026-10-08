// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'page_watch_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(pageSamples)
final pageSamplesProvider = PageSamplesProvider._();

final class PageSamplesProvider
    extends
        $FunctionalProvider<
          AsyncValue<List<PageSample>>,
          List<PageSample>,
          FutureOr<List<PageSample>>
        >
    with $FutureModifier<List<PageSample>>, $FutureProvider<List<PageSample>> {
  PageSamplesProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'pageSamplesProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$pageSamplesHash();

  @$internal
  @override
  $FutureProviderElement<List<PageSample>> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<List<PageSample>> create(Ref ref) {
    return pageSamples(ref);
  }
}

String _$pageSamplesHash() => r'5e8a2a95c5e764a1d2e236a47c980db3c0268069';

@ProviderFor(pageDocument)
final pageDocumentProvider = PageDocumentFamily._();

final class PageDocumentProvider
    extends
        $FunctionalProvider<
          AsyncValue<PageDocument>,
          PageDocument,
          FutureOr<PageDocument>
        >
    with $FutureModifier<PageDocument>, $FutureProvider<PageDocument> {
  PageDocumentProvider._({
    required PageDocumentFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'pageDocumentProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$pageDocumentHash();

  @override
  String toString() {
    return r'pageDocumentProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<PageDocument> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<PageDocument> create(Ref ref) {
    final argument = this.argument as String;
    return pageDocument(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is PageDocumentProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$pageDocumentHash() => r'653659ffd9b05be50b612f0067e846cbb28ef87c';

final class PageDocumentFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<PageDocument>, String> {
  PageDocumentFamily._()
    : super(
        retry: null,
        name: r'pageDocumentProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PageDocumentProvider call(String file) =>
      PageDocumentProvider._(argument: file, from: this);

  @override
  String toString() => r'pageDocumentProvider';
}
