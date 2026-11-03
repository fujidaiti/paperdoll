import 'package:html/dom.dart';
import 'package:html/parser.dart' as html;

export 'package:html/dom.dart' show Node;

/// A web page simplified by the server into an HTML document, which the user
/// marks to tell which parts of the page are posts. See SimplifyPage in
/// server/feature/feed/simplify.go for the format, and [PageNode] for how its
/// nodes are read.
///
/// Each node has an ID, which is its position in [nodes]. The server and the
/// client refer to the nodes of a page by their IDs. Each element carries its
/// ID in a `data-id` attribute, and a text takes the ID of its position (see
/// withNodeIDs in server/feature/feed/selector_editor.go).
class SimplifiedPage {
  new _({
    required this.url,
    required this.title,
    required this.body,
    required this.nodes,
  });

  /// Parses [source], a simplified page with node IDs. Throws a
  /// [FormatException] when the `data-id` of an element is not its position,
  /// so that an ID never refers to a node other than the one that the server
  /// gave it to, even if the HTML parsers of the two build different trees.
  factory parse(String source) {
    final document = html.parse(source);
    final body = document.body!;
    final nodes = <Node>[];
    void visit(Node n) {
      if (n is Element && n.attributes['data-id'] != '${nodes.length}') {
        throw FormatException(
          'node ${nodes.length} has the data-id ${n.attributes['data-id']}',
        );
      }
      nodes.add(n);
      n.pageChildren.forEach(visit);
    }

    visit(body);
    final head = document.head;
    return SimplifiedPage._(
      url: head?.querySelector('link[rel=canonical]')?.attributes['href'] ?? '',
      title: head?.querySelector('title')?.text ?? '',
      body: body,
      nodes: nodes,
    );
  }

  final String url;
  final String title;

  /// The root of the nodes that the user marks.
  final Element body;

  /// All nodes of [body] in document order, including [body] itself.
  final List<Node> nodes;

  late final _ids = {for (final (i, n) in nodes.indexed) n: i};

  /// The ID of [node].
  int idOf(Node node) => _ids[node]!;

  /// The node with the ID [id]. Throws a [FormatException] when there is
  /// none.
  Node nodeOf(int id) => id >= 0 && id < nodes.length
      ? nodes[id]
      : throw FormatException('no node $id');
}

/// Reads a node of a [SimplifiedPage] by the same rules as the server (see
/// tagOf and the functions after it in server/feature/feed/post_selector.go).
/// A node is a value, which is a text or an element without child elements,
/// or a container.
extension PageNode on Node {
  /// The tag name of the element, such as `h2`, `img` or `time`, or null for
  /// a plain text.
  String? get tag => this is Element ? (this as Element).localName : null;

  /// The classes of the element, without generated parts such as hashes.
  List<String> get cls => [
    for (final c in (attributes['class'] ?? '').split(' '))
      if (c.isNotEmpty) c,
  ];

  bool get isValue => this is Text || children.isEmpty;

  /// The child nodes of a container, or none for a value, since the text of a
  /// value element is a part of it rather than a node of its own.
  List<Node> get pageChildren => isValue ? const [] : nodes;

  /// The plain text of a value: the alt text of an image, or the RFC 3339
  /// timestamp of a time. Null for a container.
  String? get val => switch (this) {
    final Text t => t.data,
    _ when !isValue => null,
    _ when isImage => attributes['alt'],
    _ => nodes.whereType<Text>().map((t) => t.data).join(),
  };

  /// The link target, or the image URL of an image.
  String? get link => attributes[isImage ? 'src' : 'href'];

  bool get isImage => tag == 'img';
  bool get isTime => tag == 'time';

  /// The number of ancestors of this node.
  int get depth {
    var depth = 0;
    for (var n = parentNode; n != null; n = n.parentNode) {
      depth++;
    }
    return depth;
  }

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
  String get allText {
    if (isValue) {
      return isImage ? '' : val ?? '';
    }
    return pageChildren
        .map((c) => c.allText)
        .where((t) => t.isNotEmpty)
        .join(' ');
  }

  /// Whether [other] is this node or one of its descendants.
  bool includes(Node other) {
    for (Node? n = other; n != null; n = n.parentNode) {
      if (identical(n, this)) {
        return true;
      }
    }
    return false;
  }

  /// Whether one of the two nodes includes the other.
  bool overlaps(Node other) => includes(other) || other.includes(this);
}
