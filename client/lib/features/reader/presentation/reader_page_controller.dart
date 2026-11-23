import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:paperdoll/core/logging/app_logger.dart';
import 'package:paperdoll/features/reader/presentation/article_html.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// What the reader shows.
enum ReaderPhase {
  /// The loading cover over the WebView, while the original page loads and
  /// the article is extracted from it.
  loading,

  /// The article extracted from the original page, with JavaScript disabled.
  article,

  /// The item's content from the server, with JavaScript disabled.
  serverContent,

  /// The original page as it loaded, with JavaScript enabled.
  page,

  /// An error message, because the original page failed to load.
  pageError,
}

/// Owns the reader WebView and decides what it shows for the item at [url].
///
/// The WebView first loads the original page behind the loading cover, with
/// JavaScript enabled. Once the page's document is ready and its text has
/// stopped changing, the controller injects `assets/reader/extract.js`, which
/// runs Readability on a copy of the document and posts one JSON message to
/// the `PaperdollReader` channel:
///
/// - `v`: the contract version, `1`.
/// - `messageId`: the ID passed to `extract()`, so that a late reply to an
///   earlier extraction is dropped.
/// - `status`: `ok` (an article was found), `empty` (no article), or `error`
///   (the script threw).
/// - `url`: `document.URL`, used as the article's base URL.
/// - `title`, `byline`, `siteName`: Readability's metadata; null unless
///   `status` is `ok`.
/// - `content`: the article HTML; non-null exactly when `status` is `ok`.
/// - `error`: the exception message; non-null exactly when `status` is
///   `error`.
///
/// The script shares the JavaScript context with the page's own scripts,
/// which can rewrite anything in the message. Therefore, the content is
/// sanitized and the metadata escaped (see [renderArticle]) before the
/// article replaces the page in the same WebView, with JavaScript disabled.
///
/// Without an article, the reader shows the server's [content] if there is
/// any, and otherwise the original page itself, as a browser would.
///
/// Links never navigate the WebView away from what it shows: they are passed
/// to [openLink]. Navigations the page starts by itself while it is hidden
/// are blocked.
class ReaderPageController({
  required final String url,
  required final String? content,
  required final void Function(String url) openLink,
}) extends ChangeNotifier {
  this {
    unawaited(
      webViewController.setNavigationDelegate(
        NavigationDelegate(
          onProgress: (progress) {
            if (_showsPage) {
              _progress = progress;
              _notify();
            }
          },
          onPageStarted: (_) {
            if (_showsPage) {
              _loading = true;
              _notify();
            }
          },
          onPageFinished: (_) => _handlePageFinished(),
          onWebResourceError: _handleError,
          onNavigationRequest: _decideNavigation,
        ),
      ),
    );
    unawaited(
      webViewController.addJavaScriptChannel(
        'PaperdollReader',
        onMessageReceived: (message) => _handleMessage(message.message),
      ),
    );
    _start();
  }

  final webViewController = WebViewController();

  ReaderPhase get phase => _phase;
  var _phase = ReaderPhase.loading;

  /// The load progress of the original page, from 0 to 100.
  int get progress => _progress;
  var _progress = 0;

  /// Whether the original page is loading.
  bool get loading => _loading;
  var _loading = true;

  late final Uri? _uri = switch (Uri.tryParse(url)) {
    final uri? when uri.isScheme('http') || uri.isScheme('https') => uri,
    _ => null,
  };
  late final String? _serverContent = switch (content) {
    final content? when content.trim().isNotEmpty => content,
    _ => null,
  };

  /// Identifies the current run of the sequence that [_start] begins, so that
  /// work from an earlier run stops when it resumes.
  var _run = 0;
  var _initialLoadFinished = false;
  var _extractionStarted = false;

  /// Whether the reader has chosen what to show in place of the cover. The
  /// choice is final until [reload].
  var _decided = false;
  var _loadFailed = false;
  String? _messageId;

  /// The HTML document shown in place of the original page, kept to show it
  /// again if the web content process ends.
  ({String html, String? baseUrl})? _shownHtml;

  /// The phase that starts once [_shownHtml] has finished loading. The cover
  /// stays up until then, so the original page does not show through.
  ReaderPhase? _pendingPhase;
  var _disposed = false;
  Timer? _documentReadyPoll;
  Timer? _settlePoll;
  Timer? _deadline;
  Timer? _replyTimeout;

  /// Whether the WebView holds the original page, or is about to show it.
  bool get _showsPage => !_decided || _phase == ReaderPhase.page;

  @override
  void dispose() {
    _disposed = true;
    _cancelTimers();
    super.dispose();
  }

  /// Loads the original page again and repeats the whole sequence, including
  /// the extraction. This also works after a failed initial load, when the
  /// WebView has no page to reload.
  void reload() => _start();

  void _start() {
    final run = ++_run;
    _cancelTimers();
    _phase = ReaderPhase.loading;
    _progress = 0;
    _loading = true;
    _initialLoadFinished = false;
    _extractionStarted = false;
    _decided = false;
    _loadFailed = false;
    _messageId = null;
    _shownHtml = null;
    _pendingPhase = null;
    _notify();

    final uri = _uri;
    if (uri == null) {
      _loadFailed = true;
      _fallBack(run, 'not a web URL');
      return;
    }
    // A starting guess, to be tuned.
    _deadline = Timer(
      const Duration(seconds: 15),
      () => _fallBack(run, 'no article within 15 s', warn: true),
    );
    unawaited(_load(run, uri));
  }

  Future<void> _load(int run, Uri uri) async {
    await webViewController.setJavaScriptMode(JavaScriptMode.unrestricted);
    // Marks the current document so that the poll below does not mistake it
    // for the new one: it stays in place until the new page starts arriving.
    try {
      await webViewController.runJavaScript('window.__paperdollStale = true');
    } on Object {
      // No document to mark, e.g. after the web content process ended.
    }
    if (run != _run || _disposed) {
      return;
    }
    await webViewController.loadRequest(uri);
    // onPageFinished waits for the page's load event, which pages with ads
    // may never fire, because ad frames keep loading. Therefore, the initial
    // load also ends when the new document has been parsed.
    _documentReadyPoll?.cancel();
    _documentReadyPoll = Timer.periodic(const Duration(milliseconds: 300), (
      _,
    ) async {
      final result = await _evaluate(
        "location.href !== 'about:blank' && !window.__paperdollStale && "
        "document.readyState !== 'loading' ? 1 : 0",
      );
      if (run == _run && result == 1) {
        _finishInitialLoad();
      }
    });
  }

  void _handlePageFinished() {
    if (_pendingPhase case final phase?) {
      _pendingPhase = null;
      _phase = phase;
      _notify();
    } else if (_showsPage) {
      _finishInitialLoad();
    }
  }

  void _finishInitialLoad() {
    _documentReadyPoll?.cancel();
    _loading = false;
    _initialLoadFinished = true;
    _notify();
    if (!_decided && !_extractionStarted) {
      _extractionStarted = true;
      unawaited(_extract(_run));
    }
  }

  Future<void> _extract(int run) async {
    bool superseded() => run != _run || _decided || _disposed;

    // A PDF or an image has no article to extract.
    final isHtml = await _evaluate('/html/.test(document.contentType) ? 1 : 0');
    if (superseded()) {
      return;
    }
    if (isHtml != 1) {
      _fallBack(run, 'not an HTML document');
      return;
    }

    // Pages that render with JavaScript may still be filling in. Waits until
    // the text length stops changing for 500 ms, or 5 s at most. Both numbers
    // are starting guesses, to be tuned.
    const tick = Duration(milliseconds: 100);
    var waited = Duration.zero;
    var changedAt = Duration.zero;
    Object? lastLength;
    var measuring = false;
    _settlePoll = Timer.periodic(tick, (timer) async {
      waited += tick;
      if (measuring) {
        return;
      }
      measuring = true;
      final length = await _evaluate(
        'document.body ? document.body.innerText.length : 0',
      );
      measuring = false;
      if (superseded()) {
        timer.cancel();
        return;
      }
      if (length == null || length != lastLength) {
        lastLength = length;
        changedAt = waited;
      }
      if (waited - changedAt >= const Duration(milliseconds: 500) ||
          waited >= const Duration(seconds: 5)) {
        timer.cancel();
        unawaited(_inject(run));
      }
    });
  }

  Future<void> _inject(int run) async {
    final messageId = 'extraction-$run';
    _messageId = messageId;
    _replyTimeout = Timer(
      const Duration(seconds: 10),
      () => _fallBack(run, 'no reply within 10 s', warn: true),
    );
    try {
      await webViewController.runJavaScript(
        await rootBundle.loadString('assets/reader/extract.js'),
      );
      await webViewController.runJavaScript(
        'window.__paperdollReader.extract(${jsonEncode(messageId)})',
      );
    } on Object catch (e) {
      _fallBack(run, 'could not run the script: $e', warn: true);
    }
  }

  void _handleMessage(String raw) {
    final message = _parseMessage(raw);
    if (message == null) {
      appLogger.warning('Reader ignored a malformed message for $url');
      return;
    }
    if (message.messageId != _messageId) {
      appLogger.warning('Reader ignored a message for another extraction');
      return;
    }
    if (_decided) {
      appLogger.warning('Reader ignored a message after it chose what to show');
      return;
    }
    switch (message.status) {
      case 'ok':
        unawaited(_showArticle(_run, message));
      case 'empty':
        _fallBack(_run, 'no article found');
      default:
        _fallBack(_run, 'the script threw: ${message.error}', warn: true);
    }
  }

  Future<void> _showArticle(int run, _Message message) async {
    _decide();
    final String html;
    try {
      final template = await rootBundle.loadString('assets/reader/reader.html');
      html = await compute(renderArticle, (
        template: template,
        siteName: message.siteName,
        title: message.title,
        byline: message.byline,
        content: message.content!,
      ));
    } on Object catch (e, s) {
      appLogger.warning('Reader could not render the article', e, s);
      if (run == _run && !_disposed) {
        await _showFallback();
      }
      return;
    }
    if (run == _run && !_disposed) {
      await _showHtml(ReaderPhase.article, html, baseUrl: message.url);
    }
  }

  /// Shows the server's content if there is any, and otherwise the original
  /// page, or the error if it failed to load.
  void _fallBack(int run, String reason, {bool warn = false}) {
    if (run != _run || _decided || _disposed) {
      return;
    }
    final log = warn ? appLogger.warning : appLogger.info;
    log('Reader shows no article for $url: $reason');
    _decide();
    unawaited(_showFallback());
  }

  Future<void> _showFallback() async {
    if (_serverContent case final content?) {
      await _showHtml(
        ReaderPhase.serverContent,
        _serverContentDocument(content),
      );
    } else {
      _phase = _loadFailed ? ReaderPhase.pageError : ReaderPhase.page;
      _notify();
    }
  }

  void _decide() {
    _decided = true;
    _cancelTimers();
  }

  Future<void> _showHtml(
    ReaderPhase phase,
    String html, {
    String? baseUrl,
  }) async {
    _shownHtml = (html: html, baseUrl: baseUrl);
    _pendingPhase = phase;
    await webViewController.setJavaScriptMode(JavaScriptMode.disabled);
    await webViewController.loadHtmlString(html, baseUrl: baseUrl);
  }

  void _handleError(WebResourceError error) {
    if (!(error.isForMainFrame ?? false)) {
      return;
    }
    // iOS may end the web content process while the app is in the background
    // (e.g. while another app is open), which leaves the WebView blank.
    final terminated =
        error.errorType == WebResourceErrorType.webContentProcessTerminated;
    if (_shownHtml case final shown?) {
      if (terminated) {
        unawaited(
          webViewController.loadHtmlString(shown.html, baseUrl: shown.baseUrl),
        );
      }
      return;
    }
    if (!_decided) {
      if (terminated) {
        _start();
        return;
      }
      _documentReadyPoll?.cancel();
      _loadFailed = true;
      _fallBack(_run, 'the page failed to load: ${error.description}');
      return;
    }
    if (_phase != ReaderPhase.page) {
      return;
    }
    // Reloads the page, unless the process ended during the load itself,
    // where a reload would likely end the same way.
    if (terminated && _initialLoadFinished) {
      reload();
      return;
    }
    _documentReadyPoll?.cancel();
    _loading = false;
    _phase = ReaderPhase.pageError;
    _notify();
  }

  NavigationDecision _decideNavigation(NavigationRequest request) {
    if (!request.isMainFrame) {
      return NavigationDecision.navigate;
    }
    final url = request.url;
    final isWeb = url.startsWith('http://') || url.startsWith('https://');
    if (_shownHtml != null) {
      if (!isWeb) {
        return NavigationDecision.navigate;
      }
      _openLink(url);
      return NavigationDecision.prevent;
    }
    // Redirects during the initial load (http → https, tracking links).
    if (!_initialLoadFinished && isWeb && _showsPage) {
      return NavigationDecision.navigate;
    }
    // While the page is hidden, a navigation can only come from the page
    // itself, and must not open the in-app browser over the cover.
    if (_phase != ReaderPhase.page) {
      appLogger.info('Reader blocked a navigation of the hidden page');
      return NavigationDecision.prevent;
    }
    _openLink(url);
    return NavigationDecision.prevent;
  }

  void _openLink(String url) {
    if (!_disposed) {
      openLink(url);
    }
  }

  /// Evaluates [script] in the page and returns its result, or null if it
  /// fails. Scripts return numbers rather than booleans or strings, because
  /// the platforms return those differently.
  Future<Object?> _evaluate(String script) async {
    try {
      return await webViewController.runJavaScriptReturningResult(script);
    } on Object {
      return null;
    }
  }

  void _cancelTimers() {
    _documentReadyPoll?.cancel();
    _settlePoll?.cancel();
    _deadline?.cancel();
    _replyTimeout?.cancel();
  }

  void _notify() {
    if (!_disposed) {
      notifyListeners();
    }
  }
}

typedef _Message = ({
  String messageId,
  String status,
  String url,
  String? title,
  String? byline,
  String? siteName,
  String? content,
  String? error,
});

/// Parses a message from `extract.js`, or returns null if it breaks the
/// contract described on [ReaderPageController].
_Message? _parseMessage(String raw) {
  final Object? json;
  try {
    json = jsonDecode(raw);
  } on FormatException {
    return null;
  }
  if (json case {
    'v': 1,
    'messageId': final String messageId,
    'status': final String status,
    'url': final String url,
    'title': final String? title,
    'byline': final String? byline,
    'siteName': final String? siteName,
    'content': final String? content,
    'error': final String? error,
  }) {
    final hasMetadata = title != null || byline != null || siteName != null;
    final valid = switch (status) {
      'ok' => content != null && error == null,
      'empty' => !hasMetadata && content == null && error == null,
      'error' => !hasMetadata && content == null && error != null,
      _ => false,
    };
    // The URL becomes the article's base URL, which must not give it access
    // to anything but the web, such as local files.
    final uri = Uri.tryParse(url);
    if (!valid ||
        uri == null ||
        !(uri.isScheme('http') || uri.isScheme('https'))) {
      return null;
    }
    return (
      messageId: messageId,
      status: status,
      url: url,
      title: title,
      byline: byline,
      siteName: siteName,
      content: content,
      error: error,
    );
  }
  return null;
}

/// Wraps the server's [content] of an item into a document, as the reader
/// showed it before it extracted articles itself.
String _serverContentDocument(String content) {
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
