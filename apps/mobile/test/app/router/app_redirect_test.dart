import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/app/router/app_redirect.dart';
import 'package:siginak/app/router/app_routes.dart';
import 'package:siginak/features/session/session.dart';

void main() {
  const roomId = '0f8fad5b-d9cb-469f-a165-70867728950e';
  const unknown = SessionState.unknown();
  const signedOut = SessionState.unauthenticated();
  const signedIn = SessionState(status: SessionStatus.authenticated);
  const inRoom = SessionState(
    status: SessionStatus.authenticated,
    activeRoomId: roomId,
  );

  String? redirect(String path, SessionState s, {bool update = false}) =>
      appRedirect(uri: Uri.parse(path), session: s, updateRequired: update);

  group('güncelleme gerekli', () {
    test('her adres /update\'e yönlenir', () {
      for (final path in ['/', '/home', '/room/$roomId', '/welcome']) {
        expect(redirect(path, signedIn, update: true), AppRoutes.update);
      }
    });

    test('/update üzerinde kalınır', () {
      expect(redirect('/update', signedIn, update: true), isNull);
    });

    test('güncelleme gerekmiyorsa /update açılışa döner', () {
      expect(redirect('/update', signedIn), AppRoutes.splash);
    });
  });

  group('oturum bilinmiyor', () {
    test('açılış ekranında beklenir', () {
      expect(redirect('/', unknown), isNull);
      expect(redirect('/home', unknown), AppRoutes.splash);
    });
  });

  group('oturum yok', () {
    test('yalnızca /welcome açık', () {
      expect(redirect('/welcome', signedOut), isNull);
      for (final path in [
        '/',
        '/home',
        '/join',
        '/settings',
        '/room/$roomId',
      ]) {
        expect(redirect(path, signedOut), AppRoutes.welcome, reason: path);
      }
    });
  });

  group('oturum var', () {
    test('/ ve /welcome ana ekrana gider', () {
      expect(redirect('/', signedIn), AppRoutes.home);
      expect(redirect('/welcome', signedIn), AppRoutes.home);
    });

    test('aktif oda varsa / ve /welcome odaya gider', () {
      expect(redirect('/', inRoom), AppRoutes.room(roomId));
      expect(redirect('/welcome', inRoom), AppRoutes.room(roomId));
    });

    test('diğer adreslere dokunulmaz', () {
      for (final path in ['/home', '/join', '/settings', '/settings/blocks']) {
        expect(redirect(path, signedIn), isNull, reason: path);
      }
    });
  });

  group('kullanıcı adı onaylanmamış', () {
    const legacy = SessionState(
      status: SessionStatus.authenticated,
      activeRoomId: roomId,
      needsNickname: true,
    );

    test('yalnızca /nickname açık', () {
      expect(redirect('/nickname', legacy), isNull);
      for (final path in [
        '/',
        '/welcome',
        '/home',
        '/join',
        '/friends',
        '/settings',
        '/room/$roomId',
      ]) {
        expect(redirect(path, legacy), AppRoutes.nickname, reason: path);
      }
    });

    test('onaylanınca /nickname ana ekrana veya odaya gider', () {
      expect(redirect('/nickname', signedIn), AppRoutes.home);
      expect(redirect('/nickname', inRoom), AppRoutes.room(roomId));
      expect(redirect('/friends', signedIn), isNull);
    });
  });

  test('oda kimliği UUID olmalı', () {
    expect(AppRoutes.isValidRoomId(roomId), isTrue);
    expect(AppRoutes.isValidRoomId('abc'), isFalse);
    expect(AppRoutes.isValidRoomId('../settings'), isFalse);
  });
}
