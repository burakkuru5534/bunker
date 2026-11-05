import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/features/chat/chat.dart';

import '../../../helpers/chat.dart';
import '../../../helpers/pump_app.dart';
import '../../../helpers/small_screen.dart';

void main() {
  late List<String> sent;
  late List<PendingMessage> retried;
  late int seen;

  setUp(() {
    sent = [];
    retried = [];
    seen = 0;
  });

  Future<void> pump(
    WidgetTester tester,
    ChatState chat, {
    bool canWrite = true,
  }) async {
    await pumpWithProviders(
      tester,
      Scaffold(
        body: ChatPanel(
          chat: chat,
          myUserId: 'me',
          canWrite: canWrite,
          onSend: sent.add,
          onRetry: retried.add,
          onSeen: () => seen++,
        ),
      ),
    );
    await tester.pump();
  }

  testWidgets('Boş sohbet', (tester) async {
    await pump(tester, const ChatState());
    expect(find.text('Henüz mesaj yok. İlk sen yaz!'), findsOneWidget);
  });

  testWidgets('Mesajlar; başkalarınınkinde ad; görüldü bildirilir', (
    tester,
  ) async {
    await pump(
      tester,
      const ChatState().withHistory([
        msg(1, 'Ayşe', text: 'merhaba'),
        msg(2, 'me', text: 'selam'),
      ]),
    );
    expect(find.text('merhaba'), findsOneWidget);
    expect(find.textContaining('Ayşe ·'), findsOneWidget);
    expect(find.text('selam'), findsOneWidget);
    expect(seen, 1);
  });

  testWidgets('Gönder: boşken kapalı; kırpılır ve alan temizlenir', (
    tester,
  ) async {
    await pump(tester, const ChatState());
    IconButton send() => tester.widget<IconButton>(find.byType(IconButton));
    expect(send().onPressed, isNull);

    await tester.enterText(find.byType(TextField), '   ');
    await tester.pump();
    expect(send().onPressed, isNull);

    await tester.enterText(find.byType(TextField), '  selam  ');
    await tester.pump();
    await tester.tap(find.byTooltip('Gönder'));
    await tester.pump();

    expect(sent, ['selam']);
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      '',
    );
  });

  testWidgets('200 karakterden uzun yazılamaz', (tester) async {
    await pump(tester, const ChatState());
    await tester.enterText(find.byType(TextField), 'a' * 250);
    await tester.pump();
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text.length,
      200,
    );
  });

  testWidgets('Bekleyen ve gönderilemeyen mesaj; dokununca yeniden dene', (
    tester,
  ) async {
    const failed = PendingMessage(
      clientMsgId: 'c2',
      text: 'olmadı',
      error: ApiException(kind: ApiErrorKind.rateLimited),
    );
    await pump(
      tester,
      const ChatState().withPending([
        const PendingMessage(clientMsgId: 'c1', text: 'gidiyor'),
        failed,
      ]),
    );
    expect(find.text('Gönderiliyor…'), findsOneWidget);
    expect(find.textContaining('Tekrar denemek için dokun.'), findsOneWidget);
    expect(find.textContaining('Çok hızlı gidiyorsun.'), findsOneWidget);

    await tester.tap(find.text('olmadı'));
    expect(retried.single.clientMsgId, 'c2');

    await tester.tap(find.text('gidiyor'));
    expect(retried, hasLength(1)); // gönderilmekte olan dokunulmaz
  });

  testWidgets('İzleyici yazamaz', (tester) async {
    await pump(tester, const ChatState(), canWrite: false);
    expect(find.text('İzleyiciler yazamaz.'), findsOneWidget);
    expect(find.byType(TextField), findsNothing);
  });

  testWidgets('320 dp ve %200 yazı: taşma yok; dokunma hedefleri', (
    tester,
  ) async {
    useSmallScreen(tester);
    final chat = const ChatState()
        .withHistory([
          msg(1, 'u2', text: 'Uzun bir mesaj ' * 8),
          msg(2, 'me', text: 'kısa'),
        ])
        .withPending([
          const PendingMessage(
            clientMsgId: 'c2',
            text: 'olmadı',
            error: ApiException(kind: ApiErrorKind.rateLimited),
          ),
        ]);
    await pump(tester, chat);
    expect(tester.takeException(), isNull);
    await pump(tester, chat, canWrite: false);
    expect(tester.takeException(), isNull);
    useSmallScreen(tester, size: const Size(360, 640), textScale: 1);
    await pump(tester, chat);
    await expectAccessibleTapTargets(tester);
  });
}
