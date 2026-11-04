import 'package:flutter/foundation.dart';

/// Oyun aşaması (docs/game-design.md §2; WS'de büyük harfli adlar).
enum GamePhase {
  dealing('DEALING'),
  scenarioIntro('SCENARIO_INTRO'),
  reveal('REVEAL'),
  discussion('DISCUSSION'),
  voting('VOTING'),
  tieBreak('TIE_BREAK'),
  elimination('ELIMINATION'),
  finished('FINISHED');

  const GamePhase(this.wire);

  final String wire;

  static GamePhase? fromWire(Object? v) {
    for (final p in values) {
      if (p.wire == v) return p;
    }
    return null;
  }
}

/// Kart kategorisi; sıra açılma önceliğidir (docs/game-design.md §6).
enum CardCategory {
  profession,
  health,
  hobby,
  item,
  secret;

  static CardCategory? fromWire(Object? v) {
    for (final c in values) {
      if (c.name == v) return c;
    }
    return null;
  }
}

enum ParticipantStatus {
  alive,
  eliminated,
  left;

  static ParticipantStatus fromWire(Object? v) => switch (v) {
    'eliminated' => eliminated,
    'left' => left,
    _ => alive,
  };
}

/// Oyuncunun kendi kartı (yalnızca ona gönderilir).
@immutable
class MyCard {
  const MyCard({
    required this.category,
    required this.text,
    required this.revealed,
  });

  final CardCategory category;
  final String text;
  final bool revealed;

  MyCard asRevealed() => MyCard(category: category, text: text, revealed: true);

  @override
  bool operator ==(Object other) =>
      other is MyCard &&
      other.category == category &&
      other.text == text &&
      other.revealed == revealed;

  @override
  int get hashCode => Object.hash(category, text, revealed);
}

/// Oyuncu (koltuk sırasında). `revealed`: herkese açılmış kartları.
@immutable
class Participant {
  const Participant({
    required this.userId,
    required this.nickname,
    required this.seat,
    required this.status,
    required this.connected,
    required this.revealed,
  });

  final String userId;
  final String nickname;
  final int seat;
  final ParticipantStatus status;
  final bool connected;
  final Map<CardCategory, String> revealed;

  bool get isAlive => status == ParticipantStatus.alive;

  Participant copyWith({
    ParticipantStatus? status,
    bool? connected,
    Map<CardCategory, String>? revealed,
  }) => Participant(
    userId: userId,
    nickname: nickname,
    seat: seat,
    status: status ?? this.status,
    connected: connected ?? this.connected,
    revealed: revealed ?? this.revealed,
  );
}

/// Senaryo (mekanik etkisi yok; tartışmanın zemini).
@immutable
class ScenarioInfo {
  const ScenarioInfo({
    required this.id,
    required this.title,
    required this.disaster,
    required this.duration,
    required this.bunker,
    required this.outside,
  });

  final String id;
  final String title;
  final String disaster;
  final String duration;
  final List<String> bunker;
  final String outside;

  static ScenarioInfo fromJson(Object? json) {
    final m = _map(json);
    final bunker = m['bunker'];
    return ScenarioInfo(
      id: _str(m['id']),
      title: _str(m['title']),
      disaster: _str(m['disaster']),
      duration: _str(m['duration']),
      bunker: bunker is List
          ? List.unmodifiable(bunker.whereType<String>())
          : const [],
      outside: _str(m['outside']),
    );
  }
}

/// "x / y" ilerlemesi (oy, hazır).
typedef Progress = ({int done, int total});

/// Bir turun eleme sonucu (`game.player_eliminated`).
@immutable
class Elimination {
  const Elimination({
    required this.round,
    required this.userId,
    required this.method,
    required this.tally,
    required this.lotteryCandidates,
  });

  final int round;
  final String userId;

  /// `vote` | `spectators` | `lottery` | `left`
  final String method;

  /// Adaylara gelen oylar (kimin kime verdiği yok).
  final List<({String userId, int votes})> tally;
  final List<String> lotteryCandidates;

  static Elimination? tryParse(Object? json) {
    if (json is! Map<String, Object?>) return null;
    final user = json['user_id'];
    if (user is! String) return null;
    final tally = json['tally'];
    return Elimination(
      round: _int(json['round']),
      userId: user,
      method: _str(json['method']),
      tally: [
        if (tally is List)
          for (final t in tally.whereType<Map<String, Object?>>())
            if (t['user_id'] is String)
              (userId: t['user_id']! as String, votes: _int(t['votes'])),
      ],
      lotteryCandidates: _strings(json['lottery_candidates']),
    );
  }
}

/// Oyunun istemci görünümü: sunucu durumunun yansıması (karar sunucuda).
///
/// `session.snapshot.game` ile kurulur, `game.*` olaylarıyla [apply] edilir.
/// Süreler sunucu saatine göre: [clockOffset] = sunucu − cihaz saati.
@immutable
class GameView {
  const GameView({
    required this.id,
    required this.phase,
    required this.round,
    required this.totalRounds,
    required this.capacity,
    required this.deadline,
    required this.clockOffset,
    required this.scenario,
    required this.turnUserId,
    required this.participants,
    required this.myCards,
    required this.voted,
    required this.discussionReady,
    required this.tiebreakVoted,
    required this.readyProgress,
    required this.voteProgress,
    required this.tiebreakCandidates,
    required this.tiebreakProgress,
    required this.lastElimination,
  });

  final String id;
  final GamePhase phase;
  final int round;
  final int totalRounds;
  final int capacity;
  final DateTime? deadline;
  final Duration clockOffset;
  final ScenarioInfo? scenario;
  final String? turnUserId;

  /// Koltuk sırasında.
  final List<Participant> participants;

  /// Kategori sırasında; katılımcı değilse boş.
  final List<MyCard> myCards;

  final bool voted;
  final bool discussionReady;
  final bool tiebreakVoted;
  final Progress readyProgress;
  final Progress voteProgress;
  final List<String> tiebreakCandidates;
  final Progress tiebreakProgress;
  final Elimination? lastElimination;

  Participant? participantOf(String? userId) {
    for (final p in participants) {
      if (p.userId == userId) return p;
    }
    return null;
  }

  int get aliveCount => participants.where((p) => p.isAlive).length;

  /// Aşamanın kalan süresi (sunucu saatine göre); süre yoksa `null`.
  Duration? remaining(DateTime now) {
    final d = deadline;
    if (d == null) return null;
    final left = d.difference(now.add(clockOffset));
    return left.isNegative ? Duration.zero : left;
  }

  /// `session.snapshot.game` (bozuksa `FormatException`).
  static GameView fromSnapshot(
    Object? json, {
    required DateTime? serverTime,
    required DateTime receivedAt,
  }) {
    final m = _map(json);
    final state = m['my_action_state'];
    final s = state is Map<String, Object?> ? state : const <String, Object?>{};
    final tiebreak = m['tiebreak'];
    final tb = tiebreak is Map<String, Object?> ? tiebreak : null;
    final participants = _participants(m['participants']);
    final alive = participants.where((p) => p.isAlive).length;
    return GameView(
      id: _str(m['id']),
      phase: _phase(m['phase']),
      round: _int(m['round']),
      totalRounds: _int(m['total_rounds']),
      capacity: _int(m['capacity']),
      deadline: _time(m['phase_deadline_at']),
      clockOffset: _offset(serverTime, receivedAt),
      scenario: m['scenario'] == null
          ? null
          : ScenarioInfo.fromJson(m['scenario']),
      turnUserId: m['turn_user_id'] is String
          ? m['turn_user_id']! as String
          : null,
      participants: participants,
      myCards: _myCards(m['my_cards']),
      voted: s['voted'] == true,
      discussionReady: s['discussion_ready'] == true,
      tiebreakVoted: s['tiebreak_voted'] == true,
      readyProgress: (done: _int(m['discussion_ready_count']), total: alive),
      voteProgress: _progress(m['vote_progress']),
      tiebreakCandidates: tb == null ? const [] : _strings(tb['candidates']),
      tiebreakProgress: _progress(tb?['progress']),
      lastElimination: Elimination.tryParse(m['last_round_result']),
    );
  }

  /// `game.started` (kartlar ayrıca `game.your_cards` ile gelir).
  static GameView fromStarted(
    Map<String, Object?> p, {
    required DateTime receivedAt,
  }) => GameView(
    id: _str(p['game_id']),
    phase: GamePhase.dealing,
    round: 0,
    totalRounds: _int(p['total_rounds']),
    capacity: _int(p['capacity']),
    deadline: null,
    clockOffset: Duration.zero,
    scenario: p['scenario'] == null
        ? null
        : ScenarioInfo.fromJson(p['scenario']),
    turnUserId: null,
    participants: _participants(p['participants']),
    myCards: const [],
    voted: false,
    discussionReady: false,
    tiebreakVoted: false,
    readyProgress: (done: 0, total: 0),
    voteProgress: (done: 0, total: 0),
    tiebreakCandidates: const [],
    tiebreakProgress: (done: 0, total: 0),
    lastElimination: null,
  );

  /// Bir `game.*` olayını uygular (bilinmeyen tür → değişiklik yok).
  GameView apply(
    String type,
    Map<String, Object?> p, {
    required DateTime receivedAt,
    required String? myUserId,
  }) {
    switch (type) {
      case 'game.your_cards':
        return _copy(myCards: _myCards(p['cards']));
      case 'game.phase_changed':
        final phase = _phase(p['phase']);
        final round = _int(p['round']);
        final newRound = round != this.round;
        final alive = aliveCount;
        return _copy(
          phase: phase,
          round: round,
          deadline: () => _time(p['phase_deadline_at']),
          clockOffset: p['server_time'] == null
              ? clockOffset
              : _offset(_time(p['server_time']), receivedAt),
          turnUserId: () =>
              p['turn_user_id'] is String ? p['turn_user_id']! as String : null,
          voted: newRound ? false : voted,
          discussionReady: newRound ? false : discussionReady,
          tiebreakVoted: newRound ? false : tiebreakVoted,
          readyProgress:
              phase == GamePhase.discussion &&
                  this.phase != GamePhase.discussion
              ? (done: 0, total: alive)
              : readyProgress,
          voteProgress:
              phase == GamePhase.voting && this.phase != GamePhase.voting
              ? (done: 0, total: alive)
              : voteProgress,
        );
      case 'game.card_revealed':
        final user = p['user_id'];
        final category = CardCategory.fromWire(p['category']);
        final text = p['text'];
        if (user is! String || category == null || text is! String) {
          throw const FormatException('card_revealed');
        }
        return _copy(
          participants: [
            for (final x in participants)
              x.userId == user
                  ? x.copyWith(revealed: {...x.revealed, category: text})
                  : x,
          ],
          myCards: user == myUserId
              ? [
                  for (final c in myCards)
                    c.category == category ? c.asRevealed() : c,
                ]
              : myCards,
        );
      case 'game.discussion_ready':
        return _copy(readyProgress: _countTotal(p));
      case 'game.vote_progress':
        return _copy(voteProgress: _progress(p));
      case 'game.tiebreak_started':
        return _copy(
          tiebreakCandidates: _strings(p['candidates']),
          tiebreakProgress: (done: 0, total: 0),
          deadline: () => _time(p['phase_deadline_at']) ?? deadline,
        );
      case 'game.tiebreak_progress':
        return _copy(tiebreakProgress: _progress(p));
      case 'game.player_eliminated':
        final e = Elimination.tryParse(p);
        if (e == null) throw const FormatException('player_eliminated');
        final cards = _cardTexts(p['cards']);
        return _copy(
          participants: [
            for (final x in participants)
              x.userId == e.userId
                  ? x.copyWith(
                      status: e.method == 'left'
                          ? ParticipantStatus.left
                          : ParticipantStatus.eliminated,
                      revealed: {...x.revealed, ...cards},
                    )
                  : x,
          ],
          myCards: e.userId == myUserId
              ? [for (final c in myCards) c.asRevealed()]
              : myCards,
          lastElimination: e.method == 'left' ? lastElimination : e,
        );
      case 'game.player_connection':
        final user = p['user_id'];
        final state = p['state'];
        return _copy(
          participants: [
            for (final x in participants)
              x.userId == user
                  ? x.copyWith(
                      connected: state != 'disconnected',
                      status: state == 'left' ? ParticipantStatus.left : null,
                    )
                  : x,
          ],
        );
      case 'game.finished':
        return _copy(phase: GamePhase.finished, deadline: () => null);
      default:
        return this;
    }
  }

  GameView _copy({
    GamePhase? phase,
    int? round,
    DateTime? Function()? deadline,
    Duration? clockOffset,
    String? Function()? turnUserId,
    List<Participant>? participants,
    List<MyCard>? myCards,
    bool? voted,
    bool? discussionReady,
    bool? tiebreakVoted,
    Progress? readyProgress,
    Progress? voteProgress,
    List<String>? tiebreakCandidates,
    Progress? tiebreakProgress,
    Elimination? lastElimination,
  }) => GameView(
    id: id,
    phase: phase ?? this.phase,
    round: round ?? this.round,
    totalRounds: totalRounds,
    capacity: capacity,
    deadline: deadline != null ? deadline() : this.deadline,
    clockOffset: clockOffset ?? this.clockOffset,
    scenario: scenario,
    turnUserId: turnUserId != null ? turnUserId() : this.turnUserId,
    participants: participants != null
        ? List.unmodifiable(participants)
        : this.participants,
    myCards: myCards != null ? List.unmodifiable(myCards) : this.myCards,
    voted: voted ?? this.voted,
    discussionReady: discussionReady ?? this.discussionReady,
    tiebreakVoted: tiebreakVoted ?? this.tiebreakVoted,
    readyProgress: readyProgress ?? this.readyProgress,
    voteProgress: voteProgress ?? this.voteProgress,
    tiebreakCandidates: tiebreakCandidates ?? this.tiebreakCandidates,
    tiebreakProgress: tiebreakProgress ?? this.tiebreakProgress,
    lastElimination: lastElimination ?? this.lastElimination,
  );
}

// --- çözümleme yardımcıları (bozuk veri → FormatException) ---

Map<String, Object?> _map(Object? v) {
  if (v is Map<String, Object?>) return v;
  throw const FormatException('nesne bekleniyordu');
}

List<Object?> _list(Object? v) => v is List<Object?> ? v : const [];

String _str(Object? v) => v is String ? v : '';

int _int(Object? v) => v is int ? v : 0;

List<String> _strings(Object? v) =>
    List.unmodifiable(_list(v).whereType<String>());

DateTime? _time(Object? v) => v is String ? DateTime.tryParse(v) : null;

Duration _offset(DateTime? server, DateTime receivedAt) =>
    server == null ? Duration.zero : server.difference(receivedAt);

GamePhase _phase(Object? v) {
  final p = GamePhase.fromWire(v);
  if (p == null) throw FormatException('aşama: $v');
  return p;
}

Progress _progress(Object? v) {
  final m = v is Map<String, Object?> ? v : const <String, Object?>{};
  return (done: _int(m['cast']), total: _int(m['total']));
}

Progress _countTotal(Map<String, Object?> m) =>
    (done: _int(m['count']), total: _int(m['total']));

List<Participant> _participants(Object? v) {
  final out = <Participant>[];
  for (final p in _list(v)) {
    final m = _map(p);
    final user = m['user_id'];
    if (user is! String) throw const FormatException('participant');
    out.add(
      Participant(
        userId: user,
        nickname: _str(m['nickname']),
        seat: _int(m['seat']),
        status: ParticipantStatus.fromWire(m['status']),
        connected: m['connected'] != false,
        revealed: _cardTexts(m['revealed_cards']),
      ),
    );
  }
  out.sort((a, b) => a.seat.compareTo(b.seat));
  return List.unmodifiable(out);
}

List<MyCard> _myCards(Object? v) {
  final out = <MyCard>[
    for (final c in _list(v).whereType<Map<String, Object?>>())
      if (CardCategory.fromWire(c['category']) case final cat?)
        MyCard(
          category: cat,
          text: _str(c['text']),
          revealed: c['revealed'] == true,
        ),
  ]..sort((a, b) => a.category.index.compareTo(b.category.index));
  return List.unmodifiable(out);
}

/// `[{category, text}]` → kategori → metin (bilinmeyen kategori atlanır).
Map<CardCategory, String> _cardTexts(Object? v) {
  final out = <CardCategory, String>{};
  for (final c in _list(v).whereType<Map<String, Object?>>()) {
    final cat = CardCategory.fromWire(c['category']);
    if (cat != null) out[cat] = _str(c['text']);
  }
  return out;
}
