import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// Loads the original page at [url] in a WebView: the reader's "Web" tab.
///
/// Unlike `FeedEntryContentWebView`, JavaScript is enabled, because most
/// modern pages do not render without it. The WebView still stays on the
/// page it was opened for: redirects during the initial load (http → https,
/// tracking links) are followed, but any later navigation — a tapped link —
/// opens in the external browser instead.
///
/// Kept alive so switching reader tabs does not reload the page.
class const OriginalPageWebView({required final String url, super.key})
    extends StatefulWidget {
  @override
  State<OriginalPageWebView> createState() => _OriginalPageWebViewState();
}

class _OriginalPageWebViewState extends State<OriginalPageWebView>
    with AutomaticKeepAliveClientMixin {
  late final WebViewController _controller;
  late final Uri? _uri;
  var _progress = 0;
  var _initialLoadFinished = false;
  var _failed = false;

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    final uri = Uri.tryParse(widget.url);
    _uri = uri != null && (uri.isScheme('http') || uri.isScheme('https'))
        ? uri
        : null;
    _controller = WebViewController();
    unawaited(
      _controller.setNavigationDelegate(
        NavigationDelegate(
          onProgress: (progress) {
            if (mounted) {
              setState(() => _progress = progress);
            }
          },
          onPageFinished: (_) => _initialLoadFinished = true,
          onWebResourceError: (error) {
            if (mounted && (error.isForMainFrame ?? false)) {
              setState(() => _failed = true);
            }
          },
          onNavigationRequest: (request) {
            if (!request.isMainFrame) {
              return NavigationDecision.navigate;
            }
            final url = request.url;
            final isWeb =
                url.startsWith('http://') || url.startsWith('https://');
            if (!_initialLoadFinished && isWeb) {
              return NavigationDecision.navigate;
            }
            if (mounted) {
              unawaited(openExternalLink(context, url));
            }
            return NavigationDecision.prevent;
          },
        ),
      ),
    );
    unawaited(_controller.setJavaScriptMode(JavaScriptMode.unrestricted));
    _load();
  }

  void _load() {
    final uri = _uri;
    if (uri == null) {
      _failed = true;
      return;
    }
    unawaited(_controller.loadRequest(uri));
  }

  void _retry() {
    setState(() {
      _progress = 0;
      _initialLoadFinished = false;
      _failed = false;
      _load();
    });
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    if (_failed) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(spacingLg),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const BodyText("Couldn't load this page."),
              const Gap(spacingMd),
              FilledButton(onPressed: _retry, child: const Text('Retry')),
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
        WebViewWidget(controller: _controller),
        if (_progress < 100) LinearProgressIndicator(value: _progress / 100),
      ],
    );
  }
}
