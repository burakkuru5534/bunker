import 'dart:convert';

import 'package:fake_async/fake_async.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/realtime/realtime_client.dart';
import 'package:siginak/core/realtime/ws_envelope.dart';

import '../../helpers/fake_ws.dart';

void main() {
  const roomId = '0f8fad5b-d9cb-469f-a165-70867728950e';

  late FakeWsConnector connector;
  late RealtimeClient client;
  late List<WsEnvelope> events;
  late List<RealtimeState> states;
  String? token;
  var unauthorized = 0;
  var upgrade = 0;

  void setUpClient(FakeAsync async) {
    connector = FakeWsConnector();
    token = 'tok';
    unauthorized = 0;
    upgrade = 0;
    client = RealtimeClient(
      wsBaseUrl: Uri.parse('wss://api.example.com'),
      appVersion: '9.9.9',
      readToken: () async => token,
      connector: connector.call,
      onUnauthorized: () => unauthorized++,
      onUpgradeRequired: () => upgrade++,
      random: () => 0.5, // sapma çarpanı tam 1.0
      enableLogging: false,
    );
    events = [];
    states = [];
    client.events.listen(events.add);
    client.states.listen(states.add);
  }

  void connected(FakeAsync async) {
    client.connect(roomId);
    async.flushMicrotasks();
  }

  List<String> sentTypes(FakeWsSocket s) => s.sent
      .map((m) => (jsonDecode(m) as Map<String, Object?>)['type']! as String)
      .toList();

  test('doğru adres ve başlıklarla bağlanır; token URL\'de yer almaz', () {
    fakeAsync((async) {
      setUpClient(async);
      connected(async);

      final uri = connector.uris.single;
      expect(uri.toString(), 'wss://api.example.com/v1/ws?room_id=$roomId');
      expect(uri.toString(), isNot(contains('tok')));
      expect(connector.headers.single, {
        'X-App-Version': '9.9.9',
        'Authorization': 'Bearer tok',
      });
      expect(client.state.status, RealtimeStatus.connected);
      expect(states.map((s) => s.status), [
        RealtimeStatus.connecting,
        RealtimeStatus.connected,
      ]);
    });
  });

  group('state_version', () {
    test('snapshot sonrası sıralı olaylar iletilir, kopyalar atılır', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);
        final s = connector.last;

        s
          ..serverSend('session.snapshot', stateVersion: 10)
          ..serverSend('game.phase_changed', stateVersion: 11)
          ..serverSend('game.phase_changed', stateVersion: 11) // kopya
          ..serverSend('game.vote_progress', stateVersion: 9) // eski
          ..serverSend('game.card_revealed', stateVersion: 12);
        async.flushMicrotasks();

        expect(events.map((e) => e.type), [
          'session.snapshot',
          'game.phase_changed',
          'game.card_revealed',
        ]);
        expect(client.stateVersion, 12);
      });
    });

    test(
      'boşluk görülünce sync istenir; snapshot gelene kadar olay atlanır',
      () {
        fakeAsync((async) {
          setUpClient(async);
          connected(async);
          final s = connector.last;

          s
            ..serverSend('session.snapshot', stateVersion: 10)
            ..serverSend('game.phase_changed', stateVersion: 13) // boşluk
            ..serverSend('game.phase_changed', stateVersion: 14) // atlanır
            ..serverSend('session.snapshot', stateVersion: 14)
            ..serverSend('game.card_revealed', stateVersion: 15);
          async.flushMicrotasks();

          expect(sentTypes(s), ['sync']);
          expect(events.map((e) => e.stateVersion), [10, 14, 15]);
        });
      },
    );

    test('snapshot gelmeden sürümlü olaylar iletilmez', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);

        connector.last.serverSend('game.phase_changed', stateVersion: 5);
        async.flushMicrotasks();

        expect(events, isEmpty);
      });
    });

    test('sürümsüz olaylar (sohbet) her zaman iletilir', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);

        connector.last.serverSend('chat.message', payload: {'text': 'selam'});
        async.flushMicrotasks();

        expect(events.single.type, 'chat.message');
        expect(events.single.payload['text'], 'selam');
      });
    });

    test('bozuk JSON, yanlış sürüm ve ikili veri yok sayılır', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);
        final s = connector.last;

        s
          ..serverSendRaw('{bozuk')
          ..serverSendRaw(
            jsonEncode({'v': 2, 'type': 'x', 'payload': <String, Object?>{}}),
          )
          ..serverSendRaw([1, 2, 3]);
        async.flushMicrotasks();

        expect(events, isEmpty);
        expect(client.state.status, RealtimeStatus.connected);
      });
    });
  });

  group('yeniden bağlanma', () {
    test('beklenmedik kopmada üstel geri çekilmeyle yeniden bağlanır', () {
      fakeAsync((async) {
        setUpClient(async);
        connector.failNext = 3;
        client.connect(roomId);
        async.flushMicrotasks();
        expect(client.state.status, RealtimeStatus.reconnecting);
        expect(connector.uris, hasLength(1));

        async.elapse(const Duration(milliseconds: 499));
        expect(connector.uris, hasLength(1));
        async.elapse(const Duration(milliseconds: 1)); // 0,5 sn
        expect(connector.uris, hasLength(2));

        async.elapse(const Duration(seconds: 1)); // 1 sn
        expect(connector.uris, hasLength(3));

        async.elapse(const Duration(seconds: 2)); // 2 sn → başarılı
        expect(connector.uris, hasLength(4));
        expect(client.state.status, RealtimeStatus.connected);

        connector.last.serverClose(1006);
        async.flushMicrotasks();
        expect(client.state.status, RealtimeStatus.reconnecting);
        async.elapse(const Duration(milliseconds: 500)); // sayaç sıfırlandı
        expect(connector.uris, hasLength(5));
        expect(client.state.status, RealtimeStatus.connected);
      });
    });

    test('bekleme süresi en fazla 10 sn ve ±%20 sapmalı', () {
      fakeAsync((async) {
        setUpClient(async);
        expect(client.backoffFor(0), const Duration(milliseconds: 500));
        expect(client.backoffFor(4), const Duration(seconds: 8));
        expect(client.backoffFor(5), const Duration(seconds: 10));
        expect(client.backoffFor(50), const Duration(seconds: 10));

        final low = RealtimeClient(
          wsBaseUrl: Uri.parse('wss://x.example.com'),
          appVersion: '1',
          readToken: () async => null,
          random: () => 0,
          enableLogging: false,
        );
        final high = RealtimeClient(
          wsBaseUrl: Uri.parse('wss://x.example.com'),
          appVersion: '1',
          readToken: () async => null,
          random: () => 1,
          enableLogging: false,
        );
        expect(low.backoffFor(5), const Duration(seconds: 8));
        expect(high.backoffFor(5), const Duration(seconds: 12));
      });
    });

    for (final code in [1001, 1006, 1012, 4009]) {
      test('$code kapanışında yeniden bağlanır', () {
        fakeAsync((async) {
          setUpClient(async);
          connected(async);

          connector.last.serverClose(code);
          async
            ..flushMicrotasks()
            ..elapse(const Duration(seconds: 1));

          expect(connector.uris, hasLength(2));
        });
      });
    }

    test('yeniden bağlanınca yeni snapshot beklenir', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);
        connector.last.serverSend('session.snapshot', stateVersion: 3);
        async.flushMicrotasks();

        connector.last.serverClose(1006);
        async.elapse(const Duration(seconds: 1));
        connector.last.serverSend('game.phase_changed', stateVersion: 4);
        async.flushMicrotasks();
        expect(events.map((e) => e.stateVersion), [3]); // snapshot bekleniyor

        connector.last.serverSend('session.snapshot', stateVersion: 7);
        async.flushMicrotasks();
        expect(events.map((e) => e.stateVersion), [3, 7]);
      });
    });
  });

  group('kalıcı durma', () {
    test('4001 → unauthorized, geri çağrı, yeniden denenmez', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);

        connector.last.serverClose(4001);
        async.elapse(const Duration(seconds: 30));

        expect(client.state.status, RealtimeStatus.stopped);
        expect(client.state.stopReason, RealtimeStopReason.unauthorized);
        expect(unauthorized, 1);
        expect(connector.uris, hasLength(1));
      });
    });

    test('4026 → upgradeRequired ve geri çağrı', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);

        connector.last.serverClose(4026);
        async.elapse(const Duration(seconds: 30));

        expect(client.state.stopReason, RealtimeStopReason.upgradeRequired);
        expect(upgrade, 1);
        expect(connector.uris, hasLength(1));
      });
    });

    test('4003 → forbidden, 4008 → replaced; yeniden denenmez', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);
        connector.last.serverClose(4003);
        async.elapse(const Duration(seconds: 30));
        expect(client.state.stopReason, RealtimeStopReason.forbidden);

        client.connect(roomId); // durmuşken yeniden bağlanmak mümkün
        async.flushMicrotasks();
        connector.last.serverClose(4008);
        async.elapse(const Duration(seconds: 30));
        expect(client.state.stopReason, RealtimeStopReason.replaced);
        expect(connector.uris, hasLength(2));
      });
    });
  });

  group('kontrol', () {
    test('disconnect 1000 ile kapatır ve yeniden denemez', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);
        final s = connector.last;

        client.disconnect();
        async.elapse(const Duration(seconds: 30));

        expect(s.closedByClient, isTrue);
        expect(s.closeCode, 1000);
        expect(client.state.status, RealtimeStatus.idle);
        expect(connector.uris, hasLength(1));
      });
    });

    test('başka odaya bağlanınca eski bağlantı kapanır', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);
        final first = connector.last;

        client.connect('11111111-2222-4333-8444-555555555555');
        async.flushMicrotasks();

        expect(first.closedByClient, isTrue);
        expect(
          connector.uris.last.queryParameters['room_id'],
          '11111111-2222-4333-8444-555555555555',
        );
        expect(client.state.status, RealtimeStatus.connected);
      });
    });

    test('aynı odaya tekrar connect yeni bağlantı açmaz', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);
        connected(async);

        expect(connector.uris, hasLength(1));
      });
    });

    test('ön plana gelince: bağlıysa sync, değilse hemen bağlanır', () {
      fakeAsync((async) {
        setUpClient(async);
        connected(async);
        client.onAppResumed();
        expect(sentTypes(connector.last), ['sync']);

        connector
          ..failNext = 5
          ..last.serverClose(1006);
        async
          ..flushMicrotasks()
          ..elapse(const Duration(milliseconds: 500 + 1000 + 2000));
        expect(client.state.status, RealtimeStatus.reconnecting);
        final attempts = connector.uris.length;

        connector.failNext = 0;
        client.onAppResumed(); // 4 sn beklemeden
        async.flushMicrotasks();
        expect(connector.uris, hasLength(attempts + 1));
        expect(client.state.status, RealtimeStatus.connected);
      });
    });

    test('token yoksa Authorization gönderilmez', () {
      fakeAsync((async) {
        setUpClient(async);
        token = null;
        connected(async);

        expect(connector.headers.single.containsKey('Authorization'), isFalse);
      });
    });
  });
}
