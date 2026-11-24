import 'package:flutter/widgets.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:patrol/patrol.dart';

import 'helper.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  patrolTest(
    'Read an article extracted from the page',
    tags: 'reader-article',
    ($) async {
      await setUpServer(seederId: 'reader_article');
      await pumpAppWithAuth($);

      await $(AppDebugKey.readingListNavDestination).tap();
      await $(AppDebugKey.readingListScreen).waitUntilVisible();
      await $(AppDebugKey.readingListRow('How Tide Pools Survive the Summer'))
          .tap();
      await $(AppDebugKey.feedEntryReaderScreen).waitUntilVisible();
      // The byline comes from the page's metadata, which only the extracted
      // article shows, and the body text is in the WebView.
      await $.platform.mobile.waitUntilVisible(
        Selector(text: 'Mina Sato'),
        timeout: const Duration(seconds: 20),
      );
      await $.platform.mobile.waitUntilVisible(
        Selector(textContains: 'Anemones close up'),
      );
    },
  );
}
