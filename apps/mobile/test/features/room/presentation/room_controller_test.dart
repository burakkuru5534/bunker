import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/realtime/realtime_client_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/storage_providers.dart';
import 'package:siginak/features/room/presentation/room_controller.dart';
import 'package:siginak/features/room/room.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/fake_ws.dart';
import '../../../helpers/pump_app.dart';

void main() {
  const roomId = '44444444-4444-4444-8444-444444444444';
  late FakeBackend backend;
  late FakeWsConnector ws;
  late ProviderContainer c;

  setUp(() {
    backend = FakeBackend();
    ws = FakeWsConnector();
    c = ProviderContainer(
      overrides: [
        appConfigProvider.overrideWithValue(testConfig()),
        secureStoreProvider.overrideWithValue(InMemorySecureStore()),
        apiHttpAdapterProvider.overrideWithValue(backend.adapter),
        wsConnectorProvider.overrideWithValue(ws.call),
      ],
    );
    addTearDown(c.dispose);
  });

  Future<void> flush() async {
    for (var i = 0; i < 10; i++) {
      await Future<void>.delayed(Duration.zero);
    }
  }

  final provider = roomControllerProvider(roomId);
  RoomState read() => c.read(provider);

  /// Denetleyiciyi başlatır (canlı tutar) ve snapshot gönderir.
  Future<void> open({
    String me = 'u1',
    List<Map<String, Object?>>? members,
    int version = 5,
  }) async {
    c.listen(provider, (_, _) {});
    await flush();
    ws.last.serverSend(
      'session.snapshot',
      payload: snapshotPayload(
        roomId: roomId,
        me: me,
        version: version,
        members:
            members ??
            [wsMember('u1', 'Ayşe'), wsMember('u2', 'Mehmet', ready: true)],
      ),
    );
    await flush();
  }

  test('açılınca odaya bağlanır; snapshot odayı ve beni doldurur', () async {
    c.listen(provider, (_, _) {});
    await flush();
    expect(ws.uris.single.queryParameters['room_id'], roomId);
    expect(read().room, isNull);

    await open();

    final s = read();
    expect(s.room!.members.map((m) => m.nickname), ['Ayşe', 'Mehmet']);
    expect(s.myUserId, 'u1');
    expect(s.isHost, isTrue);
    expect(s.me!.nickname, 'Ayşe');
  });

  test('oda olayları uygulanır, sürüm ilerler', () async {
    await open();
    ws.last
      ..serverSend(
        'room.member_joined',
        stateVersion: 6,
        payload: {'member': wsMember('u3', 'Zeynep')},
      )
      ..serverSend(
        'room.ready_changed',
        stateVersion: 7,
        payload: {'user_id': 'u1', 'is_ready': true},
      )
      ..serverSend(
        'room.settings_changed',
        stateVersion: 8,
        payload: {'capacity': 1},
      );
    await flush();

    final room = read().room!;
    expect(room.members.map((m) => m.userId), ['u1', 'u2', 'u3']);
    expect(room.memberOf('u1')!.isReady, isTrue);
    expect(room.capacity, 1);
    expect(room.version, 8);
  });

  test('sahip ayrılınca member_left + host_changed (aynı sürüm)', () async {
    await open(me: 'u2');
    expect(read().isHost, isFalse);

    ws.last
      ..serverSend(
        'room.member_left',
        stateVersion: 6,
        payload: {'user_id': 'u1'},
      )
      ..serverSend(
        'room.host_changed',
        stateVersion: 6,
        payload: {'host_user_id': 'u2'},
      );
    await flush();

    expect(read().room!.members.map((m) => m.userId), ['u2']);
    expect(read().isHost, isTrue);
  });

  test('başkası atılırsa listeden çıkar; ben atılırsam çıkış', () async {
    await open(me: 'u2');
    ws.last.serverSend(
      'room.member_kicked',
      stateVersion: 6,
      payload: {'user_id': 'u2'},
    );
    await flush();

    expect(read().exit, RoomExit.kicked);
  });

  test('4003 kapanışı → forbidden çıkışı', () async {
    await open();
    ws.last.serverClose(4003);
    await flush();

    expect(read().exit, RoomExit.forbidden);
  });

  test('atılma sonrası gelen 4003 nedeni değiştirmez', () async {
    await open(me: 'u2');
    ws.last.serverSend(
      'room.member_kicked',
      stateVersion: 6,
      payload: {'user_id': 'u2'},
    );
    await flush();
    ws.last.serverClose(4003);
    await flush();

    expect(read().exit, RoomExit.kicked);
  });

  test('oyun başlar, biter, lobiye döner', () async {
    await open();
    ws.last.serverSend(
      'game.started',
      stateVersion: 1,
      payload: {'game_id': 'g1'},
    );
    await flush();
    expect(read().room!.status, RoomStatus.inGame);
    expect(read().room!.currentGameId, 'g1');
    expect(read().game, isNotNull);

    ws.last.serverSend('game.finished', stateVersion: 2);
    await flush();
    expect(read().room!.status, RoomStatus.finished);

    final lobby = snapshotPayload(
      roomId: roomId,
      me: 'u1',
      version: 6,
      members: [wsMember('u1', 'Ayşe')],
    )['room'];
    ws.last.serverSend(
      'room.returned_to_lobby',
      stateVersion: 6,
      payload: {'room': lobby},
    );
    await flush();
    expect(read().room!.status, RoomStatus.lobby);
    expect(read().room!.members, hasLength(1));
    expect(read().game, isNull);
  });

  test('bozuk olay yükü → sync istenir', () async {
    await open();
    ws.last.serverSend(
      'room.member_joined',
      stateVersion: 6,
      payload: {
        'member': {'user_id': 'u3'},
      },
    );
    await flush();

    final sent = ws.last.sent.map((m) => jsonDecode(m) as Map).toList();
    expect(sent.single['type'], 'sync');
    expect(read().room!.members, hasLength(2));
  });

  test('eylemler doğru uç noktalara gider', () async {
    final routes = <String>[];
    for (final (m, path) in [
      ('POST', '/v1/rooms/$roomId/ready'),
      ('POST', '/v1/rooms/$roomId/kick'),
      ('POST', '/v1/rooms/$roomId/start'),
    ]) {
      backend.on(m, path, (r) async {
        routes.add('$m $path ${jsonEncode(r.data)}');
        return ResponseBodyNoContent.value;
      });
    }
    backend.on('PATCH', '/v1/rooms/$roomId/settings', (r) async {
      routes.add('PATCH settings ${jsonEncode(r.data)}');
      return jsonOk(
        snapshotPayload(
          roomId: roomId,
          me: 'u1',
          members: [wsMember('u1', 'Ayşe')],
        )['room'],
      );
    });
    await open();
    final ctrl = c.read(provider.notifier);

    await ctrl.setReady(ready: true);
    await ctrl.setCapacity(1);
    await ctrl.kick('u2');
    await ctrl.start();

    expect(routes, [
      'POST /v1/rooms/$roomId/ready {"ready":true}',
      'PATCH settings {"capacity":1}',
      'POST /v1/rooms/$roomId/kick {"user_id":"u2"}',
      'POST /v1/rooms/$roomId/start null',
    ]);
  });

  test('ayrılma: istek gider, çıkış left, bağlantı kapanır', () async {
    backend.on(
      'POST',
      '/v1/rooms/$roomId/leave',
      (r) async => ResponseBodyNoContent.value,
    );
    await open();

    await c.read(provider.notifier).leave();
    await flush();

    expect(backend.calls, contains('POST /v1/rooms/$roomId/leave'));
    expect(read().exit, RoomExit.left);
    expect(ws.last.closedByClient, isTrue);
  });

  test('ekran kapanınca bağlantı kapanır', () async {
    final sub = c.listen(provider, (_, _) {});
    await flush();
    sub.close();
    await flush();
    await flush();

    expect(ws.last.closedByClient, isTrue);
  });

  group('startBlockerOf', () {
    RoomView room(List<Map<String, Object?>> members) => RoomView.fromJson(
      snapshotPayload(roomId: roomId, me: 'u1', members: members)['room'],
    );
    List<Map<String, Object?>> players(int n, {int ready = 0}) => [
      for (var i = 0; i < n; i++)
        wsMember('u$i', 'Oyuncu $i', ready: i < ready),
    ];

    test('oyuncu sayısı ve hazırlık', () {
      expect(startBlockerOf(room(players(3, ready: 3))), isA<TooFewPlayers>());
      expect(
        startBlockerOf(room(players(4, ready: 3))),
        isA<NotAllReady>()
            .having((b) => b.ready, 'ready', 3)
            .having((b) => b.total, 'total', 4),
      );
      expect(startBlockerOf(room(players(4, ready: 4))), isNull);
      expect(startBlockerOf(room(players(10, ready: 10))), isNull);
      expect(
        startBlockerOf(room(players(11, ready: 11))),
        isA<TooManyPlayers>(),
      );
    });

    test('izleyiciler sayılmaz', () {
      final members = [
        ...players(4, ready: 4),
        wsMember('s', 'İzleyici', role: 'spectator'),
      ];
      expect(startBlockerOf(room(members)), isNull);
    });
  });
}
