import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/audio/sound.dart';
import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/realtime/realtime_client.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/confirm_dialog.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/chat/chat.dart';
import 'package:siginak/features/game/game.dart';
import 'package:siginak/features/moderation/moderation.dart';
import 'package:siginak/features/room/data/room_models.dart';
import 'package:siginak/features/room/data/room_repository.dart';
import 'package:siginak/features/room/presentation/connection_strip.dart';
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
      onHide: () => _controller.onAppPaused(),
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
    final inGame =
        ref.read(roomControllerProvider(widget.roomId)).room?.status ==
        RoomStatus.inGame;
    final ok = await showConfirmDialog(
      context,
      title: l.roomLeaveTitle,
      message: inGame ? l.gameLeaveMessage : l.roomLeaveMessage,
      confirmLabel: l.roomLeaveConfirm,
      destructive: true,
    );
    if (!ok || !mounted) return;
    setState(() => _leaving = true);
    await _run(_controller.leave);
    if (mounted) setState(() => _leaving = false);
  }

  void _playerMenu(String userId, String nickname) => unawaited(
    showPlayerActions(
      context,
      userId: userId,
      nickname: nickname,
      roomId: widget.roomId,
      onBlocked: () => _controller.hideChatFrom(userId),
    ),
  );

  void _openChat() {
    final provider = roomControllerProvider(widget.roomId);
    unawaited(
      showModalBottomSheet<void>(
        context: context,
        isScrollControlled: true,
        showDragHandle: true,
        builder: (context) => FractionallySizedBox(
          heightFactor: 0.85,
          child: Consumer(
            builder: (context, ref, _) {
              final s = ref.watch(provider);
              return ChatPanel(
                chat: s.chat,
                myUserId: s.myUserId,
                canWrite: s.canChat,
                onSend: (t) => unawaited(_controller.sendChat(t)),
                onRetry: (m) => unawaited(_controller.retryChat(m)),
                onSeen: _controller.markChatSeen,
              );
            },
          ),
        ),
      ),
    );
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

    // Bağlantı yokken eylemler kapalı; son bilinen durum görünür kalır
    // (docs/ui/design-system.md "Ekran durumları").
    final online = state.connection.status == RealtimeStatus.connected;
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
    } else if (state.loadFailed &&
        (room == null ||
            (room.status == RoomStatus.inGame && state.game == null))) {
      body = ErrorView(message: l.roomLoadFailed, onRetry: _controller.reload);
    } else if (room == null) {
      body = LoadingView(message: l.roomConnecting);
    } else {
      body = switch (room.status) {
        RoomStatus.lobby => LoopingMusicScope(
          factory: lobbyMusicFactoryProvider,
          child: LobbyView(
            state: state,
            onReady: (ready) => _run(() => _controller.setReady(ready: ready)),
            onCapacity: (c) => _run(() => _controller.setCapacity(c)),
            scenarios: ref.watch(scenarioOptionsProvider),
            onScenario: (id) => _run(() => _controller.setScenario(id)),
            onRetryScenarios: () => ref.invalidate(scenarioOptionsProvider),
            onKick: (userId) => _run(() => _controller.kick(userId)),
            onStart: () => _run(_controller.start),
            onMemberMenu: (m) => _playerMenu(m.userId, m.nickname),
            online: online,
          ),
        ),
        RoomStatus.inGame when state.game != null => GameBody(
          game: state.game!,
          myUserId: state.myUserId,
          onAction: (a) => _run(() => _controller.act(a)),
          onPlayerMenu: (p) => _playerMenu(p.userId, p.nickname),
          onOpenChat: _openChat,
          chatUnread: state.chat.unreadFor(state.myUserId),
          online: online,
        ),
        // Tutarsız görüntü: denetleyici yeniden ister (loadFailed'a kadar).
        RoomStatus.inGame => LoadingView(message: l.roomConnecting),
        RoomStatus.finished => _FinishedView(
          state: state,
          onRematch: () => _run(_controller.rematch),
          onGoHome: () => _run(_controller.leave),
        ),
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
          title: Text(
            room?.status == RoomStatus.inGame ? l.gameTitle : l.roomLobbyTitle,
          ),
          actions: [
            if (room?.status == RoomStatus.lobby) const SoundToggleButton(),
            // Oyunda sohbet düğmesi Kartlarım'ın yanındadır.
            if (room != null && room.status != RoomStatus.inGame)
              _ChatButton(
                unread: state.chat.unreadFor(state.myUserId),
                onPressed: _openChat,
              ),
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
              if (!replaced)
                ConnectionStrip(
                  connection: state.connection,
                  hasRoom: room != null,
                  onRetry: _controller.retryConnection,
                  offlineMessage: _aliveInGame(state)
                      ? l.connectionOfflineInGame
                      : null,
                ),
              Expanded(child: body),
            ],
          ),
        ),
      ),
    );
  }

  /// Oyunda hayattayım: uzun kopma oyundan çıkarılma demektir.
  static bool _aliveInGame(RoomState state) =>
      state.room?.status == RoomStatus.inGame &&
      (state.game?.participantOf(state.myUserId)?.isAlive ?? false);
}

/// Sonuç: `game.finished` ile geldiyse hemen; yoksa sunucudan yüklenir.
class _FinishedView extends ConsumerWidget {
  const _FinishedView({
    required this.state,
    required this.onRematch,
    required this.onGoHome,
  });

  final RoomState state;
  final Future<void> Function() onRematch;
  final Future<void> Function() onGoHome;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = context.l10n;
    Widget view(GameResult r) => ResultView(
      result: r,
      myUserId: state.myUserId,
      isHost: state.isHost,
      onRematch: onRematch,
      onGoHome: onGoHome,
    );
    final ready = state.result;
    if (ready != null) return view(ready);
    final gameId = state.room?.currentGameId;
    if (gameId == null) return _StatusMessage(text: l.roomGameFinished);
    final provider = gameResultProvider(gameId);
    return ref
        .watch(provider)
        .when(
          data: view,
          loading: () => LoadingView(message: l.resultLoading),
          error: (e, _) => ErrorView(
            message: messageForError(l, e) ?? l.errorUnknown,
            onRetry: () => ref.invalidate(provider),
          ),
        );
  }
}

class _StatusMessage extends StatelessWidget {
  const _StatusMessage({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) =>
      EmptyState(icon: Icons.hourglass_top_rounded, message: text);
}

class _ChatButton extends StatelessWidget {
  const _ChatButton({required this.unread, required this.onPressed});

  final int unread;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    return IconButton(
      tooltip: unread > 0 ? l.chatUnread(unread) : l.chatOpen,
      onPressed: onPressed,
      icon: Badge(
        isLabelVisible: unread > 0,
        label: Text(unread > 99 ? '99+' : '$unread'),
        child: const Icon(Icons.chat_bubble_outline_rounded),
      ),
    );
  }
}
