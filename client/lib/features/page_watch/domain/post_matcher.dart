import 'package:paperdoll/features/page_watch/domain/body_markup.dart';
import 'package:paperdoll/features/page_watch/domain/page_document.dart';

/// The attributes of a post that the user can map to nodes of a page.
enum PostField {
  title('Title'),

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
  /// A title is a text value. An image or a time alone is not a body. A pub
  /// date is a time value, since the simplifier makes one from every text
  /// that is a full date.
  bool accepts(PageNode node) => switch (this) {
    title =>
      node.isValue &&
          !node.isImage &&
          !node.isTime &&
          (node.val?.isNotEmpty ?? false),
    body => !node.isImage && !node.isTime && bodyMarkup(node).isNotEmpty,
    link => node.pageLink != null,
    thumbnail => node.isImage,
    pubDate => node.isTime,
  };

  /// The value this field takes when it is mapped to [node].
  String? valueOf(PageNode node) => switch (this) {
    title || pubDate => node.val,
    body => bodyMarkup(node),
    link || thumbnail => node.link,
  };
}

/// One step of the path from a post node to a field: the child with [shape]
/// at the zero-based index [nth] among the children with that shape.
class FieldStep {
  const new({required this.shape, required this.nth});

  final String shape;
  final int nth;

  Map<String, Object?> toJson() => {'shape': shape, 'nth': nth};
}

/// The host and path of [link], which exclude patterns match.
String _hostPath(String link) {
  final uri = Uri.parse(link);
  return '${uri.host}${uri.path}';
}

/// Whether the file glob [pattern] matches [hostPath]: `*` matches any
/// characters except `/`, `**` matches any characters, and `?` matches one
/// character except `/`.
bool _globMatches(String pattern, String hostPath) {
  final re = StringBuffer('^');
  for (var i = 0; i < pattern.length; i++) {
    switch (pattern[i]) {
      case '*' when i + 1 < pattern.length && pattern[i + 1] == '*':
        re.write('.*');
        i++;
      case '*':
        re.write('[^/]*');
      case '?':
        re.write('[^/]');
      case final c:
        re.write(RegExp.escape(c));
    }
  }
  re.write(r'$');
  return RegExp(re.toString()).hasMatch(hostPath);
}

/// The pattern that matches both [pattern] and [hostPath], made by turning
/// the segments that differ into `*`, or null when there is none. Only a
/// pattern that the client generates can be merged: a host followed by
/// segments that are either `*` or literal.
String? _mergePattern(String pattern, String hostPath) {
  final a = pattern.split('/');
  final b = hostPath.split('/');
  if (pattern.contains('**') ||
      pattern.contains('?') ||
      a.any((s) => s != '*' && s.contains('*')) ||
      a.length != b.length ||
      a.first != b.first) {
    return null;
  }
  return [
    for (var i = 0; i < a.length; i++)
      if (a[i] == b[i]) a[i] else '*',
  ].join('/');
}

/// A rule that finds the posts of one shape in a page and extracts their
/// fields (see tool/semantictree/MATCHER.md).
///
/// [post] is the shape path of the post nodes from the root, and each field
/// is the path of steps from a post node to its value. A node at [post] is
/// a post only when all labeled fields are found, and when its link matches
/// none of the patterns in [exclude].
class PostMatcher {
  const new({
    required this.post,
    this.fields = const {},
    this.exclude = const [],
  });

  /// A matcher whose posts are at the shape path of [node].
  factory fromNode(PageNode node) => PostMatcher(
    post: [for (PageNode? n = node; n != null; n = n.parent) n.shape].reversed
        .toList(),
  );

  final List<String> post;
  final Map<PostField, List<FieldStep>> fields;
  final List<String> exclude;

  /// The nodes at [post] that have all labeled fields, including the
  /// excluded ones.
  List<PageNode> _candidates(PageDocument document) {
    var nodes = [if (document.root.shape == post.first) document.root];
    for (final shape in post.skip(1)) {
      nodes = [
        for (final n in nodes) ...n.children.where((c) => c.shape == shape),
      ];
    }
    return [
      for (final n in nodes)
        if (fields.values.every((steps) => _resolve(n, steps) != null)) n,
    ];
  }

  List<PageNode> matchPosts(PageDocument document) => [
    for (final n in _candidates(document))
      if (excludedBy(n) == null) n,
  ];

  /// The nodes that would be posts if their links were not excluded.
  List<PageNode> matchExcluded(PageDocument document) => [
    for (final n in _candidates(document))
      if (excludedBy(n) != null) n,
  ];

  /// The first pattern in [exclude] that matches the link of [postNode].
  String? excludedBy(PageNode postNode) {
    final link = _linkOf(postNode);
    if (link == null) {
      return null;
    }
    final hostPath = _hostPath(link);
    for (final pattern in exclude) {
      if (_globMatches(pattern, hostPath)) {
        return pattern;
      }
    }
    return null;
  }

  String? _linkOf(PageNode postNode) =>
      matchField(postNode, PostField.link)?.link;

  static PageNode? _resolve(PageNode from, List<FieldStep> steps) {
    var node = from;
    for (final step in steps) {
      final matches = node.children.where((c) => c.shape == step.shape);
      if (step.nth >= matches.length) {
        return null;
      }
      node = matches.elementAt(step.nth);
    }
    return node;
  }

  /// The node that provides [field] in [postNode], if any.
  PageNode? matchField(PageNode postNode, PostField field) =>
      switch (fields[field]) {
        final steps? => _resolve(postNode, steps),
        null => null,
      };

  /// Maps [field] to [valueNode] inside [postNode], or removes the mapping
  /// when [valueNode] is null.
  PostMatcher withField(
    PostField field,
    PageNode postNode,
    PageNode? valueNode,
  ) {
    List<FieldStep> steps(PageNode node) => [
      for (var n = node; !identical(n, postNode); n = n.parent!)
        FieldStep(
          shape: n.shape,
          nth: n.parent!.children
              .where((c) => c.shape == n.shape)
              .toList()
              .indexOf(n),
        ),
    ].reversed.toList();
    return PostMatcher(
      post: post,
      fields: {
        for (final MapEntry(:key, :value) in fields.entries)
          if (key != field) key: value,
        if (valueNode != null) field: steps(valueNode),
      },
      exclude: exclude,
    );
  }

  /// Labels the fields that are evident in [postNode]: the only link or time,
  /// the only image at the shallowest level as the thumbnail, and the only
  /// heading of the highest level as the title. Without headings, the title
  /// is the only text, such as the text of a post that is a single link.
  ///
  /// Nodes that link to the same URL count as one link, such as a card and a
  /// title in it. Of those, the post node itself is labeled, or else the one
  /// with text. A title with a link of its own provides the link of the post,
  /// even when there are other links.
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

    final links = nodes.where((n) => n.pageLink != null);
    if (links.map((n) => n.link).toSet().length == 1) {
      label(
        PostField.link,
        links.where((n) => identical(n, postNode)).firstOrNull ??
            links.where((n) => n.textContent.isNotEmpty).firstOrNull ??
            links.first,
      );
    }
    final images = nodes.where((n) => n.isImage).toList();
    if (images.isNotEmpty) {
      final top = images.map((n) => n.depth).reduce((a, b) => a < b ? a : b);
      label(PostField.thumbnail, only(images.where((n) => n.depth == top)));
    }
    label(PostField.pubDate, only(nodes.where((n) => n.isTime)));
    final headings = nodes.where((n) => n.headingLevel != null).toList();
    final PageNode? title;
    if (headings.isEmpty) {
      title = only(nodes.where(PostField.title.accepts));
    } else {
      final top = headings
          .map((n) => n.headingLevel!)
          .reduce((a, b) => a < b ? a : b);
      title = only(headings.where((n) => n.headingLevel == top));
    }
    label(PostField.title, title);
    if (title?.pageLink case final link?
        when matcher.matchField(postNode, PostField.link)?.link != link) {
      label(PostField.link, title);
    }
    return matcher;
  }

  /// Adds a pattern that excludes [link] (see "Excluding a link" in
  /// MATCHER.md), or returns null when every pattern would also exclude a
  /// post of [document] with another link.
  PostMatcher? withExcluded(PageDocument document, String link) {
    final hostPath = _hostPath(link);
    final kept = [
      for (final p in matchPosts(document))
        if (_linkOf(p) case final l? when l != link) _hostPath(l),
    ];
    bool isAccepted(String pattern) =>
        !kept.any((l) => _globMatches(pattern, l));
    PostMatcher withExclude(List<String> exclude) =>
        PostMatcher(post: post, fields: fields, exclude: exclude);

    for (final (i, pattern) in exclude.indexed) {
      if (_mergePattern(pattern, hostPath) case final merged?
          when isAccepted(merged)) {
        return withExclude([...exclude]..[i] = merged);
      }
    }
    if (isAccepted(hostPath)) {
      return withExclude([...exclude, hostPath]);
    }
    return null;
  }

  /// Removes [pattern] from [exclude], so the posts it excluded come back.
  PostMatcher withoutExcluded(String pattern) => PostMatcher(
    post: post,
    fields: fields,
    exclude: [
      for (final p in exclude)
        if (p != pattern) p,
    ],
  );

  Map<String, Object?> toJson() => {
    'post': post,
    'fields': {
      for (final MapEntry(:key, :value) in fields.entries)
        key.name: [for (final s in value) s.toJson()],
    },
    'exclude': exclude,
  };
}

/// A post extracted from a page: its node, the index of the matcher that
/// found it, and its field values.
typedef ExtractedPost = ({
  PageNode node,
  int matcher,
  Map<PostField, String> values,
});

/// The posts that [matchers] find in [document], in document order. Posts
/// with the same link are merged into the first one, which takes the fields
/// that it does not have from the later ones, since some pages split a post
/// into sibling cards with the same link, such as one with the image and one
/// with the title.
List<ExtractedPost> extractPosts(
  PageDocument document,
  List<PostMatcher> matchers,
) {
  final found = <ExtractedPost>[
    for (final (i, matcher) in matchers.indexed)
      for (final node in matcher.matchPosts(document))
        (
          node: node,
          matcher: i,
          values: {
            for (final field in matcher.fields.keys)
              if (matcher.matchField(node, field) case final n?)
                field: ?field.valueOf(n),
          },
        ),
  ]..sort((a, b) => a.node.id.compareTo(b.node.id));
  final byLink = <String, ExtractedPost>{};
  for (final post in found) {
    final link = post.values[PostField.link];
    if (link == null) {
      continue;
    }
    if (byLink[link] case final first?) {
      for (final MapEntry(:key, :value) in post.values.entries) {
        first.values.putIfAbsent(key, () => value);
      }
    } else {
      byLink[link] = post;
    }
  }
  return byLink.values.toList();
}
