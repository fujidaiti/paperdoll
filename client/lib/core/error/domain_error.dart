/// Typed errors the data layer maps transport/HTTP failures into. The
/// presentation layer pattern-matches on these to choose an error surface.
sealed class const DomainError([
  /// Server-supplied message (`{ "message": ... }`), when available.
  final String? message,
]) implements Exception {
  /// Fallback shown when [message] is null.
  String get defaultMessage;

  @override
  String toString() => 'DomainError(${message ?? defaultMessage})';
}

final class const UnauthorizedError([super.message]) extends DomainError {
  @override
  String get defaultMessage => 'Session expired. Please sign in again.';
}

final class const NotFoundError([super.message]) extends DomainError {
  @override
  String get defaultMessage => 'Not found.';
}

final class const BadRequestError([super.message]) extends DomainError {
  @override
  String get defaultMessage => 'Invalid request.';
}

final class const ConflictError([super.message]) extends DomainError {
  @override
  String get defaultMessage => 'That already exists.';
}

final class const TooManyRequestsError([super.message]) extends DomainError {
  @override
  String get defaultMessage => 'Too many requests. Please wait and try again.';
}

final class const ServerError([super.message]) extends DomainError {
  @override
  String get defaultMessage => 'Server error. Please try again.';
}

final class const NetworkError([super.message]) extends DomainError {
  @override
  String get defaultMessage => 'Network error. Check your connection.';
}

final class const UnknownError([super.message]) extends DomainError {
  @override
  String get defaultMessage => 'Something went wrong.';
}

/// Best-effort human-readable text for any thrown error, used by error UIs.
String describeError(Object error) {
  if (error is DomainError) {
    return error.message ?? error.defaultMessage;
  }
  return 'Something went wrong.';
}
