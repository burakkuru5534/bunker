import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/legal_links.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/ui/external_link.dart';
import 'package:siginak/features/session/session.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/pump_app.dart';

void main() {
  late FakeBackend backend;
  late List<Uri> opened;
  late Object? openError;
  setUp(() {
    backend = FakeBackend();
    opened = [];
    openError = null;
  });

  Future<ProviderContainer> pump(WidgetTester tester) async {
    await pumpWithProviders(
      tester,
      const WelcomeScreen(),
      overrides: [
        apiHttpAdapterProvider.overrideWithValue(backend.adapter),
        linkOpenerProvider.overrideWithValue((uri) async {
          opened.add(uri);
          if (openError != null) throw openError!;
          return true;
        }),
      ],
    );
    await tester.pumpAndSettle();
    return ProviderScope.containerOf(
      tester.element(find.byType(WelcomeScreen)),
    );
  }

  testWidgets('Başlık, kurallar ve cihaz notu görünür; hata başta yok', (
    tester,
  ) async {
    await pump(tester);

    expect(find.text('Sığınağa kim girecek?'), findsOneWidget);
    expect(find.textContaining('Son kalanlar sığınağa girer'), findsOneWidget);
    expect(find.textContaining('Hesabın bu cihaza bağlı'), findsOneWidget);
    expect(find.text('Bir takma ad yaz.'), findsNothing);
  });

  testWidgets('Yaş ve koşul notu görünür; bağlantılar açılır', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    addTearDown(tester.view.reset);
    await pump(tester);

    expect(find.textContaining('13 yaş ve üstü'), findsOneWidget);
    await tester.tap(find.text('Kullanım koşulları'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Gizlilik politikası'));
    await tester.pumpAndSettle();

    expect(opened, [LegalLinks.terms, LegalLinks.privacy]);
    expect(backend.calls, isEmpty);
  });

  testWidgets('Açıcı hata fırlatırsa kısa hata gösterilir', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    addTearDown(tester.view.reset);
    openError = StateError('tarayıcı yok');
    await pump(tester);

    await tester.tap(find.text('Gizlilik politikası'));
    await tester.pumpAndSettle();

    expect(find.text('Bağlantı açılamadı.'), findsOneWidget);
  });

  testWidgets('Boş gönderim ve geçersiz ad anlık hata gösterir, istek gitmez', (
    tester,
  ) async {
    await pump(tester);

    await tester.tap(find.text('Başla'));
    await tester.pump();
    expect(find.text('Bir takma ad yaz.'), findsOneWidget);

    await tester.enterText(find.byType(TextField), 'ab');
    await tester.pump();
    expect(find.text('En az 3 karakter olmalı.'), findsOneWidget);

    await tester.enterText(find.byType(TextField), 'ali!');
    await tester.pump();
    expect(
      find.text('Yalnızca harf, rakam, boşluk ve _ kullanabilirsin.'),
      findsOneWidget,
    );

    await tester.tap(find.text('Başla'));
    await tester.pump();
    expect(backend.calls, isEmpty);
  });

  testWidgets('16 karakterden uzun yazılamaz (sayaç)', (tester) async {
    await pump(tester);

    await tester.enterText(find.byType(TextField), 'a' * 20);
    await tester.pump();

    expect(find.text('16/16'), findsOneWidget);
  });

  testWidgets('Geçerli ad → misafir hesap açılır (kırpılmış ad)', (
    tester,
  ) async {
    final c = await pump(tester);

    await tester.enterText(find.byType(TextField), '  Burak  ');
    await tester.tap(find.text('Başla'));
    await tester.pumpAndSettle();

    expect(backend.adapter.requests.single.data, {'nickname': 'Burak'});
    final state = c.read(sessionControllerProvider);
    expect(state.status, SessionStatus.authenticated);
    expect(state.nickname, 'Burak');
  });

  testWidgets('Sunucu alan hatası satır içinde; yazınca kaybolur', (
    tester,
  ) async {
    backend.on(
      'POST',
      '/v1/sessions/guest',
      (r) async => jsonResponse(
        apiError(
          'NICKNAME_INVALID',
          details: {
            'fields': {'nickname': 'not_allowed'},
          },
        ),
        400,
      ),
    );
    await pump(tester);

    await tester.enterText(find.byType(TextField), 'Admin');
    await tester.tap(find.text('Başla'));
    await tester.pumpAndSettle();

    const msg = 'Bu takma ad kullanılamaz, başka bir tane seç.';
    expect(find.text(msg), findsOneWidget);

    await tester.enterText(find.byType(TextField), 'Adminci');
    await tester.pumpAndSettle(); // hata metni solarak kaybolur
    expect(find.text(msg), findsNothing);
  });

  testWidgets('Sıklık sınırı snackbar ile bildirilir', (tester) async {
    backend.on(
      'POST',
      '/v1/sessions/guest',
      (r) async => jsonResponse(
        apiError('RATE_LIMITED'),
        429,
        headers: {
          'retry-after': ['30'],
        },
      ),
    );
    await pump(tester);

    await tester.enterText(find.byType(TextField), 'Burak');
    await tester.tap(find.text('Başla'));
    await tester.pumpAndSettle();

    expect(
      find.text('Çok hızlı gidiyorsun. 30 saniye sonra tekrar dene.'),
      findsOneWidget,
    );
  });

  testWidgets('İstek sürerken yükleniyor; ikinci basış istek göndermez', (
    tester,
  ) async {
    final gate = Completer<void>();
    backend.on('POST', '/v1/sessions/guest', (r) async {
      await gate.future;
      return jsonResponse(apiError('INTERNAL'), 500);
    });
    await pump(tester);

    await tester.enterText(find.byType(TextField), 'Burak');
    await tester.tap(find.text('Başla'));
    // İstek zinciri (token okuma, Dio) birkaç kare sürer.
    for (var i = 0; i < 5; i++) {
      await tester.pump(const Duration(milliseconds: 10));
    }

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    await tester.tap(find.byType(FilledButton));
    await tester.pump();
    expect(backend.calls, ['POST /v1/sessions/guest']);

    gate.complete();
    await tester.pumpAndSettle();
    expect(
      find.text('Sunucuda bir sorun oluştu. Biraz sonra tekrar dene.'),
      findsOneWidget,
    );
    expect(find.byType(CircularProgressIndicator), findsNothing);
  });
}
