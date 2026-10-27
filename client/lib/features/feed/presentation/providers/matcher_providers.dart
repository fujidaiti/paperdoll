import 'package:paperdoll/core/network/dio_provider.dart';
import 'package:paperdoll/features/feed/data/matcher_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'matcher_providers.g.dart';

@riverpod
MatcherRepository matcherRepository(Ref ref) =>
    MatcherRepository(ref.watch(dioProvider));

@riverpod
Future<FeedMatchers> feedMatchers(Ref ref, int feedId) =>
    ref.watch(matcherRepositoryProvider).get(feedId);
