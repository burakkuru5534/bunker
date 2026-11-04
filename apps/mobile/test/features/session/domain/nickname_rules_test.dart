import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/features/session/domain/nickname_rules.dart';

void main() {
  test('sunucu kurallarıyla aynı sonuçlar', () {
    const cases = <String, NicknameIssue?>{
      'Burak': null,
      '  Ayşe  ': null, // kırpılır
      'IŞIK_çöğü': null,
      'Oyuncu 42': null,
      'abc': null,
      'abcdefghijklmnop': null, // 16
      '': NicknameIssue.required,
      '   ': NicknameIssue.required,
      'ab': NicknameIssue.tooShort,
      'abcdefghijklmnopq': NicknameIssue.tooLong, // 17
      'a-b-c': NicknameIssue.invalidChars,
      'ali!': NicknameIssue.invalidChars,
      'Иван': NicknameIssue.invalidChars, // Kiril
      'αβγ': NicknameIssue.invalidChars, // Yunan
      'ali😀': NicknameIssue.invalidChars,
      '123': NicknameIssue.invalid, // harf yok
      '___': NicknameIssue.invalid,
      'ali  veli': NicknameIssue.invalid, // ardışık boşluk
    };
    cases.forEach((input, want) {
      expect(checkNickname(input), want, reason: '"$input"');
    });
  });

  test('uzunluk karakter (kod noktası) olarak sayılır', () {
    expect(checkNickname('ğğğ'), isNull);
    expect(checkNickname('ş' * 16), isNull);
    expect(checkNickname('ş' * 17), NicknameIssue.tooLong);
  });

  test('sunucu kodları eşlenir', () {
    for (final issue in NicknameIssue.values) {
      expect(NicknameIssue.fromCode(issue.code), issue);
    }
    expect(NicknameIssue.fromCode('bilinmeyen'), isNull);
    expect(NicknameIssue.fromCode(null), isNull);
  });
}
