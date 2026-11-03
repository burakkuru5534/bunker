import 'package:flutter/foundation.dart';

/// Sunucudan gelen WebSocket mesaj zarfı (bkz. docs/api.md §5).
///
/// `{"v":1,"type":"...","room_id":"...","state_version":42,"ts":"...","payload":{...}}`
@immutable
class WsEnvelope {
  const WsEnvelope({
    required this.type,
    required this.payload,
    this.roomId,
    this.stateVersion,
    this.timestamp,
  });

  /// Desteklenen protokol sürümü.
  static const protocolVersion = 1;

  final String type;
  final String? roomId;
  final int? stateVersion;
  final DateTime? timestamp;
  final Map<String, Object?> payload;

  /// Geçerli bir v1 zarfıysa çözer; aksi halde `null` (mesaj yok sayılır).
  static WsEnvelope? tryParse(Object? json) {
    if (json is! Map<String, Object?>) return null;
    if (json['v'] != protocolVersion) return null;
    final type = json['type'];
    if (type is! String || type.isEmpty) return null;
    final payload = json['payload'];
    final version = json['state_version'];
    final ts = json['ts'];
    return WsEnvelope(
      type: type,
      roomId: json['room_id'] as String?,
      stateVersion: version is int ? version : null,
      timestamp: ts is String ? DateTime.tryParse(ts) : null,
      payload: payload is Map<String, Object?> ? payload : const {},
    );
  }
}
