import 'package:paperdoll/core/network/dio_provider.dart';
import 'package:paperdoll/features/feed/data/selector_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'selector_providers.g.dart';

@riverpod
SelectorRepository selectorRepository(Ref ref) =>
    SelectorRepository(ref.watch(dioProvider));

@riverpod
Future<FeedSelectors> feedSelectors(Ref ref, int feedId) =>
    ref.watch(selectorRepositoryProvider).get(feedId);
