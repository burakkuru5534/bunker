import 'package:dio/dio.dart';

import 'package:siginak/core/network/api_exception.dart';

/// `DioException`'ı uygulamanın `ApiException`'ına çevirir.
ApiException mapDioException(DioException e) {
  switch (e.type) {
    case DioExceptionType.connectionTimeout:
    case DioExceptionType.sendTimeout:
    case DioExceptionType.receiveTimeout:
    case DioExceptionType.transformTimeout:
      return ApiException(kind: ApiErrorKind.timeout, debugMessage: e.message);
    case DioExceptionType.cancel:
      return const ApiException(kind: ApiErrorKind.cancelled);
    case DioExceptionType.connectionError:
    case DioExceptionType.badCertificate:
      return ApiException(kind: ApiErrorKind.network, debugMessage: e.message);
    case DioExceptionType.badResponse:
      return _fromResponse(e.response);
    case DioExceptionType.unknown:
      final error = e.error;
      if (error is ApiException) return error;
      return ApiException(
        kind: ApiErrorKind.network,
        debugMessage: e.message ?? error?.toString(),
      );
  }
}

ApiException _fromResponse(Response<Object?>? response) {
  final status = response?.statusCode;
  final body = _errorBody(response?.data);
  final code = body?['code'] as String?;
  final requestId =
      body?['request_id'] as String? ?? response?.headers.value('x-request-id');
  final message = body?['message'] as String?;

  final kind = switch (status) {
    401 => ApiErrorKind.unauthorized,
    426 => ApiErrorKind.upgradeRequired,
    429 => ApiErrorKind.rateLimited,
    _ when code != null => ApiErrorKind.server,
    _ => ApiErrorKind.invalidResponse,
  };

  return ApiException(
    kind: kind,
    statusCode: status,
    code: code,
    requestId: requestId,
    retryAfter: _retryAfter(response?.headers.value('retry-after')),
    fieldErrors: _fieldErrors(body?['details']),
    details: switch (body?['details']) {
      final Map<String, Object?> d => d,
      _ => const {},
    },
    debugMessage: message,
  );
}

Map<String, Object?>? _errorBody(Object? data) {
  if (data is Map<String, Object?>) {
    final error = data['error'];
    if (error is Map<String, Object?>) return error;
  }
  return null;
}

Duration? _retryAfter(String? value) {
  final seconds = int.tryParse(value ?? '');
  return seconds == null || seconds < 0 ? null : Duration(seconds: seconds);
}

Map<String, String> _fieldErrors(Object? details) {
  if (details is! Map<String, Object?>) return const {};
  final fields = details['fields'];
  if (fields is! Map<String, Object?>) return const {};
  return {
    for (final entry in fields.entries)
      if (entry.value is String) entry.key: entry.value! as String,
  };
}
