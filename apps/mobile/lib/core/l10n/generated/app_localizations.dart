import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_tr.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'generated/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[Locale('tr')];

  /// No description provided for @appTitle.
  ///
  /// In tr, this message translates to:
  /// **'Sığınak'**
  String get appTitle;

  /// No description provided for @actionRetry.
  ///
  /// In tr, this message translates to:
  /// **'Tekrar dene'**
  String get actionRetry;

  /// No description provided for @actionCancel.
  ///
  /// In tr, this message translates to:
  /// **'Vazgeç'**
  String get actionCancel;

  /// No description provided for @actionGoHome.
  ///
  /// In tr, this message translates to:
  /// **'Ana ekrana dön'**
  String get actionGoHome;

  /// No description provided for @loadingLabel.
  ///
  /// In tr, this message translates to:
  /// **'{label}, yükleniyor'**
  String loadingLabel(String label);

  /// No description provided for @notFoundMessage.
  ///
  /// In tr, this message translates to:
  /// **'Aradığın sayfa bulunamadı.'**
  String get notFoundMessage;

  /// No description provided for @connectionReconnecting.
  ///
  /// In tr, this message translates to:
  /// **'Bağlantı koptu, yeniden bağlanıyoruz…'**
  String get connectionReconnecting;

  /// No description provided for @connectionRestored.
  ///
  /// In tr, this message translates to:
  /// **'Bağlandı'**
  String get connectionRestored;

  /// No description provided for @connectionOffline.
  ///
  /// In tr, this message translates to:
  /// **'Bağlantı yok'**
  String get connectionOffline;

  /// No description provided for @statusReady.
  ///
  /// In tr, this message translates to:
  /// **'Hazır'**
  String get statusReady;

  /// No description provided for @statusWaiting.
  ///
  /// In tr, this message translates to:
  /// **'Bekliyor'**
  String get statusWaiting;

  /// No description provided for @statusHost.
  ///
  /// In tr, this message translates to:
  /// **'Oda sahibi'**
  String get statusHost;

  /// No description provided for @statusSpectator.
  ///
  /// In tr, this message translates to:
  /// **'İzleyici'**
  String get statusSpectator;

  /// No description provided for @statusEliminated.
  ///
  /// In tr, this message translates to:
  /// **'Elendi'**
  String get statusEliminated;

  /// No description provided for @statusLeft.
  ///
  /// In tr, this message translates to:
  /// **'Ayrıldı'**
  String get statusLeft;

  /// No description provided for @statusTurn.
  ///
  /// In tr, this message translates to:
  /// **'Sırada'**
  String get statusTurn;

  /// No description provided for @avatarDisconnected.
  ///
  /// In tr, this message translates to:
  /// **'{name}, bağlantısı koptu'**
  String avatarDisconnected(String name);

  /// No description provided for @avatarEliminated.
  ///
  /// In tr, this message translates to:
  /// **'{name}, elendi'**
  String avatarEliminated(String name);

  /// No description provided for @inviteCodeLabel.
  ///
  /// In tr, this message translates to:
  /// **'Davet kodu, {length} karakter'**
  String inviteCodeLabel(int length);

  /// No description provided for @errorNetwork.
  ///
  /// In tr, this message translates to:
  /// **'İnternet bağlantını kontrol edip tekrar dene.'**
  String get errorNetwork;

  /// No description provided for @errorTimeout.
  ///
  /// In tr, this message translates to:
  /// **'Sunucu zamanında yanıt vermedi. Tekrar dene.'**
  String get errorTimeout;

  /// No description provided for @errorUnauthorized.
  ///
  /// In tr, this message translates to:
  /// **'Oturumun sona erdi. Yeniden başlaman gerekiyor.'**
  String get errorUnauthorized;

  /// No description provided for @errorUpgradeRequired.
  ///
  /// In tr, this message translates to:
  /// **'Uygulamanın yeni bir sürümü var. Devam etmek için güncelle.'**
  String get errorUpgradeRequired;

  /// No description provided for @errorRateLimited.
  ///
  /// In tr, this message translates to:
  /// **'Çok hızlı gidiyorsun. {seconds} saniye sonra tekrar dene.'**
  String errorRateLimited(int seconds);

  /// No description provided for @errorRateLimitedNoWait.
  ///
  /// In tr, this message translates to:
  /// **'Çok hızlı gidiyorsun. Biraz bekleyip tekrar dene.'**
  String get errorRateLimitedNoWait;

  /// No description provided for @errorUnknown.
  ///
  /// In tr, this message translates to:
  /// **'Bir şeyler ters gitti. Tekrar dene.'**
  String get errorUnknown;

  /// No description provided for @errorInternal.
  ///
  /// In tr, this message translates to:
  /// **'Sunucuda bir sorun oluştu. Biraz sonra tekrar dene.'**
  String get errorInternal;

  /// No description provided for @errorValidationFailed.
  ///
  /// In tr, this message translates to:
  /// **'Bilgileri kontrol edip tekrar dene.'**
  String get errorValidationFailed;

  /// No description provided for @errorNicknameInvalid.
  ///
  /// In tr, this message translates to:
  /// **'Takma ad 3–16 karakter olmalı; harf, rakam, boşluk ve _ kullanabilirsin.'**
  String get errorNicknameInvalid;

  /// No description provided for @errorMessageInvalid.
  ///
  /// In tr, this message translates to:
  /// **'Mesaj gönderilemedi. 1–200 karakter olmalı ve uygunsuz ifade içermemeli.'**
  String get errorMessageInvalid;

  /// No description provided for @errorForbidden.
  ///
  /// In tr, this message translates to:
  /// **'Bu işlem için yetkin yok.'**
  String get errorForbidden;

  /// No description provided for @errorNotAMember.
  ///
  /// In tr, this message translates to:
  /// **'Bu odanın üyesi değilsin.'**
  String get errorNotAMember;

  /// No description provided for @errorKicked.
  ///
  /// In tr, this message translates to:
  /// **'Bu odadan çıkarıldın, tekrar katılamazsın.'**
  String get errorKicked;

  /// No description provided for @errorNotFound.
  ///
  /// In tr, this message translates to:
  /// **'Aradığın şey bulunamadı.'**
  String get errorNotFound;

  /// No description provided for @errorInviteCodeNotFound.
  ///
  /// In tr, this message translates to:
  /// **'Bu koda ait bir oda yok ya da oda kapanmış.'**
  String get errorInviteCodeNotFound;

  /// No description provided for @errorRoomFull.
  ///
  /// In tr, this message translates to:
  /// **'Bu oda dolu. Başka bir kod dene.'**
  String get errorRoomFull;

  /// No description provided for @errorGameInProgress.
  ///
  /// In tr, this message translates to:
  /// **'Bu odada oyun başlamış. Bitince tekrar dene.'**
  String get errorGameInProgress;

  /// No description provided for @errorNicknameTakenInRoom.
  ///
  /// In tr, this message translates to:
  /// **'Bu odada aynı takma adla biri var. Takma adını değiştir.'**
  String get errorNicknameTakenInRoom;

  /// No description provided for @errorAlreadyInRoom.
  ///
  /// In tr, this message translates to:
  /// **'Zaten başka bir odadasın. Önce oradan ayrıl.'**
  String get errorAlreadyInRoom;

  /// No description provided for @errorInvalidPhase.
  ///
  /// In tr, this message translates to:
  /// **'Bu işlem şu an yapılamıyor.'**
  String get errorInvalidPhase;

  /// No description provided for @errorNotYourTurn.
  ///
  /// In tr, this message translates to:
  /// **'Sıra sende değil.'**
  String get errorNotYourTurn;

  /// No description provided for @errorCardAlreadyRevealed.
  ///
  /// In tr, this message translates to:
  /// **'Bu kart zaten açık.'**
  String get errorCardAlreadyRevealed;

  /// No description provided for @errorCategoryNotAllowed.
  ///
  /// In tr, this message translates to:
  /// **'İlk turda yalnızca meslek kartını açabilirsin.'**
  String get errorCategoryNotAllowed;

  /// No description provided for @errorAlreadyVoted.
  ///
  /// In tr, this message translates to:
  /// **'Bu turda oyunu zaten verdin.'**
  String get errorAlreadyVoted;

  /// No description provided for @errorSelfVote.
  ///
  /// In tr, this message translates to:
  /// **'Kendine oy veremezsin.'**
  String get errorSelfVote;

  /// No description provided for @errorInvalidTarget.
  ///
  /// In tr, this message translates to:
  /// **'Bu oyuncu seçilemez.'**
  String get errorInvalidTarget;

  /// No description provided for @errorNotAlive.
  ///
  /// In tr, this message translates to:
  /// **'Elendiğin için bu işlemi yapamazsın.'**
  String get errorNotAlive;

  /// No description provided for @errorNotSpectator.
  ///
  /// In tr, this message translates to:
  /// **'Eşitlik oylamasını yalnızca izleyiciler yapar.'**
  String get errorNotSpectator;

  /// No description provided for @errorStartConditionsNotMet.
  ///
  /// In tr, this message translates to:
  /// **'Başlatmak için 4–10 oyuncu ve herkesin hazır olması gerekir.'**
  String get errorStartConditionsNotMet;

  /// No description provided for @errorStaleState.
  ///
  /// In tr, this message translates to:
  /// **'Oyun durumu değişti, ekran güncelleniyor.'**
  String get errorStaleState;

  /// No description provided for @realtimeForbidden.
  ///
  /// In tr, this message translates to:
  /// **'Bu odaya artık erişimin yok.'**
  String get realtimeForbidden;

  /// No description provided for @realtimeReplaced.
  ///
  /// In tr, this message translates to:
  /// **'Bu oda başka bir cihazda ya da pencerede açıldı.'**
  String get realtimeReplaced;

  /// No description provided for @errorWidgetMessage.
  ///
  /// In tr, this message translates to:
  /// **'Bu bölüm gösterilemedi.'**
  String get errorWidgetMessage;

  /// No description provided for @welcomeTitle.
  ///
  /// In tr, this message translates to:
  /// **'Sığınağa kim girecek?'**
  String get welcomeTitle;

  /// No description provided for @welcomeRule1.
  ///
  /// In tr, this message translates to:
  /// **'Bir felaket oldu; sığınakta herkese yer yok.'**
  String get welcomeRule1;

  /// No description provided for @welcomeRule2.
  ///
  /// In tr, this message translates to:
  /// **'Kartlarını açarak kendini savun, diğerlerini sorgula.'**
  String get welcomeRule2;

  /// No description provided for @welcomeRule3.
  ///
  /// In tr, this message translates to:
  /// **'Her tur oylama var. Son kalanlar sığınağa girer.'**
  String get welcomeRule3;

  /// No description provided for @welcomeNicknameLabel.
  ///
  /// In tr, this message translates to:
  /// **'Takma adın'**
  String get welcomeNicknameLabel;

  /// No description provided for @welcomeNicknameHelper.
  ///
  /// In tr, this message translates to:
  /// **'3–16 karakter: harf, rakam, boşluk ve _'**
  String get welcomeNicknameHelper;

  /// No description provided for @welcomeDeviceNote.
  ///
  /// In tr, this message translates to:
  /// **'Hesabın bu cihaza bağlı. Uygulamayı silersen geri getirilemez.'**
  String get welcomeDeviceNote;

  /// No description provided for @welcomeStart.
  ///
  /// In tr, this message translates to:
  /// **'Başla'**
  String get welcomeStart;

  /// No description provided for @nicknameErrorRequired.
  ///
  /// In tr, this message translates to:
  /// **'Bir takma ad yaz.'**
  String get nicknameErrorRequired;

  /// No description provided for @nicknameErrorTooShort.
  ///
  /// In tr, this message translates to:
  /// **'En az 3 karakter olmalı.'**
  String get nicknameErrorTooShort;

  /// No description provided for @nicknameErrorTooLong.
  ///
  /// In tr, this message translates to:
  /// **'En fazla 16 karakter olabilir.'**
  String get nicknameErrorTooLong;

  /// No description provided for @nicknameErrorInvalidChars.
  ///
  /// In tr, this message translates to:
  /// **'Yalnızca harf, rakam, boşluk ve _ kullanabilirsin.'**
  String get nicknameErrorInvalidChars;

  /// No description provided for @nicknameErrorInvalid.
  ///
  /// In tr, this message translates to:
  /// **'En az bir harf olmalı; art arda boşluk olmamalı.'**
  String get nicknameErrorInvalid;

  /// No description provided for @nicknameErrorNotAllowed.
  ///
  /// In tr, this message translates to:
  /// **'Bu takma ad kullanılamaz, başka bir tane seç.'**
  String get nicknameErrorNotAllowed;

  /// No description provided for @nicknameErrorProfanity.
  ///
  /// In tr, this message translates to:
  /// **'Bu takma ad uygun değil, başka bir tane seç.'**
  String get nicknameErrorProfanity;

  /// No description provided for @splashChecking.
  ///
  /// In tr, this message translates to:
  /// **'Oturumun kontrol ediliyor…'**
  String get splashChecking;

  /// No description provided for @actionSave.
  ///
  /// In tr, this message translates to:
  /// **'Kaydet'**
  String get actionSave;

  /// No description provided for @actionSettings.
  ///
  /// In tr, this message translates to:
  /// **'Ayarlar'**
  String get actionSettings;

  /// No description provided for @nicknameDialogTitle.
  ///
  /// In tr, this message translates to:
  /// **'Takma adını değiştir'**
  String get nicknameDialogTitle;

  /// No description provided for @homeGreeting.
  ///
  /// In tr, this message translates to:
  /// **'Merhaba, {name}'**
  String homeGreeting(String name);

  /// No description provided for @homeCreateRoom.
  ///
  /// In tr, this message translates to:
  /// **'Oda oluştur'**
  String get homeCreateRoom;

  /// No description provided for @homeJoinRoom.
  ///
  /// In tr, this message translates to:
  /// **'Koda katıl'**
  String get homeJoinRoom;

  /// No description provided for @homeBackToRoom.
  ///
  /// In tr, this message translates to:
  /// **'Odana dön'**
  String get homeBackToRoom;

  /// No description provided for @homeHowToTitle.
  ///
  /// In tr, this message translates to:
  /// **'Nasıl oynanır?'**
  String get homeHowToTitle;

  /// No description provided for @homeHowTo1.
  ///
  /// In tr, this message translates to:
  /// **'4–10 kişi aynı odada toplanır; oda sahibi davet kodunu paylaşır.'**
  String get homeHowTo1;

  /// No description provided for @homeHowTo2.
  ///
  /// In tr, this message translates to:
  /// **'Herkese gizli 5 kart dağıtılır: meslek, sağlık, hobi, eşya ve bir sır.'**
  String get homeHowTo2;

  /// No description provided for @homeHowTo3.
  ///
  /// In tr, this message translates to:
  /// **'Her turda sırayla birer kart açılır, ardından tartışma başlar.'**
  String get homeHowTo3;

  /// No description provided for @homeHowTo4.
  ///
  /// In tr, this message translates to:
  /// **'Oylamada en çok oyu alan dışarıda kalır ve izleyici olur.'**
  String get homeHowTo4;

  /// No description provided for @homeHowTo5.
  ///
  /// In tr, this message translates to:
  /// **'Sığınağın kapasitesi kadar kişi kalınca oyun biter; kalanlar kazanır.'**
  String get homeHowTo5;

  /// No description provided for @joinHint.
  ///
  /// In tr, this message translates to:
  /// **'Oda sahibinden aldığın 6 karakterlik kodu gir.'**
  String get joinHint;

  /// No description provided for @joinSubmit.
  ///
  /// In tr, this message translates to:
  /// **'Katıl'**
  String get joinSubmit;

  /// No description provided for @joinInvalidCode.
  ///
  /// In tr, this message translates to:
  /// **'Bu kod geçerli değil. Kontrol edip tekrar dene.'**
  String get joinInvalidCode;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['tr'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'tr':
      return AppLocalizationsTr();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
