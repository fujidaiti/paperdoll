import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/error/domain_error.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/core/ui/tokens/app_radii.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/ui/widgets/heading_text.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed/domain/post_group.dart';
import 'package:paperdoll/features/feed/domain/post_selection.dart';
import 'package:paperdoll/features/feed/presentation/providers/feed_providers.dart';

/// Subscribes to a plain HTML page with no feed: the user ticks the groups of
/// post-like elements to follow, and opens each one to tell which of its
/// elements hold the post attributes.
class const FeedSubscriptionScreen({
  required final FeedCandidate candidate,
  super.key,
}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<FeedSubscriptionScreen> createState() =>
      _FeedSubscriptionScreenState();
}

class _FeedSubscriptionScreenState
    extends ConsumerState<FeedSubscriptionScreen> {
  var _subscribing = false;

  Future<void> _subscribe() async {
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    setState(() => _subscribing = true);
    try {
      await ref
          .read(subscriptionDraftProvider(widget.candidate).notifier)
          .subscribe();
      messenger.showSnackBar(
        SnackBar(
          key: AppDebugKey.subscribeSuccessSnackBar,
          content: Text('Subscribed to ${widget.candidate.title}'),
        ),
      );
      // Leaves both this screen and the search screen below it.
      router.goNamed(routeFeedsName);
    } on DomainError catch (error) {
      messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
    } finally {
      if (mounted) {
        setState(() => _subscribing = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final candidate = widget.candidate;
    final provider = subscriptionDraftProvider(candidate);
    final draft = ref.watch(provider);
    final groups = candidate.postGroups;
    return Scaffold(
      key: AppDebugKey.feedSubscriptionScreen,
      appBar: AppBar(title: Text(candidate.title)),
      // The server returns every group it found, which can reach a hundred or
      // more, so the cards are built lazily.
      body: ListView.builder(
        padding: const EdgeInsets.all(spacingSm),
        itemCount: groups.length,
        itemBuilder: (context, index) {
          final group = groups[index];
          return _PostGroupCard(
            key: AppDebugKey.postGroupCard(group.id),
            group: group,
            selection: draft.selections[group.id]!,
            ticked: draft.ticked.contains(group.id),
            onToggle: () => ref.read(provider.notifier).toggle(group.id),
            onOpen: () => context.pushNamed(
              routeFeedSubscriptionAttributesName,
              pathParameters: {'groupId': group.id.toString()},
              extra: candidate,
            ),
          );
        },
      ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(spacingMd),
          child: FilledButton(
            key: AppDebugKey.feedSubscriptionSubscribeButton,
            onPressed: draft.ticked.isEmpty || _subscribing
                ? null
                : () => unawaited(_subscribe()),
            child: const Text('Subscribe'),
          ),
        ),
      ),
    );
  }
}

class const _PostGroupCard({
  required final PostGroup group,
  required final PostSelection selection,
  required final bool ticked,
  required final VoidCallback onToggle,
  required final VoidCallback onOpen,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final hasLink = group.links.isNotEmpty;
    // The values arrays hold `sampled` entries; show at most two of them.
    final previewCount = group.sampled < 2 ? group.sampled : 2;
    return Card(
      child: InkWell(
        onTap: onOpen,
        child: Padding(
          padding: const EdgeInsets.all(spacingSm),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  // Claims the taps a disabled checkbox ignores, so that they
                  // do not reach the card and open the group.
                  GestureDetector(
                    onTap: () {},
                    child: Checkbox(
                      key: AppDebugKey.postGroupCheckbox(group.id),
                      value: ticked,
                      // A post with no URL cannot be stored.
                      onChanged: hasLink ? (_) => onToggle() : null,
                    ),
                  ),
                  Expanded(
                    child: HeadingText('${group.count} posts on this page'),
                  ),
                ],
              ),
              if (!hasLink)
                const CaptionText('This group has no link to a post.'),
              if (!selection.answered)
                const CaptionText('Guessed preview. Tap to choose.'),
              for (var i = 0; i < previewCount; i++) ...[
                const Gap(spacingSm),
                _PostPreview(group: group, selection: selection, index: i),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// The item at [index] of [group], as a post built from [selection]. Before
/// the user has answered anything, the first text and image candidates stand
/// in as a guess.
class const _PostPreview({
  required final PostGroup group,
  required final PostSelection selection,
  required final int index,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    String? valueOf(PostAttribute attribute) {
      final selector = selection.selectorOf(attribute);
      if (selector == null) {
        return null;
      }
      final candidate = attribute
          .candidatesOf(group)
          .firstWhere((c) => c.selector == selector);
      return candidate.values[index].value;
    }

    final String? title;
    final String? description;
    final String? timestamp;
    final String? image;
    if (selection.answered) {
      title = valueOf(PostAttribute.title);
      description = valueOf(PostAttribute.description);
      timestamp = valueOf(PostAttribute.timestamp);
      image = valueOf(PostAttribute.image);
    } else {
      title = group.texts.firstOrNull?.values[index].value;
      description = null;
      timestamp = null;
      image = group.images.firstOrNull?.values[index].value;
    }

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (image != null) ...[
          ClipRRect(
            borderRadius: borderRadiusCard,
            child: Image.network(
              image,
              width: iconMd,
              height: iconMd,
              fit: BoxFit.cover,
              errorBuilder: (context, error, stackTrace) =>
                  const Icon(Icons.image_outlined, size: iconMd),
            ),
          ),
          const Gap(spacingSm),
        ],
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (title != null)
                BodyText(title, maxLines: 2, overflow: TextOverflow.ellipsis),
              if (description != null)
                CaptionText(
                  description,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              if (timestamp != null) CaptionText(timestamp),
            ],
          ),
        ),
      ],
    );
  }
}
