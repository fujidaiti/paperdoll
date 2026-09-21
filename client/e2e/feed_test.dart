import 'package:flutter/widgets.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:patrol/patrol.dart';

import 'helper.dart';

void main() {
  // Initialize the Flutter binding up front so the host-facing socket is ready
  // before the first request; without it the connection can abort transiently
  // right after a test's app relaunch ("Software caused connection abort").
  WidgetsFlutterBinding.ensureInitialized();

  patrolTest('Check feeds and read a feed entry', tags: 'feed-read', (t) async {
    await setUpServer(seederId: 'feed_bbc_news');
    await pumpAppWithAuth(t);

    const targetEntryTitle = 'US signs landmark nuclear deal with Saudi Arabia';
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.feedRow('BBC News')).tap();
    await t(AppDebugKey.feedDetailScreen).waitUntilVisible();
    await t(AppDebugKey.feedEntryRow(targetEntryTitle)).tap();
    await t(AppDebugKey.feedEntryReaderScreen).waitUntilVisible();
    await t(AppDebugKey.readerTitle(targetEntryTitle)).waitUntilVisible();
  });

  patrolTest('Unsubscribe from a feed', tags: 'feed-unsubscribe', (t) async {
    await setUpServer(seederId: 'feed_bbc_news');
    await pumpAppWithAuth(t);

    const entryTitle = 'US signs landmark nuclear deal with Saudi Arabia';
    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.feedRow('BBC News')).tap();
    await t(AppDebugKey.feedDetailScreen).waitUntilVisible();

    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.unsubscribeMenuItem).tap();
    await t(AppDebugKey.unsubscribeSuccessSnackBar).waitUntilVisible();

    // The screen stays put: the feed is shared across all users, so it is not
    // deleted and its timeline is still readable. The menu now offers the
    // opposite action.
    await t(AppDebugKey.feedEntryRow(entryTitle)).waitUntilVisible();
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.subscribeMenuItem).waitUntilVisible();
    // The open menu covers the app bar's back button, so close it by tapping
    // outside it before leaving the screen.
    await t.tester.tapAt(const Offset(10, 10));
    await t.pumpAndSettle();
    await t.tester.pageBack();

    // The subscription really went away on the server. The list keeps its
    // cached page, so refresh it to see the feed disappear.
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t.tester.fling(
      t(AppDebugKey.feedsScreen).finder,
      const Offset(0, 300),
      1000,
    );
    await t('No feeds yet. Add one to get started.').waitUntilVisible();
  });

  patrolTest('Subscribe to a known web feed', tags: 'feed-subscribe', (
    t,
  ) async {
    await setUpServer(seederId: 'feed_nasa_candidate');
    await pumpAppWithAuth(t);

    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchScreen).waitUntilVisible();

    await t(AppDebugKey.feedSearchTextField)
        .enterText('http://www.nasa.gov/news-release/feed/');
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile('NASA')).tap();
    await t(AppDebugKey.subscribeSuccessSnackBar).waitUntilVisible();
    await t(AppDebugKey.feedRow('NASA')).waitUntilVisible();
  });
}
