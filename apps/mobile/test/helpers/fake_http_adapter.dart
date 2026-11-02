import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';

/// Ağa çıkmadan yanıt üreten test adaptörü; gelen istekleri kaydeder.
class FakeHttpAdapter implements HttpClientAdapter {
  FakeHttpAdapter(this.handler);

  final Future<ResponseBody> Function(RequestOptions options) handler;
  final List<RequestOptions> requests = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) {
    requests.add(options);
    return handler(options);
  }

  @override
  void close({bool force = false}) {}
}

/// JSON gövdeli test yanıtı.
ResponseBody jsonResponse(
  Object? body,
  int status, {
  Map<String, List<String>> headers = const {},
}) {
  return ResponseBody.fromString(
    body == null ? '' : jsonEncode(body),
    status,
    headers: {
      Headers.contentTypeHeader: [Headers.jsonContentType],
      ...headers,
    },
  );
}

/// docs/api.md §2 biçiminde hata gövdesi.
Map<String, Object?> apiError(
  String code, {
  String requestId = 'req-1',
  Map<String, Object?>? details,
}) => {
  'error': {
    'code': code,
    'message': 'test',
    'request_id': requestId,
    'details': details ?? <String, Object?>{},
  },
};
