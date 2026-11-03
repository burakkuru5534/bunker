import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/theme/game_theme.dart';
import 'package:siginak/core/ui/player_avatar.dart';

import '../../helpers/pump_themed.dart';

void main() {
  test('baş harf Türkçe kurallarıyla büyük harf', () {
    expect(PlayerAvatar.initialOf('ilker'), 'İ');
    expect(PlayerAvatar.initialOf('ışık'), 'I');
    expect(PlayerAvatar.initialOf('  şule'), 'Ş');
    expect(PlayerAvatar.initialOf(''), '?');
  });

  testWidgets('boyut, renk ve erişilebilirlik etiketi', (t) async {
    await pumpThemed(
      t,
      const PlayerAvatar(
        nickname: 'Ayşe',
        seat: 9,
        size: PlayerAvatarSize.large,
      ),
    );

    expect(t.getSize(find.byType(PlayerAvatar)), const Size(64, 64));
    expect(find.text('A'), findsOneWidget);
    expect(find.bySemanticsLabel('Ayşe'), findsOneWidget);
    final box = t.widget<Container>(
      find.descendant(
        of: find.byType(PlayerAvatar),
        matching: find.byWidgetPredicate(
          (w) => w is Container && w.decoration is BoxDecoration,
        ),
      ),
    );
    expect(
      (box.decoration! as BoxDecoration).color,
      GameTheme.dark.avatarColorFor(9),
    );
  });

  testWidgets('bağlantı koptu ve elendi durumları', (t) async {
    await pumpThemed(
      t,
      const Column(
        children: [
          PlayerAvatar(
            nickname: 'Can',
            seat: 1,
            presence: PlayerPresence.disconnected,
          ),
          PlayerAvatar(
            nickname: 'Deniz',
            seat: 2,
            presence: PlayerPresence.eliminated,
          ),
        ],
      ),
    );

    expect(
      find.byKey(const ValueKey('avatar-disconnected-dot')),
      findsOneWidget,
    );
    expect(
      find.byKey(const ValueKey('avatar-eliminated-strike')),
      findsOneWidget,
    );
    expect(find.bySemanticsLabel('Can, bağlantısı koptu'), findsOneWidget);
    expect(find.bySemanticsLabel('Deniz, elendi'), findsOneWidget);
  });
}
