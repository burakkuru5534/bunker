import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/network/api_exception.dart';

/// `POST /v1/rooms/{id}/messages` (docs/api.md "Sohbet").
class ChatRepository {
  ChatRepository(this._api);

  final ApiClient _api;

  /// Mesaj gönderir; aynı [clientMsgId] tekrar gelirse sunucu ilk mesajı
  /// döndürür (çift mesaj olmaz). Sunucunun mesaj kimliğini döndürür.
  Future<({int id, DateTime createdAt})> send(
    String roomId, {
    required String clientMsgId,
    required String text,
  }) async {
    final json = await _api.post(
      '/v1/rooms/$roomId/messages',
      body: {'client_msg_id': clientMsgId, 'text': text},
    );
    if (json is Map<String, Object?> &&
        json['id'] is int &&
        json['created_at'] is String) {
      final at = DateTime.tryParse(json['created_at']! as String);
      if (at != null) return (id: json['id']! as int, createdAt: at);
    }
    throw const ApiException(
      kind: ApiErrorKind.invalidResponse,
      debugMessage: 'mesaj yanıtı',
    );
  }
}

final chatRepositoryProvider = Provider<ChatRepository>(
  (ref) => ChatRepository(ref.watch(apiClientProvider)),
);
