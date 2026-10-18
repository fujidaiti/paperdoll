import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/widgets/archived_banner.dart';
import 'package:paperdoll/core/ui/widgets/empty_placeholder.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/feed_entry/presentation/widgets/feed_entry_content_webview.dart';
import 'package:paperdoll/features/feed_entry/presentation/widgets/reader_tab_view.dart';
import 'package:paperdoll/features/web_clip/domain/web_clip.dart';

/// Renders a saved web clip: its fetched HTML content in the "Article" tab (or
/// a placeholder that points to the original when the content hasn't been
/// fetched) and the original page in the "Web" tab. Opens on the "Web" tab
/// when there is no content.
class const WebClipReaderView({required final WebClip clip, super.key})
    extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final content = clip.content;
    final hasContent = content != null && content.trim().isNotEmpty;
    return ReaderTabView(
      url: clip.url,
      startOnWeb: !hasContent,
      banner: (clip.archived ?? false)
          ? const ArchivedBanner(key: AppDebugKey.webClipReaderArchivedBanner)
          : null,
      article: hasContent
          ? FeedEntryContentWebView(html: content)
          : EmptyPlaceholder(
              message: "This clip's content isn't available.",
              actionLabel: 'Open in browser',
              onAction: () => unawaited(openExternalLink(context, clip.url)),
            ),
    );
  }
}
