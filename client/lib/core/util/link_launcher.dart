import 'package:material_ui/material_ui.dart';
import 'package:url_launcher/url_launcher.dart';

/// Opens [url] in an external browser. Shows a snackbar if it can't be opened.
Future<void> openExternalLink(BuildContext context, String url) async {
  // Capture before the await so we don't touch BuildContext afterwards.
  final messenger = ScaffoldMessenger.of(context);
  final uri = Uri.tryParse(url);
  var launched = false;
  if (uri != null) {
    launched = await launchUrl(uri, mode: LaunchMode.externalApplication);
  }
  if (!launched) {
    messenger.showSnackBar(
      const SnackBar(content: Text('Could not open the link.')),
    );
  }
}

/// Opens [url] in the in-app browser. Shows a snackbar if it can't be opened.
Future<void> openInAppLink(BuildContext context, String url) async {
  final messenger = ScaffoldMessenger.of(context);
  final uri = Uri.tryParse(url);
  var launched = false;
  if (uri != null && (uri.isScheme('http') || uri.isScheme('https'))) {
    launched = await launchUrl(uri, mode: LaunchMode.inAppBrowserView);
  }
  if (!launched) {
    messenger.showSnackBar(
      const SnackBar(content: Text('Could not open the link.')),
    );
  }
}
