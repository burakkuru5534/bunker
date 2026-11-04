import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/realtime/realtime_client.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/confirm_dialog.dart';
import 'package:siginak/core/ui/connection_banner.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/room/data/room_models.dart';
import 'package:siginak/features/room/presentation/lobby_view.dart';
import 'package:siginak/features/room/presentation/room_controller.dart';
import 'package:siginak/features/session/session.dart';

/// `/room/:roomId` — oda kabuğu (docs/ui/screen-map.md §3.4).
///
/// WS bağlantısını ve sunucu durumunu yönetir; içerik `room.status`'a göre
/// seçilir (lobi; oyun ve sonuç görünümleri PLAY-* görevlerinde).
class RoomScreen extends ConsumerStatefulWidget {
  const RoomScreen({required this.roomId, required this.onExit, super.key});

  final String roomId;

  /// Odadan çıkıldı (ayrılma, atılma, erişim kaybı) → ana ekrana dön.
  final VoidCallback onExit;

  @override
  ConsumerState<RoomScreen> createState() => _RoomScreenState();
}

class _RoomScreenState extends ConsumerState<RoomScreen> {
  late final AppLifecycleListener _lifecycle;
  bool _leaving = false;

  RoomController get _controller =>
      ref.read(roomControllerProvider(widget.roomId).notifier);

  @override
  void initState() {
    super.initState();
    _lifecycle = AppLifecycleListener(
      onResume: () => _controller.onAppResumed(),
    );
  }

  @override
  void dispose() {
    _lifecycle.dispose();
    super.dispose();
  }

  /// İşlemi çalıştırır; hata olursa snackbar gösterir.
  Future<void> _run(Future<void> Function() action) async {
    try {
      await action();
    } on ApiException catch (e) {
      if (!mounted) return;
      final message = messageForApiError(context.l10n, e);
      if (message != null) showAppSnackBar(context, message, isError: true);
    }
  }

  Future<void> _confirmLeave() async {
    if (_leaving) return;
    final l = context.l10n;
    final ok = await showConfirmDialog(
      context,
      title: l.roomLeaveTitle,
      message: l.roomLeaveMessage,
      confirmLabel: l.roomLeaveConfirm,
      destructive: true,
    );
    if (!ok || !mounted) return;
    setState(() => _leaving = true);
    await _run(_controller.leave);
    if (mounted) setState(() => _leaving = false);
  }

  void _onExit(RoomExit exit) {
    final l = context.l10n;
    final message = switch (exit) {
      RoomExit.left => null,
      RoomExit.kicked => l.errorKicked,
      RoomExit.forbidden => l.realtimeForbidden,
    };
    // Kök ScaffoldMessenger: mesaj ana ekranda da görünür.
    if (message != null) showAppSnackBar(context, message, isError: true);
    ref.read(sessionControllerProvider.notifier).setActiveRoom(null);
    widget.onExit();
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final provider = roomControllerProvider(widget.roomId);
    ref.listen(provider.select((s) => s.exit), (_, exit) {
      if (exit != null) _onExit(exit);
    });
    final state = ref.watch(provider);
    final room = state.room;

    final replaced =
        state.connection.status == RealtimeStatus.stopped &&
        state.connection.stopReason == RealtimeStopReason.replaced;
    final Widget body;
    if (replaced) {
      // Aynı oda başka cihazda/pencerede açıldı: bu bağlantı kendiliğinden
      // yeniden kurulmaz (aksi halde iki cihaz birbirini düşürür).
      body = ErrorView(
        message: l.realtimeReplaced,
        onRetry: _controller.reconnect,
      );
    } else if (room == null) {
      body = LoadingView(message: l.roomConnecting);
    } else {
      body = switch (room.status) {
        RoomStatus.lobby => LobbyView(
          state: state,
          onReady: (ready) => _run(() => _controller.setReady(ready: ready)),
          onCapacity: (c) => _run(() => _controller.setCapacity(c)),
          onKick: (userId) => _run(() => _controller.kick(userId)),
          onStart: () => _run(_controller.start),
        ),
        RoomStatus.inGame => _StatusMessage(text: l.roomGameStarted),
        RoomStatus.finished => _StatusMessage(text: l.roomGameFinished),
      };
    }

    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) unawaited(_confirmLeave());
      },
      child: Scaffold(
        appBar: AppBar(
          automaticallyImplyLeading: false,
          title: Text(l.roomLobbyTitle),
          actions: [
            IconButton(
              tooltip: l.roomLeave,
              onPressed: _leaving ? null : () => unawaited(_confirmLeave()),
              icon: const Icon(Icons.logout_rounded),
            ),
          ],
        ),
        body: SafeArea(
          child: Column(
            children: [
              ConnectionBanner(state: _bannerOf(state)),
              Expanded(child: body),
            ],
          ),
        ),
      ),
    );
  }

  static ConnectionBannerState _bannerOf(RoomState state) {
    final c = state.connection;
    return switch (c.status) {
      RealtimeStatus.reconnecting => ConnectionBannerState.reconnecting,
      RealtimeStatus.connecting when state.room != null =>
        ConnectionBannerState.reconnecting,
      _ => ConnectionBannerState.hidden,
    };
  }
}

class _StatusMessage extends StatelessWidget {
  const _StatusMessage({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) =>
      EmptyState(icon: Icons.hourglass_top_rounded, message: text);
}
