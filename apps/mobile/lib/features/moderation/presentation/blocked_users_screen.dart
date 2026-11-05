import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/player_avatar.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/moderation/data/moderation_repository.dart';

/// Engellediklerim; engel kaldırılınca listeden düşer.
class BlockedUsersController extends AsyncNotifier<List<BlockedUser>> {
  @override
  Future<List<BlockedUser>> build() =>
      ref.read(moderationRepositoryProvider).blocks();

  Future<void> unblock(BlockedUser user) async {
    await ref.read(moderationRepositoryProvider).unblock(user.userId);
    final list = state.value;
    if (list != null && ref.mounted) {
      state = AsyncData([
        for (final u in list)
          if (u.userId != user.userId) u,
      ]);
    }
  }
}

final blockedUsersProvider =
    AsyncNotifierProvider.autoDispose<
      BlockedUsersController,
      List<BlockedUser>
    >(BlockedUsersController.new, retry: (_, _) => null);

/// `/settings/blocks` — engellenen oyuncular (docs/ui/screen-map.md §3.5).
class BlockedUsersScreen extends ConsumerWidget {
  const BlockedUsersScreen({super.key});

  Future<void> _unblock(
    BuildContext context,
    WidgetRef ref,
    BlockedUser u,
  ) async {
    final l = context.l10n;
    try {
      await ref.read(blockedUsersProvider.notifier).unblock(u);
      if (context.mounted) {
        showAppSnackBar(context, l.blocksUnblocked(u.nickname));
      }
    } on ApiException catch (e) {
      if (!context.mounted) return;
      final message = messageForApiError(l, e);
      if (message != null) showAppSnackBar(context, message, isError: true);
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = context.l10n;
    final blocks = ref.watch(blockedUsersProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l.blocksTitle)),
      body: SafeArea(
        child: blocks.when(
          loading: () => const LoadingView(),
          error: (e, _) => ErrorView(
            message: messageForError(l, e) ?? l.errorUnknown,
            onRetry: () => ref.invalidate(blockedUsersProvider),
          ),
          data: (list) => list.isEmpty
              ? EmptyState(icon: Icons.block_rounded, message: l.blocksEmpty)
              : ListView(
                  children: [
                    for (final (i, u) in list.indexed)
                      ListTile(
                        key: ValueKey('blocked-${u.userId}'),
                        leading: PlayerAvatar(nickname: u.nickname, seat: i),
                        title: Text(u.nickname),
                        trailing: TextButton(
                          onPressed: () => unawaited(_unblock(context, ref, u)),
                          child: Text(l.blocksUnblock),
                        ),
                      ),
                  ],
                ),
        ),
      ),
    );
  }
}
