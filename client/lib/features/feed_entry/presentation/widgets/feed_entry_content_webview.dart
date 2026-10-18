import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// Renders an entry's HTML `content` in a WebView. The controller is built
/// once in initState (not in createState) so it survives rebuilds, and the
/// state is kept alive so switching reader tabs does not reload the content.
class const FeedEntryContentWebView({required final String html, super.key})
    extends StatefulWidget {
  @override
  State<FeedEntryContentWebView> createState() =>
      _FeedEntryContentWebViewState();
}

class _FeedEntryContentWebViewState extends State<FeedEntryContentWebView>
    with AutomaticKeepAliveClientMixin {
  late final WebViewController _controller;
  var _isLoading = true;

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController();
    unawaited(
      _controller.setNavigationDelegate(
        NavigationDelegate(
          onPageFinished: (_) {
            if (mounted) {
              setState(() => _isLoading = false);
            }
          },
          // iOS may end the web content process while the app is in the
          // background (e.g. while another app is open),
          // which leaves the WebView blank. Render the content again.
          onWebResourceError: (error) {
            if (error.errorType ==
                WebResourceErrorType.webContentProcessTerminated) {
              unawaited(_controller.loadHtmlString(_document(widget.html)));
            }
          },
          // Keep the WebView pinned to the rendered content: any link the
          // user taps opens in an in-app browser instead of navigating away
          // inside the WebView.
          onNavigationRequest: (request) {
            final url = request.url;
            if (url.startsWith('http://') || url.startsWith('https://')) {
              if (mounted) {
                unawaited(openInAppBrowserLink(context, url));
              }
              return NavigationDecision.prevent;
            }
            return NavigationDecision.navigate;
          },
        ),
      ),
    );
    unawaited(_controller.setJavaScriptMode(JavaScriptMode.disabled));
    unawaited(_controller.loadHtmlString(_document(widget.html)));
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return Stack(
      children: [
        WebViewWidget(controller: _controller),
        // Opaque scrim covering the WebView until the page finishes loading,
        // so the empty WebView's dark default background never flickers.
        Positioned.fill(
          child: IgnorePointer(
            ignoring: !_isLoading,
            child: AnimatedOpacity(
              opacity: _isLoading ? 1 : 0,
              duration: const Duration(milliseconds: 150),
              child: ColoredBox(color: Theme.of(context).colorScheme.surface),
            ),
          ),
        ),
      ],
    );
  }

  String _document(String content) {
    return '''
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body {
  font-family: -apple-system, system-ui, sans-serif;
  font-size: 16px;
  line-height: 1.6;
  margin: 0;
  padding: 16px;
  color: #1a1a1a;
}
img, video, iframe { max-width: 100%; height: auto; }
pre { white-space: pre-wrap; word-wrap: break-word; }
a { color: #1565c0; }
</style>
</head>
<body>$content</body>
</html>
''';
  }
}
