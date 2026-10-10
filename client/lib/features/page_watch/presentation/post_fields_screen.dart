import 'dart:async';
import 'dart:convert';

import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/app_divider.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
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
/// with nothing when the user leaves without finishing. Leaving asks first
/// when there are changes, so that they are not lost by accident.
///
/// A labeled element is selected as a whole, so a tap inside it selects the
/// element rather than the content under the finger. A body is converted as a
/// whole, so the content inside it cannot be labeled.
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

  /// Whether the user has labeled the selection since it was selected. The
  /// labels of such a selection can still be changed on the sheet, while a
  /// selection that was already labeled shows only its labels.
  var _isLabeling = false;

  PageNode get _post => widget.args.post;

  PageNode? _fieldNode(PostField field) => _matcher.matchField(_post, field);

  /// Whether [field] can be labeled on [node]: a body cannot contain other
  /// fields, and other fields cannot be part of the body.
  bool _canLabel(PostField field, PageNode node) {
    if (field == PostField.body) {
      for (final other in PostField.values) {
        if (other == PostField.body) {
          continue;
        }
        if (_fieldNode(other) case final n? when node.contains(n)) {
          return false;
        }
      }
      return true;
    }
    final body = _fieldNode(PostField.body);
    return body == null || !body.contains(node);
  }

  /// Whether [field] cannot be added to an element labeled with [labels].
  /// An element provides at most one text value: a title, a description, a
  /// body or a date. A link can be added to any of them, such as a title
  /// that links to the post.
  bool _conflicts(PostField field, List<PostField> labels) {
    const texts = {
      PostField.title,
      PostField.description,
      PostField.body,
      PostField.pubDate,
    };
    return texts.contains(field) &&
        labels.any((label) => label != field && texts.contains(label));
  }

  /// Labels [node] as [field], or removes the label if [node] has it.
  void _toggleField(PostField field, PageNode node) {
    setState(() {
      _matcher = _matcher.withField(
        field,
        _post,
        identical(_fieldNode(field), node) ? null : node,
      );
      _isLabeling = true;
    });
  }

  /// Closes the sheet on back, and asks before leaving with changes.
  Future<void> _onBack() async {
    if (_selected != null) {
      setState(() => _selected = null);
      return;
    }
    final discard = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Discard the changes?'),
        content: const Text('The fields you labeled will be lost.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Keep labeling'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Discard'),
          ),
        ],
      ),
    );
    if ((discard ?? false) && mounted) {
      context.pop();
    }
  }

  Future<void> _onTapLine(List<PageNode> line) async {
    // A line of several pieces selects the content that holds them, and its
    // pieces, which are too small to tap, are picked from a sheet.
    final node = line.length == 1 ? line.first : line.first.parent!;
    // The innermost labeled element that the line is part of. A link around
    // other content is labeled only on its own line, as it is tinted.
    PageNode? labeled;
    for (final field in PostField.values) {
      final n = _fieldNode(field);
      if (n == null) {
        continue;
      }
      final covers = field == PostField.link && n.children.isNotEmpty
          ? identical(line.first, n)
          : line.every(n.contains);
      if (covers && (labeled == null || labeled.contains(n))) {
        labeled = n;
      }
    }
    final target = labeled ?? node;
    setState(() {
      _selected = target;
      _tapped = node;
      _isLabeling = false;
    });
    if (line.length == 1 || !identical(target, node)) {
      return;
    }
    final piece = await pickLinePiece(context, line);
    if (piece != null && mounted && identical(_selected, node)) {
      setState(() {
        _selected = piece;
        _tapped = piece;
        _isLabeling = false;
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
    final isChanged =
        jsonEncode(_matcher.toJson()) !=
        jsonEncode(widget.args.matcher.toJson());
    return PopScope(
      canPop: selected == null && !isChanged,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) {
          unawaited(_onBack());
        }
      },
      child: Scaffold(
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
      ),
    );
  }

  Widget _buildSheet(PageNode selected) {
    final from = _tapped ?? selected;
    final labels = [
      for (final field in PostField.values)
        if (identical(_fieldNode(field), selected)) field,
    ];
    // The fields that can be labeled. One that cannot is left out rather
    // than disabled, since a disabled chip does not tell why.
    final fields = [
      for (final field in PostField.values)
        if (labels.contains(field) ||
            (field.accepts(selected) && _canLabel(field, selected)))
          field,
    ];
    // A labeled element from an earlier session can be one that the slider
    // leaves out.
    var levels = selectionLevels(from, top: _post);
    if (!levels.contains(selected)) {
      levels = selectionLevels(selected, top: _post);
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        // A labeled element is selected as a whole.
        if (labels.isEmpty)
          SelectionSlider(
            selected: selected,
            levels: levels,
            onSelect: (node) => setState(() {
              _selected = node;
              _isLabeling = false;
            }),
          )
        else
          Row(
            children: [
              Expanded(child: BodyText(labels.map((f) => f.label).join(', '))),
              TextButton(
                onPressed: () => setState(() {
                  for (final field in labels) {
                    _matcher = _matcher.withField(field, _post, null);
                  }
                }),
                child: const Text('Remove'),
              ),
            ],
          ),
        if (labels.isEmpty || _isLabeling) ...[
          const Gap(spacingXs),
          if (fields.isEmpty)
            const CaptionText('This cannot be labeled.')
          else
            Wrap(
              spacing: spacingXs,
              runSpacing: spacingXs,
              children: [
                for (final field in fields)
                  FilterChip(
                    label: Text(field.label),
                    side: BorderSide(color: fieldColor(field)),
                    selectedColor: fieldColor(field).withValues(alpha: 0.2),
                    selected: labels.contains(field),
                    onSelected: _conflicts(field, labels)
                        ? null
                        : (_) => _toggleField(field, selected),
                  ),
              ],
            ),
        ],
        const Gap(spacingSm),
        TextButton(
          onPressed: () => setState(() => _selected = null),
          // A labeled element has nothing waiting to be done.
          child: Text(labels.isEmpty ? 'Cancel' : 'Close'),
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
                  child: Wrap(
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
