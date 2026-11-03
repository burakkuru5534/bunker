import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/ui/connection_banner.dart';
import 'package:siginak/core/ui/status_badge.dart';

import '../../helpers/pump_themed.dart';

void main() {
  testWidgets('tüm rozetler etiketleriyle çizilir', (t) async {
    await pumpThemed(
      t,
      Wrap(children: [for (final k in StatusBadgeKind.values) StatusBadge(k)]),
    );

    for (final label in [
      'Hazır',
      'Bekliyor',
      'Oda sahibi',
      'İzleyici',
      'Elendi',
      'Ayrıldı',
      'Sırada',
    ]) {
      expect(find.text(label), findsOneWidget);
    }
  });

  testWidgets('bağlantı şeridi durumları', (t) async {
    var retries = 0;
    Future<void> show(ConnectionBannerState s) =>
        pumpThemed(t, ConnectionBanner(state: s, onRetry: () => retries++));

    await show(ConnectionBannerState.hidden);
    expect(find.byType(Text), findsNothing);

    await show(ConnectionBannerState.reconnecting);
    await t.pumpAndSettle();
    expect(find.text('Bağlantı koptu, yeniden bağlanıyoruz…'), findsOneWidget);
    expect(find.text('Tekrar dene'), findsNothing);

    await show(ConnectionBannerState.restored);
    await t.pumpAndSettle();
    expect(find.text('Bağlandı'), findsOneWidget);

    await show(ConnectionBannerState.offline);
    await t.pumpAndSettle();
    await t.tap(find.text('Tekrar dene'));
    expect(retries, 1);
  });
}
