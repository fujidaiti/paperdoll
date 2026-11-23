import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// Owns the reader WebView and loads the original page at [url] into it.
///
/// JavaScript is enabled, because most modern pages do not render without it.
/// The WebView still stays on the page it was opened for: redirects during the
/// initial load (http → https, tracking links) are followed, but any
/// navigation after the page's document has been parsed — a tapped link — is
/// passed to [openLink] instead.
class ReaderPageController({
  required final String url,
  required final void Function(String url) openLink,
}) extends ChangeNotifier {
  this {
    unawaited(
      webViewController.setNavigationDelegate(
        NavigationDelegate(
          onProgress: (progress) {
            _progress = progress;
            _notify();
          },
          onPageStarted: (_) {
            _loading = true;
            _notify();
          },
          onPageFinished: (_) => _finishInitialLoad(),
          onWebResourceError: _handleError,
          onNavigationRequest: _decideNavigation,
        ),
      ),
    );
    unawaited(webViewController.setJavaScriptMode(JavaScriptMode.unrestricted));
    unawaited(_load());
  }

  final webViewController = WebViewController();

  /// The load progress of the page, from 0 to 100.
  int get progress => _progress;
  var _progress = 0;

  /// Whether the page is loading.
  bool get loading => _loading;
  var _loading = true;

  /// Whether the page failed to load, or [url] is not a web URL.
  bool get failed => _failed;
  var _failed = false;

  late final Uri? _uri = switch (Uri.tryParse(url)) {
    final uri? when uri.isScheme('http') || uri.isScheme('https') => uri,
    _ => null,
  };
  var _initialLoadFinished = false;
  var _disposed = false;
  Timer? _documentReadyPoll;

  @override
  void dispose() {
    _disposed = true;
    _documentReadyPoll?.cancel();
    super.dispose();
  }

  /// Loads the page again from [url], following redirects as on the first
  /// load. This also works after a failed initial load, when the WebView has
  /// no page to reload.
  void reload() {
    _progress = 0;
    _loading = true;
    _initialLoadFinished = false;
    _failed = false;
    _notify();
    unawaited(_load());
  }

  Future<void> _load() async {
    final uri = _uri;
    if (uri == null) {
      _failed = true;
      _notify();
      return;
    }
    // Marks the current document so that the poll below does not mistake it
    // for the new one: it stays in place until the new page starts arriving.
    try {
      await webViewController.runJavaScript('window.__paperdollStale = true');
    } on Object {
      // No document to mark, e.g. after the web content process ended.
    }
    await webViewController.loadRequest(uri);
    // onPageFinished waits for the page's load event, which pages with ads
    // may never fire, because ad frames keep loading. Therefore, the initial
    // load also ends when the new document has been parsed.
    _documentReadyPoll?.cancel();
    _documentReadyPoll = Timer.periodic(const Duration(milliseconds: 300), (
      _,
    ) async {
      try {
        final result = await webViewController.runJavaScriptReturningResult(
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
    _loading = false;
    _initialLoadFinished = true;
    _notify();
  }

  void _handleError(WebResourceError error) {
    if (!(error.isForMainFrame ?? false)) {
      return;
    }
    // iOS may end the web content process while the app is in the background
    // (e.g. while another app is open), which leaves the WebView blank.
    // Reload the page in that case, unless the process ended during the load
    // itself, where a reload would likely end the same way.
    if (error.errorType == WebResourceErrorType.webContentProcessTerminated &&
        _initialLoadFinished) {
      reload();
      return;
    }
    _documentReadyPoll?.cancel();
    _loading = false;
    _failed = true;
    _notify();
  }

  NavigationDecision _decideNavigation(NavigationRequest request) {
    if (!request.isMainFrame) {
      return NavigationDecision.navigate;
    }
    final url = request.url;
    final isWeb = url.startsWith('http://') || url.startsWith('https://');
    if (!_initialLoadFinished && isWeb) {
      return NavigationDecision.navigate;
    }
    if (!_disposed) {
      openLink(url);
    }
    return NavigationDecision.prevent;
  }

  void _notify() {
    if (!_disposed) {
      notifyListeners();
    }
  }
}
