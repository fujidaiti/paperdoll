import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/error/domain_error.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/app_divider.dart';
import 'package:paperdoll/core/ui/widgets/async_value_view.dart';
import 'package:paperdoll/core/ui/widgets/empty_placeholder.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed/presentation/providers/feed_providers.dart';
import 'package:paperdoll/features/feed/presentation/widgets/feed_candidate_tile.dart';

/// Feed Search / Subscribe: find a feed by URL and subscribe to it.
class const FeedSearchScreen({super.key}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<FeedSearchScreen> createState() => _FeedSearchScreenState();
}

class _FeedSearchScreenState extends ConsumerState<FeedSearchScreen> {
  final _queryController = TextEditingController();
  int? _subscribingIndex;

  @override
  void dispose() {
    _queryController.dispose();
    super.dispose();
  }

  Future<void> _search() async {
    final query = _queryController.text.trim();
    if (query.isEmpty) {
      return;
    }
    await ref.read(feedSearchControllerProvider.notifier).search(query);
  }

  Future<void> _subscribe(FeedCandidate candidate, int index) async {
    // A plain HTML page with no feed: the user has to tell which elements of
    // the page are posts before it can be subscribed to.
    if (candidate.postGroups.isNotEmpty) {
      await context.pushNamed(routeFeedSubscriptionName, extra: candidate);
      return;
    }
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    setState(() => _subscribingIndex = index);
    try {
      await ref
          .read(feedSearchControllerProvider.notifier)
          .subscribe(candidate.url);
      messenger.showSnackBar(
        SnackBar(
          key: AppDebugKey.subscribeSuccessSnackBar,
          content: Text('Subscribed to ${candidate.title}'),
        ),
      );
      router.pop();
    } on DomainError catch (error) {
      messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
    } finally {
      if (mounted) {
        setState(() => _subscribingIndex = null);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final results = ref.watch(feedSearchControllerProvider);
    return Scaffold(
      key: AppDebugKey.feedSearchScreen,
      appBar: AppBar(title: const Text('Add feed')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(spacingMd),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    key: AppDebugKey.feedSearchTextField,
                    controller: _queryController,
                    keyboardType: TextInputType.url,
                    decoration: const InputDecoration(
                      labelText: 'Feed URL',
                      border: OutlineInputBorder(),
                    ),
                    onSubmitted: (_) => unawaited(_search()),
                  ),
                ),
                const Gap(spacingSm),
                FilledButton(
                  key: AppDebugKey.feedSearchButton,
                  onPressed: () => unawaited(_search()),
                  child: const Text('Search'),
                ),
              ],
            ),
          ),
          const AppDivider(),
          Expanded(
            child: AsyncValueView<List<FeedCandidate>>(
              value: results,
              data: _buildResults,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildResults(List<FeedCandidate> candidates) {
    if (candidates.isEmpty) {
      return const EmptyPlaceholder(message: 'Search for a feed by its URL.');
    }
    return ListView.separated(
      itemCount: candidates.length,
      separatorBuilder: (context, index) => const AppDivider(),
      itemBuilder: (context, index) {
        final candidate = candidates[index];
        return FeedCandidateTile(
          key: AppDebugKey.feedCandidateTile(candidate.title),
          candidate: candidate,
          isSubscribing: _subscribingIndex == index,
          onSubscribe: () => unawaited(_subscribe(candidate, index)),
        );
      },
    );
  }
}
