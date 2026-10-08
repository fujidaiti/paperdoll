/// A web page simplified by the server into a tree of content nodes, which the
/// user annotates to tell which parts of the page are posts.
class PageDocument {
  new _({
    required this.url,
    required this.title,
    required this.root,
    required this.nodes,
  });

  /// Parses the JSON produced by the page simplifier (see
  /// tool/simplifyhtml/main.go for the format).
  factory fromJson(Map<String, Object?> json) {
    final nodes = <PageNode>[];
    PageNode parse(Map<String, Object?> json, PageNode? parent, int depth) {
      final node = PageNode._(
        id: nodes.length,
        parent: parent,
        depth: depth,
        tag: json['tag']! as String,
        classes: [...?(json['cls'] as List<Object?>?)?.cast<String>()],
        text: json['text'] as String?,
        href: json['href'] as String?,
        src: json['src'] as String?,
        alt: json['alt'] as String?,
        datetime: json['dt'] as String?,
        rich: switch (json['rich']) {
          final List<Object?> rich => InlineNode._parseAll(rich),
          _ => null,
        },
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
    return PageDocument._(
      url: json['url']! as String,
      title: json['title'] as String? ?? '',
      root: root,
      nodes: nodes,
    );
  }

  final String url;
  final String title;
  final PageNode root;

  /// All nodes in document order, indexed by [PageNode.id].
  final List<PageNode> nodes;
}

/// An element of a [PageDocument]. Runs of text inside mixed content are
/// nodes with the tag `#text`.
class PageNode {
  new _({
    required this.id,
    required this.parent,
    required this.depth,
    required this.tag,
    required this.classes,
    required this.text,
    required this.href,
    required this.src,
    required this.alt,
    required this.datetime,
    required this.rich,
  });

  /// The position of this node in document order.
  final int id;
  final PageNode? parent;
  final int depth;
  final String tag;
  final List<String> classes;

  /// The text of this node; set only on nodes without child elements.
  final String? text;
  final String? href;
  final String? src;
  final String? alt;
  final String? datetime;

  /// The same text as [text] with inline formatting; null when there is no
  /// formatting.
  final List<InlineNode>? rich;
  final List<PageNode> children = [];

  bool get isText => tag == '#text';
  bool get isImage => tag == 'img';
  bool get isEmbed => tag == 'iframe';

  /// All text in this subtree, joined with spaces.
  String get textContent {
    if (text case final text?) {
      return text;
    }
    return children
        .map((c) => c.textContent)
        .where((t) => t.isNotEmpty)
        .join(' ');
  }

  /// Whether [other] is this node or one of its descendants.
  bool contains(PageNode other) {
    for (PageNode? n = other; n != null; n = n.parent) {
      if (identical(n, this)) {
        return true;
      }
    }
    return false;
  }

  /// Whether one of the two nodes contains the other.
  bool overlaps(PageNode other) => contains(other) || other.contains(this);

  /// A short CSS-like label such as `div.card.featured`.
  String get label => [tag, ...classes].join('.');
}

/// A piece of formatted text: a `#text` run, a `br`, or a `b`, `i` or `code`
/// element around other inline nodes.
class InlineNode {
  const new _({required this.tag, this.text, this.children = const []});

  static List<InlineNode> _parseAll(List<Object?> json) => [
    for (final node in json.cast<Map<String, Object?>>())
      InlineNode._(
        tag: node['tag']! as String,
        text: node['text'] as String?,
        children: _parseAll((node['ch'] as List<Object?>?) ?? const []),
      ),
  ];

  final String tag;
  final String? text;
  final List<InlineNode> children;
}
