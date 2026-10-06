import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// Loads the original page at [url] in a WebView: the reader's "Web" tab.
///
/// Unlike `FeedEntryContentWebView`, JavaScript is enabled, because most
/// modern pages do not render without it. The WebView still stays on the
/// page it was opened for: redirects during the initial load (http → https,
/// tracking links) are followed, but any navigation after the page's document
/// has been parsed — a tapped link — opens in an in-app browser instead.
/// A floating button reloads the page.
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
  var _loading = true;
  var _initialLoadFinished = false;
  var _failed = false;
  Timer? _documentReadyPoll;

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
          onPageStarted: (_) {
            if (mounted) {
              setState(() => _loading = true);
            }
          },
          onPageFinished: (_) => _finishInitialLoad(),
          onWebResourceError: (error) {
            if (!mounted || !(error.isForMainFrame ?? false)) {
              return;
            }
            // iOS may end the web content process while the app is in the
            // background (e.g. while another app is open),
            // which leaves the WebView blank. Reload the page in that case,
            // unless the process ended during the load itself, where a
            // reload would likely end the same way.
            if (error.errorType ==
                    WebResourceErrorType.webContentProcessTerminated &&
                _initialLoadFinished) {
              _reload();
              return;
            }
            _documentReadyPoll?.cancel();
            setState(() {
              _loading = false;
              _failed = true;
            });
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
              unawaited(openInAppBrowserLink(context, url));
            }
            return NavigationDecision.prevent;
          },
        ),
      ),
    );
    unawaited(_controller.setJavaScriptMode(JavaScriptMode.unrestricted));
    unawaited(_load());
  }

  @override
  void dispose() {
    _documentReadyPoll?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    final uri = _uri;
    if (uri == null) {
      _failed = true;
      return;
    }
    // Marks the current document so that the poll below does not mistake it
    // for the new one: it stays in place until the new page starts arriving.
    try {
      await _controller.runJavaScript('window.__paperdollStale = true');
    } on Object {
      // No document to mark, e.g. after the web content process ended.
    }
    await _controller.loadRequest(uri);
    // onPageFinished waits for the page's load event, which pages with ads
    // may never fire, because ad frames keep loading. Therefore, the initial
    // load also ends when the new document has been parsed.
    _documentReadyPoll?.cancel();
    _documentReadyPoll = Timer.periodic(const Duration(milliseconds: 300), (
      _,
    ) async {
      try {
        final result = await _controller.runJavaScriptReturningResult(
          "location.href !== 'about:blank' && !window.__paperdollStale && "
          "document.readyState !== 'loading' ? 1 : 0",
        );
        // A number, because the platforms return booleans differently.
        if (result == 1) {
          _finishInitialLoad();
        }
      } on Object {
        // The document is being replaced; try again on the next tick.
      }
    });
  }

  void _finishInitialLoad() {
    _documentReadyPoll?.cancel();
    if (mounted) {
      setState(() {
        _loading = false;
        _initialLoadFinished = true;
      });
    }
  }

  /// Loads the page again from [OriginalPageWebView.url], following redirects
  /// as on the first load. This also works after a failed initial load, when
  /// the WebView has no page to reload.
  void _reload() {
    setState(() {
      _progress = 0;
      _loading = true;
      _initialLoadFinished = false;
      _failed = false;
    });
    unawaited(_load());
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
              FilledButton(onPressed: _reload, child: const Text('Retry')),
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
        if (_loading) LinearProgressIndicator(value: _progress / 100),
        Positioned(
          right: spacingMd,
          bottom: spacingMd,
          child: SafeArea(
            child: FloatingActionButton(
              key: AppDebugKey.readerWebReloadButton,
              // Disables the hero animation, which would otherwise move the
              // button between two readers during a route transition.
              heroTag: null,
              tooltip: 'Reload',
              onPressed: _reload,
              child: const Icon(Icons.refresh),
            ),
          ),
        ),
      ],
    );
  }
}
