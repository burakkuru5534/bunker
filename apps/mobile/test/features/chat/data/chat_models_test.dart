import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/features/chat/chat.dart';

import '../../../helpers/chat.dart';

void main() {
  test('tryParse', () {
    final m = ChatMessage.tryParse({
      'id': 7,
      'user_id': 'u1',
      'nickname': 'Ayşe',
      'text': 'selam',
      'created_at': '2026-11-04T10:00:00Z',
    });
    expect(m!.id, 7);
    expect(m.nickname, 'Ayşe');
    expect(ChatMessage.tryParse({'id': '7'}), isNull);
    expect(ChatMessage.tryParse('x'), isNull);
    expect(
      ChatMessage.tryParse({
        'id': 1,
        'user_id': 'u',
        'text': 't',
        'created_at': 'dün',
      }),
      isNull,
    );
  });

  test('birleştirme: kimliğe göre sıralı, tekrarsız, son gelen kazanır', () {
    var s = const ChatState().withHistory([msg(3, 'a'), msg(1, 'b')]);
    s = s.withMessage(msg(2, 'c'));
    s = s.withHistory([msg(1, 'b'), msg(3, 'a', text: '***')]);
    expect(s.messages.map((m) => m.id), [1, 2, 3]);
    expect(s.messages.last.text, '***');
  });

  test('bellekte en fazla 200 mesaj (en yeniler)', () {
    final s = const ChatState().withHistory([
      for (var i = 1; i <= 250; i++) msg(i, 'a'),
    ]);
    expect(s.messages, hasLength(ChatState.maxMessages));
    expect(s.messages.first.id, 51);
  });

  test('okunmamış: benimkiler sayılmaz; görüldü sıfırlar', () {
    var s = const ChatState().withHistory([
      msg(1, 'a'),
      msg(2, 'me'),
      msg(3, 'b'),
    ]);
    expect(s.unreadFor('me'), 2);
    s = s.seen();
    expect(s.unreadFor('me'), 0);
    s = s.withMessage(msg(4, 'a'));
    expect(s.unreadFor('me'), 1);
    expect(const ChatState().seen().lastSeenId, 0);
  });

  test('bekleyen mesaj durumları', () {
    const p = PendingMessage(clientMsgId: 'c1', text: 'x');
    expect(p.failed, isFalse);
    final f = p.failedWith(Exception('ağ'));
    expect(f.failed, isTrue);
    expect(f.clientMsgId, 'c1');
    expect(f.retrying().failed, isFalse);
    final s = const ChatState().withPending([p]).withMessage(msg(1, 'a'));
    expect(s.pending, [p]); // birleştirme bekleyenleri korur
  });
}
