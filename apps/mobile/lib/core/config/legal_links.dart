/// Yasal metinlerin herkese açık adresleri (legal/README.md; GitHub Pages).
///
/// Derlemede `--dart-define=PRIVACY_URL=…` / `TERMS_URL=…` ile değiştirilebilir.
abstract final class LegalLinks {
  static final privacy = Uri.parse(
    const String.fromEnvironment(
      'PRIVACY_URL',
      defaultValue:
          'https://burakkuru5534.github.io/siginak-legal/gizlilik.html',
    ),
  );

  static final terms = Uri.parse(
    const String.fromEnvironment(
      'TERMS_URL',
      defaultValue:
          'https://burakkuru5534.github.io/siginak-legal/kosullar.html',
    ),
  );
}
