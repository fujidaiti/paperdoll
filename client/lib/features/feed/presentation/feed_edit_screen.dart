import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/async_value_view.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed/domain/feed.dart';
import 'package:paperdoll/features/feed/presentation/providers/feed_providers.dart';
import 'package:paperdoll/features/feed/presentation/widgets/feed_icon.dart';

/// Feed Edit: the attributes of a feed, and the entry to the editor of the
/// post matchers of an unstructured feed. Feeds are shared across all users,
/// so is what is edited here.
class const FeedEditScreen({required final int id, super.key})
    extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final feed = ref.watch(feedDetailControllerProvider(id: id));
    return Scaffold(
      key: AppDebugKey.feedEditScreen,
      appBar: AppBar(title: const Text('Edit feed')),
      body: AsyncValueView<Feed>(
        value: feed,
        onRetry: () => ref.invalidate(feedDetailControllerProvider(id: id)),
        data: (feed) => _FeedAttributes(feed: feed),
      ),
    );
  }
}

class const _FeedAttributes({required final Feed feed})
    extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final description = feed.description;
    final siteUrl = feed.siteUrl;
    // A page without an RSS feed, whose posts are extracted with the matchers
    // that the user builds.
    final unstructured = feed.url == siteUrl;
    return ListView(
      padding: const EdgeInsets.all(spacingMd),
      children: [
        // TODO: Let the user change the icon.
        _Attribute(
          label: 'Icon',
          child: Align(
            alignment: Alignment.centerLeft,
            child: FeedIcon(url: feed.iconUrl),
          ),
        ),
        // TODO: Let the user change the title.
        _Attribute(label: 'Title', child: BodyText(feed.title)),
        // TODO: Let the user change the description.
        _Attribute(
          label: 'Description',
          child: BodyText(description ?? 'No description'),
        ),
        _Attribute(label: 'Feed URL', child: BodyText(feed.url)),
        if (siteUrl != null && !unstructured)
          _Attribute(
            label: 'Site',
            child: Align(
              alignment: Alignment.centerLeft,
              child: TextButton.icon(
                onPressed: () => unawaited(openExternalLink(context, siteUrl)),
                icon: const Icon(Icons.public),
                label: Text(siteUrl),
              ),
            ),
          ),
        if (unstructured) ...[
          const Gap(spacingMd),
          FilledButton(
            key: AppDebugKey.editMatchersButton,
            onPressed: () => context.pushNamed(
              routePageAnnotationName,
              pathParameters: {'id': feed.id.toString()},
            ),
            child: const Text('Edit post matchers'),
          ),
        ],
      ],
    );
  }
}

/// One attribute of the feed: a caption with its value below.
class const _Attribute({
  required final String label,
  required final Widget child,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: spacingSm),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [CaptionText(label), const Gap(spacingXs), child],
      ),
    );
  }
}
