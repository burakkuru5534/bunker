/// Uygulamanın çalıştığı ortam.
enum AppEnvironment {
  local,
  test,
  prod;

  static AppEnvironment parse(String value) {
    for (final env in AppEnvironment.values) {
      if (env.name == value) return env;
    }
    throw ArgumentError.value(value, 'APP_ENV', 'Bilinmeyen ortam');
  }
}

/// Derleme zamanında `--dart-define-from-file` ile verilen yapılandırma.
///
/// Gizli değer içermez: yalnızca herkese açık API adresleri tutulur.
/// Örnek: `flutter run --dart-define-from-file=config/local.json`
class AppConfig {
  const AppConfig({
    required this.environment,
    required this.apiBaseUrl,
    required this.wsBaseUrl,
  });

  /// `--dart-define` değerlerinden yapılandırmayı okur ve doğrular.
  factory AppConfig.fromEnvironment() {
    return AppConfig.fromValues(
      environment: const String.fromEnvironment(
        'APP_ENV',
        defaultValue: 'local',
      ),
      apiBaseUrl: const String.fromEnvironment(
        'API_BASE_URL',
        defaultValue: 'http://10.0.2.2:8080',
      ),
      wsBaseUrl: const String.fromEnvironment(
        'WS_BASE_URL',
        defaultValue: 'ws://10.0.2.2:8080',
      ),
    );
  }

  /// Ham değerlerden yapılandırma üretir; test edilebilirlik için ayrıdır.
  factory AppConfig.fromValues({
    required String environment,
    required String apiBaseUrl,
    required String wsBaseUrl,
  }) {
    final env = AppEnvironment.parse(environment);
    final api = _parseUrl(apiBaseUrl, 'API_BASE_URL', const {'http', 'https'});
    final ws = _parseUrl(wsBaseUrl, 'WS_BASE_URL', const {'ws', 'wss'});

    if (env != AppEnvironment.local) {
      if (api.scheme != 'https') {
        throw ArgumentError.value(
          apiBaseUrl,
          'API_BASE_URL',
          '$env için https zorunlu',
        );
      }
      if (ws.scheme != 'wss') {
        throw ArgumentError.value(
          wsBaseUrl,
          'WS_BASE_URL',
          '$env için wss zorunlu',
        );
      }
    }
    return AppConfig(environment: env, apiBaseUrl: api, wsBaseUrl: ws);
  }

  final AppEnvironment environment;
  final Uri apiBaseUrl;
  final Uri wsBaseUrl;

  bool get isProduction => environment == AppEnvironment.prod;

  static Uri _parseUrl(String value, String name, Set<String> schemes) {
    final uri = Uri.tryParse(value);
    if (uri == null || !schemes.contains(uri.scheme) || uri.host.isEmpty) {
      throw ArgumentError.value(
        value,
        name,
        'Geçersiz adres (izinli: ${schemes.join(', ')})',
      );
    }
    if (uri.path.isNotEmpty && uri.path != '/') {
      throw ArgumentError.value(value, name, 'Adres yol içermemeli');
    }
    return uri.replace(path: '');
  }
}
