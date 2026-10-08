import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/core/ui/widgets/app_divider.dart';
import 'package:paperdoll/core/ui/widgets/async_value_view.dart';
import 'package:paperdoll/core/ui/widgets/body_text.dart';
import 'package:paperdoll/core/ui/widgets/caption_text.dart';
import 'package:paperdoll/features/page_watch/presentation/providers/page_watch_providers.dart';

/// Lists the bundled sample pages. It stands in for searching a page URL
/// until the server can simplify arbitrary pages.
class const PageSamplesScreen({super.key}) extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Watch a web page')),
      body: AsyncValueView<List<PageSample>>(
        value: ref.watch(pageSamplesProvider),
        onRetry: () => ref.invalidate(pageSamplesProvider),
        data: (samples) => ListView.separated(
          itemCount: samples.length,
          separatorBuilder: (context, index) => const AppDivider(),
          itemBuilder: (context, index) {
            final sample = samples[index];
            return ListTile(
              title: BodyText(
                sample.title.isEmpty ? sample.file : sample.title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              subtitle: CaptionText(
                sample.url,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              onTap: () => context.pushNamed(
                routePageAnnotationName,
                pathParameters: {'file': sample.file},
              ),
            );
          },
        ),
      ),
    );
  }
}
