import 'dart:async';
import 'dart:convert';

import 'package:siginak/core/realtime/ws_socket.dart';

/// Test için sahte WebSocket.
class FakeWsSocket implements WsSocket {
  final _incoming = StreamController<Object?>();
  final List<String> sent = [];
  int? _closeCode;
  bool closedByClient = false;

  @override
  Stream<Object?> get messages => _incoming.stream;

  @override
  void send(String data) => sent.add(data);

  @override
  Future<void> close([int? code, String? reason]) async {
    closedByClient = true;
    _closeCode ??= code;
    if (!_incoming.isClosed) await _incoming.close();
  }

  @override
  int? get closeCode => _closeCode;

  /// Sunucudan v1 zarfı gönderir.
  void serverSend(String type, {int? stateVersion, Object? payload}) {
    _incoming.add(
      jsonEncode({
        'v': 1,
        'type': type,
        'room_id': 'r',
        'state_version': ?stateVersion,
        'ts': '2026-11-02T18:00:00Z',
        'payload': payload ?? <String, Object?>{},
      }),
    );
  }

  void serverSendRaw(Object? data) => _incoming.add(data);

  /// Sunucu bağlantıyı belirtilen kodla kapatır.
  void serverClose(int code) {
    _closeCode = code;
    unawaited(_incoming.close());
  }
}

/// Bağlantı isteklerini kaydeden sahte bağlayıcı.
class FakeWsConnector {
  final List<Uri> uris = [];
  final List<Map<String, String>> headers = [];
  final List<FakeWsSocket> sockets = [];

  /// `true` ise bir sonraki bağlantı denemesi başarısız olur.
  int failNext = 0;

  Future<WsSocket> call(Uri uri, Map<String, String> h) async {
    uris.add(uri);
    headers.add(h);
    if (failNext > 0) {
      failNext--;
      throw Exception('bağlanılamadı');
    }
    final socket = FakeWsSocket();
    sockets.add(socket);
    return socket;
  }

  FakeWsSocket get last => sockets.last;
}
