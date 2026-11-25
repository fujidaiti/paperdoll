import 'package:dio/dio.dart';
import 'package:paperdoll/core/network/request_runner.dart';
import 'package:paperdoll/features/feed/domain/post_field.dart';
import 'package:paperdoll/features/feed/domain/selector_editor.dart';
import 'package:paperdoll/features/feed/domain/simplified_page.dart';
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

  /// `GET /feed-selector-editors/{id}` → the editor with its stored marks.
  Future<SelectorEditor> get(int editorId) {
    return runRequest(() async {
      final res = await _dio.get<Map<String, dynamic>>(
        '/feed-selector-editors/$editorId',
      );
      return SelectorEditor.fromJson(res.data!);
    });
  }

  /// `PUT /feed-selector-editors/{id}/marks` → replaces all marks of
  /// [editor], and returns what the selectors generated from them find.
  Future<EditorState> updateMarks(
    SelectorEditor editor,
    Iterable<MarkedPost> markedPosts,
    Set<String> excludedLinks,
    Set<Node> excludedNodes,
  ) {
    return runRequest(() async {
      final res = await _dio.put<Map<String, dynamic>>(
        '/feed-selector-editors/${editor.id}/marks',
        data: {
          'marked_posts': [
            for (final p in markedPosts) markedPostToJson(editor.page, p),
          ],
          'excluded_links': excludedLinks.toList(),
          'excluded_nodes': [
            for (final n in excludedNodes) editor.page.idOf(n),
          ],
        },
      );
      return EditorState.fromJson(editor.page, res.data!);
    });
  }

  /// `POST /feed-selector-editors/{id}/commit` → saves the selectors of the
  /// editor to its feed, and deletes the editor.
  Future<void> commit(int editorId) {
    return runRequest(
      () => _dio.post<void>('/feed-selector-editors/$editorId/commit'),
    );
  }

  /// `DELETE /feed-selector-editors/{id}` → deletes the editor without saving
  /// its selectors.
  Future<void> close(int editorId) {
    return runRequest(
      () => _dio.delete<void>('/feed-selector-editors/$editorId'),
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
