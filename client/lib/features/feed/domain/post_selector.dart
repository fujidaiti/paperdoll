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

/// The nodes that [path] selects from [start], in document order. With
/// [children], each step after the first selects among the children of the
/// nodes that the previous step selects, instead of their descendants.
List<SemanticTreeNode> _selectNodes(
  SemanticTreeNode start,
  List<Step> path, {
  required bool children,
}) {
  var nodes = [start];
  for (final (i, step) in path.indexed) {
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

/// The first node that [path] selects from [postNode] and that [field]
/// accepts, if any.
SemanticTreeNode? _resolvePath(
  SemanticTreeNode postNode,
  PostField field,
  List<Step> path,
) => _selectNodes(
  postNode,
  path,
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
/// fields (see PostSelector in server/feature/feed/post_selector.go).
///
/// [post] is the path of steps to the post nodes: the first step selects
/// descendants of the root, and each later step selects children of the
/// nodes that the previous step selects. Each field has one path of steps
/// from a post node to the node of its value; an empty path is the post node
/// itself. The [title] and the [link] are required, since they define the
/// kind of posts, and the other fields are null when the posts have none. A
/// node that [post] selects is a post only when its title and link are found,
/// and when its link matches none of the patterns in [exclude]. A node that
/// contains another such node is not a post, since it is a list of posts.
class PostSelector {
  const new({
    required this.post,
    required this.title,
    required this.link,
    this.body,
    this.thumbnail,
    this.pubDate,
    this.exclude = const [],
  });

  /// Parses the JSON produced by [toJson].
  factory fromJson(Map<String, Object?> json) {
    List<Step>? pathOf(Object? json) => switch (json) {
      final List<Object?> steps => [
        for (final s in steps) Step.fromJson(s! as Map<String, Object?>),
      ],
      _ => null,
    };
    final fields = json['fields']! as Map<String, Object?>;
    return PostSelector(
      post: pathOf(json['post'])!,
      title: pathOf(fields[PostField.title.name])!,
      link: pathOf(fields[PostField.link.name])!,
      body: pathOf(fields[PostField.body.name]),
      thumbnail: pathOf(fields[PostField.thumbnail.name]),
      pubDate: pathOf(fields[PostField.pubDate.name]),
      exclude: (json['exclude'] as List<Object?>? ?? const []).cast<String>(),
    );
  }

  final List<Step> post;
  final List<Step> title;
  final List<Step> link;
  final List<Step>? body;
  final List<Step>? thumbnail;
  final List<Step>? pubDate;
  final List<String> exclude;

  /// The path of [field], if any.
  List<Step>? _pathOf(PostField field) => switch (field) {
    PostField.title => title,
    PostField.link => link,
    PostField.body => body,
    PostField.thumbnail => thumbnail,
    PostField.pubDate => pubDate,
  };

  /// The path of each field that has one.
  Map<PostField, List<Step>> get fields => {
    for (final field in PostField.values) field: ?_pathOf(field),
  };

  /// The nodes that [post] selects and that have the title and the link,
  /// including the excluded ones. The other fields are optional, since posts
  /// of one kind often lack some of them, such as the date of a featured
  /// post. Other nodes that [post] selects inside a post, such as a footer
  /// with the same classes, do not matter.
  List<SemanticTreeNode> _candidates(SemanticTree tree) {
    if (post.isEmpty) {
      return const [];
    }
    final selected = _selectNodes(tree.root, post, children: true).toSet();
    final posts = <SemanticTreeNode>[];
    // Returns whether n is or contains a selected node with the title and the
    // link.
    bool visit(SemanticTreeNode n) {
      var inner = false;
      for (final c in n.children) {
        inner = visit(c) || inner;
      }
      final isPost =
          selected.contains(n) &&
          matchField(n, PostField.title) != null &&
          matchField(n, PostField.link) != null;
      if (isPost && !inner) {
        posts.add(n);
      }
      return inner || isPost;
    }

    visit(tree.root);
    return posts;
  }

  List<SemanticTreeNode> selectPosts(SemanticTree tree) => [
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
  /// path of [field] selects and that [field] accepts, if any.
  SemanticTreeNode? matchField(SemanticTreeNode postNode, PostField field) =>
      switch (_pathOf(field)) {
        final path? => _resolvePath(postNode, field, path),
        null => null,
      };

  Map<String, Object?> toJson() => {
    'post': [for (final s in post) s.toJson()],
    'fields': {
      for (final MapEntry(:key, :value) in fields.entries)
        key.name: [for (final s in value) s.toJson()],
    },
    if (exclude.isNotEmpty) 'exclude': exclude,
  };
}

/// A post that the user labeled: its node, and the nodes of its fields.
typedef LabeledPost = ({
  SemanticTreeNode node,
  Map<PostField, SemanticTreeNode> fields,
});

/// The classes that differ between the posts of a list, such as the ID and
/// the categories of a WordPress post.
final _variedPostClass = RegExp(
  r'^(post-\d+|category-.+|tag-.+|has-post-thumbnail)$',
);

/// The classes with the word "post" or "card", which name the kind of item
/// rather than its layout or state. A word starts at the start of the class,
/// after "-" or "_", or with an uppercase letter, and ends at the end of the
/// class, before "-" or "_", or before an uppercase letter.
final _postKindClass = RegExp(
  r'(^|[-_])(post|card)($|[-_A-Z])|(Post|Card)($|[-_A-Z])',
);

/// The classes of [cls] that a post node keeps in its step: the ones that
/// [_postKindClass] matches, or all when none does, without the ones that
/// [_variedPostClass] matches. A step built from a single post would
/// otherwise keep the classes of that post alone, and miss the other posts of
/// the list.
List<String> _postClasses(List<String> cls) {
  final kept = [
    for (final c in cls)
      if (!_variedPostClass.hasMatch(c)) c,
  ];
  final kinds = kept.where(_postKindClass.hasMatch).toList();
  return kinds.isNotEmpty ? kinds : kept;
}

/// The path that selects [node] from the root: the step of [node] when it has
/// classes left (see [_postClasses]), or else the steps of the nodes from its
/// nearest ancestor with classes, or from the top of the tree below the root,
/// down to [node].
List<Step> _postPathOf(SemanticTree tree, SemanticTreeNode node) {
  final step = Step.of(tree, node);
  final path = [Step(tag: step.tag, cls: _postClasses(step.cls))];
  for (var n = node; path.first.cls.isEmpty && n.parent?.parent != null;) {
    n = n.parent!;
    path.insert(0, Step.of(tree, n));
  }
  return path;
}

/// The steps of the nodes from [postNode], exclusive, down to [node].
List<Step> _fullPathOf(
  SemanticTree tree,
  SemanticTreeNode postNode,
  SemanticTreeNode node,
) => [
  for (var n = node; !identical(n, postNode); n = n.parent!) Step.of(tree, n),
].reversed.toList();

/// The path that finds [node] as [field] in [postNode]: the shortest tail of
/// the full path from [postNode] to [node] that does, and that has a step with
/// classes when the full path has one. It falls back to the full path, with
/// [Step.nth] on the last step. Null when no path finds [node].
List<Step>? _fieldPath(
  SemanticTree tree,
  PostField field,
  SemanticTreeNode postNode,
  SemanticTreeNode node,
) {
  final fullPath = _fullPathOf(tree, postNode, node);
  bool finds(List<Step> path) =>
      identical(_resolvePath(postNode, field, path), node);
  bool hasCls(List<Step> path) => path.any((s) => s.cls.isNotEmpty);
  for (var i = fullPath.length; i >= 0; i--) {
    final path = fullPath.sublist(i);
    // The empty path finds only the post node itself.
    if ((i == fullPath.length && fullPath.isNotEmpty) ||
        (hasCls(fullPath) && !hasCls(path))) {
      continue;
    }
    if (finds(path)) {
      return path;
    }
  }
  if (fullPath.isEmpty) {
    return null;
  }
  for (var nth = 0; ; nth++) {
    final last = fullPath.last;
    final path = [
      ...fullPath.take(fullPath.length - 1),
      Step(tag: last.tag, cls: last.cls, nth: nth),
    ];
    if (_selectNodes(postNode, path, children: false).isEmpty) {
      return null;
    }
    if (finds(path)) {
      return path;
    }
  }
}

/// The selector that finds [post] with its fields. Null when no path finds
/// its title or link. A field that no path finds is left out.
PostSelector? _selectorOf(
  SemanticTree tree,
  LabeledPost post,
  List<String> exclude,
) {
  List<Step>? pathOf(PostField field) => switch (post.fields[field]) {
    final n? => _fieldPath(tree, field, post.node, n),
    null => null,
  };
  final (title, link) = (pathOf(PostField.title), pathOf(PostField.link));
  if (title == null || link == null) {
    return null;
  }
  return PostSelector(
    post: _postPathOf(tree, post.node),
    title: title,
    link: link,
    body: pathOf(PostField.body),
    thumbnail: pathOf(PostField.thumbnail),
    pubDate: pathOf(PostField.pubDate),
    exclude: exclude,
  );
}

/// Whether [a] and [b] have the same steps, with the classes of each step in
/// any order.
bool _samePaths(List<Step> a, List<Step> b) =>
    a.length == b.length &&
    a.indexed.every((e) {
      final (x, y) = (e.$2, b[e.$1]);
      return x.tag == y.tag &&
          x.nth == y.nth &&
          x.cls.length == y.cls.length &&
          x.cls.every(y.cls.contains);
    });

/// The selector that finds the posts of [a] and of [b] with their fields, or
/// null when they cannot merge. They merge when their post paths have the
/// same tags and the same classes except on the last step, and when they have
/// the same title and link paths, and no other field with different paths.
/// The last step of the merged post path keeps the classes that [a] and [b]
/// share, and the merge fails when no step has classes left, since such a
/// path selects any node of its tags.
///
/// The merged selector finds the posts of [a] and of [b] without a check on
/// the tree: its post path selects their nodes, and their fields resolve by
/// the same paths as before.
PostSelector? _merge(PostSelector a, PostSelector b) {
  if (a.post.length != b.post.length) {
    return null;
  }
  final last = a.post.length - 1;
  for (final (i, step) in a.post.indexed) {
    if (step.tag != b.post[i].tag ||
        (i < last && !_samePaths([step], [b.post[i]]))) {
      return null;
    }
  }
  final post = [
    ...a.post.take(last),
    Step(
      tag: a.post[last].tag,
      cls: [
        for (final c in a.post[last].cls)
          if (b.post[last].cls.contains(c)) c,
      ],
    ),
  ];
  if (post.every((s) => s.cls.isEmpty) ||
      !_samePaths(a.title, b.title) ||
      !_samePaths(a.link, b.link)) {
    return null;
  }
  final optional = <PostField, List<Step>?>{};
  for (final field in [
    PostField.body,
    PostField.thumbnail,
    PostField.pubDate,
  ]) {
    final (path, other) = (a._pathOf(field), b._pathOf(field));
    if (path != null && other != null && !_samePaths(path, other)) {
      return null;
    }
    optional[field] = path ?? other;
  }
  return PostSelector(
    post: post,
    title: a.title,
    link: a.link,
    body: optional[PostField.body],
    thumbnail: optional[PostField.thumbnail],
    pubDate: optional[PostField.pubDate],
    exclude: a.exclude,
  );
}

/// The selectors that find the [posts] that the user labeled, and none of the
/// posts whose links the user excluded. [excluded] holds the host and path of
/// each such link. The selectors depend on the posts and the links, not on the
/// order in which the user marked them. A link of [excluded] must not be the
/// link of a post.
///
/// Each post starts with its own selector (see [_selectorOf]), in document
/// order, and the first two selectors that can merge (see [_merge]) are
/// replaced by the merged one, until no two can merge. A post whose title or
/// link no path finds gets no selector.
List<PostSelector> generateSelectors(
  SemanticTree tree,
  Iterable<LabeledPost> posts,
  Set<String> excluded,
) {
  final exclude = excluded.toList()..sort();
  final selectors = [
    for (final post in [
      ...posts,
    ]..sort((a, b) => a.node.id.compareTo(b.node.id)))
      ?_selectorOf(tree, post, exclude),
  ];
  merging:
  while (true) {
    for (var i = 0; i < selectors.length; i++) {
      for (var j = i + 1; j < selectors.length; j++) {
        if (_merge(selectors[i], selectors[j]) case final m?) {
          selectors
            ..[i] = m
            ..removeAt(j);
          continue merging;
        }
      }
    }
    return selectors;
  }
}

/// Every post that [selectors] find in [tree], with the nodes of its fields, so
/// that the selectors can be generated again from them, since the marks of the
/// user are not saved. All of them are needed, since a merged selector keeps
/// only the classes that all of its posts share, and fewer posts would make
/// the selectors find fewer posts. A post that several
/// selectors find takes each field from the first selector that finds it.
List<LabeledPost> labelFoundPosts(
  SemanticTree tree,
  List<PostSelector> selectors,
) {
  final posts = <LabeledPost>[];
  for (final selector in selectors) {
    for (final node in selector.selectPosts(tree)) {
      final fields = {
        for (final field in selector.fields.keys)
          field: ?selector.matchField(node, field),
      };
      final i = posts.indexWhere((p) => identical(p.node, node));
      if (i < 0) {
        posts.add((node: node, fields: fields));
      } else {
        for (final MapEntry(:key, :value) in fields.entries) {
          posts[i].fields.putIfAbsent(key, () => value);
        }
      }
    }
  }
  return posts;
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

/// A post extracted from a page: its node, the index of the selector that
/// found it, and its field values.
typedef ExtractedPost = ({
  SemanticTreeNode node,
  int selector,
  Map<PostField, String> values,
});

/// The posts that [selectors] find in [tree], in document order. Posts
/// with the same link are merged into the first one, which takes the fields
/// that it does not have from the later ones, since some pages split a post
/// into sibling cards with the same link, such as one with the image and one
/// with the title.
List<ExtractedPost> extractPosts(
  SemanticTree tree,
  List<PostSelector> selectors,
) {
  final found =
      <ExtractedPost>[
          for (final (i, selector) in selectors.indexed)
            for (final node in selector.selectPosts(tree))
              (
                node: node,
                selector: i,
                values: {
                  for (final field in selector.fields.keys)
                    if (selector.matchField(node, field) case final n?)
                      field: ?field.valueOf(n),
                },
              ),
        ]
        // A node that several selectors find takes its fields from the first
        // one first, since List.sort is not stable.
        ..sort(
          (a, b) => a.node.id != b.node.id
              ? a.node.id.compareTo(b.node.id)
              : a.selector.compareTo(b.selector),
        );
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
