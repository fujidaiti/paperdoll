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

  group('sampleLabeledPosts', () {
    test('takes the first post of each matcher with its fields', () {
      final matchers = generateMatchers(
        tree,
        [labeled(articles[0])],
        {'blog.example.com/newsletters'},
      );

      final samples = sampleLabeledPosts(tree, matchers);

      expect(samples, hasLength(1));
      expect(samples.single.node, same(articles[0]));
      expect(
        samples.single.fields[PostField.link],
        same(articles[0].children.first),
      );
    });

    test('drops a matcher that finds no post', () {
      const matcher = PostMatcher(post: [Step(tag: 'section')]);

      expect(sampleLabeledPosts(tree, [matcher]), isEmpty);
    });
  });
}
