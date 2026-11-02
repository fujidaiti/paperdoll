import 'package:paperdoll/features/feed/domain/post_field.dart';
import 'package:paperdoll/features/feed/domain/simplified_page.dart';

/// A temporary editor on the server in which the user edits the post
/// selectors of an unstructured feed (see SelectorEditor in
/// server/feature/feed/selector_editor.go). The server generates the
/// selectors from the marks of the user and tells what they find, so the
/// client never runs selectors.
class SelectorEditor {
  const new({
    required this.id,
    required this.page,
    required this.labeledPosts,
    required this.excludedLinks,
    required this.state,
  });

  factory fromJson(Map<String, Object?> json) {
    final page = SimplifiedPage.parse(json['page']! as String);
    return SelectorEditor(
      id: json['id']! as int,
      page: page,
      labeledPosts: [
        for (final p in json['labeled_posts']! as List<Object?>)
          _labeledPostFromJson(page, p! as Map<String, Object?>),
      ],
      excludedLinks: {
        ...(json['excluded_links']! as List<Object?>).cast<String>(),
      },
      state: EditorState.fromJson(page, json),
    );
  }

  final int id;
  final SimplifiedPage page;

  /// The posts labeled so far. A new editor starts with the posts that the
  /// saved selectors find, since the saved selectors do not keep the marks of
  /// the user.
  final List<LabeledPost> labeledPosts;

  /// The host and path of each excluded link (see [hostPathOf]).
  final Set<String> excludedLinks;

  final EditorState state;
}

/// What the selectors generated from the marks of an editor find on its page.
class EditorState {
  const new({
    required this.matchedPosts,
    required this.excludedPosts,
    required this.preview,
  });

  factory fromJson(SimplifiedPage page, Map<String, Object?> json) {
    return EditorState(
      matchedPosts: {
        for (final p in json['matched_posts']! as List<Object?>)
          if (p case final Map<String, Object?> p)
            page.nodeOf(p['box']! as int): (
              fields: _fieldsFromJson(page, p),
              values: {
                for (final MapEntry(:key, :value)
                    in (p['values']! as Map<String, Object?>).entries)
                  PostField.values.byName(key): value! as String,
              },
            ),
      },
      excludedPosts: {
        for (final p in json['excluded_posts']! as List<Object?>)
          if (p case final Map<String, Object?> p)
            page.nodeOf(p['box']! as int): p['link']! as String,
      },
      preview: [
        for (final p in json['preview']! as List<Object?>)
          {
            for (final MapEntry(:key, :value)
                in (p! as Map<String, Object?>).entries)
              PostField.values.byName(key): value! as String,
          },
      ],
    );
  }

  /// The posts that the selectors find, in document order.
  final Map<Node, MatchedPost> matchedPosts;

  /// The nodes that would be posts if their links were not excluded, with the
  /// excluded link that excludes each.
  final Map<Node, String> excludedPosts;

  /// The values of the posts that the selectors extract, merged by link, as
  /// the server will save them.
  final List<Map<PostField, String>> preview;
}

/// A post that the selectors find: the nodes of its fields, and their values.
typedef MatchedPost = ({
  Map<PostField, Node> fields,
  Map<PostField, String> values,
});

/// The nodes of the fields in [json], a post by node IDs with the node of
/// the post under `box` and the node of each field under its name.
Map<PostField, Node> _fieldsFromJson(
  SimplifiedPage page,
  Map<String, Object?> json,
) => {
  for (final field in PostField.values)
    if (json[field.name] case final int id) field: page.nodeOf(id),
};

LabeledPost _labeledPostFromJson(
  SimplifiedPage page,
  Map<String, Object?> json,
) => (
  node: page.nodeOf(json['box']! as int),
  fields: _fieldsFromJson(page, json),
);

/// [post] by node IDs, as the server takes it.
Map<String, Object?> labeledPostToJson(SimplifiedPage page, LabeledPost post) =>
    {
      'box': page.idOf(post.node),
      for (final MapEntry(:key, :value) in post.fields.entries)
        key.name: page.idOf(value),
    };
