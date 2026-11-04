import 'package:flutter/foundation.dart';

import 'package:siginak/features/game/domain/game_models.dart';

/// Oyuncunun maç sonu kaydı.
@immutable
class ResultPlayer {
  const ResultPlayer({
    required this.userId,
    required this.nickname,
    required this.isWinner,
    required this.status,
    required this.eliminatedRound,
    required this.cards,
  });

  final String userId;
  final String nickname;
  final bool isWinner;
  final ParticipantStatus status;
  final int? eliminatedRound;

  /// Kategori sırasında, tüm kartlar.
  final List<({CardCategory category, String text})> cards;
}

/// Bir turun eleme özeti.
@immutable
class ResultRound {
  const ResultRound({
    required this.round,
    required this.eliminatedUserId,
    required this.method,
    required this.tally,
  });

  final int round;
  final String? eliminatedUserId;
  final String method;
  final List<({String userId, int votes})> tally;
}

/// Maç sonu (`game.finished` yükü = `GET /v1/games/{id}/result`).
@immutable
class GameResult {
  const GameResult({
    required this.gameId,
    required this.endReason,
    required this.scenarioTitle,
    required this.capacity,
    required this.players,
    required this.rounds,
  });

  final String gameId;

  /// `capacity_reached` | `abandoned`
  final String endReason;
  final String scenarioTitle;
  final int capacity;

  /// Koltuk sırasında.
  final List<ResultPlayer> players;
  final List<ResultRound> rounds;

  Iterable<ResultPlayer> get winners => players.where((p) => p.isWinner);
  Iterable<ResultPlayer> get losers => players.where((p) => !p.isWinner);

  String nicknameOf(String? userId) {
    for (final p in players) {
      if (p.userId == userId) return p.nickname;
    }
    return '';
  }

  /// Bozuk veride `FormatException`.
  static GameResult fromJson(Object? json) {
    final m = _map(json);
    final scenario = m['scenario'];
    return GameResult(
      gameId: _str(m['game_id']),
      endReason: _str(m['end_reason']),
      scenarioTitle: scenario is Map<String, Object?>
          ? _str(scenario['title'])
          : '',
      capacity: _int(m['capacity']),
      players: List.unmodifiable([
        for (final p in _list(m['players'])) _player(_map(p)),
      ]),
      rounds: List.unmodifiable([
        for (final r in _list(m['rounds']))
          if (r is Map<String, Object?>)
            ResultRound(
              round: _int(r['round']),
              eliminatedUserId: r['eliminated_user_id'] is String
                  ? r['eliminated_user_id']! as String
                  : null,
              method: _str(r['method']),
              tally: [
                for (final t in _list(r['tally']))
                  if (t is Map<String, Object?> && t['user_id'] is String)
                    (userId: t['user_id']! as String, votes: _int(t['votes'])),
              ],
            ),
      ]),
    );
  }

  static ResultPlayer _player(Map<String, Object?> m) {
    final user = m['user_id'];
    if (user is! String) throw const FormatException('result player');
    final cards = [
      for (final c in _list(m['cards']))
        if (c is Map<String, Object?>)
          if (CardCategory.fromWire(c['category']) case final cat?)
            (category: cat, text: _str(c['text'])),
    ]..sort((a, b) => a.category.index.compareTo(b.category.index));
    final round = m['eliminated_round'];
    return ResultPlayer(
      userId: user,
      nickname: _str(m['nickname']),
      isWinner: m['is_winner'] == true,
      status: ParticipantStatus.fromWire(m['status']),
      eliminatedRound: round is int ? round : null,
      cards: List.unmodifiable(cards),
    );
  }
}

Map<String, Object?> _map(Object? v) {
  if (v is Map<String, Object?>) return v;
  throw const FormatException('nesne bekleniyordu');
}

List<Object?> _list(Object? v) => v is List<Object?> ? v : const [];

String _str(Object? v) => v is String ? v : '';

int _int(Object? v) => v is int ? v : 0;
