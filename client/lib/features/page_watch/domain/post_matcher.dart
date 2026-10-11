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
  /// A title is a single text, so it is an element without children, which
  /// is shown as one line rather than a card. A description is plain text,
  /// so it holds no image, embed or date. An image or a date alone is not a
  /// body, nor is a link without content, but an embed alone is, such as the
  /// video of a video post. A date in text is a short text with a digit, such
  /// as "Oct 1, 2026" or "3 hours ago".
  bool accepts(PageNode node) {
    final text = node.textContent;
    return switch (this) {
      title =>
        node.children.isEmpty &&
            text.isNotEmpty &&
            node.tag != 'time' &&
            !node.isImage &&
            !node.isEmbed,
      description =>
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

/// A position in the tree of the elements of a page: the element `layer` of
/// `node` (see [PageNode.elements]).
typedef _At = ({PageNode node, int layer});

/// The innermost element of [node].
_At _inside(PageNode node) => (node: node, layer: node.elements.length - 1);

extension on _At {
  PageElement get element => node.elements[layer];

  _At get parent =>
      layer > 0 ? (node: node, layer: layer - 1) : _inside(node.parent!);

  List<_At> get children => layer < node.elements.length - 1
      ? [(node: node, layer: layer + 1)]
      : [for (final c in node.children) (node: c, layer: 0)];
}

/// One step of a selector. It matches the child elements that have [tag]
/// and at least all of [classes], so that state classes such as "featured"
/// on some posts do not break the match. [index] picks the n-th of those
/// matches (zero-based); a step without an index keeps all of them.
///
/// A step with [has] matches only the elements with a child that [has]
/// matches, such as the links around one kind of card when the links of all
/// kinds of cards have the same classes, or the one of several spans that
/// holds a time.
class SelectorStep {
  const new({required this.tag, required this.classes, this.index, this.has});

  final String tag;
  final List<String> classes;
  final int? index;
  final SelectorStep? has;

  bool _matches(_At at) =>
      at.element.tag == tag &&
      classes.every(at.element.classes.contains) &&
      (has == null || at.children.any(has!._matches));

  Map<String, Object?> toJson() => {
    'tag': tag,
    if (classes.isNotEmpty) 'cls': classes,
    'nth': ?index,
    'has': ?has?.toJson(),
  };

  @override
  String toString() => [
    tag,
    ...classes.map((c) => '.$c'),
    if (has case final has?) ':has(> $has)',
    if (index case final index?) ':nth($index)',
  ].join();
}

/// Builds the path of steps that leads from [from] down to the innermost
/// element of [node]. With [generalize], one step has no index so that the
/// selector also matches the siblings that share the shape of [node]. Its
/// classes are only the ones that those siblings share, since posts often
/// have classes of their own, such as "post-9850" or "tag-databases".
///
/// A sibling shares the shape of [node] when it is of the same kind and so
/// are most of its children. Others, such as a sidebar widget or a grid of
/// several posts, do not reduce the classes. If the shared classes, with
/// the steps below, still match such a sibling, the step keeps all classes
/// of the element, or else the innermost element of [node] requires a child
/// that the siblings of the same shape have and the others do not. If there
/// is no such child, the step matches [node] only.
///
/// When [node] has no such siblings but is the only content with text of its
/// parent, such as a link in a table row next to a spacer image, the parent
/// is what repeats. In that case, the step of the nearest such ancestor that
/// has siblings of its shape is generalized instead.
///
/// The steps below the generalized one, or all steps for the fields of a
/// post given the other posts as [others], keep only the classes that the
/// matching elements of the others share (see [_steps]).
List<SelectorStep> _buildSelector(
  _At from,
  PageNode node, {
  bool generalize = false,
  List<_At> others = const [],
}) {
  bool isSameKind(_At a, _At b) =>
      a.element.tag == b.element.tag &&
      (a.element.classes.isEmpty && b.element.classes.isEmpty ||
          a.element.classes.any(b.element.classes.contains));
  // A wrapper around a single element has the shape of that element.
  bool isSameShape(_At a, _At b) =>
      isSameKind(a, b) &&
      (a.children.length == 1 && b.children.length == 1
          ? isSameShape(a.children.single, b.children.single)
          : a.children
                        .where((c) => b.children.any((d) => isSameKind(c, d)))
                        .length *
                    2 >=
                max(a.children.length, b.children.length));

  bool hasSameShapeSibling(PageNode n) {
    final at = (node: n, layer: 0);
    return at.parent.children.any((s) => s != at && isSameShape(at, s));
  }

  final path = [for (var at = _inside(node); at != from; at = at.parent) at]
      .reversed
      .toList();

  // The node whose outermost element is generalized.
  PageNode? general;
  if (generalize) {
    general = node;
    while (!hasSameShapeSibling(general!) &&
        !identical(general.parent, from.node) &&
        general.parent!.children.every(
          (c) => identical(c, general) || c.textContent.isEmpty,
        )) {
      general = general.parent;
    }
    if (!hasSameShapeSibling(general)) {
      general = null;
    }
  }
  final g = general == null ? -1 : path.indexOf((node: general, layer: 0));
  if (g < 0) {
    return _steps(path, others);
  }

  final at = path[g];
  final element = at.element;
  final siblings = at.parent.children;
  final sameShape = [
    for (final s in siblings)
      if (isSameShape(at, s)) s,
  ];
  final classes = [
    for (final c in element.classes)
      if (sameShape.every((s) => s.element.classes.contains(c))) c,
  ];
  final below = _steps(path.sublist(g + 1), [
    for (final s in sameShape)
      if (s != at) s,
  ]);
  // Requires a child of the innermost element of the general node, which is
  // where the content of the post starts.
  final last = general!.elements.length - 1;
  final base = SelectorStep(tag: element.tag, classes: classes);
  List<SelectorStep> withHas(SelectorStep has) => [
    for (final (i, step) in [base, ...below].indexed)
      if (i == last)
        SelectorStep(
          tag: step.tag,
          classes: step.classes,
          index: step.index,
          has: has,
        )
      else
        step,
  ];
  // The first steps that match exactly the siblings of the same shape. The
  // steps below the general one can tell them apart by the elements inside,
  // such as the article in a list item.
  final unit =
      [
            [base, ...below],
            [
              SelectorStep(tag: element.tag, classes: element.classes),
              ...below,
            ],
            // Only when each of them sees the same siblings of its shape, or
            // else a child can pick some posts of one kind with some of
            // another, such as a featured post with one other post.
            if (sameShape.every(
              (s) => siblings
                  .where((t) => isSameShape(s, t))
                  .every(sameShape.contains),
            ))
              for (final child in _inside(general).children)
                withHas(
                  SelectorStep(
                    tag: child.element.tag,
                    classes: [
                      for (final c in child.element.classes)
                        if (sameShape.every(
                          (s) => _inside(s.node).children.any(
                            (d) =>
                                d.element.tag == child.element.tag &&
                                d.element.classes.contains(c),
                          ),
                        ))
                          c,
                    ],
                  ),
                ),
          ]
          .where(
            (steps) => siblings.every(
              (s) =>
                  (steps.first._matches(s) &&
                      _select(s, steps.sublist(1)).isNotEmpty) ==
                  sameShape.contains(s),
            ),
          )
          .firstOrNull;
  return [
    ..._steps(path.sublist(0, g), const []),
    ...unit ??
        [
          ..._steps([at], const []),
          ...below,
        ],
  ];
}

/// Builds a step for each element of [path], each of which is a child of the
/// previous one. A step matches its element only among its siblings. It
/// requires a child that the siblings do not have before it picks the
/// element by its index, since such a child, such as the time in the one of
/// several spans that holds the date, is more likely to stay with the
/// element when a page changes.
///
/// [others] are the elements that correspond to the parent of the first
/// element in other parts of the page, such as other posts. A step keeps
/// only the classes that the matching elements under all of them share,
/// since classes such as "wp-image-60882" belong to one post. For the same
/// reason, the classes of the element or of a required child are used only
/// when all of them have such an element.
List<SelectorStep> _steps(List<_At> path, List<_At> others) {
  var frontiers = [
    for (final o in others) [o],
  ];
  final steps = <SelectorStep>[];
  for (final at in path) {
    final element = at.element;
    final nexts = [
      for (final f in frontiers)
        [
          for (final p in f)
            ...p.children.where((c) => c.element.tag == element.tag),
        ],
    ].where((n) => n.isNotEmpty).toList();
    final classes = [
      for (final c in element.classes)
        if (nexts.every((n) => n.any((p) => p.element.classes.contains(c)))) c,
    ];
    final siblings = at.parent.children;
    bool isOnly(SelectorStep step) =>
        step._matches(at) && siblings.where(step._matches).length == 1;
    bool isShared(SelectorStep step) =>
        nexts.every((n) => n.any(step._matches));
    final base = SelectorStep(tag: element.tag, classes: classes);
    final step =
        (isOnly(base) ? base : null) ??
        [
          for (final (step, shared) in [
            (SelectorStep(tag: element.tag, classes: element.classes), true),
            for (final child in at.children)
              for (final (has, shared) in [
                (
                  SelectorStep(tag: child.element.tag, classes: const []),
                  false,
                ),
                (
                  SelectorStep(
                    tag: child.element.tag,
                    classes: child.element.classes,
                  ),
                  true,
                ),
              ])
                (
                  SelectorStep(tag: element.tag, classes: classes, has: has),
                  shared,
                ),
          ])
            if (isOnly(step) && (!shared || isShared(step))) step,
        ].firstOrNull ??
        SelectorStep(
          tag: element.tag,
          classes: classes,
          index: siblings.where(base._matches).toList().indexOf(at),
        );
    steps.add(step);
    frontiers = [for (final n in nexts) n.where(step._matches).toList()];
  }
  return steps;
}

/// Evaluates [steps] from [from]. An empty selector selects [from] itself.
List<_At> _select(_At from, List<SelectorStep> steps) {
  var current = [from];
  for (final step in steps) {
    current = [
      for (final at in current)
        ...switch (at.children.where(step._matches).toList()) {
          final matches when step.index == null => matches,
          final matches when step.index! < matches.length => [
            matches[step.index!],
          ],
          _ => const <_At>[],
        },
    ];
  }
  return current;
}

/// A rule that finds posts sharing the same shape in a page and extracts
/// their fields. [post] selects the post elements from the document root,
/// and each selector in [fields] selects a value element relative to the
/// element of a post.
class PostMatcher {
  const new({required this.post, this.fields = const {}});

  /// A matcher whose posts are [node] and its siblings of the same shape.
  factory fromNode(PageDocument document, PageNode node) => PostMatcher(
    post: _buildSelector(_inside(document.root), node, generalize: true),
  );

  final List<SelectorStep> post;
  final Map<PostField, List<SelectorStep>> fields;

  List<PageNode> matchPosts(PageDocument document) =>
      _matchPosts(document.root);

  List<PageNode> _matchPosts(PageNode root) => [
    for (final at in _select(_inside(root), post)) at.node,
  ];

  /// The element of [postNode] that [post] matches, from which the fields
  /// are selected. It is the innermost one, since [post] leads to the
  /// innermost element of the node it is built from.
  _At _postAt(PageNode postNode) {
    for (var layer = postNode.elements.length - 1; layer > 0; layer--) {
      if (post.last._matches((node: postNode, layer: layer))) {
        return (node: postNode, layer: layer);
      }
    }
    return (node: postNode, layer: 0);
  }

  /// The node that provides [field] in [postNode], if any. When the selector
  /// matches several nodes, the first one wins.
  PageNode? matchField(PageNode postNode, PostField field) =>
      switch (fields[field]) {
        final steps? => _select(_postAt(postNode), steps).firstOrNull?.node,
        null => null,
      };

  /// Maps [field] to [valueNode] inside [postNode], or removes the mapping
  /// when [valueNode] is null.
  PostMatcher withField(
    PostField field,
    PageNode postNode,
    PageNode? valueNode,
  ) {
    var root = postNode;
    while (root.parent != null) {
      root = root.parent!;
    }
    return PostMatcher(
      post: post,
      fields: {
        for (final MapEntry(:key, :value) in fields.entries)
          if (key != field) key: value,
        if (valueNode != null)
          field: _buildSelector(
            _postAt(postNode),
            valueNode,
            others: [
              for (final p in _matchPosts(root))
                if (!identical(p, postNode)) _postAt(p),
            ],
          ),
      },
    );
  }

  /// Labels the fields that are evident in [postNode]: the only link or time,
  /// the only image at the shallowest level as the thumbnail, and the only
  /// heading of the highest level as the title. A link in the title is the
  /// link of the post, even when there are other links. Without headings,
  /// the title is the only text, such as the text of a post that is a single
  /// link, or else the text of the labeled link, unless the link is a card.
  ///
  /// Links to the same URL count as one link, such as the image and the
  /// title of a card. Of those, the one with text is labeled, since it is
  /// usually the title, which can then have both labels.
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

    PageNode? onlyLink(Iterable<PageNode> nodes) {
      final links = nodes.where((n) => n.href != null);
      if (links.map((n) => n.href).toSet().length != 1) {
        return null;
      }
      return links.where((n) => n.textContent.isNotEmpty).firstOrNull ??
          links.first;
    }

    label(PostField.link, onlyLink(nodes));
    final images = nodes.where((n) => n.isImage).toList();
    if (images.isNotEmpty) {
      final top = images.map((n) => n.depth).reduce(min);
      label(PostField.thumbnail, only(images.where((n) => n.depth == top)));
    }
    label(PostField.pubDate, only(nodes.where((n) => n.tag == 'time')));
    // The level of a heading (1 for h1), or null for other nodes.
    int? level(PageNode n) => switch (RegExp(r'^h([1-6])$').firstMatch(n.tag)) {
      final m? => int.parse(m[1]!),
      null => null,
    };
    final headings = nodes.where((n) => level(n) != null).toList();
    PageNode? title;
    if (headings.isEmpty) {
      title =
          only(
            nodes.where((n) => n.children.isEmpty && n.textContent.isNotEmpty),
          ) ??
          matcher.matchField(postNode, PostField.link);
    } else {
      final top = headings.map((n) => level(n)!).reduce(min);
      title = only(headings.where((n) => level(n) == top));
      while (title != null && title.children.length == 1) {
        title = title.children.single;
      }
    }
    if (title != null) {
      label(PostField.title, title);
      if (identical(matcher.matchField(postNode, PostField.title), title)) {
        label(PostField.link, onlyLink(nodes.where(title.contains)));
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
