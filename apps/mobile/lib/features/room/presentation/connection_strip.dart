import 'dart:async';

import 'package:flutter/material.dart';

import 'package:siginak/core/realtime/realtime_client.dart';
import 'package:siginak/core/ui/connection_banner.dart';

/// Oda ekranının bağlantı şeridi (docs/ui/design-system.md `ConnectionBanner`):
///
/// - Kopunca "yeniden bağlanıyoruz" (turuncu); istemci arka planda dener.
/// - [offlineAfter] boyunca dönülemezse "Bağlantı yok" + **Tekrar dene**
///   (kırmızı); denemeler sürer, düğme beklemeyi atlar.
/// - Kopmadan sonra bağlanınca [restoredFor] boyunca "Bağlandı" (yeşil).
///
/// İlk bağlantı (oda henüz yüklenmedi) şerit değil yükleniyor görünümüdür;
/// durmuş bağlantılar (4008 vb.) oda ekranında ayrıca ele alınır.
class ConnectionStrip extends StatefulWidget {
  const ConnectionStrip({
    required this.connection,
    required this.hasRoom,
    required this.onRetry,
    this.offlineMessage,
    super.key,
  });

  static const offlineAfter = Duration(seconds: 10);
  static const restoredFor = Duration(seconds: 2);

  final RealtimeState connection;

  /// Oda en az bir kez yüklendi mi (yüklenmeden önceki bağlantı ilk bağlantıdır).
  final bool hasRoom;
  final VoidCallback onRetry;

  /// "Bağlantı yok" yerine gösterilecek metin.
  final String? offlineMessage;

  @override
  State<ConnectionStrip> createState() => _ConnectionStripState();
}

class _ConnectionStripState extends State<ConnectionStrip> {
  Timer? _offlineTimer;
  Timer? _restoredTimer;
  bool _offline = false;
  bool _restored = false;

  bool get _lost => switch (widget.connection.status) {
    RealtimeStatus.reconnecting => true,
    RealtimeStatus.connecting => widget.hasRoom,
    _ => false,
  };

  @override
  void initState() {
    super.initState();
    if (_lost) _startOfflineTimer();
  }

  @override
  void didUpdateWidget(ConnectionStrip old) {
    super.didUpdateWidget(old);
    final wasLost =
        old.connection.status == RealtimeStatus.reconnecting ||
        (old.connection.status == RealtimeStatus.connecting && old.hasRoom);
    if (_lost && _offlineTimer == null && !_offline) {
      _startOfflineTimer();
    } else if (!_lost && (wasLost || _offlineTimer != null || _offline)) {
      _offlineTimer?.cancel();
      _offlineTimer = null;
      final reconnected = widget.connection.status == RealtimeStatus.connected;
      _offline = false;
      _restored = reconnected;
      _restoredTimer?.cancel();
      if (reconnected) {
        _restoredTimer = Timer(ConnectionStrip.restoredFor, () {
          if (mounted) setState(() => _restored = false);
        });
      }
    }
  }

  void _startOfflineTimer() {
    _restoredTimer?.cancel();
    _restored = false;
    _offlineTimer = Timer(ConnectionStrip.offlineAfter, () {
      _offlineTimer = null;
      if (mounted && _lost) setState(() => _offline = true);
    });
  }

  void _retry() {
    setState(() => _offline = false);
    _offlineTimer?.cancel();
    _startOfflineTimer();
    widget.onRetry();
  }

  @override
  void dispose() {
    _offlineTimer?.cancel();
    _restoredTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = _lost
        ? (_offline
              ? ConnectionBannerState.offline
              : ConnectionBannerState.reconnecting)
        : _restored
        ? ConnectionBannerState.restored
        : ConnectionBannerState.hidden;
    return ConnectionBanner(
      state: state,
      onRetry: _retry,
      message: state == ConnectionBannerState.offline
          ? widget.offlineMessage
          : null,
    );
  }
}
