import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/player_avatar.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/friends/data/friends_repository.dart';

/// Lobiden arkadaş davet etme sayfasını açar.
Future<void> showInviteFriendsSheet(
  BuildContext context, {
  required String roomId,
  required Set<String> memberIds,
}) => showModalBottomSheet<void>(
  context: context,
  isScrollControlled: true,
  showDragHandle: true,
  builder: (_) => InviteFriendsSheet(roomId: roomId, memberIds: memberIds),
);

/// Arkadaş listesi; her satırda "Davet et" (gönderilince "Davet edildi",
/// odadakiler için "Odada").
class InviteFriendsSheet extends ConsumerStatefulWidget {
  const InviteFriendsSheet({
    required this.roomId,
    required this.memberIds,
    super.key,
  });

  final String roomId;
  final Set<String> memberIds;

  @override
  ConsumerState<InviteFriendsSheet> createState() => _InviteFriendsSheetState();
}

class _InviteFriendsSheetState extends ConsumerState<InviteFriendsSheet> {
  final _sent = <String>{};
  String? _busy;

  Future<void> _invite(FriendPerson f) async {
    if (_busy != null) return;
    setState(() => _busy = f.userId);
    try {
      await ref.read(friendsRepositoryProvider).invite(widget.roomId, f.userId);
      if (mounted) setState(() => _sent.add(f.userId));
    } on ApiException catch (e) {
      if (!mounted) return;
      final message = messageForApiError(context.l10n, e);
      if (message != null) showAppSnackBar(context, message, isError: true);
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final data = ref.watch(friendsProvider);
    final maxHeight = MediaQuery.sizeOf(context).height * 0.7;

    return SafeArea(
      child: ConstrainedBox(
        constraints: BoxConstraints(maxHeight: maxHeight),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(
            AppSpacing.screen,
            0,
            AppSpacing.screen,
            AppSpacing.screen,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(l.roomInviteTitle, style: theme.textTheme.titleLarge),
              const SizedBox(height: AppSpacing.s1),
              Text(l.roomInviteHint, style: theme.textTheme.bodySmall),
              const SizedBox(height: AppSpacing.s3),
              Flexible(
                child: switch (data) {
                  AsyncData(:final value) when value.friends.isEmpty => Padding(
                    padding: const EdgeInsets.symmetric(
                      vertical: AppSpacing.s4,
                    ),
                    child: Text(
                      l.roomInviteEmpty,
                      key: const ValueKey('invite-empty'),
                      style: theme.textTheme.bodyMedium,
                    ),
                  ),
                  AsyncData(:final value) => ListView(
                    shrinkWrap: true,
                    children: [for (final f in value.friends) _row(l, f)],
                  ),
                  AsyncError() => ErrorView(
                    message: l.friendsLoadFailed,
                    onRetry: () => ref.invalidate(friendsProvider),
                  ),
                  _ => const Padding(
                    padding: EdgeInsets.all(AppSpacing.s5),
                    child: Center(child: CircularProgressIndicator()),
                  ),
                },
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _row(AppLocalizations l, FriendPerson f) {
    final inRoom = widget.memberIds.contains(f.userId);
    final sent = _sent.contains(f.userId);
    final Widget action;
    if (inRoom) {
      action = Text(l.roomInviteInRoom);
    } else if (sent) {
      action = Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.check_rounded, size: 18),
          const SizedBox(width: AppSpacing.s1),
          Text(l.roomInviteSent),
        ],
      );
    } else {
      action = FilledButton.tonal(
        key: ValueKey('invite-send-${f.userId}'),
        onPressed: _busy == null ? () => unawaited(_invite(f)) : null,
        child: _busy == f.userId
            ? const SizedBox.square(
                dimension: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : Text(l.roomInviteSend),
      );
    }
    return Padding(
      key: ValueKey('invite-row-${f.userId}'),
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.s1),
      child: Row(
        children: [
          PlayerAvatar(
            nickname: f.nickname,
            seat: f.userId.hashCode & 0x7fffffff,
          ),
          const SizedBox(width: AppSpacing.s3),
          Expanded(
            child: Text(
              f.nickname,
              style: Theme.of(context).textTheme.bodyLarge,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          action,
        ],
      ),
    );
  }
}
