import 'package:flutter/foundation.dart';

/// Sohbet mesajı (docs/api.md `chat.message`).
@immutable
class ChatMessage {
  const ChatMessage({
    required this.id,
    required this.userId,
    required this.nickname,
    required this.text,
    required this.createdAt,
  });

  final int id;
  final String userId;
  final String nickname;

  /// Sunucuda temizlenmiş ve küfürleri maskelenmiş metin.
  final String text;
  final DateTime createdAt;

  /// Bozuk veride `null` (mesaj atlanır).
  static ChatMessage? tryParse(Object? json) {
    if (json is! Map<String, Object?>) return null;
    final id = json['id'];
    final user = json['user_id'];
    final text = json['text'];
    final at = json['created_at'] is String
        ? DateTime.tryParse(json['created_at']! as String)
        : null;
    if (id is! int || user is! String || text is! String || at == null) {
      return null;
    }
    final nick = json['nickname'];
    return ChatMessage(
      id: id,
      userId: user,
      nickname: nick is String ? nick : '',
      text: text,
      createdAt: at,
    );
  }

  @override
  bool operator ==(Object other) =>
      other is ChatMessage &&
      other.id == id &&
      other.userId == userId &&
      other.nickname == nickname &&
      other.text == text &&
      other.createdAt == createdAt;

  @override
  int get hashCode => Object.hash(id, userId, nickname, text, createdAt);
}

/// Gönderilmekte olan (veya gönderilemeyen) kendi mesajım.
@immutable
class PendingMessage {
  const PendingMessage({
    required this.clientMsgId,
    required this.text,
    this.error,
  });

  /// İdempotency anahtarı: yeniden denemede aynı kimlik gider.
  final String clientMsgId;
  final String text;

  /// Gönderilemediyse hata (yeniden denenebilir); sürüyorsa `null`.
  final Object? error;

  bool get failed => error != null;

  PendingMessage failedWith(Object e) =>
      PendingMessage(clientMsgId: clientMsgId, text: text, error: e);

  PendingMessage retrying() =>
      PendingMessage(clientMsgId: clientMsgId, text: text);
}

/// Sohbet durumu: sunucudan gelen mesajlar (kimliğe göre sıralı, tekrarsız,
/// en fazla [maxMessages]) + bekleyen kendi mesajlarım.
@immutable
class ChatState {
  const ChatState({
    this.messages = const [],
    this.pending = const [],
    this.lastSeenId = 0,
  });

  /// Bellekte tutulan en fazla mesaj (sunucu geçmişi 50).
  static const maxMessages = 200;

  final List<ChatMessage> messages;
  final List<PendingMessage> pending;

  /// Kullanıcının gördüğü son mesaj kimliği (okunmamış rozeti).
  final int lastSeenId;

  /// Benden olmayan okunmamış mesajlar.
  int unreadFor(String? me) =>
      messages.where((m) => m.id > lastSeenId && m.userId != me).length;

  /// Snapshot geçmişi: mevcut listeyle birleştirir (yeniden bağlanmada
  /// arada gelenler kaybolmaz, tekrar edenler bir kez kalır).
  ChatState withHistory(Iterable<ChatMessage> history) => _merge(history);

  ChatState withMessage(ChatMessage m) => _merge([m]);

  ChatState _merge(Iterable<ChatMessage> incoming) {
    final byId = {for (final m in messages) m.id: m};
    for (final m in incoming) {
      byId[m.id] = m;
    }
    final list = byId.values.toList()..sort((a, b) => a.id.compareTo(b.id));
    final trimmed = list.length > maxMessages
        ? list.sublist(list.length - maxMessages)
        : list;
    return ChatState(
      messages: List.unmodifiable(trimmed),
      pending: pending,
      lastSeenId: lastSeenId,
    );
  }

  ChatState withPending(List<PendingMessage> pending) => ChatState(
    messages: messages,
    pending: List.unmodifiable(pending),
    lastSeenId: lastSeenId,
  );

  ChatState seen() => ChatState(
    messages: messages,
    pending: pending,
    lastSeenId: messages.isEmpty ? lastSeenId : messages.last.id,
  );
}
