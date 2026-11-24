import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/ui/tokens/app_spacing.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/gap.dart';
import 'package:paperdoll/core/ui/widgets/loading_indicator.dart';
import 'package:paperdoll/core/util/link_launcher.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/reader/presentation/reader_page_controller.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// Shows the reader WebView in the state [controller] is in: covered while
/// the article is being extracted, then the article or the server's content
/// as they are, or the original page with a reload button and a progress bar.
class const ReaderWebView({
  required final ReaderPageController controller,
  super.key,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: controller,
      builder: (context, _) {
        final phase = controller.phase;
        if (phase == ReaderPhase.pageError) {
          return Center(
            child: Padding(
              padding: const EdgeInsets.all(spacingLg),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const BodyText("Couldn't load this page."),
                  const Gap(spacingMd),
                  FilledButton(
                    key: AppDebugKey.readerRetryButton,
                    onPressed: controller.reload,
                    child: const Text('Retry'),
                  ),
                  const Gap(spacingSm),
                  TextButton(
                    onPressed: () =>
                        unawaited(openExternalLink(context, controller.url)),
                    child: const Text('Open in browser'),
                  ),
                ],
              ),
            ),
          );
        }
        return Stack(
          children: [
            WebViewWidget(controller: controller.webViewController),
            if (phase == ReaderPhase.page && controller.loading)
              LinearProgressIndicator(value: controller.progress / 100),
            if (phase == ReaderPhase.page)
              Positioned(
                right: spacingMd,
                bottom: spacingMd,
                child: SafeArea(
                  child: FloatingActionButton(
                    key: AppDebugKey.readerReloadButton,
                    // Disables the hero animation, which would otherwise move
                    // the button between two readers during a route
                    // transition.
                    heroTag: null,
                    tooltip: 'Reload',
                    onPressed: controller.reload,
                    child: const Icon(Icons.refresh),
                  ),
                ),
              ),
            // Opaque, so that neither the original page nor the empty
            // WebView's dark default background shows through.
            if (phase == ReaderPhase.loading)
              Positioned.fill(
                key: AppDebugKey.readerLoadingCover,
                child: ColoredBox(
                  color: Theme.of(context).colorScheme.surface,
                  child: const LoadingIndicator(),
                ),
              ),
          ],
        );
      },
    );
  }
}
