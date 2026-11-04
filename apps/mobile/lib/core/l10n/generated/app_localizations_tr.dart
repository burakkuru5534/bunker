// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Turkish (`tr`).
class AppLocalizationsTr extends AppLocalizations {
  AppLocalizationsTr([String locale = 'tr']) : super(locale);

  @override
  String get appTitle => 'Sığınak';

  @override
  String get actionRetry => 'Tekrar dene';

  @override
  String get actionCancel => 'Vazgeç';

  @override
  String get actionGoHome => 'Ana ekrana dön';

  @override
  String loadingLabel(String label) {
    return '$label, yükleniyor';
  }

  @override
  String get notFoundMessage => 'Aradığın sayfa bulunamadı.';

  @override
  String get connectionReconnecting => 'Bağlantı koptu, yeniden bağlanıyoruz…';

  @override
  String get connectionRestored => 'Bağlandı';

  @override
  String get connectionOffline => 'Bağlantı yok';

  @override
  String get statusReady => 'Hazır';

  @override
  String get statusWaiting => 'Bekliyor';

  @override
  String get statusHost => 'Oda sahibi';

  @override
  String get statusSpectator => 'İzleyici';

  @override
  String get statusEliminated => 'Elendi';

  @override
  String get statusLeft => 'Ayrıldı';

  @override
  String get statusTurn => 'Sırada';

  @override
  String avatarDisconnected(String name) {
    return '$name, bağlantısı koptu';
  }

  @override
  String avatarEliminated(String name) {
    return '$name, elendi';
  }

  @override
  String inviteCodeLabel(int length) {
    return 'Davet kodu, $length karakter';
  }

  @override
  String get errorNetwork => 'İnternet bağlantını kontrol edip tekrar dene.';

  @override
  String get errorTimeout => 'Sunucu zamanında yanıt vermedi. Tekrar dene.';

  @override
  String get errorUnauthorized =>
      'Oturumun sona erdi. Yeniden başlaman gerekiyor.';

  @override
  String get errorUpgradeRequired =>
      'Uygulamanın yeni bir sürümü var. Devam etmek için güncelle.';

  @override
  String errorRateLimited(int seconds) {
    return 'Çok hızlı gidiyorsun. $seconds saniye sonra tekrar dene.';
  }

  @override
  String get errorRateLimitedNoWait =>
      'Çok hızlı gidiyorsun. Biraz bekleyip tekrar dene.';

  @override
  String get errorUnknown => 'Bir şeyler ters gitti. Tekrar dene.';

  @override
  String get errorInternal =>
      'Sunucuda bir sorun oluştu. Biraz sonra tekrar dene.';

  @override
  String get errorValidationFailed => 'Bilgileri kontrol edip tekrar dene.';

  @override
  String get errorNicknameInvalid =>
      'Takma ad 3–16 karakter olmalı; harf, rakam, boşluk ve _ kullanabilirsin.';

  @override
  String get errorMessageInvalid =>
      'Mesaj gönderilemedi. 1–200 karakter olmalı ve uygunsuz ifade içermemeli.';

  @override
  String get errorForbidden => 'Bu işlem için yetkin yok.';

  @override
  String get errorNotAMember => 'Bu odanın üyesi değilsin.';

  @override
  String get errorKicked => 'Bu odadan çıkarıldın, tekrar katılamazsın.';

  @override
  String get errorNotFound => 'Aradığın şey bulunamadı.';

  @override
  String get errorInviteCodeNotFound =>
      'Bu koda ait bir oda yok ya da oda kapanmış.';

  @override
  String get errorRoomFull => 'Bu oda dolu. Başka bir kod dene.';

  @override
  String get errorGameInProgress =>
      'Bu odada oyun başlamış. Bitince tekrar dene.';

  @override
  String get errorNicknameTakenInRoom =>
      'Bu odada aynı takma adla biri var. Takma adını değiştir.';

  @override
  String get errorAlreadyInRoom =>
      'Zaten başka bir odadasın. Önce oradan ayrıl.';

  @override
  String get errorInvalidPhase => 'Bu işlem şu an yapılamıyor.';

  @override
  String get errorNotYourTurn => 'Sıra sende değil.';

  @override
  String get errorCardAlreadyRevealed => 'Bu kart zaten açık.';

  @override
  String get errorCategoryNotAllowed =>
      'İlk turda yalnızca meslek kartını açabilirsin.';

  @override
  String get errorAlreadyVoted => 'Bu turda oyunu zaten verdin.';

  @override
  String get errorSelfVote => 'Kendine oy veremezsin.';

  @override
  String get errorInvalidTarget => 'Bu oyuncu seçilemez.';

  @override
  String get errorNotAlive => 'Elendiğin için bu işlemi yapamazsın.';

  @override
  String get errorNotSpectator =>
      'Eşitlik oylamasını yalnızca izleyiciler yapar.';

  @override
  String get errorStartConditionsNotMet =>
      'Başlatmak için 4–10 oyuncu ve herkesin hazır olması gerekir.';

  @override
  String get errorStaleState => 'Oyun durumu değişti, ekran güncelleniyor.';

  @override
  String get realtimeForbidden => 'Bu odaya artık erişimin yok.';

  @override
  String get realtimeReplaced =>
      'Bu oda başka bir cihazda ya da pencerede açıldı.';

  @override
  String get errorWidgetMessage => 'Bu bölüm gösterilemedi.';

  @override
  String get welcomeTitle => 'Sığınağa kim girecek?';

  @override
  String get welcomeRule1 => 'Bir felaket oldu; sığınakta herkese yer yok.';

  @override
  String get welcomeRule2 =>
      'Kartlarını açarak kendini savun, diğerlerini sorgula.';

  @override
  String get welcomeRule3 => 'Her tur oylama var. Son kalanlar sığınağa girer.';

  @override
  String get welcomeNicknameLabel => 'Takma adın';

  @override
  String get welcomeNicknameHelper => '3–16 karakter: harf, rakam, boşluk ve _';

  @override
  String get welcomeDeviceNote =>
      'Hesabın bu cihaza bağlı. Uygulamayı silersen geri getirilemez.';

  @override
  String get welcomeStart => 'Başla';

  @override
  String get nicknameErrorRequired => 'Bir takma ad yaz.';

  @override
  String get nicknameErrorTooShort => 'En az 3 karakter olmalı.';

  @override
  String get nicknameErrorTooLong => 'En fazla 16 karakter olabilir.';

  @override
  String get nicknameErrorInvalidChars =>
      'Yalnızca harf, rakam, boşluk ve _ kullanabilirsin.';

  @override
  String get nicknameErrorInvalid =>
      'En az bir harf olmalı; art arda boşluk olmamalı.';

  @override
  String get nicknameErrorNotAllowed =>
      'Bu takma ad kullanılamaz, başka bir tane seç.';

  @override
  String get nicknameErrorProfanity =>
      'Bu takma ad uygun değil, başka bir tane seç.';

  @override
  String get splashChecking => 'Oturumun kontrol ediliyor…';

  @override
  String get actionSave => 'Kaydet';

  @override
  String get actionSettings => 'Ayarlar';

  @override
  String get nicknameDialogTitle => 'Takma adını değiştir';

  @override
  String homeGreeting(String name) {
    return 'Merhaba, $name';
  }

  @override
  String get homeCreateRoom => 'Oda oluştur';

  @override
  String get homeJoinRoom => 'Koda katıl';

  @override
  String get homeBackToRoom => 'Odana dön';

  @override
  String get homeHowToTitle => 'Nasıl oynanır?';

  @override
  String get homeHowTo1 =>
      '4–10 kişi aynı odada toplanır; oda sahibi davet kodunu paylaşır.';

  @override
  String get homeHowTo2 =>
      'Herkese gizli 5 kart dağıtılır: meslek, sağlık, hobi, eşya ve bir sır.';

  @override
  String get homeHowTo3 =>
      'Her turda sırayla birer kart açılır, ardından tartışma başlar.';

  @override
  String get homeHowTo4 =>
      'Oylamada en çok oyu alan dışarıda kalır ve izleyici olur.';

  @override
  String get homeHowTo5 =>
      'Sığınağın kapasitesi kadar kişi kalınca oyun biter; kalanlar kazanır.';

  @override
  String get joinHint => 'Oda sahibinden aldığın 6 karakterlik kodu gir.';

  @override
  String get joinSubmit => 'Katıl';

  @override
  String get joinInvalidCode =>
      'Bu kod geçerli değil. Kontrol edip tekrar dene.';
}
