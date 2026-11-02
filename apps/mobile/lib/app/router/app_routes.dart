/// Uygulamanın rota yolları (bkz. docs/ui/screen-map.md §2).
abstract final class AppRoutes {
  static const splash = '/';
  static const welcome = '/welcome';
  static const home = '/home';
  static const join = '/join';
  static const roomPattern = '/room/:roomId';
  static const settings = '/settings';
  static const blockedUsers = '/settings/blocks';
  static const update = '/update';

  static String room(String roomId) => '/room/$roomId';

  static final _uuid = RegExp(
    r'^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$',
  );

  /// Oda kimliği UUID biçiminde mi?
  static bool isValidRoomId(String value) => _uuid.hasMatch(value);
}
