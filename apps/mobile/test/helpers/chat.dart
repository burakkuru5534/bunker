import 'package:siginak/features/chat/chat.dart';

/// Test mesajı (kimlik saniye olarak zamana da yansır).
ChatMessage msg(int id, String user, {String text = 'selam'}) => ChatMessage(
  id: id,
  userId: user,
  nickname: user,
  text: text,
  createdAt: DateTime.utc(2026, 11, 4, 10, 0, id),
);

/// `chat.message` / snapshot geçmişi JSON'u.
Map<String, Object?> chatJson(int id, String user, String nick, String text) =>
    {
      'id': id,
      'user_id': user,
      'nickname': nick,
      'text': text,
      'created_at': '2026-11-04T10:00:${id.toString().padLeft(2, '0')}Z',
    };
