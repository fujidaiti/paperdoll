import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_radii.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/core/ui/widgets/empty_placeholder.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/ui/widgets/heading_text.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed/domain/post_group.dart';
import 'package:paperdoll/features/feed/domain/post_selection.dart';
import 'package:paperdoll/features/feed/presentation/providers/feed_providers.dart';

/// Asks, one attribute per step, which candidate of a post group holds the
/// link, title, description, timestamp and image. Each answer is written to
/// the [SubscriptionDraft] right away; this screen only tracks the step.
class const AttributePickerScreen({
  required final FeedCandidate candidate,
  required final int groupId,
  super.key,
}) extends ConsumerStatefulWidget {
  @override
  ConsumerState<AttributePickerScreen> createState() =>
      _AttributePickerScreenState();
}

class _AttributePickerScreenState extends ConsumerState<AttributePickerScreen> {
  var _step = 0;

  @override
  Widget build(BuildContext context) {
    final provider = subscriptionDraftProvider(widget.candidate);
    final selection = ref.watch(provider).selections[widget.groupId]!;
    final group = widget.candidate.postGroups.firstWhere(
      (g) => g.id == widget.groupId,
    );
    final steps = [
      // With a single link candidate there is nothing to choose; the draft
      // has already selected it.
      if (group.links.length > 1) PostAttribute.link,
      for (final attribute in const [
        PostAttribute.title,
        PostAttribute.description,
        PostAttribute.timestamp,
        PostAttribute.image,
      ])
        if (attribute.candidatesOf(group).isNotEmpty) attribute,
    ];
    final attribute = steps[_step];
    final selected = selection.selectorOf(attribute);

    // The same element cannot hold two text attributes, so a text row chosen
    // in an earlier step is not offered again.
    const textAttributes = {
      PostAttribute.title,
      PostAttribute.description,
      PostAttribute.timestamp,
    };
    final takenSelectors = {
      if (textAttributes.contains(attribute))
        for (final earlier in steps.take(_step))
          if (textAttributes.contains(earlier)) selection.selectorOf(earlier),
    };
    final rows = attribute
        .candidatesOf(group)
        .where((c) => !takenSelectors.contains(c.selector))
        .toList();

    void next() {
      if (_step == steps.length - 1) {
        context.pop();
      } else {
        setState(() => _step++);
      }
    }

    return Scaffold(
      key: AppDebugKey.attributePickerScreen,
      appBar: AppBar(title: Text('Step ${_step + 1} of ${steps.length}')),
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.all(spacingMd),
            child: HeadingText('Which row holds the ${attribute.name}?'),
          ),
          Expanded(
            child: rows.isEmpty
                ? const EmptyPlaceholder(message: 'No rows left to choose.')
                : ListView.builder(
                    itemCount: rows.length,
                    itemBuilder: (context, index) {
                      final row = rows[index];
                      return _AttributeRow(
                        key: AppDebugKey.attributeRow(row.selector),
                        group: group,
                        candidate: row,
                        isImage: attribute == PostAttribute.image,
                        selected: row.selector == selected,
                        onTap: () => ref
                            .read(provider.notifier)
                            .select(group.id, attribute, row.selector),
                      );
                    },
                  ),
          ),
        ],
      ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(spacingMd),
          child: Row(
            children: [
              TextButton(
                key: AppDebugKey.attributeBackButton,
                onPressed: _step == 0
                    ? () => context.pop()
                    : () => setState(() => _step--),
                child: const Text('Back'),
              ),
              const Spacer(),
              // A post cannot be stored without a link, so the link step has
              // no "None" answer.
              if (attribute != PostAttribute.link)
                TextButton(
                  key: AppDebugKey.attributeNoneButton,
                  onPressed: () {
                    ref
                        .read(provider.notifier)
                        .select(group.id, attribute, null);
                    next();
                  },
                  child: const Text('None of them'),
                ),
              const Gap(spacingSm),
              FilledButton(
                key: AppDebugKey.attributeNextButton,
                onPressed: next,
                child: Text(_step == steps.length - 1 ? 'Done' : 'Next'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class const _AttributeRow({
  required final PostGroup group,
  required final AttributeCandidate candidate,
  required final bool isImage,
  required final bool selected,
  required final VoidCallback onTap,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    // The values arrays hold `sampled` entries; show at most two of them.
    final values = candidate.values.take(2).map((v) => v.value).toList();
    return ListTile(
      selected: selected,
      onTap: onTap,
      leading: Icon(
        selected ? Icons.radio_button_checked : Icons.radio_button_unchecked,
      ),
      title: isImage
          ? Row(
              children: [
                for (final value in values)
                  Padding(
                    padding: const EdgeInsets.only(right: spacingSm),
                    child: value == null
                        ? const Icon(Icons.hide_image_outlined, size: iconMd)
                        : ClipRRect(
                            borderRadius: borderRadiusCard,
                            child: Image.network(
                              value,
                              width: iconMd,
                              height: iconMd,
                              fit: BoxFit.cover,
                              errorBuilder: (context, error, stackTrace) =>
                                  const Icon(
                                    Icons.image_outlined,
                                    size: iconMd,
                                  ),
                            ),
                          ),
                  ),
              ],
            )
          : Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                for (final value in values)
                  BodyText(
                    value ?? '(empty)',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
              ],
            ),
      // The user's only warning that this choice leaves some posts
      // incomplete.
      subtitle: candidate.matched < group.count
          ? CaptionText('Reaches ${candidate.matched} of ${group.count} posts')
          : null,
    );
  }
}
