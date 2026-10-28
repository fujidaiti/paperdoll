import 'package:paperdoll/features/feed/domain/body_text.dart';
import 'package:paperdoll/features/feed/domain/semantic_tree.dart';

/// The attributes of a post that the user can map to nodes of a page.
enum PostField {
  title('Title'),

  /// The text of the post as one line (see [bodyText]). A body
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
  bool accepts(SemanticTreeNode node) => switch (this) {
    title =>
      node.isValue &&
          !node.isImage &&
          !node.isTime &&
          (node.val?.isNotEmpty ?? false),
    body => !node.isImage && !node.isTime && bodyText(node).isNotEmpty,
    link => node.pageLink != null,
    thumbnail => node.isImage,
    pubDate => node.isTime,
  };

  /// The value this field takes when it is mapped to [node].
  String? valueOf(SemanticTreeNode node) => switch (this) {
    title || pubDate => node.val,
    body => bodyText(node),
    link || thumbnail => node.link,
  };
}

/// Selects nodes by their tag and classes, as a step of a CSS descendant
/// selector does: the descendants of a node that have [tag] and every class in
/// [cls]. A step without [tag] selects plain texts, which have no tag. When
/// [nth] is set, only the one at that zero-based index among them in document
/// order is selected.
class Step {
  const new({this.tag, this.cls = const [], this.nth});

  /// The step of [node]: its tag and the classes that other nodes of [tree]
  /// have too. A class of [node] alone, such as an ID, cannot be shared by
  /// two posts.
  factory of(SemanticTree tree, SemanticTreeNode node) => Step(
    tag: node.tag,
    cls: [
      for (final c in node.cls)
        if ((tree.classCounts[c] ?? 0) >= 2) c,
    ],
  );

  factory fromJson(Map<String, Object?> json) => Step(
    tag: json['tag'] as String?,
    cls: (json['cls'] as List<Object?>? ?? const []).cast<String>(),
    nth: json['nth'] as int?,
  );

  final String? tag;
  final List<String> cls;
  final int? nth;

  bool matches(SemanticTreeNode node) =>
      node.tag == tag && cls.every(node.cls.contains);

  Map<String, Object?> toJson() => {
    'tag': ?tag,
    if (cls.isNotEmpty) 'cls': cls,
    'nth': ?nth,
  };
}

/// The nodes that [chain] selects from [start], in document order. With
/// [children], each step after the first selects among the children of the
/// nodes that the previous step selects, instead of their descendants.
List<SemanticTreeNode> _selectNodes(
  SemanticTreeNode start,
  List<Step> chain, {
  required bool children,
}) {
  var nodes = [start];
  for (final (i, step) in chain.indexed) {
    final next = <SemanticTreeNode>{};
    for (final n in nodes) {
      final matches = <SemanticTreeNode>[];
      void visit(SemanticTreeNode n) {
        for (final c in n.children) {
          if (step.matches(c)) {
            matches.add(c);
          }
          if (!children || i == 0) {
            visit(c);
          }
        }
      }

      visit(n);
      if (step.nth case final nth?) {
        if (nth < matches.length) {
          next.add(matches[nth]);
        }
      } else {
        next.addAll(matches);
      }
    }
    nodes = next.toList();
  }
  return nodes;
}

/// The first node that [chain] selects from [postNode] and that [field]
/// accepts, if any.
SemanticTreeNode? _resolveChain(
  SemanticTreeNode postNode,
  PostField field,
  List<Step> chain,
) => _selectNodes(
  postNode,
  chain,
  children: false,
).where(field.accepts).firstOrNull;

/// The host and path of [link], which exclude patterns match and by which
/// the user excludes a link.
String hostPathOf(String link) {
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

/// A rule that finds the posts of one kind in a page and extracts their
/// fields (see PostMatcher in server/feature/feed/post_matcher.go).
///
/// [post] is the chain of steps to the post nodes: the first step selects
/// descendants of the root, and each later step selects children of the
/// nodes that the previous step selects. Each field has chains of steps from
/// a post node to the node of its value, tried in order; an empty chain is
/// the post node itself. The title and the link have one chain each, since
/// they define the kind of posts. A node that [post] selects is a post only
/// when its title and link are found, when they are labeled, and when its link
/// matches none of the patterns in [exclude]. A node that contains another
/// node that [post] selects is not a post, since it is a list of posts.
class PostMatcher {
  const new({
    required this.post,
    this.fields = const {},
    this.exclude = const [],
  });

  /// Parses the JSON produced by [toJson].
  factory fromJson(Map<String, Object?> json) => PostMatcher(
    post: [
      for (final s in json['post']! as List<Object?>)
        Step.fromJson(s! as Map<String, Object?>),
    ],
    fields: {
      for (final MapEntry(:key, :value)
          in (json['fields']! as Map<String, Object?>).entries)
        PostField.values.byName(key): [
          for (final chain in value! as List<Object?>)
            [
              for (final s in chain! as List<Object?>)
                Step.fromJson(s! as Map<String, Object?>),
            ],
        ],
    },
    exclude: (json['exclude'] as List<Object?>? ?? const []).cast<String>(),
  );

  /// A matcher whose posts are selected as [node] is: by the step of [node]
  /// when it has classes, or else by the steps of the nodes from its nearest
  /// ancestor with classes, or from the top of the tree below the root, down
  /// to [node].
  factory fromNode(SemanticTree tree, SemanticTreeNode node) {
    final chain = [Step.of(tree, node)];
    for (var n = node; chain.first.cls.isEmpty && n.parent?.parent != null;) {
      n = n.parent!;
      chain.insert(0, Step.of(tree, n));
    }
    return PostMatcher(post: chain);
  }

  final List<Step> post;
  final Map<PostField, List<List<Step>>> fields;
  final List<String> exclude;

  /// The nodes that [post] selects and that have the labeled title and link,
  /// including the excluded ones. The other fields are optional, since posts
  /// of one kind often lack some of them, such as the date of a featured
  /// post.
  List<SemanticTreeNode> _candidates(SemanticTree tree) {
    if (post.isEmpty) {
      return const [];
    }
    final selected = _selectNodes(tree.root, post, children: true).toSet();
    final posts = <SemanticTreeNode>[];
    // Returns whether n is or contains a selected node.
    bool visit(SemanticTreeNode n) {
      var inner = false;
      for (final c in n.children) {
        inner = visit(c) || inner;
      }
      if (selected.contains(n) &&
          !inner &&
          [
            PostField.title,
            PostField.link,
          ].every((f) => !fields.containsKey(f) || matchField(n, f) != null)) {
        posts.add(n);
      }
      return inner || selected.contains(n);
    }

    visit(tree.root);
    return posts;
  }

  List<SemanticTreeNode> matchPosts(SemanticTree tree) => [
    for (final n in _candidates(tree))
      if (excludedBy(n) == null) n,
  ];

  /// The nodes that would be posts if their links were not excluded.
  List<SemanticTreeNode> matchExcluded(SemanticTree tree) => [
    for (final n in _candidates(tree))
      if (excludedBy(n) != null) n,
  ];

  /// The first pattern in [exclude] that matches the link of [postNode].
  String? excludedBy(SemanticTreeNode postNode) {
    final link = _linkOf(postNode);
    if (link == null) {
      return null;
    }
    final hostPath = hostPathOf(link);
    for (final pattern in exclude) {
      if (_globMatches(pattern, hostPath)) {
        return pattern;
      }
    }
    return null;
  }

  String? _linkOf(SemanticTreeNode postNode) =>
      matchField(postNode, PostField.link)?.link;

  /// The node that provides [field] in [postNode]: the first node that the
  /// first chain of [field] selects with a node that [field] accepts, if any.
  SemanticTreeNode? matchField(SemanticTreeNode postNode, PostField field) {
    for (final chain in fields[field] ?? const <List<Step>>[]) {
      if (_resolveChain(postNode, field, chain) case final n?) {
        return n;
      }
    }
    return null;
  }

  /// Maps [field] to [valueNode] inside [postNode].
  ///
  /// The chain that selects [valueNode] is added after the other chains of
  /// [field] when they find nothing in [postNode], and before them when they
  /// find another node, so that the posts that the other chains serve keep the
  /// nodes that they find. Nothing changes when no chain finds [valueNode].
  PostMatcher _withField(
    SemanticTree tree,
    PostField field,
    SemanticTreeNode postNode,
    SemanticTreeNode valueNode,
  ) {
    final chains = fields[field] ?? const <List<Step>>[];
    final current = matchField(postNode, field);
    if (identical(current, valueNode)) {
      return this;
    }
    final chain = _fieldChain(field, [
      (postNode, valueNode),
    ], _pathOf(tree, postNode, valueNode));
    if (chain == null) {
      return this;
    }
    return PostMatcher(
      post: post,
      fields: {
        ...fields,
        field: current == null ? [...chains, chain] : [chain, ...chains],
      },
      exclude: exclude,
    );
  }

  /// The chain that finds each node of [targets] as [field] in its post node:
  /// the shortest tail of [path] that does, and that has a step with classes
  /// when [path] has one. It falls back to all of [path], with [Step.nth] on
  /// the last step. Null when no chain finds the nodes.
  static List<Step>? _fieldChain(
    PostField field,
    List<(SemanticTreeNode, SemanticTreeNode)> targets,
    List<Step> path,
  ) {
    bool findsAll(List<Step> chain) => targets.every(
      (t) => identical(_resolveChain(t.$1, field, chain), t.$2),
    );
    bool hasCls(List<Step> chain) => chain.any((s) => s.cls.isNotEmpty);
    for (var i = path.length; i >= 0; i--) {
      final chain = path.sublist(i);
      // The empty chain finds only the post node itself.
      if ((i == path.length && path.isNotEmpty) ||
          (hasCls(path) && !hasCls(chain))) {
        continue;
      }
      if (findsAll(chain)) {
        return chain;
      }
    }
    if (path.isEmpty) {
      return null;
    }
    for (var nth = 0; ; nth++) {
      final last = path.last;
      final chain = [
        ...path.take(path.length - 1),
        Step(tag: last.tag, cls: last.cls, nth: nth),
      ];
      if (targets.every(
        (t) => _selectNodes(t.$1, chain, children: false).isEmpty,
      )) {
        return null;
      }
      if (findsAll(chain)) {
        return chain;
      }
    }
  }

  Map<String, Object?> toJson() => {
    'post': [for (final s in post) s.toJson()],
    'fields': {
      for (final MapEntry(:key, :value) in fields.entries)
        key.name: [
          for (final chain in value) [for (final s in chain) s.toJson()],
        ],
    },
    'exclude': exclude,
  };
}

/// A post that the user labeled: its node, and the nodes of its fields.
typedef LabeledPost = ({
  SemanticTreeNode node,
  Map<PostField, SemanticTreeNode> fields,
});

/// This matcher widened to find [node] too: each step of [chain] keeps only the
/// classes that it shares with [node], or with the ancestor of [node] at its
/// level. Null when [node] or such an ancestor has another tag.
List<Step>? _widen(SemanticTree tree, List<Step> chain, SemanticTreeNode node) {
  final widened = [...chain];
  SemanticTreeNode? n = node;
  for (var i = widened.length - 1; i >= 0; i--) {
    if (i < widened.length - 1) {
      n = n?.parent;
    }
    if (n == null || n.tag != widened[i].tag) {
      return null;
    }
    final shared = Step.of(tree, n).cls;
    widened[i] = Step(
      tag: widened[i].tag,
      cls: [
        for (final c in widened[i].cls)
          if (shared.contains(c)) c,
      ],
      nth: widened[i].nth,
    );
  }
  return widened;
}

/// The steps of the nodes from [postNode], exclusive, down to [node].
List<Step> _pathOf(
  SemanticTree tree,
  SemanticTreeNode postNode,
  SemanticTreeNode node,
) => [
  for (var n = node; !identical(n, postNode); n = n.parent!) Step.of(tree, n),
].reversed.toList();

/// The fields that make a node a post. A matcher finds each of them by one
/// chain, so posts that reach them by different paths are of different kinds.
const _keyFields = [PostField.title, PostField.link];

/// The shape of the posts of [group]: the chain that selects them, and for
/// each of [_keyFields], the path from a post node to the node of the field.
/// Each is the one of the first post, whose steps keep only the classes that
/// the other posts share (see [_widen]). Null when the posts have different
/// tags on any of them.
({List<Step> post, Map<PostField, List<Step>?> paths})? _shapeOf(
  SemanticTree tree,
  List<LabeledPost> group,
) {
  List<Step>? pathOf(LabeledPost p, PostField field) =>
      switch (p.fields[field]) {
        final n? => _pathOf(tree, p.node, n),
        null => null,
      };
  final first = group.first;
  var chain = PostMatcher.fromNode(tree, first.node).post;
  final paths = {for (final f in _keyFields) f: pathOf(first, f)};
  for (final post in group.skip(1)) {
    final widened = _widen(tree, chain, post.node);
    if (widened == null) {
      return null;
    }
    chain = widened;
    for (final f in _keyFields) {
      final (path, other) = (paths[f], pathOf(post, f));
      if (path == null || other == null) {
        if (path != other) {
          return null;
        }
        continue;
      }
      if (path.length != other.length) {
        return null;
      }
      for (final (i, s) in path.indexed) {
        if (s.tag != other[i].tag) {
          return null;
        }
        path[i] = Step(
          tag: s.tag,
          cls: [
            for (final c in s.cls)
              if (other[i].cls.contains(c)) c,
          ],
        );
      }
    }
  }
  return (post: chain, paths: paths);
}

/// The matchers that find the [posts] that the user labeled, and none of the
/// posts whose links the user excluded. [excluded] holds the host and path of
/// each such link. The matchers depend on the posts and the links, not on the
/// order in which the user marked them. A link of [excluded] must not be the
/// link of a post.
///
/// The posts are grouped by their shapes, in document order: a post joins the
/// first group whose shape it shares (see [_shapeOf]), and otherwise starts a
/// new group. Each group results in one matcher, which finds the title and the
/// link of its posts by one chain each, and the other fields by as many chains
/// as their nodes need. A post that its group does not find with its fields,
/// such as one whose fields an earlier post's chains find in other nodes, is
/// moved to a group of its own.
List<PostMatcher> generateMatchers(
  SemanticTree tree,
  Iterable<LabeledPost> posts,
  Set<String> excluded,
) {
  final exclude = excluded.toList()..sort();
  final groups = <List<LabeledPost>>[];
  for (final post in [
    ...posts,
  ]..sort((a, b) => a.node.id.compareTo(b.node.id))) {
    final group = groups
        .where((g) => _shapeOf(tree, [...g, post]) != null)
        .firstOrNull;
    if (group == null) {
      groups.add([post]);
    } else {
      group.add(post);
    }
  }

  final matchers = <PostMatcher>[];
  for (var i = 0; i < groups.length; i++) {
    final group = groups[i];
    final shape = _shapeOf(tree, group)!;
    var matcher = PostMatcher(
      post: shape.post,
      fields: {
        for (final f in _keyFields)
          if (shape.paths[f] case final path?)
            if (PostMatcher._fieldChain(f, [
                  for (final p in group) (p.node, p.fields[f]!),
                ], path)
                case final chain?)
              f: [chain],
      },
      exclude: exclude,
    );
    for (final post in group) {
      for (final MapEntry(:key, :value) in post.fields.entries) {
        if (!_keyFields.contains(key)) {
          matcher = matcher._withField(tree, key, post.node, value);
        }
      }
    }
    final found = matcher.matchPosts(tree);
    final isFound = group.every(
      (p) =>
          found.any((n) => identical(n, p.node)) &&
          p.fields.entries.every(
            (f) => identical(matcher.matchField(p.node, f.key), f.value),
          ),
    );
    if (group.length > 1 && !isFound) {
      groups.add([group.removeLast()]);
      i--;
    } else {
      matchers.add(matcher);
    }
  }
  return matchers;
}

/// One post for each of [matchers] that finds one in [tree], with the nodes
/// of its fields, so that the user can see what each matcher finds.
List<LabeledPost> sampleLabeledPosts(
  SemanticTree tree,
  List<PostMatcher> matchers,
) {
  final samples = <LabeledPost>[];
  for (final matcher in matchers) {
    final node = matcher.matchPosts(tree).firstOrNull;
    if (node != null && !samples.any((s) => identical(s.node, node))) {
      samples.add((
        node: node,
        fields: {
          for (final field in matcher.fields.keys)
            field: ?matcher.matchField(node, field),
        },
      ));
    }
  }
  return samples;
}

/// The fields that are evident in [postNode]: the only link or time, the only
/// image at the shallowest level as the thumbnail, and the only heading of the
/// highest level as the title. Without headings, the title is the only text,
/// such as the text of a post that is a single link. Without such a heading or
/// text, a text labeled as the link is also the title, such as the title link
/// of a card with other texts.
///
/// The body is the only paragraph, which is a plain text long enough not to be
/// a title or metadata such as an author. A container with two or more
/// paragraphs stands for them, unless it is the post node or holds other
/// fields, since a body cannot contain other fields.
///
/// Nodes that link to the same URL count as one link, such as a card and a
/// title in it. Of those, the post node itself is labeled, or else the one
/// with text. A title with a link of its own provides the link of the post,
/// even when there are other links.
Map<PostField, SemanticTreeNode> evidentFields(SemanticTreeNode postNode) {
  final nodes = <SemanticTreeNode>[];
  void collect(SemanticTreeNode n) {
    nodes.add(n);
    n.children.forEach(collect);
  }

  collect(postNode);
  SemanticTreeNode? only(Iterable<SemanticTreeNode> matches) =>
      matches.length == 1 ? matches.first : null;
  final fields = <PostField, SemanticTreeNode>{};
  void label(PostField field, SemanticTreeNode? node) {
    if (node != null && field.accepts(node)) {
      fields[field] = node;
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
  final headings = nodes
      .where((n) => n.isValue && n.headingLevel != null)
      .toList();
  SemanticTreeNode? title;
  if (headings.isEmpty) {
    title = only(nodes.where(PostField.title.accepts));
  } else {
    final top = headings
        .map((n) => n.headingLevel!)
        .reduce((a, b) => a < b ? a : b);
    title = only(headings.where((n) => n.headingLevel == top));
  }
  title ??= fields[PostField.link];
  label(PostField.title, title);
  if (title?.pageLink case final link?
      when fields[PostField.link]?.link != link) {
    label(PostField.link, title);
  }

  final labeled = fields.values.toList();
  final paragraphs = nodes.where(
    (n) =>
        n.isValue &&
        n.headingLevel == null &&
        !n.isImage &&
        !n.isTime &&
        n.link == null &&
        (n.val?.length ?? 0) >= 80 &&
        !labeled.any((l) => identical(l, n)),
  );
  final containers = {
    for (final p in paragraphs)
      if (!identical(p.parent, postNode) &&
          paragraphs.where((q) => identical(q.parent, p.parent)).length >= 2 &&
          !labeled.any(p.parent!.contains))
        p.parent!,
  };
  label(
    PostField.body,
    only([
      ...containers,
      ...paragraphs.where((p) => !containers.contains(p.parent)),
    ]),
  );
  return fields;
}

/// A post extracted from a page: its node, the index of the matcher that
/// found it, and its field values.
typedef ExtractedPost = ({
  SemanticTreeNode node,
  int matcher,
  Map<PostField, String> values,
});

/// The posts that [matchers] find in [tree], in document order. Posts
/// with the same link are merged into the first one, which takes the fields
/// that it does not have from the later ones, since some pages split a post
/// into sibling cards with the same link, such as one with the image and one
/// with the title.
List<ExtractedPost> extractPosts(
  SemanticTree tree,
  List<PostMatcher> matchers,
) {
  final found = <ExtractedPost>[
    for (final (i, matcher) in matchers.indexed)
      for (final node in matcher.matchPosts(tree))
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
