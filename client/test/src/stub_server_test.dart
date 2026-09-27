import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'stub_server.dart';

void main() {
  late StubServer server;
  late Dio dio;

  setUp(() {
    server = StubServer();
    dio = Dio(BaseOptions(baseUrl: 'http://mock'))..interceptors.add(server);
  });

  test('resolves a matching GET with the stubbed body and 200', () async {
    server.stubGet('/feeds', body: {'feeds': <dynamic>[]});

    final res = await dio.get<dynamic>('/feeds');

    expect(res.statusCode, 200);
    expect(res.data, {'feeds': <dynamic>[]});
    expect(server.unmatched, isEmpty);
  });

  test('resolves a custom success status instead of throwing', () async {
    server.stubGet('/thing', status: 204, body: null);

    final res = await dio.get<dynamic>('/thing');

    expect(res.statusCode, 204);
    expect(server.unmatched, isEmpty);
  });

  test('rejects a non-2xx status with the response attached', () async {
    server.stubPost('/signin', status: 400, body: {'message': 'bad'});

    final err = await _catchDio(() => dio.post<dynamic>('/signin'));

    expect(err.type, DioExceptionType.badResponse);
    expect(err.response?.statusCode, 400);
    expect(err.response?.data, {'message': 'bad'});
    expect(server.unmatched, isEmpty);
  });

  test('matches a POST whose body equals the declared data', () async {
    server.stubPost(
      '/signin',
      body: {'ok': true},
      bodyMatcher: {'email': 'a', 'p': 1},
    );

    final res = await dio.post<dynamic>(
      '/signin',
      data: {'email': 'a', 'p': 1},
    );

    expect(res.data, {'ok': true});
    expect(server.unmatched, isEmpty);
  });

  test(
    'matches when the request body is a superset of the declared data',
    () async {
      server.stubPost(
        '/signin',
        body: {'ok': true},
        bodyMatcher: {'email': 'a'},
      );

      final res = await dio.post<dynamic>(
        '/signin',
        data: {'email': 'a', 'device': 'x'},
      );

      expect(res.data, {'ok': true});
      expect(server.unmatched, isEmpty);
    },
  );

  test('omitted data matches any body, including none', () async {
    server.stubPost('/signin', body: {'ok': true});

    final withBody = await dio.post<dynamic>('/signin', data: {'anything': 1});
    final withoutBody = await dio.post<dynamic>('/signin');

    expect(withBody.data, {'ok': true});
    expect(withoutBody.data, {'ok': true});
    expect(server.unmatched, isEmpty);
  });

  test('does not match when a declared value differs', () async {
    server.stubPost('/signin', body: {'ok': true}, bodyMatcher: {'email': 'a'});

    final err = await _catchDio(
      () => dio.post<dynamic>('/signin', data: {'email': 'b'}),
    );

    expect(err.type, DioExceptionType.unknown);
    expect(server.unmatched, ['POST /signin']);
  });

  test('does not match when the request lacks a declared key', () async {
    server.stubPost('/signin', body: {'ok': true}, bodyMatcher: {'email': 'a'});

    final err = await _catchDio(
      () => dio.post<dynamic>('/signin', data: {'password': 'p'}),
    );

    expect(err.type, DioExceptionType.unknown);
    expect(server.unmatched, ['POST /signin']);
  });

  test('records and rejects a request to a route with no stub', () async {
    final err = await _catchDio(() => dio.get<dynamic>('/nope'));

    expect(err.type, DioExceptionType.unknown);
    expect(server.unmatched, ['GET /nope']);
  });

  test('onGet answers differently as captured state changes', () async {
    var count = 0;
    server.onGet('/counter', respond: (_) => (200, {'count': ++count}));

    final first = await dio.get<dynamic>('/counter');
    final second = await dio.get<dynamic>('/counter');

    expect(first.data, {'count': 1});
    expect(second.data, {'count': 2});
    expect(server.unmatched, isEmpty);
  });

  test('an onPut side effect is reflected by a later onGet', () async {
    var subscribed = false;
    server
      ..onGet(
        '/feeds',
        respond: (_) => (
          200,
          {
            'feeds': subscribed ? ['nasa'] : <String>[],
          },
        ),
      )
      ..onPut(
        '/feeds',
        respond: (_) {
          subscribed = true;
          return (200, {'ok': true});
        },
      );

    final before = await dio.get<dynamic>('/feeds');
    final put = await dio.put<dynamic>('/feeds', data: {'url': 'x'});
    final after = await dio.get<dynamic>('/feeds');

    expect(before.data, {'feeds': <String>[]});
    expect(put.data, {'ok': true});
    expect(after.data, {
      'feeds': ['nasa'],
    });
    expect(server.unmatched, isEmpty);
  });

  test('the onPut responder receives the request body', () async {
    Object? seen;
    server.onPut(
      '/feeds',
      respond: (body) {
        seen = body;
        return (200, {'ok': true});
      },
    );

    await dio.put<dynamic>('/feeds', data: {'url': 'x'});

    expect(seen, {'url': 'x'});
    expect(server.unmatched, isEmpty);
  });

  test('a dynamic non-2xx status rejects with the response attached', () async {
    server.onPut('/feeds', respond: (_) => (409, {'message': 'conflict'}));

    final err = await _catchDio(() => dio.put<dynamic>('/feeds'));

    expect(err.type, DioExceptionType.badResponse);
    expect(err.response?.statusCode, 409);
    expect(err.response?.data, {'message': 'conflict'});
    expect(server.unmatched, isEmpty);
  });

  test('resolves a matching DELETE with 204 and no body', () async {
    server.stubDelete('/feeds/1');

    final res = await dio.delete<dynamic>('/feeds/1');

    expect(res.statusCode, 204);
    expect(server.unmatched, isEmpty);
  });

  test('an onDelete side effect is reflected by a later onGet', () async {
    var subscribed = true;
    server
      ..onGet('/feeds/1', respond: (_) => (200, {'subscribed': subscribed}))
      ..onDelete(
        '/feeds/1',
        respond: (_) {
          subscribed = false;
          return (204, null);
        },
      );

    final before = await dio.get<dynamic>('/feeds/1');
    final deleted = await dio.delete<dynamic>('/feeds/1');
    final after = await dio.get<dynamic>('/feeds/1');

    expect(before.data, {'subscribed': true});
    expect(deleted.statusCode, 204);
    expect(after.data, {'subscribed': false});
    expect(server.unmatched, isEmpty);
  });

  test('the last matching registration wins', () async {
    server.stubGet('/newspapers/today', body: {'id': 1});
    server.stubGet('/newspapers/today', body: {'id': 2});

    final res = await dio.get<dynamic>('/newspapers/today');

    expect(res.data, {'id': 2});
  });
}

/// Runs [request] and returns the [DioException] it throws, failing the test if
/// it doesn't throw one.
Future<DioException> _catchDio(Future<void> Function() request) async {
  try {
    await request();
  } on DioException catch (e) {
    return e;
  }
  fail('Expected the request to throw a DioException');
}
