import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/app_typography.dart';

/// Davet kodu alfabesi (docs/api.md §3): 0/O ve 1/I/L yok.
const inviteCodeAlphabet = '23456789ABCDEFGHJKMNPQRSTUVWXYZ';

/// Davet kodu uzunluğu.
const inviteCodeLength = 6;

/// Girdiyi büyük harfe çevirir, alfabe dışı karakterleri atar, 6 ile sınırlar.
class InviteCodeFormatter extends TextInputFormatter {
  const InviteCodeFormatter();

  /// Ham metni geçerli davet kodu karakterlerine indirger.
  static String normalize(String raw) {
    final buffer = StringBuffer();
    for (final ch in raw.toUpperCase().split('')) {
      if (inviteCodeAlphabet.contains(ch)) buffer.write(ch);
      if (buffer.length == inviteCodeLength) break;
    }
    return buffer.toString();
  }

  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    final text = normalize(newValue.text);
    return TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: text.length),
    );
  }
}

/// 6 kutulu davet kodu girişi.
///
/// Arkada tek bir görünmez `TextField` vardır (yapıştırma ve klavye
/// davranışı doğal çalışır); kutular yalnızca görsel gösterimdir.
class InviteCodeInput extends StatefulWidget {
  const InviteCodeInput({
    this.controller,
    this.onChanged,
    this.onCompleted,
    this.hasError = false,
    this.autofocus = false,
    super.key,
  });

  final TextEditingController? controller;
  final ValueChanged<String>? onChanged;

  /// 6 karakter tamamlanınca çağrılır.
  final ValueChanged<String>? onCompleted;
  final bool hasError;
  final bool autofocus;

  @override
  State<InviteCodeInput> createState() => _InviteCodeInputState();
}

class _InviteCodeInputState extends State<InviteCodeInput> {
  late final TextEditingController _controller =
      widget.controller ?? TextEditingController();
  final _focus = FocusNode();

  @override
  void initState() {
    super.initState();
    _controller.addListener(_rebuild);
    _focus.addListener(_rebuild);
  }

  void _rebuild() => setState(() {});

  @override
  void dispose() {
    _controller.removeListener(_rebuild);
    if (widget.controller == null) _controller.dispose();
    _focus.dispose();
    super.dispose();
  }

  void _onChanged(String value) {
    widget.onChanged?.call(value);
    if (value.length == inviteCodeLength) widget.onCompleted?.call(value);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final text = _controller.text;
    final activeIndex = _focus.hasFocus
        ? text.length.clamp(0, inviteCodeLength - 1)
        : -1;

    return Semantics(
      label: context.l10n.inviteCodeLabel(inviteCodeLength),
      value: text,
      textField: true,
      child: GestureDetector(
        onTap: _focus.requestFocus,
        child: Stack(
          children: [
            // Görünmez gerçek giriş alanı.
            Positioned.fill(
              child: Opacity(
                opacity: 0,
                child: TextField(
                  controller: _controller,
                  focusNode: _focus,
                  autofocus: widget.autofocus,
                  autocorrect: false,
                  enableSuggestions: false,
                  showCursor: false,
                  textCapitalization: TextCapitalization.characters,
                  keyboardType: TextInputType.visiblePassword,
                  inputFormatters: const [InviteCodeFormatter()],
                  onChanged: _onChanged,
                ),
              ),
            ),
            ExcludeSemantics(
              child: Row(
                children: [
                  for (var i = 0; i < inviteCodeLength; i++) ...[
                    if (i > 0) const SizedBox(width: AppSpacing.s2),
                    Expanded(
                      child: Container(
                        key: ValueKey('invite-box-$i'),
                        height: 60,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: scheme.surfaceContainerHigh,
                          borderRadius: BorderRadius.circular(AppRadius.s),
                          border: Border.all(
                            color: widget.hasError
                                ? scheme.error
                                : i == activeIndex
                                ? scheme.primary
                                : scheme.outline,
                            width: widget.hasError || i == activeIndex ? 2 : 1,
                          ),
                        ),
                        child: Text(
                          i < text.length ? text[i] : '',
                          style: theme.textTheme.headlineMedium?.copyWith(
                            fontFamily: AppFonts.display,
                          ),
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
