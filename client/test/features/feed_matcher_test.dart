import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:paperdoll/features/feed/domain/post_matcher.dart';
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

  List<String> links(List<PostMatcher> matchers) => [
    for (final p in extractPosts(tree, matchers)) p.values[PostField.link]!,
  ];

  group('generateMatchers', () {
    test('finds the posts that look like a labeled post', () {
      final matchers = generateMatchers(tree, [labeled(articles[0])], {});

      expect(matchers, hasLength(1));
      expect(links(matchers), [
        'https://blog.example.com/first',
        'https://blog.example.com/newsletters?weekly',
        'https://blog.example.com/newsletters?monthly',
      ]);
    });

    test('excludes the host and path of a link, and nothing else', () {
      final matchers = generateMatchers(
        tree,
        [labeled(articles[0])],
        {'blog.example.com/newsletters'},
      );

      expect(links(matchers), ['https://blog.example.com/first']);
    });

    test('does not depend on the order of the labeled posts', () {
      String generate(List<LabeledPost> posts) =>
          jsonEncode(generateMatchers(tree, posts, {}));

      expect(
        generate([labeled(articles[0]), labeled(articles[2])]),
        generate([labeled(articles[2]), labeled(articles[0])]),
      );
    });
  });

  group('labelFoundPosts', () {
    test('labels every post that the matchers find with its fields', () {
      final matchers = generateMatchers(tree, [labeled(articles[0])], {});

      final posts = labelFoundPosts(tree, matchers);

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

    test('takes each field of a post from the first matcher that has it', () {
      const titles = PostMatcher(
        post: [Step(tag: 'article')],
        fields: {
          PostField.title: [
            [Step(tag: 'h2')],
          ],
          PostField.link: [
            [Step(tag: 'h2')],
          ],
        },
      );
      const bodies = PostMatcher(
        post: [Step(tag: 'article')],
        fields: {
          PostField.title: [
            [Step(tag: 'p')],
          ],
          PostField.link: [
            [Step(tag: 'h2')],
          ],
          PostField.body: [
            [Step(tag: 'p')],
          ],
        },
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

    test('labels nothing when the matchers find no post', () {
      const matcher = PostMatcher(post: [Step(tag: 'section')]);

      expect(labelFoundPosts(tree, [matcher]), isEmpty);
    });
  });
}
