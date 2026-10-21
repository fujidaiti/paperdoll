import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/app.dart';
import 'package:paperdoll/core/logging/app_logger.dart';
import 'package:paperdoll/core/logging/logging_provider_observer.dart';

void main() {
  configureLogging();
  final container = createPaperdollContainer(
    observers: [const LoggingProviderObserver()],
  );
  runApp(
    UncontrolledProviderScope(
      container: container,
      child: const PaperdollApp(),
    ),
  );
}
