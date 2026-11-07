import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';

import 'package:siginak/core/network/api_client.dart' show TokenReader;
import 'package:siginak/core/realtime/ws_envelope.dart';
import 'package:siginak/core/realtime/ws_socket.dart';

/// Gerçek zamanlı bağlantının durumu.
enum RealtimeStatus {
  /// Bağlantı istenmedi.
  idle,

  /// İlk bağlantı kuruluyor.
  connecting,

  /// Bağlı.
  connected,

  /// Bağlantı koptu; bekleyip yeniden denenecek.
  reconnecting,

  /// Uygulama arka planda; bağlantı maliyet için kapatıldı. Öne gelince
  /// ([RealtimeClient.onAppResumed]) aynı odaya yeniden bağlanır.
  suspended,

  /// Yeniden denenmeyecek şekilde durdu (bkz. [RealtimeStopReason]).
  stopped,
}

/// Bağlantının kalıcı olarak durma nedeni (docs/api.md §5 kapanış kodları).
enum RealtimeStopReason {
  /// 4001 — oturum geçersiz.
  unauthorized,

  /// 4003 — oda üyesi değil veya atıldı.
  forbidden,

  /// 4026 — uygulama sürümü eski.
  upgradeRequired,

  /// 4008 — aynı kullanıcının yeni bağlantısı bunun yerini aldı.
  replaced,
}

/// Sürüm akışı: oda olayları ve oyun olayları ayrı sayaç taşır.
enum VersionStream {
  room,
  game;

  /// Olay türünün akışı; sürümsüz türler için `null`.
  static VersionStream? of(String type) => type.startsWith('room.')
      ? room
      : type.startsWith('game.')
      ? game
      : null;
}

@immutable
class RealtimeState {
  const RealtimeState(this.status, {this.stopReason, this.roomId});

  final RealtimeStatus status;
  final RealtimeStopReason? stopReason;
  final String? roomId;

  @override
  bool operator ==(Object other) =>
      other is RealtimeState &&
      other.status == status &&
      other.stopReason == stopReason &&
      other.roomId == roomId;

  @override
  int get hashCode => Object.hash(status, stopReason, roomId);
}

/// Sunucu → istemci WebSocket istemcisi (ADR-0006, ADR-0007).
///
/// - Bir anda tek odaya bağlanır (`/v1/ws?room_id=...`).
/// - Token `Authorization` başlığıyla gider; **URL'ye yazılmaz**.
/// - Sürümler iki ayrı akışta izlenir (docs/api.md §5 "Zarf"): `room.*`
///   olayları oda sürümünü, `game.*` olayları oyun sürümünü taşır. Başlangıç
///   değerleri `session.snapshot` içinden gelir (`room.version`,
///   `game.state_version`). Aynı işlemin olayları aynı sürümü paylaşır
///   (ör. `member_left` + `host_changed`, `game.started` + `your_cards` +
///   `phase_changed`): bilinen sürümün +1'i yeni bir grubu açar, açık grubun
///   sürümüyle gelen olaylar da iletilir. Eski/kopya olaylar atılır; boşlukta
///   `sync` istenir (yeni snapshot gelene kadar sürümlü olay atlanır).
/// - Beklenmedik kopmalarda üstel geri çekilmeyle yeniden bağlanır
///   (0,5 sn → en fazla 10 sn, ±%20 rastgele sapma).
/// - 4001 / 4003 / 4008 / 4026 kodlarında yeniden denemez.
/// - Uygulama arka planda [backgroundGrace] kadar kalırsa bağlantıyı kapatır
///   ([RealtimeStatus.suspended]); öne gelince yeniden bağlanır ve
///   `session.snapshot` ile eşitlenir. Kısa geçişler (ör. oda kodunu
///   paylaşmak) bağlantıyı düşürmez. Neden: Cloud Run açık WebSocket varken
///   instance'ı faturalar (ADR-0011, release-checklist A13).
class RealtimeClient {
  RealtimeClient({
    required this._wsBaseUrl,
    required this._appVersion,
    required this._readToken,
    this._connector = connectIoWebSocket,
    this._onUnauthorized,
    this._onUpgradeRequired,
    double Function()? random,
    this._enableLogging = kDebugMode,
  }) : _random = random ?? math.Random().nextDouble;

  static const _initialBackoff = Duration(milliseconds: 500);
  static const _maxBackoff = Duration(seconds: 10);

  /// Arka planda bu süre geçince bağlantı kapatılır.
  static const backgroundGrace = Duration(seconds: 30);

  final Uri _wsBaseUrl;
  final String _appVersion;
  final TokenReader _readToken;
  final WsConnector _connector;
  final VoidCallback? _onUnauthorized;
  final VoidCallback? _onUpgradeRequired;
  final double Function() _random;
  final bool _enableLogging;

  final _events = StreamController<WsEnvelope>.broadcast();
  final _states = StreamController<RealtimeState>.broadcast();

  RealtimeState _state = const RealtimeState(RealtimeStatus.idle);
  String? _roomId;
  WsSocket? _socket;
  StreamSubscription<Object?>? _subscription;
  Timer? _retryTimer;
  Timer? _suspendTimer;
  int _attempt = 0;
  int _generation = 0;
  final Map<VersionStream, int?> _known = {};
  final Set<VersionStream> _openBatch = {};
  bool _awaitingSnapshot = false;
  bool _disposed = false;

  /// Doğrulanmış ve sıraya konmuş sunucu olayları.
  Stream<WsEnvelope> get events => _events.stream;

  /// Bağlantı durumu değişiklikleri.
  Stream<RealtimeState> get states => _states.stream;

  RealtimeState get state => _state;

  /// Akışın bilinen son sürümü (snapshot öncesi `null`).
  int? versionOf(VersionStream stream) => _known[stream];

  /// Odaya bağlanır. Başka bir odaya bağlıysa önce o bağlantıyı kapatır.
  void connect(String roomId) {
    _assertNotDisposed();
    final active =
        _state.status != RealtimeStatus.idle &&
        _state.status != RealtimeStatus.stopped &&
        _state.status != RealtimeStatus.suspended;
    if (_roomId == roomId && active) return;

    _teardown(closeCode: 1000);
    _roomId = roomId;
    _attempt = 0;
    _resetVersions();
    _awaitingSnapshot = true;
    _setState(RealtimeStatus.connecting);
    unawaited(_open());
  }

  /// Bağlantıyı kapatır; yeniden denenmez.
  Future<void> disconnect() async {
    _cancelSuspend();
    final socket = _socket;
    _teardown();
    _roomId = null;
    _resetVersions();
    _setState(RealtimeStatus.idle);
    await socket?.close(1000);
  }

  /// Sunucudan tam durum görüntüsü ister.
  void requestSync() {
    _awaitingSnapshot = true;
    _socket?.send(
      jsonEncode({'v': WsEnvelope.protocolVersion, 'type': 'sync'}),
    );
  }

  /// Uygulama arka plana geçtiğinde çağrılır: [backgroundGrace] sonunda hâlâ
  /// arka plandaysa bağlantıyı kapatır. Bağlantı yoksa veya durmuşsa bir şey
  /// yapmaz.
  void onAppPaused() {
    if (_roomId == null ||
        _state.status == RealtimeStatus.idle ||
        _state.status == RealtimeStatus.stopped ||
        _state.status == RealtimeStatus.suspended) {
      return;
    }
    _suspendTimer?.cancel();
    _suspendTimer = Timer(backgroundGrace, _suspend);
  }

  /// Uygulama ön plana geldiğinde çağrılır: bağlıysa eşitler, değilse
  /// beklemeden yeniden bağlanır (arka planda kapatılmışsa da).
  void onAppResumed() {
    _cancelSuspend();
    if (_roomId == null || _state.status == RealtimeStatus.stopped) return;
    if (_state.status == RealtimeStatus.connected) {
      requestSync();
      return;
    }
    _retryTimer?.cancel();
    _retryTimer = null;
    _attempt = 0;
    unawaited(_open());
  }

  Future<void> dispose() async {
    if (_disposed) return;
    await disconnect();
    _disposed = true;
    await _events.close();
    await _states.close();
  }

  // --- iç işleyiş ---

  void _suspend() {
    _suspendTimer = null;
    if (_roomId == null ||
        _state.status == RealtimeStatus.idle ||
        _state.status == RealtimeStatus.stopped) {
      return;
    }
    _log('arka planda, bağlantı kapatıldı');
    // Normal kapanış: sunucu oyuncuyu "bağlantısı koptu" olarak gösterir;
    // öne gelince yeni bağlantı snapshot ile durumu yeniden kurar.
    _teardown(closeCode: 1000);
    _attempt = 0;
    _setState(RealtimeStatus.suspended);
  }

  void _cancelSuspend() {
    _suspendTimer?.cancel();
    _suspendTimer = null;
  }

  Future<void> _open() async {
    final roomId = _roomId;
    if (roomId == null) return;
    final generation = ++_generation;

    try {
      final token = await _readToken();
      final uri = _wsBaseUrl.replace(
        path: '/v1/ws',
        queryParameters: {'room_id': roomId},
      );
      final headers = {
        'X-App-Version': _appVersion,
        if (token != null && token.isNotEmpty) 'Authorization': 'Bearer $token',
      };
      final socket = await _connector(uri, headers);

      if (generation != _generation) {
        // Bu sırada disconnect/connect çağrıldı; bu bağlantı artık gereksiz.
        await socket.close(1000);
        return;
      }
      _socket = socket;
      _attempt = 0;
      _awaitingSnapshot = true;
      _log('bağlandı');
      _setState(RealtimeStatus.connected);
      _subscription = socket.messages.listen(
        _onMessage,
        onError: (Object _) {},
        onDone: () => _onClosed(generation, socket),
      );
    } on Object catch (e) {
      if (generation != _generation) return;
      _log('bağlanamadı: ${e.runtimeType}');
      _scheduleRetry();
    }
  }

  void _onMessage(Object? data) {
    if (data is! String) return;
    final Object? json;
    try {
      json = jsonDecode(data);
    } on FormatException {
      return;
    }
    final envelope = WsEnvelope.tryParse(json);
    if (envelope == null) return;

    if (envelope.type == 'session.snapshot') {
      _applySnapshotVersions(envelope.payload);
      _awaitingSnapshot = false;
      _events.add(envelope);
      return;
    }

    final version = envelope.stateVersion;
    final stream = VersionStream.of(envelope.type);
    if (version == null || stream == null) {
      _events.add(envelope); // ör. chat.message, game.player_connection
      return;
    }
    if (_awaitingSnapshot) return; // snapshot gelince zaten güncellenecek

    final known = _known[stream];
    if (known == null || version == known + 1) {
      _known[stream] = version;
      _openBatch.add(stream);
      _deliver(envelope);
    } else if (version == known && _openBatch.contains(stream)) {
      _deliver(envelope); // aynı işlemin bir sonraki olayı
    } else if (version <= known) {
      // kopya veya snapshot'tan önce kuyruğa girmiş eski olay: yok say
    } else {
      _log('sürüm boşluğu (${stream.name}): $known → $version, sync');
      requestSync();
    }
  }

  void _deliver(WsEnvelope envelope) {
    // Lobiye dönüş: sonraki oyun sürümleri yeniden 1'den başlar.
    if (envelope.type == 'room.returned_to_lobby') {
      _known[VersionStream.game] = null;
      _openBatch.remove(VersionStream.game);
    }
    _events.add(envelope);
  }

  void _applySnapshotVersions(Map<String, Object?> payload) {
    _resetVersions();
    final room = payload['room'];
    if (room is Map<String, Object?> && room['version'] is int) {
      _known[VersionStream.room] = room['version']! as int;
    }
    final game = payload['game'];
    if (game is Map<String, Object?> && game['state_version'] is int) {
      _known[VersionStream.game] = game['state_version']! as int;
    }
  }

  void _resetVersions() {
    _known.clear();
    _openBatch.clear();
  }

  void _onClosed(int generation, WsSocket socket) {
    if (generation != _generation) return;
    _subscription = null;
    _socket = null;
    final code = socket.closeCode;
    _log('kapandı: $code');

    switch (code) {
      case 4001:
        _stop(RealtimeStopReason.unauthorized);
        _onUnauthorized?.call();
      case 4003:
        _stop(RealtimeStopReason.forbidden);
      case 4008:
        _stop(RealtimeStopReason.replaced);
      case 4026:
        _stop(RealtimeStopReason.upgradeRequired);
        _onUpgradeRequired?.call();
      default:
        _scheduleRetry();
    }
  }

  void _scheduleRetry() {
    if (_roomId == null || _disposed) return;
    final delay = backoffFor(_attempt++);
    _setState(RealtimeStatus.reconnecting);
    _retryTimer?.cancel();
    _retryTimer = Timer(delay, () {
      _retryTimer = null;
      unawaited(_open());
    });
  }

  /// `attempt` numaralı deneme için bekleme süresi (0'dan başlar).
  @visibleForTesting
  Duration backoffFor(int attempt) {
    final base = _initialBackoff * math.pow(2, math.min(attempt, 10)).toInt();
    final capped = base > _maxBackoff ? _maxBackoff : base;
    final jitter = 0.8 + 0.4 * _random(); // ±%20
    return capped * jitter;
  }

  void _stop(RealtimeStopReason reason) {
    _retryTimer?.cancel();
    _retryTimer = null;
    _setState(RealtimeStatus.stopped, stopReason: reason);
  }

  void _teardown({int? closeCode}) {
    _generation++;
    _retryTimer?.cancel();
    _retryTimer = null;
    unawaited(_subscription?.cancel());
    _subscription = null;
    final socket = _socket;
    _socket = null;
    if (closeCode != null && socket != null) {
      unawaited(socket.close(closeCode));
    }
  }

  void _setState(RealtimeStatus status, {RealtimeStopReason? stopReason}) {
    final next = RealtimeState(status, stopReason: stopReason, roomId: _roomId);
    if (next == _state) return;
    _state = next;
    if (!_states.isClosed) _states.add(next);
  }

  void _assertNotDisposed() {
    if (_disposed) throw StateError('RealtimeClient kapatıldı.');
  }

  void _log(String message) {
    // Token, başlık ve mesaj içeriği loglanmaz.
    if (_enableLogging) debugPrint('[ws] $message');
  }
}
