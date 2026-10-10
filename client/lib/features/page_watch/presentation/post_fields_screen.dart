import 'dart:async';

import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/app_divider.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/features/page_watch/domain/page_document.dart';
import 'package:paperdoll/features/page_watch/domain/post_matcher.dart';
import 'package:paperdoll/features/page_watch/presentation/widgets/page_lines.dart';

/// The arguments of [PostFieldsScreen]: the group of posts to label, and the
/// post of the group whose content is shown.
typedef PostFieldsArgs = ({
  PageDocument document,
  PostMatcher matcher,
  PageNode post,
  int groupNumber,
});

/// Lets the user label the content of one post as the post's fields. The
/// other posts of the group get their fields with the same selectors.
///
/// The screen pops with the updated [PostMatcher] when the user is done, or
/// with nothing when the user leaves without finishing.
///
/// A body is converted as a whole, so a tap inside it selects the whole body
/// and the content inside it cannot be labeled.
class const PostFieldsScreen({required final PostFieldsArgs args, super.key})
    extends StatefulWidget {
  @override
  State<PostFieldsScreen> createState() => _PostFieldsScreenState();
}

class _PostFieldsScreenState extends State<PostFieldsScreen> {
  late var _matcher = widget.args.matcher;
  late final _lines = layOutLines([_post]);
  PageNode? _selected;

  /// The content the user tapped last. The selection can be narrowed back
  /// down to it after being widened.
  PageNode? _tapped;

  PageNode get _post => widget.args.post;

  PageNode? _fieldNode(PostField field) => _matcher.matchField(_post, field);

  /// Why [field] cannot be labeled on [node], or null if it can.
  String? _fieldRejection(PostField field, PageNode node) {
    if (field == PostField.body) {
      for (final other in PostField.values) {
        if (other == PostField.body) {
          continue;
        }
        if (_fieldNode(other) case final n? when node.contains(n)) {
          return 'A body cannot contain the ${other.label.toLowerCase()}.';
        }
      }
    } else if (_fieldNode(PostField.body) case final body?
        when body.contains(node)) {
      return 'This is part of the body.';
    }
    return null;
  }

  void _toggleField(PostField field, PageNode node) {
    setState(() {
      _matcher = _matcher.withField(
        field,
        _post,
        identical(_fieldNode(field), node) ? null : node,
      );
    });
  }

  Future<void> _onTapLine(List<PageNode> line) async {
    // A line of several pieces selects the content that holds them, and its
    // pieces, which are too small to tap, are picked from a sheet.
    final node = line.length == 1 ? line.first : line.first.parent!;
    final target = switch (_fieldNode(PostField.body)) {
      final body? when line.every(body.contains) => body,
      _ => node,
    };
    setState(() {
      _selected = identical(_selected, target) ? null : target;
      _tapped = _selected == null ? null : node;
    });
    if (line.length == 1 || !identical(_selected, node)) {
      return;
    }
    final piece = await pickLinePiece(context, line);
    if (piece != null && mounted && identical(_selected, node)) {
      setState(() {
        _selected = piece;
        _tapped = piece;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final selected = _selected;
    // A link around other content is labeled or selected as a link when the
    // user taps its own line, so only that line is tinted. Reached by widening
    // or labeled as the body, it stands for its whole content.
    final areas = <PageArea>[
      for (final field in PostField.values)
        if (_fieldNode(field) case final node?)
          (
            node: node,
            color: fieldColor(field),
            linkOnly: field == PostField.link,
          ),
      if (selected != null)
        selectionArea(selected, linkOnly: identical(selected, _tapped)),
    ];
    final postCount = _matcher.matchPosts(widget.args.document).length;
    final hasLink = _matcher.fields.containsKey(PostField.link);
    return Scaffold(
      appBar: AppBar(
        title: Text('Group ${widget.args.groupNumber}: label the fields'),
        actions: [
          TextButton(
            onPressed: hasLink ? () => context.pop(_matcher) : null,
            child: const Text('Done'),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              padding: const EdgeInsets.symmetric(vertical: spacingSm),
              itemCount: _lines.length,
              itemBuilder: (context, index) {
                final line = _lines[index];
                return PageLine(
                  key: ValueKey(line.first.id),
                  nodes: line,
                  areas: areas,
                  onTap: (line) => unawaited(_onTapLine(line)),
                );
              },
            ),
          ),
          const AppDivider(),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.all(spacingMd),
              child: SizedBox(
                width: double.infinity,
                child: _buildPanel(postCount: postCount, hasLink: hasLink),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPanel({required int postCount, required bool hasLink}) {
    final selected = _selected;
    final narrower = selected == null
        ? null
        : identical(selected, _fieldNode(PostField.body))
        ? null
        : narrowTowards(selected, _tapped);
    final wider = selected != null && !identical(selected, _post)
        ? selected.parent
        : null;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Wrap(
          spacing: spacingXs,
          runSpacing: spacingXs,
          children: [
            for (final field in PostField.values)
              _FieldStatus(
                field: field,
                isSet: _matcher.fields.containsKey(field),
              ),
          ],
        ),
        const Gap(spacingXs),
        CaptionText(
          hasLink
              ? 'The fields apply to all $postCount posts of this group.'
              : 'A link is required.',
        ),
        const Gap(spacingSm),
        if (selected == null)
          const CaptionText('Tap the content of the post to label it.')
        else ...[
          SelectionHeader(
            selected: selected,
            onNarrow: narrower == null
                ? null
                : () => setState(() => _selected = narrower),
            onWiden: wider == null
                ? null
                : () => setState(() => _selected = wider),
          ),
          const Gap(spacingXs),
          if (PostField.values.where((f) => f.accepts(selected)).isEmpty)
            const CaptionText('This has no value to extract.')
          else ...[
            Wrap(
              spacing: spacingXs,
              runSpacing: spacingXs,
              children: [
                for (final field in PostField.values)
                  if (field.accepts(selected))
                    FilterChip(
                      label: Text(field.label),
                      side: BorderSide(color: fieldColor(field)),
                      selectedColor: fieldColor(field).withValues(alpha: 0.2),
                      selected: identical(_fieldNode(field), selected),
                      onSelected: _fieldRejection(field, selected) == null
                          ? (_) => _toggleField(field, selected)
                          : null,
                    ),
              ],
            ),
            for (final field in PostField.values)
              if (field.accepts(selected))
                if (_fieldRejection(field, selected) case final rejection?) ...[
                  const Gap(spacingXs),
                  CaptionText(rejection),
                ],
          ],
        ],
      ],
    );
  }
}

class const _FieldStatus({
  required final PostField field,
  required final bool isSet,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          isSet ? Icons.check_circle : Icons.radio_button_unchecked,
          size: 16,
          color: fieldColor(field),
        ),
        const Gap(2),
        CaptionText(field.label),
        const Gap(spacingSm),
      ],
    );
  }
}
