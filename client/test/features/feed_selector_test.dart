import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:paperdoll/features/feed/domain/post_selector.dart';
import 'package:paperdoll/features/feed/domain/simplified_page.dart';

import '../src/fixture.dart';

void main() {
  final page = SimplifiedPage.parse(fixture.pages.exampleBlogWithNewsletters);
  final articles = page.body.pageChildren;

  /// [article] labeled as a post, with its heading as the title and the link.
  LabeledPost labeled(Node article) => (
    node: article,
    fields: {
      PostField.title: article.pageChildren.first,
      PostField.link: article.pageChildren.first,
    },
  );

  List<String> links(List<PostSelector> selectors) => [
    for (final p in extractPosts(page, selectors)) p.values[PostField.link]!,
  ];

  group('generateSelectors', () {
    test('finds the posts that look like a labeled post', () {
      final selectors = generateSelectors(page, [labeled(articles[0])], {});

      expect(selectors, hasLength(1));
      expect(links(selectors), [
        'https://blog.example.com/first',
        'https://blog.example.com/newsletters?weekly',
        'https://blog.example.com/newsletters?monthly',
      ]);
    });

    test('excludes the host and path of a link, and nothing else', () {
      final selectors = generateSelectors(
        page,
        [labeled(articles[0])],
        {'blog.example.com/newsletters'},
      );

      expect(links(selectors), ['https://blog.example.com/first']);
    });

    test('does not depend on the order of the labeled posts', () {
      String generate(List<LabeledPost> posts) =>
          jsonEncode(generateSelectors(page, posts, {}));

      expect(
        generate([labeled(articles[0]), labeled(articles[2])]),
        generate([labeled(articles[2]), labeled(articles[0])]),
      );
    });
  });

  group('labelFoundPosts', () {
    test('labels every post that the selectors find with its fields', () {
      final selectors = generateSelectors(page, [labeled(articles[0])], {});

      final posts = labelFoundPosts(page, selectors);

      expect([for (final p in posts) p.node], articles);
      expect(
        [for (final p in posts) p.fields],
        [
          for (final a in articles)
            {
              PostField.title: a.pageChildren.first,
              PostField.link: a.pageChildren.first,
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

      final posts = labelFoundPosts(page, [titles, bodies]);

      expect([for (final p in posts) p.node], articles);
      expect(
        [for (final p in posts) p.fields],
        [
          for (final a in articles)
            {
              PostField.title: a.pageChildren.first,
              PostField.link: a.pageChildren.first,
              PostField.body: a.pageChildren.last,
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

      expect(labelFoundPosts(page, [selector]), isEmpty);
    });
  });
}
