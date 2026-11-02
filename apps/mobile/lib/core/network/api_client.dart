import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'package:siginak/core/network/api_error_mapper.dart';
import 'package:siginak/core/network/api_exception.dart';

/// Geçerli oturum token'ını döndürür; oturum yoksa `null`.
typedef TokenReader = Future<String?> Function();

/// REST API istemcisi (bkz. docs/api.md §1).
///
/// - Tüm yollar `/v1/...` biçiminde verilir; taban adres yapılandırmadan gelir.
/// - Her isteğe `X-App-Version`, varsa `Authorization: Bearer` eklenir.
/// - Hatalar her zaman [ApiException] olarak fırlatılır.
/// - Token ve istek/yanıt gövdeleri **asla** loglanmaz.
class ApiClient {
  ApiClient({
    required Uri baseUrl,
    required String appVersion,
    required TokenReader readToken,
    VoidCallback? onUnauthorized,
    VoidCallback? onUpgradeRequired,
    HttpClientAdapter? adapter,
    bool enableLogging = kDebugMode,
  }) : _dio = Dio(
         BaseOptions(
           baseUrl: baseUrl.toString(),
           connectTimeout: const Duration(seconds: 10),
           sendTimeout: const Duration(seconds: 10),
           receiveTimeout: const Duration(seconds: 15),
           contentType: Headers.jsonContentType,
           responseType: ResponseType.json,
           headers: {'Accept': 'application/json', 'X-App-Version': appVersion},
         ),
       ) {
    if (adapter != null) _dio.httpClientAdapter = adapter;
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          final token = await readToken();
          if (token != null && token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          options.extra['startedAt'] = DateTime.now();
          handler.next(options);
        },
        onResponse: (response, handler) {
          if (enableLogging) _log(response.requestOptions, response.statusCode);
          handler.next(response);
        },
        onError: (error, handler) {
          if (enableLogging) {
            _log(error.requestOptions, error.response?.statusCode);
          }
          switch (error.response?.statusCode) {
            case 401:
              onUnauthorized?.call();
            case 426:
              onUpgradeRequired?.call();
          }
          handler.next(error);
        },
      ),
    );
  }

  final Dio _dio;

  Future<Object?> get(String path, {Map<String, Object?>? query}) =>
      _send(() => _dio.get<Object?>(path, queryParameters: query));

  Future<Object?> post(String path, {Object? body}) =>
      _send(() => _dio.post<Object?>(path, data: body));

  Future<Object?> patch(String path, {Object? body}) =>
      _send(() => _dio.patch<Object?>(path, data: body));

  Future<Object?> delete(String path) =>
      _send(() => _dio.delete<Object?>(path));

  /// İsteği çalıştırır; 204/boş gövde için `null`, aksi halde çözülmüş JSON.
  Future<Object?> _send(Future<Response<Object?>> Function() request) async {
    try {
      final response = await request();
      final data = response.data;
      if (data is String && data.isEmpty) return null;
      return data;
    } on DioException catch (e) {
      throw mapDioException(e);
    } on ApiException {
      rethrow;
    } on Object catch (e) {
      throw ApiException(
        kind: ApiErrorKind.invalidResponse,
        debugMessage: e.toString(),
      );
    }
  }

  void close() => _dio.close(force: true);

  static void _log(RequestOptions options, int? status) {
    final startedAt = options.extra['startedAt'];
    final ms = startedAt is DateTime
        ? DateTime.now().difference(startedAt).inMilliseconds
        : null;
    // Yalnızca yöntem, yol, durum ve süre: başlık, token, gövde loglanmaz.
    debugPrint(
      '[api] ${options.method} ${options.path} → ${status ?? '-'}'
      '${ms != null ? ' (${ms}ms)' : ''}',
    );
  }
}
