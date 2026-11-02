import 'package:paperdoll/features/feed/domain/body_text.dart';
import 'package:paperdoll/features/feed/domain/simplified_page.dart';

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
  bool accepts(Node node) => switch (this) {
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
}

/// The host and path of [link], which exclude patterns match and by which
/// the user excludes a link.
String hostPathOf(String link) {
  final uri = Uri.parse(link);
  return '${uri.host}${uri.path}';
}

/// A post that the user labeled: its node, and the nodes of its fields.
typedef LabeledPost = ({Node node, Map<PostField, Node> fields});

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
Map<PostField, Node> evidentFields(Node postNode) {
  final nodes = <Node>[];
  void collect(Node n) {
    nodes.add(n);
    n.pageChildren.forEach(collect);
  }

  collect(postNode);
  Node? only(Iterable<Node> matches) =>
      matches.length == 1 ? matches.first : null;
  final fields = <PostField, Node>{};
  void label(PostField field, Node? node) {
    if (node != null && field.accepts(node)) {
      fields[field] = node;
    }
  }

  final links = nodes.where((n) => n.pageLink != null);
  if (links.map((n) => n.link).toSet().length == 1) {
    label(
      PostField.link,
      links.where((n) => identical(n, postNode)).firstOrNull ??
          links.where((n) => n.allText.isNotEmpty).firstOrNull ??
          links.first,
    );
  }
  final images = nodes.where((n) => n.isImage).toList();
  if (images.isNotEmpty) {
    final depths = {for (final n in images) n: n.depth};
    final top = depths.values.reduce((a, b) => a < b ? a : b);
    label(PostField.thumbnail, only(images.where((n) => depths[n] == top)));
  }
  label(PostField.pubDate, only(nodes.where((n) => n.isTime)));
  final headings = nodes
      .where((n) => n.isValue && n.headingLevel != null)
      .toList();
  Node? title;
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
      if (!identical(p.parentNode, postNode) &&
          paragraphs
                  .where((q) => identical(q.parentNode, p.parentNode))
                  .length >=
              2 &&
          !labeled.any(p.parentNode!.includes))
        p.parentNode!,
  };
  label(
    PostField.body,
    only([
      ...containers,
      ...paragraphs.where((p) => !containers.contains(p.parentNode)),
    ]),
  );
  return fields;
}
