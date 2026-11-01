import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:paperdoll/features/feed/domain/post_selector.dart';
import 'package:paperdoll/features/feed/domain/semantic_tree.dart';

import '../src/fixture.dart';

void main() {
  final tree = SemanticTree.fromJson(fixture.pages.exampleBlogWithNewsletters);
  final articles = tree.root.children;

  /// [article] labeled as a post, with its heading as the title and the link.
  LabeledPost labeled(SemanticTreeNode article) => (
    node: article,
    fields: {
      PostField.title: article.children.first,
      PostField.link: article.children.first,
    },
  );

  List<String> links(List<PostSelector> selectors) => [
    for (final p in extractPosts(tree, selectors)) p.values[PostField.link]!,
  ];

  group('generateSelectors', () {
    test('finds the posts that look like a labeled post', () {
      final selectors = generateSelectors(tree, [labeled(articles[0])], {});

      expect(selectors, hasLength(1));
      expect(links(selectors), [
        'https://blog.example.com/first',
        'https://blog.example.com/newsletters?weekly',
        'https://blog.example.com/newsletters?monthly',
      ]);
    });

    test('excludes the host and path of a link, and nothing else', () {
      final selectors = generateSelectors(
        tree,
        [labeled(articles[0])],
        {'blog.example.com/newsletters'},
      );

      expect(links(selectors), ['https://blog.example.com/first']);
    });

    test('does not depend on the order of the labeled posts', () {
      String generate(List<LabeledPost> posts) =>
          jsonEncode(generateSelectors(tree, posts, {}));

      expect(
        generate([labeled(articles[0]), labeled(articles[2])]),
        generate([labeled(articles[2]), labeled(articles[0])]),
      );
    });
  });

  group('labelFoundPosts', () {
    test('labels every post that the selectors find with its fields', () {
      final selectors = generateSelectors(tree, [labeled(articles[0])], {});

      final posts = labelFoundPosts(tree, selectors);

      expect([for (final p in posts) p.node], articles);
      expect(
        [for (final p in posts) p.fields],
        [
          for (final a in articles)
            {
              PostField.title: a.children.first,
              PostField.link: a.children.first,
            },
        ],
      );
    });

    test('takes each field of a post from the first selector that has it', () {
      const titles = PostSelector(
        post: [Step(tag: 'article')],
        title: [Step(tag: 'h2')],
        link: [Step(tag: 'h2')],
      );
      const bodies = PostSelector(
        post: [Step(tag: 'article')],
        title: [Step(tag: 'p')],
        link: [Step(tag: 'h2')],
        body: [Step(tag: 'p')],
      );

      final posts = labelFoundPosts(tree, [titles, bodies]);

      expect([for (final p in posts) p.node], articles);
      expect(
        [for (final p in posts) p.fields],
        [
          for (final a in articles)
            {
              PostField.title: a.children.first,
              PostField.link: a.children.first,
              PostField.body: a.children.last,
            },
        ],
      );
    });

    test('labels nothing when the selectors find no post', () {
      const selector = PostSelector(
        post: [Step(tag: 'section')],
        title: [Step(tag: 'h2')],
        link: [Step(tag: 'h2')],
      );

      expect(labelFoundPosts(tree, [selector]), isEmpty);
    });
  });
}
