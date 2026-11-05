import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/network/api_exception.dart';

/// Rapor nedeni (docs/api.md "Moderasyon").
enum ReportReason { harassment, hate, spam, cheating, other }

/// Engellediğim oyuncu.
@immutable
class BlockedUser {
  const BlockedUser({
    required this.userId,
    required this.nickname,
    required this.createdAt,
  });

  final String userId;
  final String nickname;
  final DateTime createdAt;
}

/// Engelleme ve raporlama uç noktaları.
class ModerationRepository {
  ModerationRepository(this._api);

  final ApiClient _api;

  /// Engellediklerim (en yeni önce).
  Future<List<BlockedUser>> blocks() async {
    final json = await _api.get('/v1/blocks');
    if (json is! List) throw _invalid('liste bekleniyordu');
    return [
      for (final b in json)
        if (b is Map<String, Object?>)
          BlockedUser(
            userId: b['user_id'] is String
                ? b['user_id']! as String
                : throw _invalid('user_id'),
            nickname: b['nickname'] is String ? b['nickname']! as String : '',
            createdAt:
                DateTime.tryParse(b['created_at'] as String? ?? '') ??
                DateTime.fromMillisecondsSinceEpoch(0),
          ),
    ];
  }

  /// Engeller (tekrar engellemek değişiklik yapmaz).
  Future<void> block(String userId) async {
    await _api.post('/v1/blocks', body: {'user_id': userId});
  }

  /// Engeli kaldırır (olmayan engel de başarılıdır).
  Future<void> unblock(String userId) async {
    await _api.delete('/v1/blocks/$userId');
  }

  /// Raporlar. Not boşsa gönderilmez. Aynı kişi aynı odada açık raporla
  /// tekrar raporlanırsa sunucu yeni kayıt açmaz.
  Future<void> report({
    required String reportedUserId,
    required String roomId,
    required ReportReason reason,
    String? note,
  }) async {
    final trimmed = note?.trim() ?? '';
    await _api.post(
      '/v1/reports',
      body: {
        'reported_user_id': reportedUserId,
        'room_id': roomId,
        'reason': reason.name,
        if (trimmed.isNotEmpty) 'note': trimmed,
      },
    );
  }

  static ApiException _invalid(String what) => ApiException(
    kind: ApiErrorKind.invalidResponse,
    debugMessage: 'engellenenler: $what',
  );
}

final moderationRepositoryProvider = Provider<ModerationRepository>(
  (ref) => ModerationRepository(ref.watch(apiClientProvider)),
);
