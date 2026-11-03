import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/realtime/realtime_client.dart';

void main() {
  final l = lookupAppLocalizations(const Locale('tr'));

  /// docs/api.md §2'deki hata kodları.
  List<String> documentedCodes() {
    final doc = File('../../docs/api.md').readAsStringSync();
    final section = doc.substring(
      doc.indexOf('### Hata kodları'),
      doc.indexOf('## 3.'),
    );
    return RegExp(r'\| `([A-Z_]+)` \|')
        .allMatches(section)
        .map((m) => m.group(1)!)
        .toList();
  }

  test('api.md içindeki her hata kodunun kendine özgü metni var', () {
    final codes = documentedCodes();
    expect(codes.length, greaterThanOrEqualTo(25));

    for (final code in codes) {
      final message = messageForErrorCode(l, code);
      expect(message, isNotEmpty, reason: code);
      expect(message, isNot(l.errorUnknown), reason: '$code genel metne düştü');
    }
  });

  test('bilinmeyen kod genel metne düşer', () {
    expect(messageForErrorCode(l, 'YENI_BIR_KOD'), l.errorUnknown);
    expect(messageForErrorCode(l, null), l.errorUnknown);
  });

  group('ApiException türleri', () {
    String? msg(ApiException e) => messageForApiError(l, e);

    test('ağ, zaman aşımı, oturum, sürüm', () {
      expect(
        msg(const ApiException(kind: ApiErrorKind.network)),
        l.errorNetwork,
      );
      expect(
        msg(const ApiException(kind: ApiErrorKind.timeout)),
        l.errorTimeout,
      );
      expect(
        msg(const ApiException(kind: ApiErrorKind.unauthorized)),
        l.errorUnauthorized,
      );
      expect(
        msg(const ApiException(kind: ApiErrorKind.upgradeRequired)),
        l.errorUpgradeRequired,
      );
    });

    test('iptal edilen istek kullanıcıya gösterilmez', () {
      expect(msg(const ApiException(kind: ApiErrorKind.cancelled)), isNull);
    });

    test('sıklık sınırı bekleme süresini yazar', () {
      final withWait = msg(
        const ApiException(
          kind: ApiErrorKind.rateLimited,
          retryAfter: Duration(seconds: 7),
        ),
      );
      expect(withWait, contains('7 saniye'));
      expect(
        msg(const ApiException(kind: ApiErrorKind.rateLimited)),
        l.errorRateLimitedNoWait,
      );
    });

    test('sunucu kodu metne çevrilir', () {
      expect(
        msg(const ApiException(kind: ApiErrorKind.server, code: 'ROOM_FULL')),
        l.errorRoomFull,
      );
    });

    test('biçimsiz yanıt: 5xx sunucu hatası, diğerleri genel', () {
      expect(
        msg(
          const ApiException(
            kind: ApiErrorKind.invalidResponse,
            statusCode: 502,
          ),
        ),
        l.errorInternal,
      );
      expect(
        msg(const ApiException(kind: ApiErrorKind.invalidResponse)),
        l.errorUnknown,
      );
    });
  });

  test('WebSocket durma nedenlerinin metni var', () {
    for (final r in RealtimeStopReason.values) {
      expect(messageForRealtimeStop(l, r), isNotEmpty, reason: r.name);
    }
    expect(
      messageForRealtimeStop(l, RealtimeStopReason.replaced),
      l.realtimeReplaced,
    );
  });

  test('bilinmeyen hata türleri genel metne düşer', () {
    expect(messageForError(l, StateError('x')), l.errorUnknown);
    expect(
      messageForError(l, const ApiException(kind: ApiErrorKind.network)),
      l.errorNetwork,
    );
  });
}
