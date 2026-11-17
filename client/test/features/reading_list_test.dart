import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/debug_keys.dart';
import 'package:patrol_finders/patrol_finders.dart';

import '../src/boilerplate.dart';
import '../src/fixture.dart';
import '../src/stub_server.dart';

void main() {
  patrolWidgetTest('Read a saved web clip in reading list', (t) async {
    final item = fixture.readingList.buildingEffectiveAgents;
    final webClip = fixture.webClips.buildingEffectiveAgents;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/reading-list',
        body: api.GetReadingList200Response(items: [item]).toJson(),
      )
      ..stubGet('/web-clips/${webClip.id}', body: webClip.toJson());
    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.readingListNavDestination).tap();
    expect(t(AppDebugKey.readingListScreen), findsOneWidget);
    await t(AppDebugKey.readingListRow(item.title)).tap();
    expect(t(AppDebugKey.webClipReaderScreen), findsOneWidget);
    expect(t(AppDebugKey.readerTitle(item.title)), findsOneWidget);
    expect(
      t.tester.getSemantics(find.byKey(AppDebugKey.readerArticleTab)),
      isSemantics(isSelected: true),
    );
  });

  patrolWidgetTest('Read a saved feed entry in reading list', (t) async {
    final item = fixture.readingList.nuclearDeal;
    final entry = fixture.entries.nuclearDeal;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/reading-list',
        body: api.GetReadingList200Response(items: [item]).toJson(),
      )
      ..stubGet('/feed-entries/${entry.id}', body: entry.toJson());
    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.readingListNavDestination).tap();
    expect(t(AppDebugKey.readingListScreen), findsOneWidget);
    await t(AppDebugKey.readingListRow(item.title)).tap();
    expect(t(AppDebugKey.feedEntryReaderScreen), findsOneWidget);
    expect(t(AppDebugKey.readerTitle(item.title)), findsOneWidget);
    expect(
      t.tester.getSemantics(find.byKey(AppDebugKey.readerArticleTab)),
      isSemantics(isSelected: true),
    );
  });

  patrolWidgetTest('Read a web clip without content on the web', (t) async {
    final item = fixture.readingList.claudeCodeBestPractices;
    final webClip = fixture.webClips.claudeCodeBestPractices;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/reading-list',
        body: api.GetReadingList200Response(items: [item]).toJson(),
      )
      ..stubGet('/web-clips/${webClip.id}', body: webClip.toJson());
    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.readingListNavDestination).tap();
    await t(AppDebugKey.readingListRow(item.title)).tap();
    expect(t(AppDebugKey.webClipReaderScreen), findsOneWidget);
    expect(
      t.tester.getSemantics(find.byKey(AppDebugKey.readerWebTab)),
      isSemantics(isSelected: true),
    );

    await t(AppDebugKey.readerArticleTab).tap();
    expect(t("This clip's content isn't available."), findsOneWidget);
  });

  patrolWidgetTest('Read a saved feed entry on the web', (t) async {
    final item = fixture.readingList.nuclearDeal;
    final entry = fixture.entries.nuclearDeal;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/reading-list',
        body: api.GetReadingList200Response(items: [item]).toJson(),
      )
      ..stubGet('/feed-entries/${entry.id}', body: entry.toJson());
    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.readingListNavDestination).tap();
    await t(AppDebugKey.readingListRow(item.title)).tap();
    expect(t(AppDebugKey.feedEntryReaderScreen), findsOneWidget);
    expect(
      t.tester.getSemantics(find.byKey(AppDebugKey.readerArticleTab)),
      isSemantics(isSelected: true),
    );

    await t(AppDebugKey.readerWebTab).tap();
    expect(
      t.tester.getSemantics(find.byKey(AppDebugKey.readerWebTab)),
      isSemantics(isSelected: true),
    );
  });

  patrolWidgetTest('Archive a reading list item by swiping', (t) async {
    final item = fixture.readingList.buildingEffectiveAgents;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/reading-list',
        body: api.GetReadingList200Response(items: [item]).toJson(),
      )
      ..stubPatch(
        '/reading-list/${item.id}',
        status: 204,
        body: <String, dynamic>{},
        bodyMatcher: api.SetReadingListItemArchivedStatusRequest(archived: true)
            .toJson(),
      );
    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.readingListNavDestination).tap();
    expect(t(AppDebugKey.readingListRow(item.title)), findsOneWidget);
    await t.tester.fling(
      t(AppDebugKey.readingListRow(item.title)).finder,
      const Offset(100, 0),
      1000,
    );
    await t(AppDebugKey.archiveSuccessSnackBar).waitUntilVisible();
    expect(t(AppDebugKey.readingListRow(item.title)), findsNothing);
  });

  patrolWidgetTest('Read an archived item from the archived screen', (t) async {
    final item = fixture.readingList.buildingEffectiveAgents;
    final webClip = fixture.webClips.buildingEffectiveAgents;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/reading-list',
        body: api.GetReadingList200Response(items: []).toJson(),
      )
      ..stubGet(
        '/reading-list/archived',
        body: api.GetReadingList200Response(items: [item]).toJson(),
      )
      ..stubGet('/web-clips/${webClip.id}', body: webClip.toJson());
    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.readingListNavDestination).tap();
    expect(t(AppDebugKey.readingListScreen), findsOneWidget);
    await t(AppDebugKey.archivedButton).tap();
    expect(t(AppDebugKey.archivedReadingListScreen), findsOneWidget);
    await t(AppDebugKey.readingListRow(item.title)).tap();
    expect(t(AppDebugKey.webClipReaderScreen), findsOneWidget);
    expect(t(AppDebugKey.readerTitle(item.title)), findsOneWidget);
  });

  patrolWidgetTest('Read an archived feed entry from the archived screen', (
    t,
  ) async {
    final item = fixture.readingList.nuclearDeal;
    final entry = fixture.entries.nuclearDeal;
    final server = StubServer.withDefaultResponses()
      ..stubGet(
        '/reading-list',
        body: api.GetReadingList200Response(items: []).toJson(),
      )
      ..stubGet(
        '/reading-list/archived',
        body: api.GetReadingList200Response(items: [item]).toJson(),
      )
      ..stubGet('/feed-entries/${entry.id}', body: entry.toJson());
    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.readingListNavDestination).tap();
    expect(t(AppDebugKey.readingListScreen), findsOneWidget);
    await t(AppDebugKey.archivedButton).tap();
    expect(t(AppDebugKey.archivedReadingListScreen), findsOneWidget);
    await t(AppDebugKey.readingListRow(item.title)).tap();
    expect(t(AppDebugKey.feedEntryReaderScreen), findsOneWidget);
    expect(t(AppDebugKey.readerTitle(item.title)), findsOneWidget);
  });

  patrolWidgetTest('Save a feed entry to the reading list in reader', (
    t,
  ) async {
    final feed = fixture.feeds.bbcNews;
    final entry = fixture.entries.nuclearDeal;

    final server = StubServer.withDefaultResponses()
      ..stubGet('/feeds', body: api.GetFeeds200Response(feeds: [feed]).toJson())
      ..stubGet('/feeds/${feed.id}', body: feed.toJson())
      ..stubGet(
        '/feeds/${feed.id}/timeline',
        body: api.GetFeedTimeline200Response(entries: [entry]).toJson(),
      )
      ..stubGet('/feed-entries/${entry.id}', body: entry.toJson())
      ..stubPost(
        '/reading-list',
        status: 201,
        body: api.ReadingListItem(
          id: 42,
          resourceId: entry.id,
          kind: api.ReadingListItemKindEnum.feedEntry,
          title: entry.title,
          savedAt: DateTime.utc(2026, 7, 1),
        ).toJson(),
        bodyMatcher: api.SaveToReadingListRequestOneOf1(feedEntryId: entry.id)
            .toJson(),
      );
    await pumpAppWithAuth(t, server);

    await t(AppDebugKey.feedsNavDestination).tap();
    expect(t(AppDebugKey.feedsScreen), findsOneWidget);
    await t(AppDebugKey.feedRow(feed.title)).tap();
    expect(t(AppDebugKey.feedDetailScreen), findsOneWidget);
    await t(AppDebugKey.feedEntryRow(entry.title)).tap();
    expect(t(AppDebugKey.feedEntryReaderScreen), findsOneWidget);
    expect(find.byIcon(Icons.bookmark_border), findsOneWidget);
    await t(AppDebugKey.feedEntryReaderBookmarkButton).tap();
    expect(find.byIcon(Icons.bookmark), findsOneWidget);
    expect(
      find.byKey(AppDebugKey.saveToReadingListSuccessSnackBar),
      findsOneWidget,
    );
  });
}
