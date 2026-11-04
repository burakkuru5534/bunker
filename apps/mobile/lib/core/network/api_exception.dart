import 'package:flutter/foundation.dart';

/// API çağrısı hatasının türü.
enum ApiErrorKind {
  /// Sunucuya ulaşılamadı (bağlantı yok, DNS, TLS…).
  network,

  /// Bağlantı / gönderme / alma zaman aşımı.
  timeout,

  /// İstek iptal edildi.
  cancelled,

  /// 401 — oturum geçersiz veya süresi dolmuş.
  unauthorized,

  /// 426 — uygulama sürümü artık desteklenmiyor.
  upgradeRequired,

  /// 429 — sıklık sınırı.
  rateLimited,

  /// Sunucu standart hata gövdesiyle yanıt verdi (`code` dolu).
  server,

  /// Yanıt beklenen biçimde değil.
  invalidResponse,
}

/// API hatası (bkz. docs/api.md §2).
///
/// Kullanıcıya gösterilecek metin `code` / `kind`'a göre l10n'dan seçilir
/// (MOB-009); `debugMessage` yalnızca hata ayıklama içindir.
@immutable
class ApiException implements Exception {
  const ApiException({
    required this.kind,
    this.statusCode,
    this.code,
    this.requestId,
    this.retryAfter,
    this.fieldErrors = const {},
    this.details = const {},
    this.debugMessage,
  });

  final ApiErrorKind kind;

  /// HTTP durum kodu (yanıt alındıysa).
  final int? statusCode;

  /// Sunucunun makine için sabit hata kodu (ör. `ROOM_FULL`).
  final String? code;

  /// Loglarla eşleştirme kimliği.
  final String? requestId;

  /// 429 yanıtındaki `Retry-After`.
  final Duration? retryAfter;

  /// `VALIDATION_FAILED` için alan → hata eşlemesi (`details.fields`).
  final Map<String, String> fieldErrors;

  /// Hata gövdesindeki `details` (ör. `ALREADY_IN_ROOM` → `room_id`).
  final Map<String, Object?> details;

  final String? debugMessage;

  @override
  String toString() =>
      'ApiException(${kind.name}, status: $statusCode, code: $code, '
      'requestId: $requestId)';
}
