import 'package:flutter_test/flutter_test.dart';
import 'package:siginak/core/config/app_config.dart';

void main() {
  group('AppConfig', () {
    test('dart-define verilmezse local emülatör adreslerini kullanır', () {
      final config = AppConfig.fromEnvironment();

      expect(config.environment, AppEnvironment.local);
      expect(config.apiBaseUrl.toString(), 'http://10.0.2.2:8080');
      expect(config.wsBaseUrl.toString(), 'ws://10.0.2.2:8080');
      expect(config.isProduction, isFalse);
    });

    test('local ortamda http/ws kabul edilir', () {
      final config = AppConfig.fromValues(
        environment: 'local',
        apiBaseUrl: 'http://192.168.1.20:8080',
        wsBaseUrl: 'ws://192.168.1.20:8080',
      );

      expect(config.apiBaseUrl.host, '192.168.1.20');
      expect(config.apiBaseUrl.port, 8080);
    });

    test('prod ortamda https/wss ile oluşur', () {
      final config = AppConfig.fromValues(
        environment: 'prod',
        apiBaseUrl: 'https://api.example.com/',
        wsBaseUrl: 'wss://api.example.com',
      );

      expect(config.isProduction, isTrue);
      expect(config.apiBaseUrl.toString(), 'https://api.example.com');
    });

    test('test/prod ortamında düz http reddedilir', () {
      expect(
        () => AppConfig.fromValues(
          environment: 'test',
          apiBaseUrl: 'http://api.example.com',
          wsBaseUrl: 'wss://api.example.com',
        ),
        throwsArgumentError,
      );
    });

    test('prod ortamında düz ws reddedilir', () {
      expect(
        () => AppConfig.fromValues(
          environment: 'prod',
          apiBaseUrl: 'https://api.example.com',
          wsBaseUrl: 'ws://api.example.com',
        ),
        throwsArgumentError,
      );
    });

    test('bilinmeyen ortam reddedilir', () {
      expect(
        () => AppConfig.fromValues(
          environment: 'staging',
          apiBaseUrl: 'https://api.example.com',
          wsBaseUrl: 'wss://api.example.com',
        ),
        throwsArgumentError,
      );
    });

    test('yanlış şema veya boş host reddedilir', () {
      expect(
        () => AppConfig.fromValues(
          environment: 'local',
          apiBaseUrl: 'ws://10.0.2.2:8080',
          wsBaseUrl: 'ws://10.0.2.2:8080',
        ),
        throwsArgumentError,
      );
      expect(
        () => AppConfig.fromValues(
          environment: 'local',
          apiBaseUrl: 'not a url',
          wsBaseUrl: 'ws://10.0.2.2:8080',
        ),
        throwsArgumentError,
      );
    });

    test('yol içeren adres reddedilir', () {
      expect(
        () => AppConfig.fromValues(
          environment: 'prod',
          apiBaseUrl: 'https://api.example.com/v1',
          wsBaseUrl: 'wss://api.example.com',
        ),
        throwsArgumentError,
      );
    });
  });
}
