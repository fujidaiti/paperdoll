import 'package:flutter_test/flutter_test.dart';
import 'package:openapi/api.dart' as api;
import 'package:paperdoll/debug_keys.dart';
import 'package:patrol_finders/patrol_finders.dart';

import '../src/boilerplate.dart';
import '../src/fixture.dart';
import '../src/stub_server.dart';

void main() {
  patrolWidgetTest("Check today's issue and read a story", (t) async {
    final story = fixture.stories.nuclearDeal;
    final entry = fixture.entries.nuclearDeal;

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
    await pumpAppWithAuth(t, server);

    expect(t(AppDebugKey.todayScreen), findsOneWidget);
    await t(AppDebugKey.storyCard(story.title)).tap();
    await t(AppDebugKey.feedEntryReaderScreen).waitUntilVisible();
    await t(AppDebugKey.readerTitle(entry.title)).waitUntilVisible();
  });
}
