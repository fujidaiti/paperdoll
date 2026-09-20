import 'dart:convert';

import 'package:collection/collection.dart';
import 'package:dio/dio.dart';
import 'package:openapi/api.dart' as api;

import 'fixture.dart';

/// Computes a `(status, responseBody)` for a matched request from its
/// already-decoded request body. Evaluated at request time, so it may read or
/// mutate state a test captured to answer differently across calls.
typedef StubResponder = (int status, Object? body) Function(
  Object? requestBody,
);

/// A Dio interceptor that answers registered routes with canned responses and
/// records any request no route matched. When several routes match the same
/// request, the last registered one wins.
class StubServer() extends Interceptor {
  /// A server pre-stubbed with the three shell tabs populated from [fixture]:
  /// enough for any test to boot and navigate without stubbing anything itself.
  /// Only the list endpoints the shell loads on boot are answered here; nested
  /// resources (a feed's timeline, a story's entry) are fetched on navigation,
  /// so a test that drills into a tab still stubs those itself — and overrides
  /// any tab it needs in a specific state (the last registration wins).
  factory withDefaultResponses() {
    return StubServer()
      ..stubGet(
        '/newspapers/today',
        body: api.GetTodaysNewspaper200Response(
          id: 1,
          publishedAt: DateTime.utc(2026, 7, 1),
          stories: [fixture.stories.nuclearDeal],
        ).toJson(),
      )
      ..stubGet(
        '/reading-list',
        body: api.GetReadingList200Response(
          items: [
            fixture.readingList.buildingEffectiveAgents,
            fixture.readingList.nuclearDeal,
          ],
        ).toJson(),
      )
      ..stubGet(
        '/feeds',
        body: api.GetFeeds200Response(
          feeds: [
            fixture.feeds.bbcNews,
            fixture.feeds.nasa,
            fixture.feeds.stackOverflow,
            fixture.feeds.wikipedia,
          ],
        ).toJson(),
      );
  }

  final _routes = <_Route>[];

  /// `'METHOD /path'` for every request no registered route matched.
  final unmatched = <String>[];

  void stubGet(String path, {int status = 200, Object? body}) =>
      _routes.add(_Route('GET', path, null, (_) => (status, body)));

  void stubPost(
    String path, {
    int status = 200,
    Object? body,
    Object? bodyMatcher,
  }) => _routes.add(_Route('POST', path, bodyMatcher, (_) => (status, body)));

  void stubPut(
    String path, {
    int status = 200,
    Object? body,
    Object? bodyMatcher,
  }) => _routes.add(_Route('PUT', path, bodyMatcher, (_) => (status, body)));

  void stubPatch(
    String path, {
    int status = 200,
    Object? body,
    Object? bodyMatcher,
  }) => _routes.add(_Route('PATCH', path, bodyMatcher, (_) => (status, body)));

  /// Like [stubGet], but computes the response per request via [respond] so it
  /// can vary with state a test captured (e.g. answer differently before and
  /// after a [onPut] mutates that state).
  void onGet(String path, {required StubResponder respond}) =>
      _routes.add(_Route('GET', path, null, respond));

  /// Like [stubPost], but computes the response per request via [respond]. The
  /// responder receives the request body, so it can both capture what the app
  /// sent and answer differently across calls (see [onGet]).
  void onPost(
    String path, {
    required StubResponder respond,
    Object? bodyMatcher,
  }) => _routes.add(_Route('POST', path, bodyMatcher, respond));

  /// Like [stubPut], but computes the response per request via [respond]. The
  /// responder receives the request body and may mutate captured state (see
  /// [onGet]).
  void onPut(
    String path, {
    required StubResponder respond,
    Object? bodyMatcher,
  }) => _routes.add(_Route('PUT', path, bodyMatcher, respond));

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    final route = _routes.reversed.firstWhereOrNull(
      (r) =>
          r.method == options.method &&
          r.path == options.uri.path &&
          r.matchesBody(options.data),
    );

    if (route == null) {
      unmatched.add('${options.method} ${options.uri.path}');
      // Reject (rather than hang) so the app doesn't wait forever; the
      // teardown assertion in pumpApp — not this rejection — fails the test.
      return handler.reject(
        DioException(requestOptions: options, type: DioExceptionType.unknown),
        true,
      );
    }

    final (status, body) = route.responder(options.data);
    final response = Response<dynamic>(
      requestOptions: options,
      statusCode: status,
      data: _asTransportJson(body),
    );
    if (options.validateStatus(status)) {
      handler.resolve(response);
    } else {
      handler.reject(
        DioException(
          requestOptions: options,
          response: response,
          type: DioExceptionType.badResponse,
        ),
        // Run the app's ErrorInterceptor so the status maps to a DomainError.
        true,
      );
    }
  }
}

/// Mimics real transport: over the wire the response body is JSON-encoded and
/// re-decoded, so nested generated models (e.g. a `Feed` inside a
/// `GetFeeds200Response`) arrive as plain `Map`s. Generated `toJson()` is
/// shallow — it leaves list/object members as live objects — so a round-trip
/// here is what turns them into the `Map`/`List` tree `fromJson` expects.
Object? _asTransportJson(Object? body) =>
    body == null ? null : jsonDecode(jsonEncode(body));

class _Route(
  final String method,
  final String path,
  final Object? bodyMatcher,
  final StubResponder responder,
) {
  /// Whether a request carrying [actual] as its body matches this route.
  /// A `null` [bodyMatcher] matches any body; a `Map` matches as a top-level
  /// subset (extra keys ignored); anything else is compared with deep equality.
  bool matchesBody(Object? actual) {
    final expected = bodyMatcher;
    if (expected == null) {
      return true;
    }
    if (expected is Map && actual is Map) {
      return expected.entries.every(
        (e) =>
            actual.containsKey(e.key) &&
            const DeepCollectionEquality().equals(actual[e.key], e.value),
      );
    }
    return const DeepCollectionEquality().equals(expected, actual);
  }
}
