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

  /// The element shown on the selection sheet, while the sheet is shown.
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
      _selected = target;
      _tapped = node;
    });
    if (line.length == 1 || !identical(target, node)) {
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
    return Scaffold(
      appBar: AppBar(
        title: Text('Group ${widget.args.groupNumber}: label the fields'),
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
          if (selected != null)
            SelectionSheet(child: _buildSheet(selected))
          else
            _StatusFooter(
              matcher: _matcher,
              onDone: () => context.pop(_matcher),
            ),
        ],
      ),
    );
  }

  Widget _buildSheet(PageNode selected) {
    final narrower = identical(selected, _fieldNode(PostField.body))
        ? null
        : narrowTowards(selected, _tapped);
    final wider = identical(selected, _post) ? null : selected.parent;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
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
          // A body that contains other fields only disables its chip.
          for (final field in PostField.values)
            if (field != PostField.body && field.accepts(selected))
              if (_fieldRejection(field, selected) case final rejection?) ...[
                const Gap(spacingXs),
                CaptionText(rejection),
              ],
        ],
        const Gap(spacingSm),
        TextButton(
          onPressed: () => setState(() => _selected = null),
          child: const Text('Cancel'),
        ),
      ],
    );
  }
}

/// Which fields are labeled so far, with a button to finish labeling, which
/// is enabled once the link is labeled.
class const _StatusFooter({
  required final PostMatcher matcher,
  required final VoidCallback onDone,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final hasLink = matcher.fields.containsKey(PostField.link);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        const AppDivider(),
        SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.all(spacingMd),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Wrap(
                        spacing: spacingXs,
                        runSpacing: spacingXs,
                        children: [
                          for (final field in PostField.values)
                            _FieldStatus(
                              field: field,
                              isSet: matcher.fields.containsKey(field),
                            ),
                        ],
                      ),
                      if (!hasLink) ...[
                        const Gap(spacingXs),
                        const CaptionText('A link is required.'),
                      ],
                    ],
                  ),
                ),
                const Gap(spacingSm),
                FilledButton(
                  onPressed: hasLink ? onDone : null,
                  child: const Text('Done'),
                ),
              ],
            ),
          ),
        ),
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
