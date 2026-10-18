import 'package:paperdoll/core/network/dio_provider.dart';
import 'package:paperdoll/features/page_watch/data/matcher_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'page_watch_providers.g.dart';

@riverpod
MatcherRepository matcherRepository(Ref ref) =>
    MatcherRepository(ref.watch(dioProvider));

@riverpod
Future<FeedMatchers> feedMatchers(Ref ref, int feedId) =>
    ref.watch(matcherRepositoryProvider).get(feedId);
