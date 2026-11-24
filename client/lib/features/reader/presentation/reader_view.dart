import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/features/reader/presentation/reader_page_controller.dart';
import 'package:paperdoll/features/reader/presentation/reader_web_view.dart';

/// Reader body shared by the Story, Feed Entry, and Web Clip Readers: the
/// article extracted from the page at [url], or, without one, the server's
/// [content] or the page itself (see [ReaderPageController]).
///
/// What it shows stays for the lifetime of the screen, so that an article the
/// user is reading is never replaced.
class const ReaderView({
  required final String url,

  /// The item's content from the server, shown when no article is extracted.
  required final String? content,

  /// Shown above the reader, e.g. the archived banner.
  final Widget? banner,
  super.key,
}) extends StatefulWidget {
  @override
  State<ReaderView> createState() => _ReaderViewState();
}

class _ReaderViewState extends State<ReaderView> {
  late final _controller = ReaderPageController(
    url: widget.url,
    content: widget.content,
    openLink: (url) => unawaited(openInAppBrowserLink(context, url)),
  );

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ?widget.banner,
        Expanded(child: ReaderWebView(controller: _controller)),
      ],
    );
  }
}
