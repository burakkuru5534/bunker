import 'package:siginak/core/l10n/l10n.dart';

/// Takma ad alan hatası; değerleri sunucunun `details.fields.nickname`
/// kodlarıyla aynıdır (docs/api.md §3).
enum NicknameIssue {
  required('required'),
  tooShort('too_short'),
  tooLong('too_long'),
  invalidChars('invalid_chars'),
  invalid('invalid'),
  notAllowed('not_allowed'),
  profanity('profanity'),

  /// Kullanıcı adı başka bir hesapta (`NICKNAME_TAKEN`).
  taken('taken');

  const NicknameIssue(this.code);

  /// Sunucu kodu.
  final String code;

  static NicknameIssue? fromCode(String? code) {
    for (final v in values) {
      if (v.code == code) return v;
    }
    return null;
  }

  String message(AppLocalizations l) => switch (this) {
    required => l.nicknameErrorRequired,
    tooShort => l.nicknameErrorTooShort,
    tooLong => l.nicknameErrorTooLong,
    invalidChars => l.nicknameErrorInvalidChars,
    invalid => l.nicknameErrorInvalid,
    notAllowed => l.nicknameErrorNotAllowed,
    profanity => l.nicknameErrorProfanity,
    taken => l.nicknameErrorTaken,
  };
}

/// Takma ad uzunluk sınırları (karakter).
const nicknameMinLength = 3;
const nicknameMaxLength = 16;

final _latinLetter = RegExp(r'^(?=\p{L})\p{Script=Latin}$', unicode: true);

/// Sunucu kurallarının istemci kopyası (anlık geri bildirim için; karar
/// sunucudadır — ayrılmış adlar ve küfür yalnızca sunucuda denetlenir).
///
/// Geçerliyse `null`.
NicknameIssue? checkNickname(String raw) {
  final nick = raw.trim();
  final length = nick.runes.length;
  if (length == 0) return NicknameIssue.required;
  if (length < nicknameMinLength) return NicknameIssue.tooShort;
  if (length > nicknameMaxLength) return NicknameIssue.tooLong;

  var hasLetter = false;
  var prevSpace = false;
  for (final rune in nick.runes) {
    final ch = String.fromCharCode(rune);
    if (_latinLetter.hasMatch(ch)) {
      hasLetter = true;
    } else if (ch == ' ') {
      if (prevSpace) return NicknameIssue.invalid;
    } else if (!_isDigitOrUnderscore(ch)) {
      return NicknameIssue.invalidChars;
    }
    prevSpace = ch == ' ';
  }
  return hasLetter ? null : NicknameIssue.invalid;
}

bool _isDigitOrUnderscore(String ch) =>
    ch == '_' || (ch.codeUnitAt(0) >= 0x30 && ch.codeUnitAt(0) <= 0x39);
