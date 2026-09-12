import 'package:dio/dio.dart';

import 'package:paperdoll/core/error/domain_error.dart';

/// Translates transport/HTTP failures into typed [DomainError]s, attached as
/// the rejected [DioException.error] so repositories can rethrow them.
class const ErrorInterceptor() extends Interceptor {
  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    handler.reject(
      DioException(
        requestOptions: err.requestOptions,
        response: err.response,
        type: err.type,
        error: _mapError(err),
        stackTrace: err.stackTrace,
      ),
    );
  }

  DomainError _mapError(DioException err) {
    final status = err.response?.statusCode;
    final message = _extractMessage(err.response?.data);
    if (status != null) {
      if (status == 400) {
        return BadRequestError(message);
      }
      if (status == 401) {
        return UnauthorizedError(message);
      }
      if (status == 404) {
        return NotFoundError(message);
      }
      if (status == 409) {
        return ConflictError(message);
      }
      if (status == 429) {
        return TooManyRequestsError(message);
      }
      if (status >= 500) {
        return ServerError(message);
      }
    }
    switch (err.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.sendTimeout:
      case DioExceptionType.receiveTimeout:
      case DioExceptionType.connectionError:
        return const NetworkError();
      case DioExceptionType.badResponse:
      case DioExceptionType.cancel:
      case DioExceptionType.badCertificate:
      case DioExceptionType.unknown:
        return UnknownError(message);
    }
  }

  String? _extractMessage(Object? data) {
    if (data is Map<String, dynamic>) {
      final Object? message = data['message'];
      if (message is String) {
        return message;
      }
    }
    return null;
  }
}
