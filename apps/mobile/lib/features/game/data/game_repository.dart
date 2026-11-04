import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/util/uuid.dart';
import 'package:siginak/features/game/domain/game_models.dart';

/// Oyun eylemi (docs/api.md "Oyun"): tür + yük.
sealed class GameAction {
  const GameAction();

  String get type;
  Map<String, Object?> get payload;

  /// Aynı eylemin tekrarı için kimlik anahtarı (idempotency).
  String get key => '$type:$payload';
}

class RevealCard extends GameAction {
  const RevealCard(this.category);
  final CardCategory category;
  @override
  String get type => 'reveal_card';
  @override
  Map<String, Object?> get payload => {'category': category.name};
}

class DiscussionReady extends GameAction {
  const DiscussionReady();
  @override
  String get type => 'discussion_ready';
  @override
  Map<String, Object?> get payload => const {};
}

class Vote extends GameAction {
  const Vote(this.targetUserId);
  final String targetUserId;
  @override
  String get type => 'vote';
  @override
  Map<String, Object?> get payload => {'target_user_id': targetUserId};
}

class TiebreakVote extends GameAction {
  const TiebreakVote(this.targetUserId);
  final String targetUserId;
  @override
  String get type => 'tiebreak_vote';
  @override
  Map<String, Object?> get payload => {'target_user_id': targetUserId};
}

/// `POST /v1/games/{id}/actions`.
///
/// Aynı eylem (tür + yük + tur) başarıya ulaşana kadar aynı `action_id` ile
/// gönderilir: ağ hatasından sonra yeniden deneme çift işlem yaratmaz
/// (sunucu `duplicate: true` döner). Sonuç ekrana WS olaylarıyla yansır.
class GameRepository {
  GameRepository(this._api, {String Function()? newId})
    : _newId = newId ?? uuidV4;

  final ApiClient _api;
  final String Function() _newId;
  final Map<String, String> _pending = {};

  Future<void> act(String gameId, int round, GameAction action) async {
    final key = '$gameId:$round:${action.key}';
    final id = _pending[key] ??= _newId();
    await _api.post(
      '/v1/games/$gameId/actions',
      body: {'action_id': id, 'type': action.type, 'payload': action.payload},
    );
    _pending.remove(key);
  }
}

final gameRepositoryProvider = Provider<GameRepository>(
  (ref) => GameRepository(ref.watch(apiClientProvider)),
);
