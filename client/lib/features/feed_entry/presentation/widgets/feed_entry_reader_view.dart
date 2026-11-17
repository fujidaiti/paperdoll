import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/archived_banner.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed_entry/domain/feed_entry.dart';
import 'package:paperdoll/features/feed_entry/presentation/widgets/feed_entry_content_webview.dart';
import 'package:paperdoll/features/feed_entry/presentation/widgets/reader_tab_view.dart';

/// Shared reading layout for both the Story Reader and the Feed Entry Reader:
/// the entry's rendered HTML content in the "Article" tab (or its description
/// when there is none) and the original page in the "Web" tab. Opens on the
/// "Web" tab when there is no content.
class const FeedEntryReaderView({required final FeedEntry entry, super.key})
    extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final content = entry.content;
    final hasContent = content != null && content.trim().isNotEmpty;
    return ReaderTabView(
      url: entry.url,
      startOnWeb: !hasContent,
      banner: (entry.archived ?? false)
          ? const ArchivedBanner(key: AppDebugKey.feedEntryReaderArchivedBanner)
          : null,
      article: hasContent
          ? FeedEntryContentWebView(html: content)
          : Padding(
              padding: const EdgeInsets.all(spacingMd),
              child: BodyText(entry.description ?? 'No content available.'),
            ),
    );
  }
}
