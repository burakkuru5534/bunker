import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:url_launcher/url_launcher.dart';

/// Bağlantıyı cihazın tarayıcısında açar; açıldıysa `true`.
typedef LinkOpener = Future<bool> Function(Uri uri);

/// Testler sahte açıcıyla override eder.
final linkOpenerProvider = Provider<LinkOpener>(
  (ref) =>
      (uri) => launchUrl(uri, mode: LaunchMode.externalApplication),
);

/// Bağlantıyı açar; açılamazsa kısa bir hata gösterir.
Future<void> openExternalLink(
  BuildContext context,
  WidgetRef ref,
  Uri uri,
) async {
  var opened = false;
  try {
    opened = await ref.read(linkOpenerProvider)(uri);
  } on Object {
    opened = false;
  }
  if (!opened && context.mounted) {
    showAppSnackBar(context, context.l10n.legalOpenFailed, isError: true);
  }
}
