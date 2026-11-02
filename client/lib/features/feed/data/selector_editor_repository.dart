import 'package:dio/dio.dart';
import 'package:paperdoll/core/network/request_runner.dart';
import 'package:paperdoll/features/feed/domain/post_field.dart';
import 'package:paperdoll/features/feed/domain/selector_editor.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Opens, edits, commits and closes the selector editors of unstructured
/// feeds on the server, and remembers the editor of each feed on the device
/// until it ends, so that the editor can be resumed after the app is killed.
class const SelectorEditorRepository(
  final Dio _dio,
  final SharedPreferencesAsync _preferences,
) {
  /// `POST /feeds/{id}/selector-editors` → a new editor of the feed.
  Future<SelectorEditor> open(int feedId) {
    return runRequest(() async {
      final res = await _dio.post<Map<String, dynamic>>(
        '/feeds/$feedId/selector-editors',
      );
      return SelectorEditor.fromJson(res.data!);
    });
  }

  /// `GET /feeds/{id}/selector-editors/{eid}` → the editor with its stored
  /// marks.
  Future<SelectorEditor> get(int feedId, int editorId) {
    return runRequest(() async {
      final res = await _dio.get<Map<String, dynamic>>(
        '/feeds/$feedId/selector-editors/$editorId',
      );
      return SelectorEditor.fromJson(res.data!);
    });
  }

  /// `PUT /feeds/{id}/selector-editors/{eid}/marks` → replaces all marks of
  /// [editor], and returns what the selectors generated from them find.
  Future<EditorState> updateMarks(
    int feedId,
    SelectorEditor editor,
    Iterable<LabeledPost> labeledPosts,
    Set<String> excludedLinks,
  ) {
    return runRequest(() async {
      final res = await _dio.put<Map<String, dynamic>>(
        '/feeds/$feedId/selector-editors/${editor.id}/marks',
        data: {
          'labeled_posts': [
            for (final p in labeledPosts) labeledPostToJson(editor.page, p),
          ],
          'excluded_links': excludedLinks.toList(),
        },
      );
      return EditorState.fromJson(editor.page, res.data!);
    });
  }

  /// `POST /feeds/{id}/selector-editors/{eid}/commit` → saves the selectors
  /// of the editor to the feed, and deletes the editor.
  Future<void> commit(int feedId, int editorId) {
    return runRequest(
      () => _dio.post<void>('/feeds/$feedId/selector-editors/$editorId/commit'),
    );
  }

  /// `DELETE /feeds/{id}/selector-editors/{eid}` → deletes the editor without
  /// saving its selectors.
  Future<void> close(int feedId, int editorId) {
    return runRequest(
      () => _dio.delete<void>('/feeds/$feedId/selector-editors/$editorId'),
    );
  }

  /// The ID of the editor of the feed that has not ended on this device, if
  /// any.
  Future<int?> storedEditorId(int feedId) =>
      _preferences.getInt(_editorIdKey(feedId));

  /// Remembers [editorId] as the editor of the feed, or forgets the editor of
  /// the feed when it is null.
  Future<void> storeEditorId(int feedId, int? editorId) => editorId == null
      ? _preferences.remove(_editorIdKey(feedId))
      : _preferences.setInt(_editorIdKey(feedId), editorId);

  static String _editorIdKey(int feedId) => 'selector_editor.$feedId';
}
