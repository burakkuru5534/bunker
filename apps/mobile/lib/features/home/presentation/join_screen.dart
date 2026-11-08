import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/invite_code_input.dart';
import 'package:siginak/features/home/presentation/home_screen.dart'
    show existingRoomOf;
import 'package:siginak/features/room/room.dart';
import 'package:siginak/features/session/session.dart';

/// `/join` — davet koduyla odaya katılma (docs/ui/screen-map.md §3.3).
class JoinScreen extends ConsumerStatefulWidget {
  const JoinScreen({required this.onOpenRoom, this.initialCode, super.key});

  /// Davetten gelindiyse hazır kod; geçerliyse ekran açılır açılmaz katılır.
  final String? initialCode;

  /// Katılındı (veya kullanıcı zaten bir odadaydı) → oda ekranına geç.
  final ValueChanged<String> onOpenRoom;

  @override
  ConsumerState<JoinScreen> createState() => _JoinScreenState();
}

class _JoinScreenState extends ConsumerState<JoinScreen> {
  final _code = TextEditingController();
  bool _joining = false;
  ApiException? _error;

  @override
  void initState() {
    super.initState();
    final initial = widget.initialCode?.trim().toUpperCase() ?? '';
    if (initial.length == inviteCodeLength) {
      _code.text = initial;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) unawaited(_join());
      });
    }
    _code.addListener(_onCodeChanged);
  }

  @override
  void dispose() {
    _code
      ..removeListener(_onCodeChanged)
      ..dispose();
    super.dispose();
  }

  void _onCodeChanged() {
    if (_error != null) setState(() => _error = null);
    setState(() {}); // "Katıl" düğmesinin etkinliği
  }

  bool get _complete => _code.text.length == inviteCodeLength;

  void _openRoom(String roomId) {
    ref.read(sessionControllerProvider.notifier).setActiveRoom(roomId);
    widget.onOpenRoom(roomId);
  }

  Future<void> _join() async {
    if (!_complete || _joining) return;
    setState(() {
      _joining = true;
      _error = null;
    });
    try {
      final room = await ref.read(roomRepositoryProvider).join(_code.text);
      if (mounted) _openRoom(room.id);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
    } finally {
      if (mounted) setState(() => _joining = false);
    }
  }

  Future<void> _changeNicknameAndRetry() async {
    final changed = await showChangeNicknameDialog(context);
    if (changed && mounted) await _join();
  }

  String? _message(AppLocalizations l) {
    final e = _error;
    if (e == null) return null;
    if (e.code == 'VALIDATION_FAILED' && e.fieldErrors['invite_code'] != null) {
      return l.joinInvalidCode;
    }
    return messageForApiError(l, e);
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final message = _message(l);
    final existing = _error == null ? null : existingRoomOf(_error!);
    final nicknameTaken = _error?.code == 'NICKNAME_TAKEN_IN_ROOM';

    return Scaffold(
      appBar: AppBar(title: Text(l.homeJoinRoom)),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(AppSpacing.screen),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 480),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(l.joinHint, style: theme.textTheme.bodyLarge),
                  const SizedBox(height: AppSpacing.s5),
                  InviteCodeInput(
                    controller: _code,
                    autofocus: widget.initialCode == null,
                    hasError: message != null,
                    onCompleted: (_) => unawaited(_join()),
                  ),
                  if (message != null) ...[
                    const SizedBox(height: AppSpacing.s3),
                    Text(
                      message,
                      key: const ValueKey('join-error'),
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.error,
                      ),
                    ),
                  ],
                  if (nicknameTaken) ...[
                    const SizedBox(height: AppSpacing.s3),
                    AppButton.secondary(
                      label: l.nicknameDialogTitle,
                      icon: Icons.edit_outlined,
                      onPressed: () => unawaited(_changeNicknameAndRetry()),
                    ),
                  ],
                  if (existing != null) ...[
                    const SizedBox(height: AppSpacing.s3),
                    AppButton.secondary(
                      label: l.homeBackToRoom,
                      icon: Icons.meeting_room_outlined,
                      onPressed: () => _openRoom(existing),
                    ),
                  ],
                  const SizedBox(height: AppSpacing.s5),
                  AppButton.primary(
                    label: l.joinSubmit,
                    isLoading: _joining,
                    onPressed: _complete ? () => unawaited(_join()) : null,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
