import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/features/reader/presentation/reader_page_controller.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// Shows the original page at [url] in the reader WebView, which
/// [ReaderPageController] loads: the reader's "Web" tab. Links open in an
/// in-app browser, and a floating button reloads the page.
///
/// Kept alive so switching reader tabs does not reload the page.
class const ReaderWebView({required final String url, super.key})
    extends StatefulWidget {
  @override
  State<ReaderWebView> createState() => _ReaderWebViewState();
}

class _ReaderWebViewState extends State<ReaderWebView>
    with AutomaticKeepAliveClientMixin {
  late final _controller = ReaderPageController(
    url: widget.url,
    openLink: (url) => unawaited(openInAppBrowserLink(context, url)),
  );

  @override
  bool get wantKeepAlive => true;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return ListenableBuilder(
      listenable: _controller,
      builder: (context, _) {
        if (_controller.failed) {
          return Center(
            child: Padding(
              padding: const EdgeInsets.all(spacingLg),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const BodyText("Couldn't load this page."),
                  const Gap(spacingMd),
                  FilledButton(
                    onPressed: _controller.reload,
                    child: const Text('Retry'),
                  ),
                  const Gap(spacingSm),
                  TextButton(
                    onPressed: () =>
                        unawaited(openExternalLink(context, widget.url)),
                    child: const Text('Open in browser'),
                  ),
                ],
              ),
            ),
          );
        }
        return Stack(
          children: [
            WebViewWidget(controller: _controller.webViewController),
            if (_controller.loading)
              LinearProgressIndicator(value: _controller.progress / 100),
            Positioned(
              right: spacingMd,
              bottom: spacingMd,
              child: SafeArea(
                child: FloatingActionButton(
                  // Disables the hero animation, which would otherwise move
                  // the button between two readers during a route transition.
                  heroTag: null,
                  tooltip: 'Reload',
                  onPressed: _controller.reload,
                  child: const Icon(Icons.refresh),
                ),
              ),
            ),
          ],
        );
      },
    );
  }
}
