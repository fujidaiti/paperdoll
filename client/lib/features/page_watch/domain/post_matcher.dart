import 'package:paperdoll/features/page_watch/domain/body_markup.dart';
import 'package:paperdoll/features/page_watch/domain/page_document.dart';

/// The attributes of a post that the user can map to nodes of a page.
enum PostField {
  title('Title'),
  description('Description'),

  /// The content of the post as marked-up text (see [bodyMarkup]). A body
  /// node cannot contain the nodes of other fields.
  body('Body'),
  link('Link'),
  thumbnail('Thumbnail'),
  pubDate('Pub date');

  new(this.label);

  final String label;

  /// Whether [node] can provide a value for this field.
  bool accepts(PageNode node) => switch (this) {
    title || description => node.textContent.isNotEmpty,
    body => bodyMarkup(node).isNotEmpty,
    link => node.href != null,
    thumbnail => node.isImage,
    pubDate => node.datetime != null || node.textContent.isNotEmpty,
  };

  /// The value this field takes when it is mapped to [node].
  String? valueOf(PageNode node) => switch (this) {
    title || description => node.textContent,
    body => bodyMarkup(node),
    link => node.href,
    thumbnail => node.src,
    pubDate => node.datetime ?? node.textContent,
  };
}

/// One step of a selector. It matches the child nodes that have [tag] and at
/// least all of [classes], so that state classes such as "featured" on some
/// posts do not break the match. [index] picks the n-th of those matches
/// (zero-based); a step without an index keeps all of them.
class SelectorStep {
  const new({required this.tag, required this.classes, this.index});

  final String tag;
  final List<String> classes;
  final int? index;

  bool matches(PageNode node) =>
      node.tag == tag && classes.every(node.classes.contains);

  Map<String, Object?> toJson() => {
    'tag': tag,
    if (classes.isNotEmpty) 'cls': classes,
    'nth': ?index,
  };

  @override
  String toString() => [
    tag,
    ...classes.map((c) => '.$c'),
    if (index case final index?) ':nth($index)',
  ].join();
}

/// Builds the path of steps that leads from [ancestor] down to [node]. A step
/// gets an index only when its tag and classes match more than one sibling.
/// With [generalize], the last step has no index so that the selector also
/// matches the siblings that share the shape of [node].
List<SelectorStep> buildSelector(
  PageNode ancestor,
  PageNode node, {
  bool generalize = false,
}) {
  final steps = <SelectorStep>[];
  for (var n = node; !identical(n, ancestor); n = n.parent!) {
    final step = SelectorStep(tag: n.tag, classes: n.classes);
    final isLast = steps.isEmpty;
    final matches = n.parent!.children.where(step.matches).toList();
    steps.add(
      matches.length > 1 && !(isLast && generalize)
          ? SelectorStep(
              tag: n.tag,
              classes: n.classes,
              index: matches.indexOf(n),
            )
          : step,
    );
  }
  return steps.reversed.toList();
}

/// Evaluates [steps] from [from]. An empty selector selects [from] itself.
List<PageNode> select(PageNode from, List<SelectorStep> steps) {
  var current = [from];
  for (final step in steps) {
    current = [
      for (final node in current)
        ...switch (node.children.where(step.matches).toList()) {
          final matches when step.index == null => matches,
          final matches when step.index! < matches.length => [
            matches[step.index!],
          ],
          _ => const <PageNode>[],
        },
    ];
  }
  return current;
}

/// A rule that finds posts sharing the same shape in a page and extracts
/// their fields. [post] selects the post nodes from the document root, and
/// each selector in [fields] selects a value node relative to a post node.
class PostMatcher {
  const new({required this.post, this.fields = const {}});

  /// A matcher whose posts are [node] and its siblings of the same shape.
  factory fromNode(PageDocument document, PageNode node) =>
      PostMatcher(post: buildSelector(document.root, node, generalize: true));

  final List<SelectorStep> post;
  final Map<PostField, List<SelectorStep>> fields;

  List<PageNode> matchPosts(PageDocument document) =>
      select(document.root, post);

  /// The node that provides [field] in [postNode], if any. When the selector
  /// matches several nodes, the first one wins.
  PageNode? matchField(PageNode postNode, PostField field) =>
      switch (fields[field]) {
        final steps? => select(postNode, steps).firstOrNull,
        null => null,
      };

  /// Maps [field] to [valueNode] inside [postNode], or removes the mapping
  /// when [valueNode] is null.
  PostMatcher withField(
    PostField field,
    PageNode postNode,
    PageNode? valueNode,
  ) => PostMatcher(
    post: post,
    fields: {
      for (final MapEntry(:key, :value) in fields.entries)
        if (key != field) key: value,
      if (valueNode != null) field: buildSelector(postNode, valueNode),
    },
  );

  Map<String, Object?> toJson() => {
    'post': [for (final s in post) s.toJson()],
    'fields': {
      for (final MapEntry(:key, :value) in fields.entries)
        key.name: [for (final s in value) s.toJson()],
    },
  };
}
