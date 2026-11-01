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
/// nodes that the previous step selects. Each field has one chain of steps
/// from a post node to the node of its value; an empty chain is the post node
/// itself. The [title] and the [link] are required, since they define the
/// kind of posts, and the other fields are null when the posts have none. A
/// node that [post] selects is a post only when its title and link are found,
/// and when its link matches none of the patterns in [exclude]. A node that
/// contains another such node is not a post, since it is a list of posts.
class PostMatcher {
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
    List<Step>? chainOf(Object? json) => switch (json) {
      final List<Object?> steps => [
        for (final s in steps) Step.fromJson(s! as Map<String, Object?>),
      ],
      _ => null,
    };
    final fields = json['fields']! as Map<String, Object?>;
    return PostMatcher(
      post: chainOf(json['post'])!,
      title: chainOf(fields[PostField.title.name])!,
      link: chainOf(fields[PostField.link.name])!,
      body: chainOf(fields[PostField.body.name]),
      thumbnail: chainOf(fields[PostField.thumbnail.name]),
      pubDate: chainOf(fields[PostField.pubDate.name]),
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

  /// The chain of [field], if any.
  List<Step>? _chainOf(PostField field) => switch (field) {
    PostField.title => title,
    PostField.link => link,
    PostField.body => body,
    PostField.thumbnail => thumbnail,
    PostField.pubDate => pubDate,
  };

  /// The chain of each field that has one.
  Map<PostField, List<Step>> get fields => {
    for (final field in PostField.values) field: ?_chainOf(field),
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
  /// chain of [field] selects and that [field] accepts, if any.
  SemanticTreeNode? matchField(SemanticTreeNode postNode, PostField field) =>
      switch (_chainOf(field)) {
        final chain? => _resolveChain(postNode, field, chain),
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

/// The chain that selects [node] from the root: the step of [node] when it has
/// classes left (see [_postClasses]), or else the steps of the nodes from its
/// nearest ancestor with classes, or from the top of the tree below the root,
/// down to [node].
List<Step> _postChainOf(SemanticTree tree, SemanticTreeNode node) {
  final step = Step.of(tree, node);
  final chain = [Step(tag: step.tag, cls: _postClasses(step.cls))];
  for (var n = node; chain.first.cls.isEmpty && n.parent?.parent != null;) {
    n = n.parent!;
    chain.insert(0, Step.of(tree, n));
  }
  return chain;
}

/// The steps of the nodes from [postNode], exclusive, down to [node].
List<Step> _pathOf(
  SemanticTree tree,
  SemanticTreeNode postNode,
  SemanticTreeNode node,
) => [
  for (var n = node; !identical(n, postNode); n = n.parent!) Step.of(tree, n),
].reversed.toList();

/// The chain that finds [node] as [field] in [postNode]: the shortest tail of
/// the path from [postNode] to [node] that does, and that has a step with
/// classes when the path has one. It falls back to the whole path, with
/// [Step.nth] on the last step. Null when no chain finds [node].
List<Step>? _fieldChain(
  SemanticTree tree,
  PostField field,
  SemanticTreeNode postNode,
  SemanticTreeNode node,
) {
  final path = _pathOf(tree, postNode, node);
  bool finds(List<Step> chain) =>
      identical(_resolveChain(postNode, field, chain), node);
  bool hasCls(List<Step> chain) => chain.any((s) => s.cls.isNotEmpty);
  for (var i = path.length; i >= 0; i--) {
    final chain = path.sublist(i);
    // The empty chain finds only the post node itself.
    if ((i == path.length && path.isNotEmpty) ||
        (hasCls(path) && !hasCls(chain))) {
      continue;
    }
    if (finds(chain)) {
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
    if (_selectNodes(postNode, chain, children: false).isEmpty) {
      return null;
    }
    if (finds(chain)) {
      return chain;
    }
  }
}

/// The matcher that finds [post] with its fields. Null when no chain finds
/// its title or link. A field that no chain finds is left out.
PostMatcher? _matcherOf(
  SemanticTree tree,
  LabeledPost post,
  List<String> exclude,
) {
  List<Step>? chainOf(PostField field) => switch (post.fields[field]) {
    final n? => _fieldChain(tree, field, post.node, n),
    null => null,
  };
  final (title, link) = (chainOf(PostField.title), chainOf(PostField.link));
  if (title == null || link == null) {
    return null;
  }
  return PostMatcher(
    post: _postChainOf(tree, post.node),
    title: title,
    link: link,
    body: chainOf(PostField.body),
    thumbnail: chainOf(PostField.thumbnail),
    pubDate: chainOf(PostField.pubDate),
    exclude: exclude,
  );
}

/// Whether [a] and [b] have the same steps, with the classes of each step in
/// any order.
bool _sameChains(List<Step> a, List<Step> b) =>
    a.length == b.length &&
    a.indexed.every((e) {
      final (x, y) = (e.$2, b[e.$1]);
      return x.tag == y.tag &&
          x.nth == y.nth &&
          x.cls.length == y.cls.length &&
          x.cls.every(y.cls.contains);
    });

/// The matcher that finds the posts of [a] and of [b] with their fields, or
/// null when they cannot merge. They merge when their post chains have the
/// same tags and the same classes except on the last step, and when they have
/// the same title and link chains, and no other field with different chains.
/// The last step of the merged post chain keeps the classes that [a] and [b]
/// share, and the merge fails when no step has classes left, since such a
/// chain selects any node of its tags.
///
/// The merged matcher finds the posts of [a] and of [b] without a check on
/// the tree: its post chain selects their nodes, and their fields resolve by
/// the same chains as before.
PostMatcher? _merge(PostMatcher a, PostMatcher b) {
  if (a.post.length != b.post.length) {
    return null;
  }
  final last = a.post.length - 1;
  for (final (i, step) in a.post.indexed) {
    if (step.tag != b.post[i].tag ||
        (i < last && !_sameChains([step], [b.post[i]]))) {
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
      !_sameChains(a.title, b.title) ||
      !_sameChains(a.link, b.link)) {
    return null;
  }
  final optional = <PostField, List<Step>?>{};
  for (final field in [
    PostField.body,
    PostField.thumbnail,
    PostField.pubDate,
  ]) {
    final (chain, other) = (a._chainOf(field), b._chainOf(field));
    if (chain != null && other != null && !_sameChains(chain, other)) {
      return null;
    }
    optional[field] = chain ?? other;
  }
  return PostMatcher(
    post: post,
    title: a.title,
    link: a.link,
    body: optional[PostField.body],
    thumbnail: optional[PostField.thumbnail],
    pubDate: optional[PostField.pubDate],
    exclude: a.exclude,
  );
}

/// The matchers that find the [posts] that the user labeled, and none of the
/// posts whose links the user excluded. [excluded] holds the host and path of
/// each such link. The matchers depend on the posts and the links, not on the
/// order in which the user marked them. A link of [excluded] must not be the
/// link of a post.
///
/// Each post starts with its own matcher (see [_matcherOf]), in document
/// order, and the first two matchers that can merge (see [_merge]) are
/// replaced by the merged one, until no two can merge. A post whose title or
/// link no chain finds gets no matcher.
List<PostMatcher> generateMatchers(
  SemanticTree tree,
  Iterable<LabeledPost> posts,
  Set<String> excluded,
) {
  final exclude = excluded.toList()..sort();
  final matchers = [
    for (final post in [
      ...posts,
    ]..sort((a, b) => a.node.id.compareTo(b.node.id)))
      ?_matcherOf(tree, post, exclude),
  ];
  merging:
  while (true) {
    for (var i = 0; i < matchers.length; i++) {
      for (var j = i + 1; j < matchers.length; j++) {
        if (_merge(matchers[i], matchers[j]) case final m?) {
          matchers
            ..[i] = m
            ..removeAt(j);
          continue merging;
        }
      }
    }
    return matchers;
  }
}

/// Every post that [matchers] find in [tree], with the nodes of its fields, so
/// that the matchers can be generated again from them, since the marks of the
/// user are not saved. All of them are needed, since a merged matcher keeps
/// only the classes that all of its posts share, and fewer posts would make
/// the matchers find fewer posts. A post that several
/// matchers find takes each field from the first matcher that finds it.
List<LabeledPost> labelFoundPosts(
  SemanticTree tree,
  List<PostMatcher> matchers,
) {
  final posts = <LabeledPost>[];
  for (final matcher in matchers) {
    for (final node in matcher.matchPosts(tree)) {
      final fields = {
        for (final field in matcher.fields.keys)
          field: ?matcher.matchField(node, field),
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
  final found =
      <ExtractedPost>[
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
        ]
        // A node that several matchers find takes its fields from the first one
        // first, since List.sort is not stable.
        ..sort(
          (a, b) => a.node.id != b.node.id
              ? a.node.id.compareTo(b.node.id)
              : a.matcher.compareTo(b.matcher),
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
