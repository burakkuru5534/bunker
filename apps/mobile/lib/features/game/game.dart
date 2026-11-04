/// `game` feature'ının diğer katmanlara açık giriş noktası.
library;

export 'package:siginak/features/game/data/game_repository.dart'
    show
        DiscussionReady,
        GameAction,
        GameRepository,
        RevealCard,
        TiebreakVote,
        Vote,
        gameRepositoryProvider;

export 'package:siginak/features/game/domain/game_models.dart'
    show
        CardCategory,
        Elimination,
        GamePhase,
        GameView,
        MyCard,
        Participant,
        ParticipantStatus,
        ScenarioInfo;
export 'package:siginak/features/game/domain/game_result.dart'
    show GameResult, ResultPlayer, ResultRound;
export 'package:siginak/features/game/presentation/game_body.dart'
    show GameBody;
export 'package:siginak/features/game/presentation/result_view.dart'
    show ResultView, gameResultProvider;
