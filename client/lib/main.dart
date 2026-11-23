import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/app.dart';
import 'package:paperdoll/core/logging/app_logger.dart';
import 'package:paperdoll/core/logging/logging_provider_observer.dart';

void main() {
  configureLogging();
  // Readability ships inside assets/reader/extract.js rather than as a Dart
  // package, so its license is not registered automatically.
  LicenseRegistry.addLicense(
    () => Stream.value(
      const LicenseEntryWithLineBreaks([
        '@mozilla/readability',
      ], _readabilityLicense),
    ),
  );
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

const _readabilityLicense = '''
Copyright (c) 2010 Arc90 Inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.''';
