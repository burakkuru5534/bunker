import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/core/ui/brand_logo.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';
import 'package:siginak/features/session/session.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/pump_app.dart';
import '../../../helpers/small_screen.dart';

void main() {
  late FakeBackend backend;
  late List<String> opened;
  setUp(() {
    backend = FakeBackend();
    opened = [];
  });

  Future<ProviderContainer> pump(
    WidgetTester tester, {
    VoidCallback? onJoin,
    VoidCallback? onOpenSettings,
    VoidCallback? onOpenFriends,
    ValueChanged<String>? onJoinInvite,
    AppConfig? config,
  }) async {
    final store = InMemorySecureStore();
    await SessionStore(store).save(
      StoredSession(
        token: 't',
        expiresAt: DateTime.now().add(const Duration(days: 30)),
        userId: 'u-guest',
        nickname: 'Ayşe',
      ),
    );
    await pumpWithProviders(
      tester,
      HomeScreen(
        onJoin: onJoin ?? () {},
        onOpenSettings: onOpenSettings ?? () {},
        onOpenRoom: opened.add,
        onOpenFriends: onOpenFriends ?? () {},
        onJoinInvite: onJoinInvite ?? (_) {},
      ),
      config: config,
      secureStore: store,
      overrides: [apiHttpAdapterProvider.overrideWithValue(backend.adapter)],
    );
    await tester.pumpAndSettle();
    return ProviderScope.containerOf(tester.element(find.byType(HomeScreen)));
  }

  testWidgets('Logo, selamlama ve local ortam etiketi görünür', (tester) async {
    await pump(tester);

    final logo = tester.widget<Image>(
      find.descendant(of: find.byType(BrandLogo), matching: find.byType(Image)),
    );
    expect((logo.image as AssetImage).assetName, logoAsset);
    expect(logo.semanticLabel, 'BUNKER — Son Kalanlar');
    expect(find.text('SIĞINAK'), findsNothing);
    expect(find.text('Merhaba, Ayşe'), findsOneWidget);
    expect(find.textContaining('Ortam: local'), findsOneWidget);
  });

  test('ortam etiketi yalnızca debug derlemede ve üretim dışında', () {
    AppConfig config(String env) => AppConfig.fromValues(
      environment: env,
      apiBaseUrl: env == 'local'
          ? 'http://10.0.2.2:8080'
          : 'https://api.example.com',
      wsBaseUrl: env == 'local'
          ? 'ws://10.0.2.2:8080'
          : 'wss://api.example.com',
    );
    expect(showEnvironmentLabel(config('local'), debugBuild: true), isTrue);
    expect(showEnvironmentLabel(config('test'), debugBuild: true), isTrue);
    expect(showEnvironmentLabel(config('prod'), debugBuild: true), isFalse);
    // Release derlemesi (mağaza ekran görüntüleri, kapalı test): hiç görünmez.
    expect(showEnvironmentLabel(config('test'), debugBuild: false), isFalse);
    expect(showEnvironmentLabel(config('local'), debugBuild: false), isFalse);
  });

  testWidgets('Production ortamında ortam etiketi gösterilmez', (tester) async {
    final prod = AppConfig.fromValues(
      environment: 'prod',
      apiBaseUrl: 'https://api.example.com',
      wsBaseUrl: 'wss://api.example.com',
    );
    await pump(tester, config: prod);

    expect(find.textContaining('Ortam:'), findsNothing);
  });

  testWidgets('Koda katıl ve ayarlar geri çağrıları tetiklenir', (
    tester,
  ) async {
    var joined = 0;
    var settings = 0;
    await pump(
      tester,
      onJoin: () => joined++,
      onOpenSettings: () => settings++,
    );

    await tester.tap(find.text('Koda katıl'));
    await tester.tap(find.byTooltip('Ayarlar'));

    expect(joined, 1);
    expect(settings, 1);
  });

  testWidgets('Oda oluştur → oda açılır, aktif oda güncellenir', (
    tester,
  ) async {
    final c = await pump(tester);

    await tester.tap(find.text('Oda oluştur'));
    await tester.pumpAndSettle();

    expect(backend.calls.last, 'POST /v1/rooms');
    expect(opened, [FakeBackend.createdRoomId]);
    expect(
      c.read(sessionControllerProvider).activeRoomId,
      FakeBackend.createdRoomId,
    );
  });

  testWidgets('Ağ hatasında snackbar; oda açılmaz, tekrar denenebilir', (
    tester,
  ) async {
    await pump(tester);
    backend.on('POST', '/v1/rooms', networkFailure);

    await tester.tap(find.text('Oda oluştur'));
    await tester.pumpAndSettle();

    expect(
      find.text('İnternet bağlantını kontrol edip tekrar dene.'),
      findsOneWidget,
    );
    expect(opened, isEmpty);

    backend.restore('POST', '/v1/rooms');
    await tester.tap(find.text('Oda oluştur'));
    await tester.pumpAndSettle();
    expect(opened, [FakeBackend.createdRoomId]);
  });

  testWidgets('Zaten odadaysa uyarı ve "Odana dön"', (tester) async {
    const existing = '33333333-3333-4333-8333-333333333333';
    backend.on(
      'POST',
      '/v1/rooms',
      (r) async => jsonResponse(
        apiError('ALREADY_IN_ROOM', details: {'room_id': existing}),
        409,
      ),
    );
    await pump(tester);

    await tester.tap(find.text('Oda oluştur'));
    await tester.pumpAndSettle();

    expect(
      find.text('Zaten başka bir odadasın. Önce oradan ayrıl.'),
      findsOneWidget,
    );
    expect(opened, isEmpty);

    await tester.tap(find.text('Odana dön'));
    await tester.pump();
    expect(opened, [existing]);
  });

  testWidgets('Oluşturma sürerken ikinci basış istek göndermez', (
    tester,
  ) async {
    final gate = Completer<void>();
    backend.on('POST', '/v1/rooms', (r) async {
      await gate.future;
      return jsonResponse(apiError('INTERNAL'), 500);
    });
    await pump(tester);

    await tester.tap(find.text('Oda oluştur'));
    for (var i = 0; i < 5; i++) {
      await tester.pump(const Duration(milliseconds: 10));
    }
    await tester.tap(find.byType(FilledButton));
    await tester.pump();

    expect(backend.calls.where((c) => c == 'POST /v1/rooms'), hasLength(1));
    gate.complete();
    await tester.pumpAndSettle();
    expect(
      find.text('Sunucuda bir sorun oluştu. Biraz sonra tekrar dene.'),
      findsOneWidget,
    );
    expect(opened, isEmpty);
  });

  group('Davetler ve arkadaşlar', () {
    Map<String, Object?> invite(String room, String code, String from) => {
      'room_id': room,
      'invite_code': code,
      'from': {'user_id': 'u-$from', 'nickname': from},
      'players': 3,
    };
    late Map<String, Object?> inbox;
    setUp(() {
      inbox = {
        'invites': [
          invite('r1', 'AAAAAA', 'Ali'),
          invite('r2', 'BBBBBB', 'Can'),
        ],
        'incoming_requests': 2,
      };
      backend.on('GET', '/v1/inbox', (r) async => jsonOk(inbox));
    });

    int inboxCalls() => backend.calls.where((c) => c == 'GET /v1/inbox').length;

    testWidgets('davet kartları ve istek rozeti görünür; Katıl kodu verir', (
      tester,
    ) async {
      final joined = <String>[];
      var friends = 0;
      await pump(
        tester,
        onJoinInvite: joined.add,
        onOpenFriends: () => friends++,
      );

      expect(find.text('Ali seni lobisine çağırıyor'), findsOneWidget);
      expect(find.text('Can seni lobisine çağırıyor'), findsOneWidget);
      expect(find.text('3 oyuncu bekliyor'), findsNWidgets(2));
      expect(find.byTooltip('Arkadaşlar, 2 yeni istek'), findsOneWidget);
      expect(find.text('2'), findsOneWidget);

      await tester.tap(
        find.descendant(
          of: find.byKey(const ValueKey('invite-r2')),
          matching: find.byKey(const ValueKey('invite-join')),
        ),
      );
      expect(joined, ['BBBBBB']);

      await tester.tap(find.byKey(const ValueKey('home-friends')));
      expect(friends, 1);
    });

    testWidgets('istek yoksa rozet gizli', (tester) async {
      inbox = {'invites': <Object?>[], 'incoming_requests': 0};
      await pump(tester);
      expect(find.byTooltip('Arkadaşlar'), findsOneWidget);
      final badge = tester.widget<Badge>(find.byType(Badge));
      expect(badge.isLabelVisible, isFalse);
      expect(find.textContaining('lobisine çağırıyor'), findsNothing);
    });

    testWidgets('Reddet kartı hemen gizler ve sunucuya bildirir', (
      tester,
    ) async {
      backend.on('DELETE', '/v1/invites/r1', (r) async {
        inbox = {
          'invites': [invite('r2', 'BBBBBB', 'Can')],
          'incoming_requests': 0,
        };
        return ResponseBodyNoContent.value;
      });
      await pump(tester);

      await tester.tap(
        find.descendant(
          of: find.byKey(const ValueKey('invite-r1')),
          matching: find.byKey(const ValueKey('invite-decline')),
        ),
      );
      await tester.pumpAndSettle();

      expect(backend.calls, contains('DELETE /v1/invites/r1'));
      expect(find.text('Ali seni lobisine çağırıyor'), findsNothing);
      expect(find.text('Can seni lobisine çağırıyor'), findsOneWidget);
    });

    testWidgets('Reddetme hatası ekranı bozmaz', (tester) async {
      backend.on(
        'DELETE',
        '/v1/invites/r1',
        (r) async => jsonResponse(apiError('NOT_FOUND'), 404),
      );
      await pump(tester);
      await tester.tap(
        find.descendant(
          of: find.byKey(const ValueKey('invite-r1')),
          matching: find.byKey(const ValueKey('invite-decline')),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Ali seni lobisine çağırıyor'), findsNothing);
      expect(tester.takeException(), isNull);
    });

    testWidgets('20 sn\'de bir yoklanır; arka planda durur, önde tazelenir', (
      tester,
    ) async {
      await pump(tester);
      expect(inboxCalls(), 1);

      await tester.pump(const Duration(seconds: 19));
      await tester.pump();
      expect(inboxCalls(), 1);
      await tester.pump(const Duration(seconds: 1));
      await tester.pump();
      expect(inboxCalls(), 2);

      // Yeni davet bir sonraki yoklamada görünür.
      inbox = {
        'invites': [invite('r3', 'CCCCCC', 'Ece')],
        'incoming_requests': 0,
      };
      await tester.pump(const Duration(seconds: 20));
      await tester.pumpAndSettle();
      expect(find.text('Ece seni lobisine çağırıyor'), findsOneWidget);
      final before = inboxCalls();

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
      await tester.pump(const Duration(seconds: 60));
      await tester.pump();
      expect(inboxCalls(), before);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();
      expect(inboxCalls(), before + 1);
      await tester.pump(const Duration(seconds: 20));
      await tester.pump();
      expect(inboxCalls(), before + 2);
    });

    testWidgets('yoklama hatasında son davetler kalır', (tester) async {
      await pump(tester);
      backend.on(
        'GET',
        '/v1/inbox',
        (r) async => jsonResponse(apiError('INTERNAL'), 500),
      );
      await tester.pump(const Duration(seconds: 20));
      await tester.pumpAndSettle();
      expect(find.text('Ali seni lobisine çağırıyor'), findsOneWidget);
    });

    testWidgets('320 dp ve %200 yazı: davet kartı taşmaz', (tester) async {
      useSmallScreen(tester);
      await pump(tester);
      expect(tester.takeException(), isNull);
    });
  });

  testWidgets('"Nasıl oynanır?" açılır', (tester) async {
    await pump(tester);
    expect(find.textContaining('Herkese gizli 5 kart'), findsNothing);

    await tester.tap(find.text('Nasıl oynanır?'));
    await tester.pumpAndSettle();

    expect(find.textContaining('Herkese gizli 5 kart'), findsOneWidget);
  });

  testWidgets('320 dp ve %200 yazı: taşma yok; dokunma hedefleri', (
    tester,
  ) async {
    useSmallScreen(tester);
    await pump(tester);
    expect(tester.takeException(), isNull);
    useSmallScreen(tester, size: const Size(360, 640), textScale: 1);
    await tester.pumpAndSettle();
    await expectAccessibleTapTargets(tester);
  });
}
