import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:paperdoll/features/page_watch/domain/post_matcher.dart';
import 'package:paperdoll/features/page_watch/domain/semantic_tree.dart';
import 'package:paperdoll/features/page_watch/presentation/widgets/page_lines.dart';

// The server's test pages (see server/feature/feed/testdata/README.md) check
// that the client generates the matchers of each page, from the posts that a
// user of the client marks, and that the matchers extract the posts of the
// fixture. The server's tests do the same with their own port of the client's
// generator, so that the two implementations agree.

const _testdata = '../server/feature/feed/testdata';

typedef _Values = Map<PostField, String>;

/// The field values that [matcher] extracts from [post].
_Values _valuesOf(PostMatcher matcher, SemanticTreeNode post) => {
  for (final field in matcher.fields.keys)
    if (matcher.matchField(post, field) case final n?) field: ?field.valueOf(n),
};

bool _equalValues(_Values? a, _Values b) =>
    a != null &&
    a.length == b.length &&
    a.entries.every((e) => b[e.key] == e.value);

/// The node in [post] that the user labels as [field] to get the value
/// [want], or null when there is none. The user taps a line of the value and
/// widens the selection until it shows the value: a line with the value
/// itself, or for a body, a line of its text. A line that is not part of the
/// value, such as the date above a body, would reach a wider node with the
/// same text.
SemanticTreeNode? _fieldNode(
  SemanticTree tree,
  SemanticTreeNode post,
  PostField field,
  String want,
) {
  bool isValueNode(SemanticTreeNode n) =>
      field.valueOf(n) == want && field.accepts(n);
  for (final line in tree.nodes) {
    if (!post.contains(line)) {
      continue;
    }
    if (field == PostField.body) {
      if (!line.isValue || line.isImage || line.isTime) {
        continue;
      }
    } else if (!isValueNode(line)) {
      continue;
    }
    for (final n in selectionLevels(line, top: post)) {
      if (isValueNode(n)) {
        return n;
      }
    }
  }
  return null;
}

/// The matchers that a user of the client builds for [tree], who wants to
/// extract exactly [posts], and the number of marks that the user makes. The
/// client generates the matchers from the posts that the user marked and the
/// links that the user excluded (see [generateMatchers]), and the user
/// repeats, until every post is extracted with its values:
///
///  1. If a node that the matchers find is wrong, the user marks it. A node
///     whose link is not in [posts] is marked as not a post, which excludes
///     the host and path of its link, and unmarks the posts that the user
///     marked with them. A node that has a value that differs from the one in
///     [posts] is labeled as a post with the nodes of the values.
///  2. Otherwise, the user takes the first post that is not extracted with its
///     values, taps the line of its link, widens the selection until it holds
///     all values of the post, and labels the nodes of the values as its
///     fields.
///
/// In every round, every labeled post must be found and no excluded link must
/// be extracted. It throws when a step cannot be done, when a guarantee does
/// not hold, or when the marks come back to the same state.
(List<PostMatcher>, int) _perfectMatchers(
  SemanticTree tree,
  List<_Values> posts,
) {
  final want = {for (final p in posts) p[PostField.link]!: p};
  bool isWrong(_Values v) {
    final p = want[v[PostField.link]];
    return p == null || v.entries.any((e) => p[e.key] != e.value);
  }

  final labeled = <SemanticTreeNode, LabeledPost>{};
  final excluded = <String>{};
  // Labels [post] as a post with the nodes of the values in [target]. Returns
  // false when a value has no node.
  bool label(SemanticTreeNode post, _Values target) {
    final fields = <PostField, SemanticTreeNode>{};
    for (final MapEntry(:key, :value) in target.entries) {
      final n = _fieldNode(tree, post, key, value);
      if (n == null) {
        return false;
      }
      fields[key] = n;
    }
    labeled[post] = (node: post, fields: fields);
    excluded.remove(hostPathOf(target[PostField.link]!));
    return true;
  }

  var marks = 0;
  final seen = <String>{};
  while (true) {
    final matchers = generateMatchers(tree, labeled.values, excluded);

    // The matcher that the user sees a node as found by is the first one.
    final finder = <SemanticTreeNode, PostMatcher>{};
    for (final m in matchers) {
      for (final n in m.matchPosts(tree)) {
        finder.putIfAbsent(n, () => m);
      }
    }
    for (final n in labeled.keys) {
      if (!finder.containsKey(n)) {
        throw StateError(
          'the labeled post ${labeled[n]!.fields[PostField.link]?.link} is not '
          'found: ${jsonEncode(matchers)}',
        );
      }
    }
    final got = <String, _Values>{};
    for (final p in extractPosts(tree, matchers)) {
      if (excluded.contains(hostPathOf(p.values[PostField.link]!))) {
        throw StateError('the excluded link is extracted: ${p.values}');
      }
      got[p.values[PostField.link]!] = p.values;
    }

    final state = jsonEncode([
      matchers,
      excluded.toList()..sort(),
      [for (final n in labeled.keys) n.id]..sort(),
    ]);
    if (!seen.add(state)) {
      throw StateError('the marks come back to the same state: $state');
    }

    // A node of a post that has a value that differs from the one in posts,
    // and that does not hold the values in posts, such as a card of another
    // title with the link of the post, cannot be labeled. The user cannot
    // exclude it either, since that also excludes the post.
    var acted = false;
    for (final n in tree.nodes) {
      final m = finder[n];
      if (m == null || !isWrong(_valuesOf(m, n))) {
        continue;
      }
      final link = _valuesOf(m, n)[PostField.link]!;
      if (want[link] case final w?) {
        if (!label(n, w)) {
          continue;
        }
      } else {
        final hostPath = hostPathOf(link);
        excluded.add(hostPath);
        labeled.removeWhere(
          (_, p) => hostPathOf(p.fields[PostField.link]!.link!) == hostPath,
        );
      }
      marks++;
      acted = true;
      break;
    }
    if (acted) {
      continue;
    }

    final target = posts
        .where((p) => !_equalValues(got[p[PostField.link]], p))
        .firstOrNull;
    if (target == null) {
      return (matchers, marks);
    }
    final link = target[PostField.link]!;
    final start = tree.nodes
        .where((n) => PostField.link.accepts(n) && n.link == link)
        .firstOrNull;
    if (start == null) {
      throw StateError('post $link: no node has the link');
    }
    final post = selectionLevels(start).where((n) {
      return target.entries.every(
        (e) => _fieldNode(tree, n, e.key, e.value) != null,
      );
    }).firstOrNull;
    if (post == null) {
      throw StateError('post $link: no node holds all of its values: $target');
    }
    // The client does not let a new post overlap a post that it finds.
    for (final MapEntry(key: n, value: m) in finder.entries) {
      if (!identical(n, post) && n.overlaps(post)) {
        throw StateError(
          'post $link overlaps with the post '
          '${_valuesOf(m, n)[PostField.link]}: ${jsonEncode(m)}',
        );
      }
    }
    marks++;
    if (!label(post, target)) {
      throw StateError('post $link: a value has no node');
    }
  }
}

Object? _readJson(String path) => jsonDecode(File(path).readAsStringSync());

void main() {
  final fixtures =
      Directory('$_testdata/fixture')
          .listSync()
          .map((f) => f.uri.pathSegments.last)
          .where((f) => f.endsWith('.fixture.json'))
          .map((f) => f.substring(0, f.length - '.fixture.json'.length))
          .toList()
        ..sort();

  for (final name in fixtures) {
    group(name, () {
      final tree = SemanticTree.fromJson(
        _readJson('$_testdata/tree/$name.json')! as Map<String, Object?>,
      );
      final posts = [
        for (final p
            in _readJson('$_testdata/fixture/$name.fixture.json')!
                as List<Object?>)
          {
            for (final MapEntry(:key, :value)
                in (p! as Map<String, Object?>).entries)
              PostField.values.byName(key): value! as String,
          },
      ];
      final goldens = [
        for (final m
            in _readJson('$_testdata/matcher/$name.json')! as List<Object?>)
          PostMatcher.fromJson(m! as Map<String, Object?>),
      ];

      test('generates the golden matchers from the marks of a user', () {
        final (matchers, _) = _perfectMatchers(tree, posts);

        expect(
          jsonDecode(jsonEncode(matchers)),
          jsonDecode(jsonEncode(goldens)),
        );
      });

      test('extracts the posts in the fixture with the golden matchers', () {
        // The posts are compared by link, since a fixture does not always
        // list them in the order of the page.
        expect(
          {
            for (final p in extractPosts(tree, goldens))
              p.values[PostField.link]!: p.values,
          },
          {for (final p in posts) p[PostField.link]!: p},
        );
      });
    });
  }
}
