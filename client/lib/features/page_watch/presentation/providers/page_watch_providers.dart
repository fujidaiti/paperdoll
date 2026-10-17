import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:paperdoll/features/page_watch/domain/page_document.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'page_watch_providers.g.dart';

// The server does not simplify pages yet, so the documents are generated from
// sample pages by tool/semantictree and bundled as assets.
const _assetDir = 'assets/page_watch';

/// A bundled sample page that can be annotated.
typedef PageSample = ({String file, String url, String title});

@riverpod
Future<List<PageSample>> pageSamples(Ref ref) async {
  final json = jsonDecode(await rootBundle.loadString('$_assetDir/index.json'));
  return [
    for (final entry in (json as List<Object?>).cast<Map<String, Object?>>())
      (
        file: entry['file']! as String,
        url: entry['url']! as String,
        title: entry['title']! as String,
      ),
  ];
}

@riverpod
Future<PageDocument> pageDocument(Ref ref, String file) async {
  final json = jsonDecode(await rootBundle.loadString('$_assetDir/$file'));
  return PageDocument.fromJson(json as Map<String, Object?>);
}
