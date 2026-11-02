import 'package:flutter/widgets.dart';

import 'package:siginak/app/app.dart';
import 'package:siginak/core/config/app_config.dart';

void main() {
  final config = AppConfig.fromEnvironment();
  runApp(SiginakApp(config: config));
}
