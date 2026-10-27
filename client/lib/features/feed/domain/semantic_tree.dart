/// A web page simplified by the server into a semantic tree, which the user
/// annotates to tell which parts of the page are posts.
class SemanticTree {
  new _({
    required this.url,
    required this.title,
    required this.root,
    required this.nodes,
    required this.classCounts,
  });

  /// Parses the JSON built by the server (see SemanticTree in
  /// server/feature/feed/semantic_tree.go for the format).
  factory fromJson(Map<String, Object?> json) {
    final nodes = <SemanticTreeNode>[];
    SemanticTreeNode parse(
      Map<String, Object?> json,
      SemanticTreeNode? parent,
      int depth,
    ) {
      final node = SemanticTreeNode._(
        id: nodes.length,
        parent: parent,
        depth: depth,
        tag: json['tag'] as String?,
        cls: (json['cls'] as List<Object?>? ?? const []).cast<String>(),
        val: json['val'] as String?,
        link: json['link'] as String?,
      );
      nodes.add(node);
      for (final child in (json['ch'] as List<Object?>?) ?? const []) {
        node.children.add(
          parse(child! as Map<String, Object?>, node, depth + 1),
        );
      }
      return node;
    }

    final root = parse(json['root']! as Map<String, Object?>, null, 0);
    final classCounts = <String, int>{};
    for (final n in nodes) {
      for (final c in n.cls) {
        classCounts.update(c, (count) => count + 1, ifAbsent: () => 1);
      }
    }
    return SemanticTree._(
      url: json['url']! as String,
      title: json['title'] as String? ?? '',
      root: root,
      nodes: nodes,
      classCounts: classCounts,
    );
  }

  final String url;
  final String title;
  final SemanticTreeNode root;

  /// All nodes in document order, indexed by [SemanticTreeNode.id].
  final List<SemanticTreeNode> nodes;

  /// The number of nodes that have each class.
  final Map<String, int> classCounts;
}

/// A node of a [SemanticTree]: a value without children, or a container.
class SemanticTreeNode {
  new _({
    required this.id,
    required this.parent,
    required this.depth,
    required this.tag,
    required this.cls,
    required this.val,
    required this.link,
  });

  /// The position of this node in document order.
  final int id;
  final SemanticTreeNode? parent;
  final int depth;

  /// The tag name of the element that this node comes from, such as `h2`,
  /// `img` or `time`, or null for a plain text that does not come from one
  /// element.
  final String? tag;

  /// The classes of that element, without generated parts such as hashes.
  final List<String> cls;

  /// The plain text of a value: the alt text of an image, or the RFC 3339
  /// timestamp of a time.
  final String? val;

  /// The link target, or the image URL of an image.
  final String? link;
  final List<SemanticTreeNode> children = [];

  bool get isValue => children.isEmpty;
  bool get isImage => tag == 'img';
  bool get isTime => tag == 'time';

  /// The level of a heading (1 for h1), or null for other nodes.
  int? get headingLevel => switch (tag) {
    'h1' || 'h2' || 'h3' || 'h4' || 'h5' || 'h6' => int.parse(tag![1]),
    _ => null,
  };

  /// The link of the page that this node points to: [link], except for an
  /// image, whose link is the image itself.
  String? get pageLink => isImage ? null : link;

  /// All text in this subtree without the alt texts of images, joined with
  /// spaces.
  String get textContent {
    if (isValue) {
      return isImage ? '' : val ?? '';
    }
    return children
        .map((c) => c.textContent)
        .where((t) => t.isNotEmpty)
        .join(' ');
  }

  /// Whether [other] is this node or one of its descendants.
  bool contains(SemanticTreeNode other) {
    for (SemanticTreeNode? n = other; n != null; n = n.parent) {
      if (identical(n, this)) {
        return true;
      }
    }
    return false;
  }

  /// Whether one of the two nodes contains the other.
  bool overlaps(SemanticTreeNode other) =>
      contains(other) || other.contains(this);
}
