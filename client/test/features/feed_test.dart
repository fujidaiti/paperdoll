import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/debug_keys.dart';
import 'package:patrol_finders/patrol_finders.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../src/boilerplate.dart';
import '../src/fixture.dart';
import '../src/stub_server.dart';

void main() {
  patrolWidgetTest('Check feeds and read a feed entry', (t) async {
    final bbcNews = fixture.feeds.bbcNews;
    final entry = fixture.entries.nuclearDeal;
    final timeline = [
      entry,
      fixture.entries.houthiStrikes,
      fixture.entries.moonshotAi,
    ];

    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/feeds',
        body: api.GetFeeds200Response(
          feeds: [
            bbcNews,
            fixture.feeds.stackOverflow,
            fixture.feeds.wikipedia,
          ],
        ).toJson(),
      )
      ..stubGet('/feeds/${bbcNews.id}', body: bbcNews.toJson())
      ..stubGet(
        '/feeds/${bbcNews.id}/timeline',
        body: api.GetFeedTimeline200Response(entries: timeline).toJson(),
      )
      ..stubGet('/feed-entries/${entry.id}', body: entry.toJson());

    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.feedRow(bbcNews.title)).tap();
    await t(AppDebugKey.feedDetailScreen).waitUntilVisible();
    await t(AppDebugKey.feedEntryRow(entry.title)).tap();
    await t(AppDebugKey.feedEntryReaderScreen).waitUntilVisible();
    await t(AppDebugKey.readerTitle(entry.title)).waitUntilVisible();
    expect(
      t.tester.getSemantics(find.byKey(AppDebugKey.readerArticleTab)),
      isSemantics(isSelected: true),
    );
  });

  patrolWidgetTest('Unsubscribe from a feed and subscribe again', (t) async {
    final bbcNews = fixture.feeds.bbcNews;
    final server = _stubFeedSubscription(bbcNews);

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, bbcNews);

    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.unsubscribeMenuItem).tap();
    await t(AppDebugKey.unsubscribeSuccessSnackBar).waitUntilVisible();

    // The feed is shared across all users, so it is not deleted: the screen
    // stays put and its timeline is still readable.
    expect(t(AppDebugKey.feedDetailScreen), findsOneWidget);
    expect(
      t(AppDebugKey.feedEntryRow(fixture.entries.nuclearDeal.title)),
      findsOneWidget,
    );

    // The menu now offers the opposite action.
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.subscribeMenuItem).tap();
    await t(AppDebugKey.subscribeSuccessSnackBar).waitUntilVisible();

    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.unsubscribeMenuItem).waitUntilVisible();
  });

  patrolWidgetTest('Undo an unsubscribe from the snackbar', (t) async {
    final bbcNews = fixture.feeds.bbcNews;
    final server = _stubFeedSubscription(bbcNews);

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, bbcNews);

    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.unsubscribeMenuItem).tap();
    await t(AppDebugKey.unsubscribeSuccessSnackBar).waitUntilVisible();

    await t('Undo').tap();
    await t(AppDebugKey.subscribeSuccessSnackBar).waitUntilVisible();

    // The subscription is back, so the menu offers to drop it again.
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.unsubscribeMenuItem).waitUntilVisible();
  });

  patrolWidgetTest('Subscribe to a known web feed', (t) async {
    final candidate = fixture.feedCandidates.nasa;
    final nasa = fixture.feeds.nasa;
    final subscriptions = <api.Feed>[];
    final server = StubServer.withDefaultResponses()
      ..onGet(
        '/feeds',
        respond: (_) =>
            (200, api.GetFeeds200Response(feeds: subscriptions).toJson()),
      )
      ..onPut(
        '/feeds',
        bodyMatcher: api.SubscribeToFeedRequest(url: nasa.url).toJson(),
        respond: (_) {
          subscriptions.add(nasa);
          return (200, nasa.toJson());
        },
      )
      ..stubGet(
        '/feeds/search',
        body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
      );

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchScreen).waitUntilVisible();

    await t(AppDebugKey.feedSearchTextField).enterText(nasa.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
    await t(AppDebugKey.subscribeSuccessSnackBar).waitUntilVisible();
    await t(AppDebugKey.feedRow(nasa.title)).waitUntilVisible();
  });

  patrolWidgetTest('Subscribe to a web page without a feed', (t) async {
    final candidate = fixture.feedCandidates.exampleBlog;
    final blog = fixture.feeds.exampleBlog;
    final server = StubServer.withDefaultResponses()
      ..stubPut('/feeds', body: blog.toJson())
      ..stubGet(
        '/feeds/search',
        body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
      )
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogEditor,
      );

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchTextField).enterText(blog.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();

    // The page has no selectors yet, so the app asks for them.
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();
    await t('First post').waitUntilVisible();
    expect(
      t.tester.widget<TextButton>(t(AppDebugKey.saveSelectorsButton)).onPressed,
      isNull,
    );
    // The editor is remembered until it ends, so that it can be continued
    // after the app is killed.
    expect(
      await SharedPreferencesAsync().getInt('selector_editor.${blog.id}'),
      42,
    );
  });

  patrolWidgetTest('Mark a post on a web page', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogEditor,
      )
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        bodyMatcher: {
          'labeled_posts': [
            {'box': 2, 'title': 2, 'link': 2},
          ],
          'excluded_links': <Object?>[],
          'excluded_nodes': <Object?>[],
        },
        body: fixture.pages.exampleBlogHeadingsFound,
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();

    // The heading is a post with itself as the title and the link.
    await t('First post').tap();
    await t('Mark as post').tap();
    await t('Done').tap();

    // The server finds the other heading too.
    await t('2 post(s) found.').waitUntilVisible();
    expect(
      t.tester.widget<TextButton>(t(AppDebugKey.saveSelectorsButton)).onPressed,
      isNotNull,
    );
  });

  patrolWidgetTest('Save the post selectors of a web page', (t) async {
    final candidate = fixture.feedCandidates.exampleBlog;
    final blog = fixture.feeds.exampleBlog;
    final server = StubServer.withDefaultResponses()
      ..stubPut('/feeds', body: blog.toJson())
      ..stubGet(
        '/feeds/search',
        body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
      )
      // Another user saved selectors after this user subscribed, so the
      // editor starts with the posts that they find.
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogSavedEditor,
      )
      ..stubPost('/feeds/${blog.id}/selector-editors/42/commit', status: 204);

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchTextField).enterText(blog.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();

    await t(AppDebugKey.saveSelectorsButton).tap();
    await t(AppDebugKey.saveSelectorsSuccessSnackBar).waitUntilVisible();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    // The committed editor has ended, so it is not offered to continue.
    expect(
      await SharedPreferencesAsync().getInt('selector_editor.${blog.id}'),
      isNull,
    );
  });

  patrolWidgetTest('Edit the post selectors of a web page', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogSavedEditor,
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.feedEditScreen).waitUntilVisible();
    expect(t(blog.url), findsOneWidget);

    await t(AppDebugKey.editSelectorsButton).tap();
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();
    // The posts are shown as cards of their extracted values, without the
    // content that is not extracted, such as the summaries.
    await t('First post').waitUntilVisible();
    expect(t('https://blog.example.com/second'), findsOneWidget);
    expect(t('The first post.'), findsNothing);

    // The preview shows the same cards, one for each post that is saved.
    await t('Preview posts').tap();
    await t('2 post(s)').waitUntilVisible();
    // The screen behind the sheet has the cards too.
    expect(t('https://blog.example.com/second'), findsNWidgets(2));
    await t.tester.tapAt(const Offset(10, 10));
    await t.pumpAndSettle();

    // A card selects its post.
    await t('First post').tap();
    await t('Edit fields').waitUntilVisible();
  });

  patrolWidgetTest('Mark a post as not a post by its link', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogWithNewslettersEditor,
      )
      // The newsletters are unmarked, since their link is excluded.
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        bodyMatcher: {
          'labeled_posts': [
            {'box': 1, 'title': 2, 'link': 2},
          ],
          'excluded_links': ['blog.example.com/newsletters'],
          'excluded_nodes': <Object?>[],
        },
        body: fixture.pages.exampleBlogWithNewslettersExcluded,
      )
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        bodyMatcher: {
          'labeled_posts': [
            {'box': 1, 'title': 2, 'link': 2},
          ],
          'excluded_links': <Object?>[],
          'excluded_nodes': <Object?>[],
        },
        body: fixture.pages.exampleBlogWithNewslettersAllFound,
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();
    await t('3 post(s) found.').waitUntilVisible();

    // The link of a newsletter is excluded by its host and path, so the other
    // newsletter, whose link differs only in the query, is excluded too.
    await t('Weekly digest').tap();
    await t('Not a post').tap();
    await t('1 post(s) found.').waitUntilVisible();

    // The excluded posts are shown as such, and can be restored.
    await t('Weekly digest').tap();
    await t('Marked as not a post.').waitUntilVisible();
    await t('Restore').tap();
    await t('3 post(s) found.').waitUntilVisible();
  });

  patrolWidgetTest('Exclude a section of a web page', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogWithMenuEditor,
      )
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        bodyMatcher: {
          'labeled_posts': [
            {'box': 5, 'title': 6, 'link': 6},
          ],
          'excluded_links': <Object?>[],
          'excluded_nodes': [1],
        },
        body: fixture.pages.exampleBlogWithMenuExcluded,
      )
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        bodyMatcher: {
          'labeled_posts': [
            {'box': 5, 'title': 6, 'link': 6},
          ],
          'excluded_links': <Object?>[],
          'excluded_nodes': <Object?>[],
        },
        body: fixture.pages.exampleBlogWithMenuAllFound,
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();
    await t('4 post(s) found.').waitUntilVisible();

    // A tap on the text of the menu selects the whole menu, whose links are
    // found as posts.
    await t('Menu').tap();
    await t('Exclude').tap();

    // The menu is shown as one line instead of its content.
    await t('2 post(s) found.').waitUntilVisible();
    expect(t('About'), findsNothing);
    await t('Excluded: 3 texts, 2 links').tap();
    await t('Excluded from posts.').waitUntilVisible();
    await t('Restore').tap();
    await t('4 post(s) found.').waitUntilVisible();
    expect(t('About'), findsOneWidget);
  });

  patrolWidgetTest('Exclude a section with marked posts', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final labeled = [
      {'box': 3, 'title': 3, 'link': 3},
      {'box': 4, 'title': 4, 'link': 4},
      {'box': 5, 'title': 6, 'link': 6},
      {'box': 8, 'title': 9, 'link': 9},
    ];
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogWithMenuSavedEditor,
      )
      // The links of the menu keep their marks while the menu is excluded.
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        bodyMatcher: {
          'labeled_posts': labeled,
          'excluded_links': <Object?>[],
          'excluded_nodes': [1],
        },
        body: fixture.pages.exampleBlogWithMenuExcluded,
      )
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        bodyMatcher: {
          'labeled_posts': labeled,
          'excluded_links': <Object?>[],
          'excluded_nodes': <Object?>[],
        },
        body: fixture.pages.exampleBlogWithMenuAllFound,
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();
    await t('4 post(s) found.').waitUntilVisible();

    // The links of the menu are marked, so the app asks first.
    await t('Menu').tap();
    await t('Exclude').tap();
    await t(
      'This contains 2 marked post(s). They are not posts while it is '
      'excluded.',
    ).waitUntilVisible();
    await t(AlertDialog).$('Exclude').tap();
    await t('2 post(s) found.').waitUntilVisible();

    // The links of the menu are still marked, so they are found again.
    await t('Excluded: 3 texts, 2 links').tap();
    await t('Restore').tap();
    await t('4 post(s) found.').waitUntilVisible();
    expect(t('About'), findsOneWidget);
  });

  patrolWidgetTest('Exclude a found post', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogWithMenuEditor,
      )
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        bodyMatcher: {
          'labeled_posts': [
            {'box': 5, 'title': 6, 'link': 6},
          ],
          'excluded_links': <Object?>[],
          'excluded_nodes': [8],
        },
        body: fixture.pages.exampleBlogWithMenuSecondPostExcluded,
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();
    await t('4 post(s) found.').waitUntilVisible();

    // The card of the post is replaced by the line of the excluded node.
    await t('Second post').tap();
    await t(find.byTooltip('More')).tap();
    await t('Exclude').tap();
    await t('3 post(s) found.').waitUntilVisible();
    expect(t('Second post'), findsNothing);
    expect(t('Excluded: 2 texts, 1 link'), findsOneWidget);
  });

  patrolWidgetTest('Tell that the editor expired when marking a post', (
    t,
  ) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogEditor,
      )
      ..stubPut(
        '/feeds/${blog.id}/selector-editors/42/marks',
        status: 410,
        body: {'message': 'The selector editor expired'},
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();

    await t('First post').tap();
    await t('Mark as post').tap();
    await t('Done').tap();

    // The editor expired, so the marks cannot be saved anymore.
    await t(AppDebugKey.selectorEditorExpiredDialog).waitUntilVisible();
    await t('OK').tap();
    await t(AppDebugKey.feedEditScreen).waitUntilVisible();
    expect(t(AppDebugKey.pageAnnotationScreen), findsNothing);
    expect(
      await SharedPreferencesAsync().getInt('selector_editor.${blog.id}'),
      isNull,
    );
  });

  patrolWidgetTest('Tell that the editor expired when saving', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogSavedEditor,
      )
      ..stubPost(
        '/feeds/${blog.id}/selector-editors/42/commit',
        status: 410,
        body: {'message': 'The selector editor expired'},
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();

    await t(AppDebugKey.saveSelectorsButton).tap();

    // The editor expired, so the selectors are not saved.
    await t(AppDebugKey.selectorEditorExpiredDialog).waitUntilVisible();
    await t('OK').tap();
    await t(AppDebugKey.feedEditScreen).waitUntilVisible();
    expect(t(AppDebugKey.pageAnnotationScreen), findsNothing);
    expect(
      await SharedPreferencesAsync().getInt('selector_editor.${blog.id}'),
      isNull,
    );
  });

  patrolWidgetTest('Leave the editor of a web page without saving', (t) async {
    final blog = fixture.feeds.exampleBlog;
    var closed = false;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogSavedEditor,
      )
      ..onDelete(
        '/feeds/${blog.id}/selector-editors/42',
        respond: (_) {
          closed = true;
          return (204, null);
        },
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();

    // Going back asks first, and canceling keeps the editor open.
    await t.tester.pageBack();
    await t.pumpAndSettle();
    await t('Leave without saving?').waitUntilVisible();
    await t('Cancel').tap();
    expect(t(AppDebugKey.pageAnnotationScreen), findsOneWidget);
    expect(closed, isFalse);

    await t.tester.pageBack();
    await t.pumpAndSettle();
    await t('Leave').tap();
    await t(AppDebugKey.feedEditScreen).waitUntilVisible();
    expect(t(AppDebugKey.pageAnnotationScreen), findsNothing);
    expect(closed, isTrue);
    expect(
      await SharedPreferencesAsync().getInt('selector_editor.${blog.id}'),
      isNull,
    );
  });

  patrolWidgetTest('Continue the previous edits of a web page', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubGet(
        '/feeds/${blog.id}/selector-editors/41',
        body: {...fixture.pages.exampleBlogSavedEditor, 'id': 41},
      );

    await pumpAppWithAuth(t, server);
    // The app was killed while the editor 41 was open.
    await SharedPreferencesAsync().setInt('selector_editor.${blog.id}', 41);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();

    await t('Continue your previous edits?').waitUntilVisible();
    await t('Continue').tap();
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();
    await t('2 post(s) found.').waitUntilVisible();
    expect(
      await SharedPreferencesAsync().getInt('selector_editor.${blog.id}'),
      41,
    );
  });

  patrolWidgetTest('Start over the previous edits of a web page', (t) async {
    final blog = fixture.feeds.exampleBlog;
    var closed = false;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubGet(
        '/feeds/${blog.id}/selector-editors/41',
        body: {...fixture.pages.exampleBlogSavedEditor, 'id': 41},
      )
      ..onDelete(
        '/feeds/${blog.id}/selector-editors/41',
        respond: (_) {
          closed = true;
          return (204, null);
        },
      )
      ..stubPost(
        '/feeds/${blog.id}/selector-editors',
        status: 201,
        body: fixture.pages.exampleBlogEditor,
      );

    await pumpAppWithAuth(t, server);
    // The app was killed while the editor 41 was open.
    await SharedPreferencesAsync().setInt('selector_editor.${blog.id}', 41);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.editSelectorsButton).tap();

    await t('Continue your previous edits?').waitUntilVisible();
    await t('Start over').tap();
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();
    // The new editor has no marks.
    await t('First post').waitUntilVisible();
    expect(t('2 post(s) found.'), findsNothing);
    expect(closed, isTrue);
    expect(
      await SharedPreferencesAsync().getInt('selector_editor.${blog.id}'),
      42,
    );
  });

  for (final (status, reason) in [(410, 'expired'), (404, 'was deleted')]) {
    patrolWidgetTest('Start a new editor when the previous one $reason', (
      t,
    ) async {
      final blog = fixture.feeds.exampleBlog;
      final server = _stubFeedSubscription(blog)
        ..stubGet(
          '/feeds',
          body: api.GetFeeds200Response(feeds: [blog]).toJson(),
        )
        ..stubGet(
          '/feeds/${blog.id}/selector-editors/41',
          status: status,
          body: {'message': 'The selector editor $reason'},
        )
        ..stubPost(
          '/feeds/${blog.id}/selector-editors',
          status: 201,
          body: fixture.pages.exampleBlogEditor,
        );

      await pumpAppWithAuth(t, server);
      // The app was killed while the editor 41 was open.
      await SharedPreferencesAsync().setInt('selector_editor.${blog.id}', 41);
      await _openTimeline(t, blog);
      await t(AppDebugKey.feedDetailMenuButton).tap();
      await t(AppDebugKey.editFeedMenuItem).tap();
      await t(AppDebugKey.editSelectorsButton).tap();

      // The previous editor cannot be saved anymore, so the app does not ask
      // about it.
      await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();
      expect(t('Continue your previous edits?'), findsNothing);
      expect(
        await SharedPreferencesAsync().getInt('selector_editor.${blog.id}'),
        42,
      );
    });
  }

  patrolWidgetTest('Edit a web feed', (t) async {
    final nasa = fixture.feeds.nasa;
    final server = _stubFeedSubscription(
      nasa,
    )..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [nasa]).toJson());

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, nasa);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.feedEditScreen).waitUntilVisible();
    expect(t(nasa.url), findsOneWidget);
    // The selectors are only for pages without a feed.
    expect(t(AppDebugKey.editSelectorsButton), findsNothing);
  });
}

/// Stubs [feed]'s header, its timeline, and the two subscription routes, with
/// a `subscribed` flag that the DELETE and PUT flip — so the app bar menu
/// reflects what the server last reported, including after a refresh. The feed
/// and its timeline keep being served either way, because a feed is shared
/// across all users and unsubscribing only drops the caller's subscription.
StubServer _stubFeedSubscription(api.Feed feed) {
  var subscribed = true;
  return StubServer.withDefaultResponses()
    ..onGet(
      '/feeds/${feed.id}',
      respond: (_) => (200, {...feed.toJson(), 'subscribed': subscribed}),
    )
    ..onDelete(
      '/feeds/${feed.id}',
      respond: (_) {
        subscribed = false;
        return (204, null);
      },
    )
    ..onPut(
      '/feeds',
      bodyMatcher: api.SubscribeToFeedRequest(url: feed.url).toJson(),
      respond: (_) {
        subscribed = true;
        return (200, {...feed.toJson(), 'subscribed': true});
      },
    )
    ..stubGet(
      '/feeds/${feed.id}/timeline',
      body: api.GetFeedTimeline200Response(
        entries: [fixture.entries.nuclearDeal],
      ).toJson(),
    );
}

/// Navigates from the shell's Today tab to [feed]'s timeline screen.
Future<void> _openTimeline(PatrolTester t, api.Feed feed) async {
  await t(AppDebugKey.feedsNavDestination).tap();
  await t(AppDebugKey.feedsScreen).waitUntilVisible();
  await t(AppDebugKey.feedRow(feed.title)).tap();
  await t(AppDebugKey.feedDetailScreen).waitUntilVisible();
}
