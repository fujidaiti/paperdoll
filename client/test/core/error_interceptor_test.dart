import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:paperdoll/core/error/domain_error.dart';
import 'package:paperdoll/core/network/error_interceptor.dart';

const _message = 'Email already registered';

void main() {
  // The sign-up flow tells 409 (the address is taken) and 429 (the send
  // throttle) apart, so neither may fall through to UnknownError.
  test('Map every status the API answers with to its own error', () async {
    expect(await mapStatus(400), isA<BadRequestError>());
    expect(await mapStatus(401), isA<UnauthorizedError>());
    expect(await mapStatus(404), isA<NotFoundError>());
    expect(await mapStatus(409), isA<ConflictError>());
    expect(await mapStatus(429), isA<TooManyRequestsError>());
    expect(await mapStatus(500), isA<ServerError>());
  });

  test('Carry the server message into the mapped error', () async {
    expect((await mapStatus(409)).message, _message);
  });
}

/// Runs a request that comes back [status] through [ErrorInterceptor] and
/// returns the error it attached, which is what `runRequest` rethrows.
Future<DomainError> mapStatus(int status) async {
  final dio = Dio(BaseOptions(baseUrl: 'http://mock'))
    ..interceptors.add(const ErrorInterceptor())
    ..httpClientAdapter = _FixedStatusAdapter(status);
  try {
    await dio.get<void>('/signup');
  } on DioException catch (e) {
    return e.error! as DomainError;
  }
  fail('$status did not fail the request');
}

/// Answers every request with [status] and the error body shape the API uses.
class _FixedStatusAdapter(final int status) implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    return ResponseBody.fromString(
      jsonEncode({'message': _message}),
      status,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}
