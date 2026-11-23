import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/debug_keys.dart';
import 'package:patrol_finders/patrol_finders.dart';
import 'package:webview_flutter_platform_interface/webview_flutter_platform_interface.dart';

import '../src/boilerplate.dart';
import '../src/fake_webview.dart';
import '../src/fixture.dart';
import '../src/stub_server.dart';

void main() {
  patrolWidgetTest('Read an article extracted from the page', (t) async {
    final webView = await _openStory(
      t,
      fixture.stories.houthiStrikes,
      fixture.entries.houthiStrikes,
      reply: (id) => _message(id, 'ok', content: '<p>Extracted body</p>'),
    );
    expect(t(AppDebugKey.readerLoadingCover), findsOneWidget);

    await _finishHtml(t, webView);
    expect(t(AppDebugKey.readerLoadingCover), findsNothing);
    expect(webView.htmlLoads.single.html, contains('<p>Extracted body</p>'));
    expect(webView.htmlLoads.single.mode, JavaScriptMode.disabled);
    expect(t(AppDebugKey.readerReloadButton), findsNothing);
    expect(t(AppDebugKey.readerRetryButton), findsNothing);
  });

  patrolWidgetTest('Read the server content of a page without an article', (
    t,
  ) async {
    final entry = fixture.entries.nuclearDeal;
    final webView = await _openStory(
      t,
      fixture.stories.nuclearDeal,
      entry,
      reply: (id) => _message(id, 'empty'),
    );

    await _finishHtml(t, webView);
    expect(t(AppDebugKey.readerLoadingCover), findsNothing);
    expect(webView.htmlLoads.single.html, contains(entry.content));
    expect(t(AppDebugKey.readerReloadButton), findsNothing);
    expect(t(AppDebugKey.readerRetryButton), findsNothing);
  });

  patrolWidgetTest('Read the original page without an article or content', (
    t,
  ) async {
    final platform = await _pumpTodayWith(
      t,
      fixture.stories.houthiStrikes,
      fixture.entries.houthiStrikes,
    );
    platform.extractionReply = (id) => _message(id, 'empty');
    await t(AppDebugKey.storyCard(fixture.stories.houthiStrikes.title)).tap();
    final webView = platform.controllers.single;

    await pumpUntil(t.tester, () => t(AppDebugKey.readerReloadButton).exists);
    expect(t(AppDebugKey.readerLoadingCover), findsNothing);
    expect(webView.htmlLoads, isEmpty);

    // Reload covers the page again and repeats the extraction.
    platform.extractionReply = null;
    await t(AppDebugKey.readerReloadButton).tap();
    expect(t(AppDebugKey.readerLoadingCover), findsOneWidget);
    expect(webView.requests, hasLength(2));
  });

  patrolWidgetTest('Retry a page that failed to load', (t) async {
    final webView = await _openStory(
      t,
      fixture.stories.houthiStrikes,
      fixture.entries.houthiStrikes,
    );

    webView.failPage();
    await t.pump();
    expect(t(AppDebugKey.readerLoadingCover), findsNothing);
    expect(t("Couldn't load this page."), findsOneWidget);

    await t(AppDebugKey.readerRetryButton).tap();
    expect(t(AppDebugKey.readerLoadingCover), findsOneWidget);
    expect(webView.requests, hasLength(2));
  });

  patrolWidgetTest('Keep the page hidden when it navigates by itself', (
    t,
  ) async {
    final webView = await _openStory(
      t,
      fixture.stories.houthiStrikes,
      fixture.entries.houthiStrikes,
    );

    expect(
      await webView.navigate('https://ads.example/'),
      NavigationDecision.prevent,
    );
    await t.pump();
    expect(t(AppDebugKey.readerLoadingCover), findsOneWidget);
  });
}

Future<FakeWebViewPlatform> _pumpTodayWith(
  PatrolTester t,
  api.Story story,
  api.FeedEntry entry,
) {
  final server = StubServer.withDefaultResponses()
    ..stubGet(
      '/newspapers/today',
      body: api.GetTodaysNewspaper200Response(
        id: 1,
        publishedAt: DateTime.utc(2026, 7, 1),
        stories: [story],
      ).toJson(),
    )
    ..stubGet('/feed-entries/${entry.id}', body: entry.toJson());
  return pumpAppWithAuth(t, server);
}

/// Opens [story] in the reader, with [reply] as the page's reply to the
/// extraction, and returns the reader's WebView. Opening it lets several
/// seconds of fake time pass, while the page loads and settles.
Future<FakeWebViewController> _openStory(
  PatrolTester t,
  api.Story story,
  api.FeedEntry entry, {
  String? Function(String messageId)? reply,
}) async {
  final platform = await _pumpTodayWith(t, story, entry);
  platform.extractionReply = reply;
  await t(AppDebugKey.storyCard(story.title)).tap();
  await t(AppDebugKey.feedEntryReaderScreen).waitUntilVisible();
  return platform.controllers.single;
}

/// Waits for the reader to load an HTML string, and finishes loading it.
Future<void> _finishHtml(PatrolTester t, FakeWebViewController webView) async {
  await pumpUntil(t.tester, () => webView.htmlLoads.isNotEmpty);
  webView.finishPage('about:blank');
  await t.pump();
}

String _message(String id, String status, {String? content}) => jsonEncode({
  'v': 1,
  'messageId': id,
  'status': status,
  'url': 'https://www.bbc.co.uk/news/articles/1',
  'title': null,
  'byline': null,
  'siteName': null,
  'content': content,
  'error': null,
});
