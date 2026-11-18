import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
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

    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.feedRow('BBC News')).tap();
    await t(AppDebugKey.feedDetailScreen).waitUntilVisible();
    await t(
      AppDebugKey.feedEntryRow(
        'US signs landmark nuclear deal with Saudi Arabia',
      ),
    ).waitUntilVisible();
    await t(AppDebugKey.feedDetailMenuButton).tap();
    await t(AppDebugKey.unsubscribeMenuItem).tap();
    await t(AppDebugKey.unsubscribeSuccessSnackBar).waitUntilVisible();

    await t.tester.pageBack();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    // Refresh the page and see if the feed disappears from the list.
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

  patrolTest('Build the post selectors of a web page', tags: 'feed-selectors', (
    t,
  ) async {
    // The number of posts in posts.fixture.json of the page, in
    // server/feature/feed/testdata/webpage/blog.codinghorror.com. The marks
    // below are the ones in its marks.fixture.json.
    const postCount = 3;

    await setUpServer(seederId: 'feed_web_page');
    await pumpAppWithAuth(t);

    // Widens the selection to the third element around the tapped line, which
    // is the article on the editor and its content on the fields screen.
    Future<void> selectThirdLevel() async {
      final slider = t.tester.getRect(find.byType(Slider));
      final lastLevel = t.tester.widget<Slider>(find.byType(Slider)).max;
      await t.tester.tapAt(
        Offset(slider.left + slider.width * 2 / lastLevel, slider.center.dy),
      );
      await t.pumpAndSettle();
    }

    await t(AppDebugKey.feedsNavDestination).tap();
    await t(AppDebugKey.feedsScreen).waitUntilVisible();
    await t(AppDebugKey.addFeedButton).tap();
    await t(AppDebugKey.feedSearchScreen).waitUntilVisible();
    await t(AppDebugKey.feedSearchTextField)
        .enterText('http://blog.codinghorror.com/');
    await t(AppDebugKey.feedSearchButton).tap();
    await t(AppDebugKey.feedCandidateTile('Coding Horror')).tap();
    await t(AppDebugKey.selectorEditorScreen).waitUntilVisible();

    // The app guesses the title, the link and the pub date of the post.
    await t(RegExp('Vibe Code')).tap();
    await selectThirdLevel();
    await t('Mark as post').tap();

    // The body is the only field that the app does not guess.
    await t(RegExp('^About a year ago')).scrollTo().tap();
    await selectThirdLevel();
    await t(find.widgetWithText(FilterChip, 'Body')).tap();
    await t('Close').tap();
    await t('Done').tap();

    await t('$postCount post(s) found.').waitUntilVisible();
    await t(AppDebugKey.saveSelectorsButton).tap();
    await t(AppDebugKey.saveSelectorsSuccessSnackBar).waitUntilVisible();

    await t(AppDebugKey.feedRow('Coding Horror')).tap();
    await t(AppDebugKey.feedDetailScreen).waitUntilVisible();

    // The timeline builds only the rows on screen, so the rows are collected
    // while scrolling to its end.
    final timeline = find
        .descendant(
          of: t(AppDebugKey.feedDetailScreen).finder,
          matching: find.byType(Scrollable),
        )
        .first;
    final entryRows = find.byWidgetPredicate(
      (widget) => switch (widget.key) {
        ValueKey<String>(:final value) => value.startsWith('feedEntry:'),
        _ => false,
      },
    );
    final seen = <Key>{};
    while (true) {
      // A page that is still loading extends the list after this wait.
      await t.pump(const Duration(seconds: 1));
      seen.addAll(t.tester.widgetList(entryRows).map((row) => row.key!));
      final position = t.tester.state<ScrollableState>(timeline).position;
      final before = position.pixels;
      await t.tester.drag(timeline, const Offset(0, -400));
      await t.pumpAndSettle();
      if (position.pixels == before) {
        break;
      }
    }
    expect(seen, hasLength(postCount));
  });
}
