import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/game_theme.dart';

enum ConnectionBannerState {
  /// Şerit görünmez.
  hidden,

  /// "Bağlantı koptu, yeniden bağlanıyoruz…" (turuncu).
  reconnecting,

  /// "Bağlandı" — kısa süre gösterilip gizlenir (yeşil).
  restored,

  /// "Bağlantı yok" + Tekrar dene (kırmızı).
  offline,
}

/// Ekranın üstündeki bağlantı durumu şeridi.
class ConnectionBanner extends StatelessWidget {
  const ConnectionBanner({
    required this.state,
    this.onRetry,
    this.message,
    super.key,
  });

  final ConnectionBannerState state;
  final VoidCallback? onRetry;

  /// Durumun varsayılan metni yerine gösterilir (ör. oyunda daha açık uyarı).
  final String? message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final game = context.gameTheme;
    final l10n = context.l10n;

    final (String text, Color color, IconData icon)? content = switch (state) {
      ConnectionBannerState.hidden => null,
      ConnectionBannerState.reconnecting => (
        l10n.connectionReconnecting,
        theme.colorScheme.primary,
        Icons.wifi_off_rounded,
      ),
      ConnectionBannerState.restored => (
        l10n.connectionRestored,
        game.success,
        Icons.wifi_rounded,
      ),
      ConnectionBannerState.offline => (
        l10n.connectionOffline,
        game.danger,
        Icons.cloud_off_rounded,
      ),
    };

    return AnimatedSize(
      duration: AppMotion.medium,
      curve: AppMotion.enter,
      child: content == null
          ? const SizedBox(width: double.infinity)
          : Semantics(
              liveRegion: true,
              child: Container(
                width: double.infinity,
                constraints: const BoxConstraints(minHeight: 44),
                padding: const EdgeInsets.symmetric(
                  horizontal: AppSpacing.s4,
                  vertical: AppSpacing.s1,
                ),
                color: Color.alphaBlend(
                  content.$2.withValues(alpha: 0.16),
                  theme.colorScheme.surface,
                ),
                child: Row(
                  children: [
                    Icon(content.$3, size: 18, color: content.$2),
                    const SizedBox(width: AppSpacing.s2),
                    Expanded(
                      child: Text(
                        message ?? content.$1,
                        style: theme.textTheme.labelLarge?.copyWith(
                          color: content.$2,
                        ),
                      ),
                    ),
                    if (state == ConnectionBannerState.offline &&
                        onRetry != null)
                      TextButton(
                        onPressed: onRetry,
                        child: Text(l10n.actionRetry),
                      ),
                  ],
                ),
              ),
            ),
    );
  }
}
