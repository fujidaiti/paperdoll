import 'dart:async';

import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:plugin_platform_interface/plugin_platform_interface.dart';
import 'package:webview_flutter_platform_interface/webview_flutter_platform_interface.dart';

/// Installs an in-memory WebView platform so that screens embedding a
/// `WebViewWidget` can build under the Flutter test framework, where the real
/// platform view is unavailable (building one otherwise throws
/// `WebViewPlatform.instance != null`). The fake renders nothing.
///
/// It records what the app asks the WebView to do, and lets a test act as the
/// page: finish loading, fail, request navigations, answer scripts, and post
/// channel messages. Reach a WebView through [FakeWebViewPlatform.controllers].
///
/// By default every page is a ready HTML document with a fixed amount of text,
/// and the reader's extraction script gets no reply; set
/// [FakeWebViewPlatform.extractionReply] to answer it.
FakeWebViewPlatform installFakeWebViewPlatform() {
  // The reader loads its script and template with rootBundle, which caches
  // each asset as a Future. A Future cached by an earlier test belongs to that
  // test's fake time, and never completes in a later one.
  rootBundle.clear();
  final platform = FakeWebViewPlatform();
  WebViewPlatform.instance = platform;
  return platform;
}

/// Pumps frames, 100 ms apart, until [condition] holds. In between, it lets
/// real asynchronous work run, which fake time alone does not complete, such
/// as the isolate the reader renders an article in.
Future<void> pumpUntil(WidgetTester tester, bool Function() condition) async {
  for (var i = 0; i < 100; i++) {
    if (condition()) {
      return;
    }
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    await tester.pump(const Duration(milliseconds: 100));
  }
  fail('The condition was not met within 100 frames.');
}

/// The [WebViewPlatform] installed by [installFakeWebViewPlatform].
class FakeWebViewPlatform extends WebViewPlatform {
  /// The WebViews created so far, oldest first.
  final controllers = <FakeWebViewController>[];

  /// The message the page posts to the `PaperdollReader` channel when the
  /// reader calls `extract(messageId)`, or null to post nothing.
  String? Function(String messageId)? extractionReply;

  @override
  PlatformWebViewController createPlatformWebViewController(
    PlatformWebViewControllerCreationParams params,
  ) {
    final controller = FakeWebViewController(params, this);
    controllers.add(controller);
    return controller;
  }

  @override
  PlatformNavigationDelegate createPlatformNavigationDelegate(
    PlatformNavigationDelegateCreationParams params,
  ) => _FakeNavigationDelegate(params);

  @override
  PlatformWebViewWidget createPlatformWebViewWidget(
    PlatformWebViewWidgetCreationParams params,
  ) => _FakeWebViewWidget(params);
}

/// A loaded HTML string, with the JavaScript mode at the time of loading.
typedef HtmlLoad = ({String html, String? baseUrl, JavaScriptMode? mode});

class FakeWebViewController extends PlatformWebViewController
    with MockPlatformInterfaceMixin {
  new(super.params, this._platform) : super.implementation();

  final FakeWebViewPlatform _platform;
  _FakeNavigationDelegate? _delegate;
  final _channels = <String, JavaScriptChannelParams>{};

  JavaScriptMode? javaScriptMode;

  /// The URLs loaded with `loadRequest`, oldest first.
  final requests = <Uri>[];

  /// The HTML strings loaded with `loadHtmlString`, oldest first.
  final htmlLoads = <HtmlLoad>[];

  /// The scripts run with `runJavaScript`, oldest first.
  final scripts = <String>[];

  /// Answers `runJavaScriptReturningResult`; throwing makes the call fail.
  /// The default describes a ready HTML document with 100 characters of text.
  Object Function(String script) evaluate = (script) {
    if (script.contains('readyState') || script.contains('contentType')) {
      return 1;
    }
    if (script.contains('innerText')) {
      return 100;
    }
    return 0;
  };

  /// Acts as the page finishing its load.
  void finishPage(String url) => _delegate?.onPageFinished?.call(url);

  /// Acts as the main frame failing to load, or the web content process
  /// ending when [type] is [WebResourceErrorType.webContentProcessTerminated].
  void failPage({WebResourceErrorType type = WebResourceErrorType.hostLookup}) {
    _delegate?.onWebResourceError?.call(
      WebResourceError(
        errorCode: -1,
        description: 'Failed',
        errorType: type,
        isForMainFrame: true,
      ),
    );
  }

  /// Acts as a main-frame navigation to [url], and returns the app's decision.
  Future<NavigationDecision> navigate(String url) async {
    final callback = _delegate?.onNavigationRequest;
    if (callback == null) {
      return NavigationDecision.navigate;
    }
    return await callback(NavigationRequest(url: url, isMainFrame: true));
  }

  /// Acts as the page posting [message] to the channel named [channel].
  void postMessage(String channel, String message) {
    _channels[channel]?.onMessageReceived(JavaScriptMessage(message: message));
  }

  @override
  Future<void> setJavaScriptMode(JavaScriptMode javaScriptMode) async {
    this.javaScriptMode = javaScriptMode;
  }

  @override
  Future<void> setPlatformNavigationDelegate(
    PlatformNavigationDelegate handler,
  ) async {
    _delegate = handler as _FakeNavigationDelegate;
  }

  @override
  Future<void> addJavaScriptChannel(
    JavaScriptChannelParams javaScriptChannelParams,
  ) async {
    _channels[javaScriptChannelParams.name] = javaScriptChannelParams;
  }

  @override
  Future<void> loadHtmlString(String html, {String? baseUrl}) async {
    htmlLoads.add((html: html, baseUrl: baseUrl, mode: javaScriptMode));
  }

  @override
  Future<void> loadRequest(LoadRequestParams params) async {
    requests.add(params.uri);
  }

  @override
  Future<void> runJavaScript(String javaScript) async {
    scripts.add(javaScript);
    final call = RegExp(r'__paperdollReader\.extract\("([^"]*)"\)')
        .firstMatch(javaScript);
    final reply = call == null
        ? null
        : _platform.extractionReply?.call(call[1]!);
    if (reply != null) {
      scheduleMicrotask(() => postMessage('PaperdollReader', reply));
    }
  }

  @override
  Future<Object> runJavaScriptReturningResult(String javaScript) async =>
      evaluate(javaScript);
}

class _FakeNavigationDelegate extends PlatformNavigationDelegate
    with MockPlatformInterfaceMixin {
  new(super.params) : super.implementation();

  NavigationRequestCallback? onNavigationRequest;
  PageEventCallback? onPageFinished;
  WebResourceErrorCallback? onWebResourceError;

  @override
  Future<void> setOnNavigationRequest(
    NavigationRequestCallback onNavigationRequest,
  ) async => this.onNavigationRequest = onNavigationRequest;

  @override
  Future<void> setOnPageStarted(PageEventCallback onPageStarted) async {}

  @override
  Future<void> setOnPageFinished(PageEventCallback onPageFinished) async =>
      this.onPageFinished = onPageFinished;

  @override
  Future<void> setOnProgress(ProgressCallback onProgress) async {}

  @override
  Future<void> setOnWebResourceError(
    WebResourceErrorCallback onWebResourceError,
  ) async => this.onWebResourceError = onWebResourceError;

  @override
  Future<void> setOnUrlChange(UrlChangeCallback onUrlChange) async {}

  @override
  Future<void> setOnHttpError(HttpResponseErrorCallback onHttpError) async {}

  @override
  Future<void> setOnHttpAuthRequest(
    HttpAuthRequestCallback onHttpAuthRequest,
  ) async {}

  @override
  Future<void> setOnSSlAuthError(SslAuthErrorCallback onSslAuthError) async {}
}

class _FakeWebViewWidget extends PlatformWebViewWidget
    with MockPlatformInterfaceMixin {
  new(super.params) : super.implementation();

  @override
  Widget build(BuildContext context) => const SizedBox.shrink();
}
