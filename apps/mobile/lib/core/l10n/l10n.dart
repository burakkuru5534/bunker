import 'package:flutter/widgets.dart';

import 'package:siginak/core/l10n/generated/app_localizations.dart';

export 'package:siginak/core/l10n/generated/app_localizations.dart';

/// `context.l10n` kısayolu.
extension L10nContext on BuildContext {
  AppLocalizations get l10n => AppLocalizations.of(this);
}
