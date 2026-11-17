import 'package:material_ui/material_ui.dart';
import 'package:url_launcher/url_launcher.dart';

/// Opens [url] in an external browser. Shows a snackbar if it can't be opened.
Future<void> openExternalLink(BuildContext context, String url) =>
    _launch(context, url, LaunchMode.externalApplication);

/// Opens [url] in an in-app browser (SFSafariViewController on iOS, Custom
/// Tabs on Android), so the user returns to the app by closing it. Non-web
/// URLs such as `mailto:` open in their external app, because the in-app
/// browser cannot show them. Shows a snackbar if it can't be opened.
Future<void> openInAppBrowserLink(BuildContext context, String url) {
  final isWeb = url.startsWith('http://') || url.startsWith('https://');
  return _launch(
    context,
    url,
    isWeb ? LaunchMode.inAppBrowserView : LaunchMode.externalApplication,
  );
}

Future<void> _launch(BuildContext context, String url, LaunchMode mode) async {
  // Capture before the await so we don't touch BuildContext afterwards.
  final messenger = ScaffoldMessenger.of(context);
  final uri = Uri.tryParse(url);
  var launched = false;
  if (uri != null) {
    launched = await launchUrl(uri, mode: mode);
  }
  if (!launched) {
    messenger.showSnackBar(
      const SnackBar(content: Text('Could not open the link.')),
    );
  }
}
