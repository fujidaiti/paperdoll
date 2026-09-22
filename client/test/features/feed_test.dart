import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed/data/feed_repository_impl.dart';
import 'package:paperdoll/features/feed/domain/post_selection.dart';
import 'package:paperdoll/features/feed/presentation/providers/feed_providers.dart';
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
        // Written out rather than built from api.SubscribeToFeedRequest,
        // whose toJson always writes an empty `selectors` list. A feed URL is
        // subscribed to with the url alone, so the app sends no `selectors`,
        // and bodyMatcher matches a subset of what the app actually sent.
        bodyMatcher: {'url': nasa.url},
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

  patrolWidgetTest('Subscribe to a web page with no feed', (t) async {
    final candidate = fixture.feedCandidates.exampleBlog;
    final exampleBlog = fixture.feeds.exampleBlog;
    final subscriptions = <api.Feed>[];
    Object? sentBody;
    final server = StubServer.withDefaultResponses()
      ..onGet(
        '/feeds',
        respond: (_) =>
            (200, api.GetFeeds200Response(feeds: subscriptions).toJson()),
      )
      ..onPut(
        '/feeds',
        respond: (body) {
          sentBody = body;
          subscriptions.add(exampleBlog);
          return (200, exampleBlog.toJson());
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
    await t(AppDebugKey.feedSearchTextField).enterText(candidate.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
    await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();

    await t(AppDebugKey.postGroupCheckbox(0)).tap();
    await t(AppDebugKey.postGroupCard(0)).tap();
    await t(AppDebugKey.attributePickerScreen).waitUntilVisible();
    await t(AppDebugKey.attributeRow('div > h3')).tap();
    await t(AppDebugKey.attributeNextButton).tap();
    await t(AppDebugKey.attributeNoneButton).tap();
    await t(AppDebugKey.attributeRow('div > time')).tap();
    await t(AppDebugKey.attributeNextButton).tap();
    await t(AppDebugKey.attributeRow('img')).tap();
    await t(AppDebugKey.attributeNextButton).tap();
    await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();
    await t(AppDebugKey.feedSubscriptionSubscribeButton).tap();

    await t(AppDebugKey.subscribeSuccessSnackBar).waitUntilVisible();
    expect(sentBody, {
      'url': candidate.url,
      'selectors': [
        {
          'root': 'main > article',
          'link': ':scope',
          'title': 'div > h3',
          'timestamp': 'div > time',
          'image': 'img',
        },
      ],
    });
    await t(AppDebugKey.feedRow(exampleBlog.title)).waitUntilVisible();
  });

  patrolWidgetTest('Skip the link question when the group offers one link', (
    t,
  ) async {
    final candidate = fixture.feedCandidates.exampleBlog;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/feeds/search',
        body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
      );

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchTextField).enterText(candidate.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
    await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();

    await t(AppDebugKey.postGroupCard(0)).tap();
    await t(AppDebugKey.attributePickerScreen).waitUntilVisible();
    expect(t('Which row holds the title?'), findsOneWidget);
    await t(AppDebugKey.attributeBackButton).tap();

    await t(AppDebugKey.postGroupCard(2)).scrollTo().tap();
    await t(AppDebugKey.attributePickerScreen).waitUntilVisible();
    expect(t('Which row holds the link?'), findsOneWidget);
  });

  patrolWidgetTest(
    'Keep the choices when a group is unticked and ticked again',
    (t) async {
      final candidate = fixture.feedCandidates.exampleBlog;
      final server = StubServer.withDefaultResponses()
        ..stubGet(
          '/feeds/search',
          body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
        );

      await pumpAppWithAuth(t, server);
      await t(AppDebugKey.feedsNavDestination).tap();
      await t(AppDebugKey.feedsScreen).waitUntilVisible();
      await t(AppDebugKey.addFeedButton).tap();
      await t(AppDebugKey.feedSearchTextField).enterText(candidate.url);
      await t(AppDebugKey.feedSearchButton).tap();
      await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
      await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();

      await t(AppDebugKey.postGroupCheckbox(0)).tap();
      await t(AppDebugKey.postGroupCard(0)).tap();
      await t(AppDebugKey.attributeRow('div > h3')).tap();
      await t(AppDebugKey.attributeBackButton).tap();
      await t(AppDebugKey.postGroupCheckbox(0)).tap();
      await t(AppDebugKey.postGroupCheckbox(0)).tap();
      expect(
        t.tester.widget<Checkbox>(t(AppDebugKey.postGroupCheckbox(0))).value,
        isTrue,
      );

      await t(AppDebugKey.postGroupCard(0)).tap();
      expect(
        t.tester
            .widget<ListTile>(
              find.descendant(
                of: t(AppDebugKey.attributeRow('div > h3')),
                matching: find.byType(ListTile),
              ),
            )
            .selected,
        isTrue,
      );
    },
  );

  patrolWidgetTest(
    'Keep the choices when returning from the attribute screen',
    (t) async {
      final candidate = fixture.feedCandidates.exampleBlog;
      final server = StubServer.withDefaultResponses()
        ..stubGet(
          '/feeds/search',
          body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
        );

      await pumpAppWithAuth(t, server);
      await t(AppDebugKey.feedsNavDestination).tap();
      await t(AppDebugKey.feedsScreen).waitUntilVisible();
      await t(AppDebugKey.addFeedButton).tap();
      await t(AppDebugKey.feedSearchTextField).enterText(candidate.url);
      await t(AppDebugKey.feedSearchButton).tap();
      await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
      await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();

      await t(AppDebugKey.postGroupCard(0)).tap();
      await t(AppDebugKey.attributeRow('div > h3')).tap();
      await t(AppDebugKey.attributeBackButton).tap();
      await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();
      final card = t(AppDebugKey.postGroupCard(0));
      expect(card.$('Why we moved to a monorepo'), findsOneWidget);
      expect(card.$('2026-09-12'), findsNothing);
      expect(card.$('Guessed preview. Tap to choose.'), findsNothing);

      await t(AppDebugKey.postGroupCard(0)).tap();
      expect(
        t.tester
            .widget<ListTile>(
              find.descendant(
                of: t(AppDebugKey.attributeRow('div > h3')),
                matching: find.byType(ListTile),
              ),
            )
            .selected,
        isTrue,
      );
    },
  );

  patrolWidgetTest('A group with no link cannot be subscribed to', (t) async {
    final candidate = fixture.feedCandidates.exampleBlog;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/feeds/search',
        body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
      );

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchTextField).enterText(candidate.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
    await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();

    await t(AppDebugKey.postGroupCheckbox(1)).scrollTo().tap();
    expect(
      t.tester.widget<Checkbox>(t(AppDebugKey.postGroupCheckbox(1))).value,
      isFalse,
    );
    expect(
      t.tester
          .widget<FilledButton>(t(AppDebugKey.feedSubscriptionSubscribeButton))
          .onPressed,
      isNull,
    );
  });

  patrolWidgetTest('Warn when a selector does not reach every post', (t) async {
    final candidate = fixture.feedCandidates.exampleBlog;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/feeds/search',
        body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
      );

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchTextField).enterText(candidate.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
    await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();

    await t(AppDebugKey.postGroupCard(0)).tap();
    expect(
      t(AppDebugKey.attributeRow('div > time')).$('Reaches 23 of 24 posts'),
      findsOneWidget,
    );
    expect(
      t(AppDebugKey.attributeRow('div > h3')).$(RegExp('^Reaches')),
      findsNothing,
    );
  });

  patrolWidgetTest(
    'A text already chosen as the title is not offered as the description',
    (t) async {
      final candidate = fixture.feedCandidates.exampleBlog;
      final server = StubServer.withDefaultResponses()
        ..stubGet(
          '/feeds/search',
          body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
        );

      await pumpAppWithAuth(t, server);
      await t(AppDebugKey.feedsNavDestination).tap();
      await t(AppDebugKey.feedsScreen).waitUntilVisible();
      await t(AppDebugKey.addFeedButton).tap();
      await t(AppDebugKey.feedSearchTextField).enterText(candidate.url);
      await t(AppDebugKey.feedSearchButton).tap();
      await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
      await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();

      await t(AppDebugKey.postGroupCard(0)).tap();
      await t(AppDebugKey.attributeRow('div > h3')).tap();
      await t(AppDebugKey.attributeNextButton).tap();
      expect(t('Which row holds the description?'), findsOneWidget);
      expect(t(AppDebugKey.attributeRow('div > h3')), findsNothing);
      expect(t(AppDebugKey.attributeRow('div > time')), findsOneWidget);
    },
  );

  patrolWidgetTest('Report a rejected selector set', (t) async {
    final candidate = fixture.feedCandidates.exampleBlog;
    final server = StubServer.withDefaultResponses()
      ..stubPut(
        '/feeds',
        status: 400,
        body: api.Error(message: 'The link selector matches nothing.').toJson(),
      )
      ..stubGet(
        '/feeds/search',
        body: api.SearchFeeds200Response(feeds: [candidate]).toJson(),
      );

    await pumpAppWithAuth(t, server);
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchTextField).enterText(candidate.url);
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile(candidate.title)).tap();
    await t(AppDebugKey.feedSubscriptionScreen).waitUntilVisible();

    await t(AppDebugKey.postGroupCheckbox(0)).tap();
    await t(AppDebugKey.postGroupCard(0)).tap();
    await t(AppDebugKey.attributeRow('div > h3')).tap();
    await t(AppDebugKey.attributeBackButton).tap();
    await t(AppDebugKey.feedSubscriptionSubscribeButton).tap();

    await t('The link selector matches nothing.').waitUntilVisible();
    expect(t(AppDebugKey.feedSubscriptionScreen), findsOneWidget);
    expect(
      t.tester.widget<Checkbox>(t(AppDebugKey.postGroupCheckbox(0))).value,
      isTrue,
    );
    expect(
      t(AppDebugKey.postGroupCard(0)).$('Why we moved to a monorepo'),
      findsOneWidget,
    );
  });

  // Building the request body from several groups is a pure mapping, so it is
  // tested on the notifier directly instead of through a long run of taps.
  group('SubscriptionDraft request', () {
    test('Two ticked groups produce two entries in group order', () async {
      final dio = Dio(BaseOptions(baseUrl: 'http://mock'))
        ..interceptors.add(
          StubServer()..stubGet(
            '/feeds/search',
            body: api.SearchFeeds200Response(
              feeds: [fixture.feedCandidates.exampleBlog],
            ).toJson(),
          ),
        );
      final candidate = (await FeedRepositoryImpl(
        dio,
      ).search(fixture.feedCandidates.exampleBlog.url)).single;
      final container = ProviderContainer();
      addTearDown(container.dispose);
      final provider = subscriptionDraftProvider(candidate);
      container.listen(provider, (_, _) {});

      container.read(provider.notifier)
        ..toggle(2)
        ..toggle(0)
        ..select(0, PostAttribute.title, 'div > h3')
        ..select(2, PostAttribute.link, 'a.more');

      expect(
        container.read(provider.notifier).request().map((s) => s.toJson()),
        [
          {
            'root': 'main > article',
            'link': ':scope',
            'title': 'div > h3',
            'description': null,
            'image': null,
            'timestamp': null,
          },
          {
            'root': 'section.news > div',
            'link': 'a.more',
            'title': null,
            'description': null,
            'image': null,
            'timestamp': null,
          },
        ],
      );
    });

    test(
      'A group with no chosen attributes produces root and link only',
      () async {
        final dio = Dio(BaseOptions(baseUrl: 'http://mock'))
          ..interceptors.add(
            StubServer()..stubGet(
              '/feeds/search',
              body: api.SearchFeeds200Response(
                feeds: [fixture.feedCandidates.exampleBlog],
              ).toJson(),
            ),
          );
        final candidate = (await FeedRepositoryImpl(
          dio,
        ).search(fixture.feedCandidates.exampleBlog.url)).single;
        final container = ProviderContainer();
        addTearDown(container.dispose);
        final provider = subscriptionDraftProvider(candidate);
        container.listen(provider, (_, _) {});

        container.read(provider.notifier).toggle(2);

        expect(
          container.read(provider.notifier).request().map((s) => s.toJson()),
          [
            {
              'root': 'section.news > div',
              'link': 'a.headline',
              'title': null,
              'description': null,
              'image': null,
              'timestamp': null,
            },
          ],
        );
      },
    );

    test('An unticked group produces nothing', () async {
      final dio = Dio(BaseOptions(baseUrl: 'http://mock'))
        ..interceptors.add(
          StubServer()..stubGet(
            '/feeds/search',
            body: api.SearchFeeds200Response(
              feeds: [fixture.feedCandidates.exampleBlog],
            ).toJson(),
          ),
        );
      final candidate = (await FeedRepositoryImpl(
        dio,
      ).search(fixture.feedCandidates.exampleBlog.url)).single;
      final container = ProviderContainer();
      addTearDown(container.dispose);
      final provider = subscriptionDraftProvider(candidate);
      container.listen(provider, (_, _) {});

      container.read(provider.notifier)
        ..toggle(0)
        ..select(2, PostAttribute.title, 'a.headline')
        ..toggle(1);

      expect(
        container.read(provider.notifier).request().map((s) => s.toJson()),
        [
          {
            'root': 'main > article',
            'link': ':scope',
            'title': null,
            'description': null,
            'image': null,
            'timestamp': null,
          },
        ],
      );
    });
  });
}
