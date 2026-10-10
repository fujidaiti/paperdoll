import 'dart:math';

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

  /// Whether [node] can provide a sensible value for this field.
  ///
  /// A title or a description is plain text, so it holds no image, embed
  /// or date. An image or a date alone is not a body, nor is a link without
  /// content, but an embed alone is, such as the video of a video post. A
  /// date in text is a short text with a digit, such as "Oct 1, 2026" or "3
  /// hours ago".
  bool accepts(PageNode node) {
    final text = node.textContent;
    return switch (this) {
      title || description =>
        text.isNotEmpty &&
            !_anyInside(node, (n) => n.tag == 'time' || n.isImage || n.isEmbed),
      body =>
        node.tag != 'time' &&
            !node.isImage &&
            !(node.href != null && node.children.isEmpty) &&
            bodyMarkup(node).isNotEmpty,
      link => node.href != null,
      thumbnail => node.isImage,
      pubDate =>
        node.datetime != null ||
            (node.children.isEmpty &&
                text.length <= 40 &&
                text.contains(RegExp(r'\d'))),
    };
  }

  /// The value this field takes when it is mapped to [node].
  String? valueOf(PageNode node) => switch (this) {
    title || description => node.textContent,
    body => bodyMarkup(node),
    link => node.href,
    thumbnail => node.src,
    pubDate => node.datetime ?? node.textContent,
  };
}

/// Whether [node] or any node inside it passes [test].
bool _anyInside(PageNode node, bool Function(PageNode n) test) =>
    test(node) || node.children.any((c) => _anyInside(c, test));

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
/// matches the siblings that share the shape of [node]. Its classes are only
/// the ones that those siblings share, since posts often have classes of
/// their own, such as "post-9850" or "tag-databases".
///
/// A sibling shares the shape of [node] when it is of the same kind and so
/// are most of its children. Others, such as a sidebar widget or a grid of
/// several posts, do not reduce the classes. If the shared classes still
/// match such a sibling, the step keeps all classes of [node].
List<SelectorStep> buildSelector(
  PageNode ancestor,
  PageNode node, {
  bool generalize = false,
}) {
  bool isSameKind(PageNode a, PageNode b) =>
      a.tag == b.tag &&
      (a.classes.isEmpty && b.classes.isEmpty ||
          a.classes.any(b.classes.contains));
  bool isSameShape(PageNode a, PageNode b) =>
      isSameKind(a, b) &&
      a.children.where((c) => b.children.any((d) => isSameKind(c, d))).length *
              2 >=
          max(a.children.length, b.children.length);

  final steps = <SelectorStep>[];
  for (var n = node; !identical(n, ancestor); n = n.parent!) {
    final isLast = steps.isEmpty;
    var step = SelectorStep(tag: n.tag, classes: n.classes);
    if (isLast && generalize) {
      final siblings = n.parent!.children;
      final shared = SelectorStep(
        tag: n.tag,
        classes: [
          for (final c in n.classes)
            if (siblings
                .where((s) => isSameShape(n, s))
                .every((s) => s.classes.contains(c)))
              c,
        ],
      );
      if (siblings.every((s) => !shared.matches(s) || isSameShape(n, s))) {
        step = shared;
      }
    }
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

  /// Labels the fields that are evident in [postNode]: the only link, image
  /// or time, and the only heading of the highest level as the title. A link
  /// in the title is the link of the post, even when there are other links.
  ///
  /// A heading around a single node is shown as that node, so the innermost
  /// such node is labeled. As a result, a title that is a link is labeled on
  /// the link, which then has both labels and is selected as one element.
  PostMatcher withEvidentFields(PageNode postNode) {
    final nodes = <PageNode>[];
    void collect(PageNode n) {
      nodes.add(n);
      n.children.forEach(collect);
    }

    collect(postNode);
    PageNode? only(Iterable<PageNode> matches) =>
        matches.length == 1 ? matches.first : null;
    var matcher = this;
    void label(PostField field, PageNode? node) {
      if (node != null && field.accepts(node)) {
        matcher = matcher.withField(field, postNode, node);
      }
    }

    label(PostField.link, only(nodes.where((n) => n.href != null)));
    label(PostField.thumbnail, only(nodes.where((n) => n.isImage)));
    label(PostField.pubDate, only(nodes.where((n) => n.tag == 'time')));
    // The level of a heading (1 for h1), or null for other nodes.
    int? level(PageNode n) => switch (RegExp(r'^h([1-6])$').firstMatch(n.tag)) {
      final m? => int.parse(m[1]!),
      null => null,
    };
    final headings = nodes.where((n) => level(n) != null).toList();
    if (headings.isNotEmpty) {
      final top = headings.map((n) => level(n)!).reduce(min);
      if (only(headings.where((n) => level(n) == top)) case var title?) {
        while (title.children.length == 1) {
          title = title.children.single;
        }
        label(PostField.title, title);
        if (identical(matcher.matchField(postNode, PostField.title), title)) {
          label(
            PostField.link,
            only(nodes.where((n) => n.href != null && title.contains(n))),
          );
        }
      }
    }
    return matcher;
  }

  Map<String, Object?> toJson() => {
    'post': [for (final s in post) s.toJson()],
    'fields': {
      for (final MapEntry(:key, :value) in fields.entries)
        key.name: [for (final s in value) s.toJson()],
    },
  };
}
