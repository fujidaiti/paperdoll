import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed_entry/presentation/widgets/original_page_webview.dart';

/// Reader body shared by the Story, Feed Entry, and Web Clip Readers: an
/// "Article" tab with the parsed content and a "Web" tab with the original
/// page at [url], switched by a tab bar at the top.
///
/// `TabBarView` builds a page only once it becomes visible, so the inactive
/// tab loads nothing until the user opens it. Swiping between tabs is disabled
/// because horizontal swipes belong to the page content (carousels, wide
/// tables, code blocks).
class const ReaderTabView({
  required final Widget article,
  required final String url,

  /// Whether to open on the "Web" tab, for items without parsed content.
  required final bool startOnWeb,

  /// Shown between the tab bar and the pages, e.g. the archived banner.
  final Widget? banner,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 2,
      initialIndex: startOnWeb ? 1 : 0,
      child: Column(
        children: [
          const TabBar(
            tabs: [
              Tab(key: AppDebugKey.readerArticleTab, text: 'Article'),
              Tab(key: AppDebugKey.readerWebTab, text: 'Web'),
            ],
          ),
          ?banner,
          Expanded(
            child: TabBarView(
              physics: const NeverScrollableScrollPhysics(),
              children: [
                article,
                OriginalPageWebView(url: url),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
