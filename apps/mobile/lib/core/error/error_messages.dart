import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/realtime/realtime_client.dart';

/// API hata kodları (docs/api.md §2) → kullanıcı metni.
///
/// Bilinmeyen kodlar genel hata metnine düşer; böylece sunucuya yeni kod
/// eklenmesi uygulamayı bozmaz.
String messageForErrorCode(AppLocalizations l, String? code) {
  return switch (code) {
    'VALIDATION_FAILED' => l.errorValidationFailed,
    'NICKNAME_INVALID' => l.errorNicknameInvalid,
    'MESSAGE_INVALID' => l.errorMessageInvalid,
    'UNAUTHORIZED' => l.errorUnauthorized,
    'FORBIDDEN' => l.errorForbidden,
    'NOT_A_MEMBER' => l.errorNotAMember,
    'KICKED' => l.errorKicked,
    'NOT_FOUND' => l.errorNotFound,
    'INVITE_CODE_NOT_FOUND' => l.errorInviteCodeNotFound,
    'ROOM_FULL' => l.errorRoomFull,
    'GAME_IN_PROGRESS' => l.errorGameInProgress,
    'NICKNAME_TAKEN_IN_ROOM' => l.errorNicknameTakenInRoom,
    'NICKNAME_TAKEN' => l.nicknameErrorTaken,
    'ALREADY_FRIENDS' => l.errorAlreadyFriends,
    'FRIEND_LIMIT' => l.errorFriendLimit,
    'ALREADY_IN_ROOM' => l.errorAlreadyInRoom,
    'INVALID_PHASE' => l.errorInvalidPhase,
    'NOT_YOUR_TURN' => l.errorNotYourTurn,
    'CARD_ALREADY_REVEALED' => l.errorCardAlreadyRevealed,
    'CATEGORY_NOT_ALLOWED' => l.errorCategoryNotAllowed,
    'ALREADY_VOTED' => l.errorAlreadyVoted,
    'SELF_VOTE' => l.errorSelfVote,
    'INVALID_TARGET' => l.errorInvalidTarget,
    'NOT_ALIVE' => l.errorNotAlive,
    'NOT_SPECTATOR' => l.errorNotSpectator,
    'START_CONDITIONS_NOT_MET' => l.errorStartConditionsNotMet,
    'STALE_STATE' => l.errorStaleState,
    'UPGRADE_REQUIRED' => l.errorUpgradeRequired,
    'RATE_LIMITED' => l.errorRateLimitedNoWait,
    'INTERNAL' => l.errorInternal,
    _ => l.errorUnknown,
  };
}

/// Bir [ApiException] için kullanıcıya gösterilecek metin.
///
/// İptal edilen istekler için `null` döner (kullanıcıya gösterilmez).
String? messageForApiError(AppLocalizations l, ApiException e) {
  return switch (e.kind) {
    ApiErrorKind.cancelled => null,
    ApiErrorKind.network => l.errorNetwork,
    ApiErrorKind.timeout => l.errorTimeout,
    ApiErrorKind.unauthorized => l.errorUnauthorized,
    ApiErrorKind.upgradeRequired => l.errorUpgradeRequired,
    ApiErrorKind.rateLimited => switch (e.retryAfter) {
      final wait? when wait.inSeconds > 0 => l.errorRateLimited(wait.inSeconds),
      _ => l.errorRateLimitedNoWait,
    },
    ApiErrorKind.invalidResponse =>
      (e.statusCode ?? 0) >= 500 ? l.errorInternal : l.errorUnknown,
    ApiErrorKind.server => messageForErrorCode(l, e.code),
  };
}

/// Gerçek zamanlı bağlantının kalıcı durma nedeni → kullanıcı metni.
String messageForRealtimeStop(AppLocalizations l, RealtimeStopReason reason) {
  return switch (reason) {
    RealtimeStopReason.unauthorized => l.errorUnauthorized,
    RealtimeStopReason.forbidden => l.realtimeForbidden,
    RealtimeStopReason.replaced => l.realtimeReplaced,
    RealtimeStopReason.upgradeRequired => l.errorUpgradeRequired,
  };
}

/// Herhangi bir hatayı kullanıcı metnine çevirir (bilinmeyenler genel metin).
String? messageForError(AppLocalizations l, Object error) {
  if (error is ApiException) return messageForApiError(l, error);
  return l.errorUnknown;
}
