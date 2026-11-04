import 'package:flutter/material.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/chat/data/chat_models.dart';

/// Mesaj uzunluk sınırı (docs/api.md §3).
const chatMaxLength = 200;

/// Sohbet paneli (docs/ui/screen-map.md §3.4 "Sohbet"): mesajlar, bekleyen
/// kendi mesajlarım ve yazma alanı (izleyicide not).
class ChatPanel extends StatefulWidget {
  const ChatPanel({
    required this.chat,
    required this.myUserId,
    required this.canWrite,
    required this.onSend,
    required this.onRetry,
    required this.onSeen,
    super.key,
  });

  final ChatState chat;
  final String? myUserId;

  /// `false` → yazma alanı yerine "İzleyiciler yazamaz".
  final bool canWrite;
  final ValueChanged<String> onSend;
  final ValueChanged<PendingMessage> onRetry;

  /// Panel açıkken yeni mesajlar görülmüş sayılır.
  final VoidCallback onSeen;

  @override
  State<ChatPanel> createState() => _ChatPanelState();
}

class _ChatPanelState extends State<ChatPanel> {
  final _text = TextEditingController();
  final _scroll = ScrollController();

  @override
  void initState() {
    super.initState();
    _text.addListener(() => setState(() {}));
    _afterFrame();
  }

  @override
  void didUpdateWidget(ChatPanel old) {
    super.didUpdateWidget(old);
    final grew =
        widget.chat.messages.length != old.chat.messages.length ||
        widget.chat.pending.length != old.chat.pending.length;
    if (grew) _afterFrame();
  }

  /// Yeni mesajda en alta kaydır ve görüldü işaretle.
  void _afterFrame() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (widget.chat.unreadFor(widget.myUserId) > 0 ||
          widget.chat.messages.isNotEmpty &&
              widget.chat.messages.last.id > widget.chat.lastSeenId) {
        widget.onSeen();
      }
      if (_scroll.hasClients) {
        _scroll.jumpTo(_scroll.position.maxScrollExtent);
      }
    });
  }

  @override
  void dispose() {
    _text.dispose();
    _scroll.dispose();
    super.dispose();
  }

  void _send() {
    final text = _text.text.trim();
    if (text.isEmpty) return;
    widget.onSend(text);
    _text.clear();
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final chat = widget.chat;
    final empty = chat.messages.isEmpty && chat.pending.isEmpty;

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: AppSpacing.screen),
          child: Row(
            children: [Text(l.chatTitle, style: theme.textTheme.titleLarge)],
          ),
        ),
        Expanded(
          child: empty
              ? EmptyState(
                  icon: Icons.chat_bubble_outline_rounded,
                  message: l.chatEmpty,
                )
              : ListView(
                  controller: _scroll,
                  padding: const EdgeInsets.all(AppSpacing.s3),
                  children: [
                    for (final m in chat.messages)
                      _Bubble(
                        key: ValueKey('msg-${m.id}'),
                        mine: m.userId == widget.myUserId,
                        nickname: m.nickname,
                        text: m.text,
                        time: m.createdAt.toLocal(),
                      ),
                    for (final p in chat.pending)
                      _Bubble(
                        key: ValueKey('pending-${p.clientMsgId}'),
                        mine: true,
                        nickname: '',
                        text: p.text,
                        status: p.failed
                            ? '${_failureText(l, p.error!)} ${l.chatRetryHint}'
                            : l.chatSending,
                        failed: p.failed,
                        onTap: p.failed ? () => widget.onRetry(p) : null,
                      ),
                  ],
                ),
        ),
        const Divider(height: 1),
        Padding(
          padding: EdgeInsets.fromLTRB(
            AppSpacing.s3,
            AppSpacing.s2,
            AppSpacing.s2,
            AppSpacing.s2 + MediaQuery.viewInsetsOf(context).bottom,
          ),
          child: widget.canWrite
              ? Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: _text,
                        maxLength: chatMaxLength,
                        minLines: 1,
                        maxLines: 4,
                        textInputAction: TextInputAction.send,
                        onSubmitted: (_) => _send(),
                        decoration: InputDecoration(
                          hintText: l.chatHint,
                          counterText: '',
                          isDense: true,
                        ),
                      ),
                    ),
                    IconButton(
                      tooltip: l.chatSend,
                      onPressed: _text.text.trim().isEmpty ? null : _send,
                      icon: const Icon(Icons.send_rounded),
                    ),
                  ],
                )
              : Padding(
                  padding: const EdgeInsets.all(AppSpacing.s2),
                  child: Text(
                    l.chatSpectatorNote,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ),
        ),
      ],
    );
  }

  static String _failureText(AppLocalizations l, Object error) =>
      messageForError(l, error) ?? l.errorUnknown;
}

class _Bubble extends StatelessWidget {
  const _Bubble({
    required this.mine,
    required this.nickname,
    required this.text,
    this.time,
    this.status,
    this.failed = false,
    this.onTap,
    super.key,
  });

  final bool mine;
  final String nickname;
  final String text;
  final DateTime? time;
  final String? status;
  final bool failed;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final t = time;
    final meta = [
      if (!mine && nickname.isNotEmpty) nickname,
      if (t != null)
        '${t.hour.toString().padLeft(2, '0')}:${t.minute.toString().padLeft(2, '0')}',
    ].join(' · ');
    return Align(
      alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 300),
        child: GestureDetector(
          onTap: onTap,
          child: Container(
            margin: const EdgeInsets.only(bottom: AppSpacing.s2),
            padding: const EdgeInsets.symmetric(
              horizontal: AppSpacing.s3,
              vertical: AppSpacing.s2,
            ),
            decoration: BoxDecoration(
              color: mine
                  ? scheme.primaryContainer
                  : scheme.surfaceContainerHigh,
              borderRadius: BorderRadius.circular(AppRadius.m),
              border: failed ? Border.all(color: scheme.error) : null,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (meta.isNotEmpty)
                  Text(meta, style: theme.textTheme.labelSmall),
                Text(text, style: theme.textTheme.bodyMedium),
                if (status != null)
                  Text(
                    status!,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: failed ? scheme.error : scheme.onSurfaceVariant,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
