import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/debug_keys.dart';
import 'package:patrol_finders/patrol_finders.dart';

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
      ..stubGet(
        '/feeds/${blog.id}/matcher',
        body: {'page': fixture.pages.exampleBlog},
      );

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchTextField).enterText(blog.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();

    // The page has no matchers yet, so the app asks for them.
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();
    await t('First post').waitUntilVisible();
    expect(
      t.tester.widget<TextButton>(t(AppDebugKey.saveMatchersButton)).onPressed,
      isNull,
    );
  });

  patrolWidgetTest('Save the post matchers of a web page', (t) async {
    final candidate = fixture.feedCandidates.exampleBlog;
    final blog = fixture.feeds.exampleBlog;
    final matcher = fixture.pages.exampleBlogMatcher;
    final server = StubServer.withDefaultResponses()
      ..stubPut('/feeds', body: blog.toJson())
      ..stubGet(
        '/feeds/search',
        body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
      )
      // Another user saved matchers after this user subscribed, so the
      // editor starts with them.
      ..stubGet(
        '/feeds/${blog.id}/matcher',
        body: {
          'page': fixture.pages.exampleBlog,
          'matchers': [matcher],
        },
      )
      ..stubPut(
        '/feeds/${blog.id}/matcher',
        status: 204,
        bodyMatcher: {
          'matchers': [matcher],
        },
      );

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchTextField).enterText(blog.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();

    await t(AppDebugKey.saveMatchersButton).tap();
    await t(AppDebugKey.saveMatchersSuccessSnackBar).waitUntilVisible();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
  });

  patrolWidgetTest('Edit the post matchers of a web page', (t) async {
    final blog = fixture.feeds.exampleBlog;
    final server = _stubFeedSubscription(blog)
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [blog]).toJson())
      ..stubGet(
        '/feeds/${blog.id}/matcher',
        body: {
          'page': fixture.pages.exampleBlog,
          'matchers': [fixture.pages.exampleBlogMatcher],
        },
      );

    await pumpAppWithAuth(t, server);
    await _openTimeline(t, blog);
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.editFeedMenuItem).tap();
    await t(AppDebugKey.feedEditScreen).waitUntilVisible();
    expect(t(blog.url), findsOneWidget);

    await t(AppDebugKey.editMatchersButton).tap();
    await t(AppDebugKey.pageAnnotationScreen).waitUntilVisible();
    // The posts are shown as cards of their extracted values, without the
    // content that is not extracted, such as the summaries.
    await t('First post').waitUntilVisible();
    expect(
      t('Group 1 · Not extracted: Body, Thumbnail, Pub date'),
      findsNWidgets(2),
    );
    expect(t('The first post.'), findsNothing);

    // The preview shows the same cards, one for each post that is saved.
    await t('Preview posts').tap();
    await t('2 post(s)').waitUntilVisible();
    expect(
      t('Group 1 · Not extracted: Body, Thumbnail, Pub date'),
      findsNWidgets(4),
    );
    await t.tester.tapAt(const Offset(10, 10));
    await t.pumpAndSettle();

    // A card selects its post.
    await t('First post').tap();
    await t('Edit fields').waitUntilVisible();
  });

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
    // The matchers are only for pages without a feed.
    expect(t(AppDebugKey.editMatchersButton), findsNothing);
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
