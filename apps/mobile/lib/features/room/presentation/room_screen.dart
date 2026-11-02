import 'package:flutter/material.dart';

import 'package:siginak/core/ui/placeholder_screen.dart';

/// `/room/:roomId` — oda kabuğu: lobi, oyun ve sonuç görünümleri
/// sunucu durumuna göre burada seçilir (bkz. docs/ui/screen-map.md §3.4).
class RoomScreen extends StatelessWidget {
  const RoomScreen({required this.roomId, super.key});

  final String roomId;

  @override
  Widget build(BuildContext context) {
    return PlaceholderScreen(
      title: 'Oda',
      task: 'ROOM-011 / PLAY',
      children: [Text('Oda: $roomId')],
    );
  }
}
