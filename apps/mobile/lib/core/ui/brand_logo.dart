import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';

/// Logo görseli (branding/make_icons.py ile üretilir).
const logoAsset = 'assets/images/logo.webp';

/// "BUNKER — Son Kalanlar" logosu; ekran okuyucuya uygulama adını söyler.
class BrandLogo extends StatelessWidget {
  const BrandLogo({this.maxWidth = 360, super.key});

  final double maxWidth;

  @override
  Widget build(BuildContext context) => ConstrainedBox(
    constraints: BoxConstraints(maxWidth: maxWidth),
    child: Image.asset(
      logoAsset,
      semanticLabel: context.l10n.appTitle,
      fit: BoxFit.contain,
    ),
  );
}
