import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

/// Gömülü yazı tiplerinin OFL lisanslarını uygulamanın lisans sayfasına ekler.
void registerFontLicenses() {
  LicenseRegistry.addLicense(() async* {
    for (final (family, path) in const [
      ('Oswald', 'assets/fonts/oswald/OFL.txt'),
      ('Inter', 'assets/fonts/inter/OFL.txt'),
    ]) {
      final text = await rootBundle.loadString(path);
      yield LicenseEntryWithLineBreaks([family], text);
    }
  });
}
