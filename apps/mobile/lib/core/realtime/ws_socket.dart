import 'dart:async';
import 'dart:io';

/// WebSocket bağlantısının test edilebilir soyutlaması.
abstract interface class WsSocket {
  /// Sunucudan gelen ham mesajlar; bağlantı kapanınca akış biter.
  Stream<Object?> get messages;

  void send(String data);

  Future<void> close([int? code, String? reason]);

  /// Bağlantı kapandıktan sonra kapanış kodu (yoksa `null`).
  int? get closeCode;
}

/// Verilen adrese ve başlıklarla bağlanır.
typedef WsConnector = Future<WsSocket> Function(
  Uri uri,
  Map<String, String> headers,
);

/// `dart:io` tabanlı gerçek bağlantı.
Future<WsSocket> connectIoWebSocket(
  Uri uri,
  Map<String, String> headers,
) async {
  // Soket _IoWsSocket'e sarılıp döndürülür; kapatma çağıranın sorumluluğunda.
  // ignore: close_sinks
  final socket = await WebSocket.connect(
    uri.toString(),
    headers: headers,
  ).timeout(const Duration(seconds: 10));
  // Sessizce kopan bağlantıları yakalamak için istemci de ping gönderir.
  socket.pingInterval = const Duration(seconds: 30);
  return _IoWsSocket(socket);
}

class _IoWsSocket implements WsSocket {
  _IoWsSocket(this._socket);

  final WebSocket _socket;

  @override
  Stream<Object?> get messages => _socket;

  @override
  void send(String data) => _socket.add(data);

  @override
  Future<void> close([int? code, String? reason]) =>
      _socket.close(code, reason);

  @override
  int? get closeCode => _socket.closeCode;
}
