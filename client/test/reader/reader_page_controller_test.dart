import 'dart:convert';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:paperdoll/features/reader/presentation/reader_page_controller.dart';
import 'package:webview_flutter_platform_interface/webview_flutter_platform_interface.dart';

import '../src/fake_webview.dart';

const _url = 'https://coastal.example/posts/1';
const _serverContent = '<p>Server content</p>';

void main() {
  late FakeWebViewPlatform platform;
  late List<String> openedLinks;

  setUp(() {
    platform = installFakeWebViewPlatform();
    openedLinks = [];
  });

  /// Creates a controller and lets it start loading the page. It is disposed
  /// with the widget tree at the end of the test.
  Future<ReaderPageController> createController(
    WidgetTester tester, {
    String url = _url,
    String? content,
  }) async {
    final controller = ReaderPageController(
      url: url,
      content: content,
      openLink: openedLinks.add,
    );
    await tester.pumpWidget(_Owner(controller: controller));
    return controller;
  }

  FakeWebViewController webView() => platform.controllers.single;

  /// The ID of the extraction the reader has started, or null.
  String? extractionId() {
    final scripts = webView().scripts.reversed;
    for (final script in scripts) {
      final match = RegExp(r'__paperdollReader\.extract\("([^"]*)"\)')
          .firstMatch(script);
      if (match != null) {
        return match[1];
      }
    }
    return null;
  }

  Future<void> waitForExtraction(WidgetTester tester) =>
      pumpUntil(tester, () => extractionId() != null);

  /// Waits for the reader to load an HTML string, and finishes loading it.
  Future<HtmlLoad> waitForHtml(WidgetTester tester) async {
    await pumpUntil(tester, () => webView().htmlLoads.isNotEmpty);
    webView().finishPage('about:blank');
    await tester.pump();
    return webView().htmlLoads.single;
  }

  group('extraction', () {
    testWidgets('shows the article in place of the page', (tester) async {
      platform.extractionReply = (id) => _okMessage(
        id,
        url: 'https://coastal.example/posts/1?redirected',
        title: 'Tide <Pools>',
        content: '<p>Article body</p><script>alert(1)</script>',
      );
      final controller = await createController(
        tester,
        content: _serverContent,
      );

      expect(controller.phase, ReaderPhase.loading);
      expect(webView().requests, [Uri.parse(_url)]);
      expect(webView().javaScriptMode, JavaScriptMode.unrestricted);

      final load = await waitForHtml(tester);
      expect(controller.phase, ReaderPhase.article);
      expect(load.mode, JavaScriptMode.disabled);
      expect(load.baseUrl, 'https://coastal.example/posts/1?redirected');
      expect(load.html, contains('<p>Article body</p>'));
      expect(load.html, contains('Tide &lt;Pools&gt;'));
      expect(load.html, isNot(contains('alert(1)')));
      expect(load.html, contains('Content-Security-Policy'));
      expect(
        webView().scripts,
        contains(contains('window.__paperdollReader = { extract }')),
      );
    });

    testWidgets('keeps the cover until the article has loaded', (tester) async {
      platform.extractionReply = _okMessage;
      final controller = await createController(tester);

      await pumpUntil(tester, () => webView().htmlLoads.isNotEmpty);
      expect(controller.phase, ReaderPhase.loading);

      webView().finishPage('about:blank');
      expect(controller.phase, ReaderPhase.article);
    });

    testWidgets('waits for the text to stop changing', (tester) async {
      var length = 0;
      final controller = await createController(tester);
      webView().evaluate = (script) =>
          script.contains('innerText') ? length += 10 : 1;

      await tester.pump(const Duration(seconds: 3));
      expect(extractionId(), isNull);

      webView().evaluate = (script) => script.contains('innerText') ? 42 : 1;
      await tester.pump(const Duration(milliseconds: 400));
      expect(extractionId(), isNull);
      await waitForExtraction(tester);
      expect(controller.phase, ReaderPhase.loading);
    });

    testWidgets('extracts after 5 s even if the text keeps changing', (
      tester,
    ) async {
      var length = 0;
      await createController(tester);
      webView().evaluate = (script) =>
          script.contains('innerText') ? length += 10 : 1;

      // The page is ready on the first 300 ms poll.
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump(const Duration(milliseconds: 4800));
      expect(extractionId(), isNull);
      await tester.pump(const Duration(milliseconds: 300));
      await waitForExtraction(tester);
    });

    testWidgets('does not extract from a document that is not HTML', (
      tester,
    ) async {
      final controller = await createController(tester);
      webView().evaluate = (script) => script.contains('contentType') ? 0 : 1;

      await tester.pump(const Duration(seconds: 2));
      expect(extractionId(), isNull);
      expect(controller.phase, ReaderPhase.page);
    });
  });

  group('fallback', () {
    testWidgets('shows server content when no article is found', (
      tester,
    ) async {
      platform.extractionReply = _emptyMessage;
      final controller = await createController(
        tester,
        content: _serverContent,
      );

      final load = await waitForHtml(tester);
      expect(controller.phase, ReaderPhase.serverContent);
      expect(load.html, contains(_serverContent));
      expect(load.mode, JavaScriptMode.disabled);
      expect(load.baseUrl, isNull);
    });

    testWidgets('shows the page without an article or server content', (
      tester,
    ) async {
      platform.extractionReply = _emptyMessage;
      final controller = await createController(tester, content: '  ');

      await waitForExtraction(tester);
      expect(controller.phase, ReaderPhase.page);
      expect(webView().htmlLoads, isEmpty);
      expect(webView().javaScriptMode, JavaScriptMode.unrestricted);
    });

    testWidgets('falls back when the script throws', (tester) async {
      platform.extractionReply = (id) =>
          jsonEncode({..._fields(id), 'status': 'error', 'error': 'boom'});
      final controller = await createController(tester);

      await waitForExtraction(tester);
      expect(controller.phase, ReaderPhase.page);
    });

    testWidgets('falls back without a reply within 10 s', (tester) async {
      final controller = await createController(
        tester,
        content: _serverContent,
      );

      await waitForExtraction(tester);
      await tester.pump(const Duration(seconds: 9));
      expect(controller.phase, ReaderPhase.loading);
      await tester.pump(const Duration(seconds: 1));
      await waitForHtml(tester);
      expect(controller.phase, ReaderPhase.serverContent);
    });

    testWidgets('falls back 15 s after opening', (tester) async {
      final controller = await createController(tester);
      // The page never becomes ready.
      webView().evaluate = (_) => 0;

      await tester.pump(const Duration(seconds: 14));
      expect(controller.phase, ReaderPhase.loading);
      await tester.pump(const Duration(seconds: 1));
      expect(controller.phase, ReaderPhase.page);
    });

    testWidgets('falls back when the page fails to load', (tester) async {
      final controller = await createController(tester);

      webView().failPage();
      await tester.pump();
      expect(controller.phase, ReaderPhase.pageError);
      expect(extractionId(), isNull);
    });

    testWidgets('shows server content when the page fails to load', (
      tester,
    ) async {
      final controller = await createController(
        tester,
        content: _serverContent,
      );

      webView().failPage();
      await waitForHtml(tester);
      expect(controller.phase, ReaderPhase.serverContent);
    });

    testWidgets('shows the error for a URL that is not a web URL', (
      tester,
    ) async {
      final controller = await createController(
        tester,
        url: 'file:///etc/passwd',
      );

      expect(controller.phase, ReaderPhase.pageError);
      expect(webView().requests, isEmpty);
    });

    testWidgets('shows server content for a URL that is not a web URL', (
      tester,
    ) async {
      final controller = await createController(
        tester,
        url: '',
        content: _serverContent,
      );

      await waitForHtml(tester);
      expect(controller.phase, ReaderPhase.serverContent);
    });
  });

  group('messages', () {
    for (final (name, message) in [
      ('not JSON', (String id) => '{'),
      ('another version', (String id) => jsonEncode({..._ok(id), 'v': 2})),
      ('no content', (String id) => jsonEncode({..._ok(id), 'content': null})),
      (
        'metadata without an article',
        (String id) => jsonEncode({..._fields(id), 'title': 'Title'}),
      ),
      (
        'a URL that is not a web URL',
        (String id) => jsonEncode({..._ok(id), 'url': 'file:///etc/passwd'}),
      ),
      ('another extraction', (String id) => _okMessage('$id-old')),
    ]) {
      testWidgets('ignores a message with $name', (tester) async {
        final controller = await createController(tester);

        await waitForExtraction(tester);
        webView().postMessage('PaperdollReader', message(extractionId()!));
        await tester.pump();
        expect(controller.phase, ReaderPhase.loading);

        // The timeout still applies.
        await tester.pump(const Duration(seconds: 10));
        expect(controller.phase, ReaderPhase.page);
        expect(webView().htmlLoads, isEmpty);
      });
    }

    testWidgets('keeps the first result', (tester) async {
      platform.extractionReply = _okMessage;
      final controller = await createController(
        tester,
        content: _serverContent,
      );

      await waitForHtml(tester);
      webView().postMessage('PaperdollReader', _emptyMessage(extractionId()!));
      webView().postMessage('PaperdollReader', _okMessage(extractionId()!));
      await tester.pump(const Duration(seconds: 1));
      expect(controller.phase, ReaderPhase.article);
      expect(webView().htmlLoads, hasLength(1));
    });
  });

  group('navigation', () {
    testWidgets('follows redirects during the first load', (tester) async {
      await createController(tester);

      expect(
        await webView().navigate('https://coastal.example/redirected'),
        NavigationDecision.navigate,
      );
    });

    testWidgets('blocks navigations of the hidden page', (tester) async {
      final controller = await createController(tester);

      await waitForExtraction(tester);
      expect(
        await webView().navigate('https://ads.example/'),
        NavigationDecision.prevent,
      );
      expect(openedLinks, isEmpty);
      expect(controller.phase, ReaderPhase.loading);
    });

    testWidgets('opens links of the page in the in-app browser', (
      tester,
    ) async {
      platform.extractionReply = _emptyMessage;
      await createController(tester);

      await waitForExtraction(tester);
      expect(
        await webView().navigate('https://coastal.example/2'),
        NavigationDecision.prevent,
      );
      expect(openedLinks, ['https://coastal.example/2']);
    });

    testWidgets('lets the article load at its base URL', (tester) async {
      platform.extractionReply = _okMessage;
      await createController(tester);

      await pumpUntil(tester, () => webView().htmlLoads.isNotEmpty);
      // WebKit asks to navigate to the base URL of a loading HTML string.
      expect(await webView().navigate(_url), NavigationDecision.navigate);
      webView().finishPage(_url);
      // Once loaded, a link to the same URL opens in the in-app browser.
      expect(await webView().navigate(_url), NavigationDecision.prevent);
      expect(openedLinks, [_url]);
    });

    testWidgets('opens links of the article in the in-app browser', (
      tester,
    ) async {
      platform.extractionReply = _okMessage;
      await createController(tester);

      await waitForHtml(tester);
      expect(
        await webView().navigate('https://coastal.example/2'),
        NavigationDecision.prevent,
      );
      expect(openedLinks, ['https://coastal.example/2']);
    });
  });

  group('reload', () {
    testWidgets('repeats the whole sequence', (tester) async {
      platform.extractionReply = _emptyMessage;
      final controller = await createController(tester);
      await waitForExtraction(tester);
      expect(controller.phase, ReaderPhase.page);

      platform.extractionReply = _okMessage;
      controller.reload();
      await tester.pump();
      expect(controller.phase, ReaderPhase.loading);
      expect(webView().requests, hasLength(2));

      await waitForHtml(tester);
      expect(controller.phase, ReaderPhase.article);
    });

    testWidgets('drops the reply to the extraction before a reload', (
      tester,
    ) async {
      platform.extractionReply = _emptyMessage;
      final controller = await createController(tester);
      await waitForExtraction(tester);
      final oldId = extractionId()!;

      platform.extractionReply = null;
      controller.reload();
      await tester.pump(const Duration(seconds: 2));
      webView().postMessage('PaperdollReader', _okMessage(oldId));
      await tester.pump();
      expect(controller.phase, ReaderPhase.loading);
    });
  });

  group('web content process termination', () {
    const terminated = WebResourceErrorType.webContentProcessTerminated;

    testWidgets('restarts while the cover is up', (tester) async {
      final controller = await createController(tester);
      await waitForExtraction(tester);

      webView().failPage(type: terminated);
      await tester.pump();
      expect(controller.phase, ReaderPhase.loading);
      expect(webView().requests, hasLength(2));
    });

    testWidgets('loads the article again', (tester) async {
      platform.extractionReply = _okMessage;
      final controller = await createController(tester);
      final load = await waitForHtml(tester);

      webView().failPage(type: terminated);
      expect(controller.phase, ReaderPhase.article);
      expect(webView().htmlLoads, [load, load]);
      expect(webView().requests, hasLength(1));
    });

    testWidgets('reloads the page', (tester) async {
      platform.extractionReply = _emptyMessage;
      final controller = await createController(tester);
      await waitForExtraction(tester);
      expect(controller.phase, ReaderPhase.page);

      webView().failPage(type: terminated);
      await tester.pump();
      expect(controller.phase, ReaderPhase.loading);
      expect(webView().requests, hasLength(2));
    });
  });
}

Map<String, Object?> _fields(String id) => {
  'v': 1,
  'messageId': id,
  'status': 'empty',
  'url': _url,
  'title': null,
  'byline': null,
  'siteName': null,
  'content': null,
  'error': null,
};

Map<String, Object?> _ok(
  String id, {
  String url = _url,
  String? title = 'Title',
  String content = '<p>Article body</p>',
}) => {
  ..._fields(id),
  'status': 'ok',
  'url': url,
  'title': title,
  'content': content,
};

String _okMessage(
  String id, {
  String url = _url,
  String? title = 'Title',
  String content = '<p>Article body</p>',
}) => jsonEncode(_ok(id, url: url, title: title, content: content));

String _emptyMessage(String id) => jsonEncode(_fields(id));

/// Disposes [controller] when the widget tree is disposed.
class const _Owner({required final ReaderPageController controller})
    extends StatefulWidget {
  @override
  State<_Owner> createState() => _OwnerState();
}

class _OwnerState extends State<_Owner> {
  @override
  void dispose() {
    widget.controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => const SizedBox();
}
