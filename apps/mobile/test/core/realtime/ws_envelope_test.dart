import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/realtime/ws_envelope.dart';

void main() {
  test('geçerli v1 zarfı çözülür', () {
    final e = WsEnvelope.tryParse({
      'v': 1,
      'type': 'game.card_revealed',
      'room_id': 'r1',
      'state_version': 42,
      'ts': '2026-11-02T18:00:00Z',
      'payload': {'category': 'health'},
    })!;

    expect(e.type, 'game.card_revealed');
    expect(e.roomId, 'r1');
    expect(e.stateVersion, 42);
    expect(e.timestamp, DateTime.utc(2026, 11, 2, 18));
    expect(e.payload, {'category': 'health'});
  });

  test('geçersiz zarflar null döner', () {
    expect(WsEnvelope.tryParse(null), isNull);
    expect(WsEnvelope.tryParse([1]), isNull);
    expect(WsEnvelope.tryParse({'v': 2, 'type': 'x'}), isNull);
    expect(WsEnvelope.tryParse({'v': 1}), isNull);
    expect(WsEnvelope.tryParse({'v': 1, 'type': ''}), isNull);
  });

  test('eksik alanlar güvenli varsayılan alır', () {
    final e = WsEnvelope.tryParse({'v': 1, 'type': 'chat.message'})!;

    expect(e.stateVersion, isNull);
    expect(e.payload, isEmpty);
  });
}
