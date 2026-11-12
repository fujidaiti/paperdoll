import 'package:paperdoll/core/network/dio_provider.dart';
import 'package:paperdoll/features/feed/data/selector_editor_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:shared_preferences/shared_preferences.dart';

part 'selector_providers.g.dart';

@riverpod
SelectorEditorRepository selectorEditorRepository(Ref ref) =>
    SelectorEditorRepository(ref.watch(dioProvider), SharedPreferencesAsync());
