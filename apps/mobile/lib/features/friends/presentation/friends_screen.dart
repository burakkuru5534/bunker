import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/app_text_field.dart';
import 'package:siginak/core/ui/confirm_dialog.dart';
import 'package:siginak/core/ui/player_avatar.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/friends/data/friends_repository.dart';
import 'package:siginak/features/session/session.dart';

/// `/friends` — kullanıcı adıyla arkadaş ekleme, gelen/giden istekler ve
/// arkadaş listesi.
class FriendsScreen extends ConsumerStatefulWidget {
  const FriendsScreen({super.key});

  @override
  ConsumerState<FriendsScreen> createState() => _FriendsScreenState();
}

class _FriendsScreenState extends ConsumerState<FriendsScreen> {
  final _name = TextEditingController();

  /// Süren işlem (aynı satıra iki kez basılmasın).
  String? _busy;
  String? _addError;

  FriendsRepository get _repo => ref.read(friendsRepositoryProvider);

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  void _refresh() {
    ref
      ..invalidate(friendsProvider)
      ..invalidate(inboxProvider);
  }

  Future<void> _run(String key, Future<void> Function() action) async {
    if (_busy != null) return;
    setState(() => _busy = key);
    try {
      await action();
      _refresh();
    } on ApiException catch (e) {
      if (!mounted) return;
      final message = messageForApiError(context.l10n, e);
      if (message != null) showAppSnackBar(context, message, isError: true);
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  Future<void> _add() async {
    final l = context.l10n;
    final name = _name.text.trim();
    if (name.isEmpty || _busy != null) return;
    setState(() {
      _busy = 'add';
      _addError = null;
    });
    try {
      final (outcome, user) = await _repo.request(name);
      if (!mounted) return;
      _name.clear();
      showAppSnackBar(
        context,
        outcome == FriendRequestOutcome.friends
            ? l.friendsNowFriends(user.nickname)
            : l.friendsRequestSent(user.nickname),
      );
      _refresh();
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _addError = switch (e.code) {
          'NOT_FOUND' => l.friendsNotFound,
          'VALIDATION_FAILED' when e.fieldErrors['nickname'] == 'not_allowed' =>
            l.friendsSelf,
          _ => messageForApiError(l, e),
        };
      });
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  Future<void> _remove(FriendPerson f) async {
    final l = context.l10n;
    final ok = await showConfirmDialog(
      context,
      title: l.friendsRemoveTitle(f.nickname),
      message: l.friendsRemoveMessage,
      confirmLabel: l.friendsRemove,
      destructive: true,
    );
    if (ok && mounted) {
      await _run('remove:${f.userId}', () => _repo.remove(f.userId));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final data = ref.watch(friendsProvider);

    Widget header(String text) => Padding(
      padding: const EdgeInsets.fromLTRB(0, AppSpacing.s5, 0, AppSpacing.s2),
      child: Text(text, style: theme.textTheme.titleMedium),
    );

    return Scaffold(
      appBar: AppBar(title: Text(l.friendsTitle)),
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: () async {
            _refresh();
            await ref
                .read(friendsProvider.future)
                .catchError(
                  (_) => const FriendsData(
                    friends: [],
                    incoming: [],
                    outgoing: [],
                  ),
                );
          },
          child: ListView(
            padding: const EdgeInsets.all(AppSpacing.screen),
            children: [
              AppTextField(
                key: const ValueKey('friend-name'),
                label: l.friendsAddLabel,
                hintText: l.friendsAddHint,
                controller: _name,
                maxLength: nicknameMaxLength,
                errorText: _addError,
                textInputAction: TextInputAction.send,
                onChanged: (_) {
                  if (_addError != null) setState(() => _addError = null);
                },
                onSubmitted: (_) => unawaited(_add()),
              ),
              const SizedBox(height: AppSpacing.s2),
              AppButton.primary(
                key: const ValueKey('friend-add'),
                label: l.friendsAddButton,
                icon: Icons.person_add_alt_1_rounded,
                isLoading: _busy == 'add',
                onPressed: _busy == null ? () => unawaited(_add()) : null,
              ),
              ...switch (data) {
                AsyncData(:final value) => [
                  if (value.incoming.isNotEmpty) ...[
                    header(l.friendsIncoming),
                    for (final p in value.incoming)
                      _PersonTile(
                        key: ValueKey('incoming-${p.userId}'),
                        person: p,
                        actions: [
                          TextButton(
                            onPressed: _busy == null
                                ? () => unawaited(
                                    _run(
                                      'decline:${p.userId}',
                                      () => _repo.deleteRequest(p.userId),
                                    ),
                                  )
                                : null,
                            child: Text(l.friendsDecline),
                          ),
                          FilledButton(
                            onPressed: _busy == null
                                ? () => unawaited(
                                    _run(
                                      'accept:${p.userId}',
                                      () => _repo.accept(p.userId),
                                    ),
                                  )
                                : null,
                            child: Text(l.friendsAccept),
                          ),
                        ],
                      ),
                  ],
                  header(l.friendsList),
                  if (value.friends.isEmpty)
                    Text(l.friendsEmpty, style: theme.textTheme.bodyMedium),
                  for (final p in value.friends)
                    _PersonTile(
                      key: ValueKey('friend-${p.userId}'),
                      person: p,
                      actions: [
                        IconButton(
                          tooltip: l.friendsRemove,
                          onPressed: _busy == null
                              ? () => unawaited(_remove(p))
                              : null,
                          icon: const Icon(Icons.person_remove_outlined),
                        ),
                      ],
                    ),
                  if (value.outgoing.isNotEmpty) ...[
                    header(l.friendsOutgoing),
                    for (final p in value.outgoing)
                      _PersonTile(
                        key: ValueKey('outgoing-${p.userId}'),
                        person: p,
                        actions: [
                          TextButton(
                            onPressed: _busy == null
                                ? () => unawaited(
                                    _run(
                                      'cancel:${p.userId}',
                                      () => _repo.deleteRequest(p.userId),
                                    ),
                                  )
                                : null,
                            child: Text(l.friendsCancel),
                          ),
                        ],
                      ),
                  ],
                ],
                AsyncError() => [
                  const SizedBox(height: AppSpacing.s5),
                  ErrorView(
                    message: l.friendsLoadFailed,
                    onRetry: () => ref.invalidate(friendsProvider),
                  ),
                ],
                _ => [
                  const SizedBox(height: AppSpacing.s5),
                  const Center(child: CircularProgressIndicator()),
                ],
              },
            ],
          ),
        ),
      ),
    );
  }
}

class _PersonTile extends StatelessWidget {
  const _PersonTile({required this.person, required this.actions, super.key});

  final FriendPerson person;
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) {
    final name = Row(
      children: [
        PlayerAvatar(
          nickname: person.nickname,
          seat: person.userId.hashCode & 0x7fffffff,
        ),
        const SizedBox(width: AppSpacing.s3),
        Expanded(
          child: Text(
            person.nickname,
            style: Theme.of(context).textTheme.bodyLarge,
            overflow: TextOverflow.ellipsis,
          ),
        ),
      ],
    );
    final buttons = Wrap(
      spacing: AppSpacing.s1,
      alignment: WrapAlignment.end,
      children: actions,
    );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.s1),
      child: LayoutBuilder(
        builder: (context, box) {
          // Dar ekran / büyük yazı: düğmeler adın altına iner.
          final compact =
              box.maxWidth < 360 ||
              MediaQuery.textScalerOf(context).scale(1) > 1.3;
          if (!compact) {
            return Row(
              children: [
                Expanded(child: name),
                buttons,
              ],
            );
          }
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              name,
              Align(alignment: Alignment.centerRight, child: buttons),
            ],
          );
        },
      ),
    );
  }
}
