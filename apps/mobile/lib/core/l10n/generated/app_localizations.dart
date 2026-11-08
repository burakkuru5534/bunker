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
  /// **'BUNKER — Son Kalanlar'**
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

  /// No description provided for @connectionOfflineInGame.
  ///
  /// In tr, this message translates to:
  /// **'Bağlantı yok. Kısa sürede dönmezsen oyundan çıkarılırsın.'**
  String get connectionOfflineInGame;

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

  /// No description provided for @roomLobbyTitle.
  ///
  /// In tr, this message translates to:
  /// **'Lobi'**
  String get roomLobbyTitle;

  /// No description provided for @roomLeave.
  ///
  /// In tr, this message translates to:
  /// **'Odadan ayrıl'**
  String get roomLeave;

  /// No description provided for @roomLeaveTitle.
  ///
  /// In tr, this message translates to:
  /// **'Odadan ayrılmak istiyor musun?'**
  String get roomLeaveTitle;

  /// No description provided for @roomLeaveMessage.
  ///
  /// In tr, this message translates to:
  /// **'Lobiden ayrılırsan davet koduyla tekrar katılabilirsin.'**
  String get roomLeaveMessage;

  /// No description provided for @roomLeaveConfirm.
  ///
  /// In tr, this message translates to:
  /// **'Ayrıl'**
  String get roomLeaveConfirm;

  /// No description provided for @roomInviteCode.
  ///
  /// In tr, this message translates to:
  /// **'Davet kodu'**
  String get roomInviteCode;

  /// No description provided for @roomCopy.
  ///
  /// In tr, this message translates to:
  /// **'Kopyala'**
  String get roomCopy;

  /// No description provided for @roomCopied.
  ///
  /// In tr, this message translates to:
  /// **'Davet kodu kopyalandı'**
  String get roomCopied;

  /// No description provided for @roomCapacity.
  ///
  /// In tr, this message translates to:
  /// **'Sığınak kapasitesi'**
  String get roomCapacity;

  /// No description provided for @roomCapacityHelp.
  ///
  /// In tr, this message translates to:
  /// **'Oyun sonunda sığınağa girecek kişi sayısı'**
  String get roomCapacityHelp;

  /// No description provided for @roomScenario.
  ///
  /// In tr, this message translates to:
  /// **'Felaket senaryosu'**
  String get roomScenario;

  /// No description provided for @roomScenarioHelp.
  ///
  /// In tr, this message translates to:
  /// **'Oda sahibi seçer. Seçilmezse her oyunda rastgele gelir.'**
  String get roomScenarioHelp;

  /// No description provided for @roomScenarioRandom.
  ///
  /// In tr, this message translates to:
  /// **'Rastgele'**
  String get roomScenarioRandom;

  /// No description provided for @roomScenarioChoose.
  ///
  /// In tr, this message translates to:
  /// **'Senaryo seç'**
  String get roomScenarioChoose;

  /// No description provided for @roomScenarioLoading.
  ///
  /// In tr, this message translates to:
  /// **'Senaryolar yükleniyor…'**
  String get roomScenarioLoading;

  /// No description provided for @roomScenarioLoadFailed.
  ///
  /// In tr, this message translates to:
  /// **'Senaryolar yüklenemedi.'**
  String get roomScenarioLoadFailed;

  /// No description provided for @roomCapacityValue.
  ///
  /// In tr, this message translates to:
  /// **'{count} kişi'**
  String roomCapacityValue(int count);

  /// No description provided for @roomPlayers.
  ///
  /// In tr, this message translates to:
  /// **'Oyuncular ({count}/{max})'**
  String roomPlayers(int count, int max);

  /// No description provided for @roomYou.
  ///
  /// In tr, this message translates to:
  /// **'sen'**
  String get roomYou;

  /// No description provided for @roomReady.
  ///
  /// In tr, this message translates to:
  /// **'Hazırım'**
  String get roomReady;

  /// No description provided for @roomStart.
  ///
  /// In tr, this message translates to:
  /// **'Başlat'**
  String get roomStart;

  /// No description provided for @roomStartTooFew.
  ///
  /// In tr, this message translates to:
  /// **'Başlatmak için en az {min} oyuncu gerekli ({count}/{min})'**
  String roomStartTooFew(int count, int min);

  /// No description provided for @roomStartTooMany.
  ///
  /// In tr, this message translates to:
  /// **'En fazla {max} oyuncuyla başlatılabilir'**
  String roomStartTooMany(int max);

  /// No description provided for @roomStartNotReady.
  ///
  /// In tr, this message translates to:
  /// **'Herkesin hazır olması bekleniyor ({ready}/{total})'**
  String roomStartNotReady(int ready, int total);

  /// No description provided for @roomWaitingHost.
  ///
  /// In tr, this message translates to:
  /// **'Oda sahibinin oyunu başlatması bekleniyor.'**
  String get roomWaitingHost;

  /// No description provided for @roomKick.
  ///
  /// In tr, this message translates to:
  /// **'Odadan at'**
  String get roomKick;

  /// No description provided for @roomKickTitle.
  ///
  /// In tr, this message translates to:
  /// **'{name} odadan atılsın mı?'**
  String roomKickTitle(String name);

  /// No description provided for @roomKickMessage.
  ///
  /// In tr, this message translates to:
  /// **'Atılan oyuncu bu odaya tekrar katılamaz.'**
  String get roomKickMessage;

  /// No description provided for @roomKickConfirm.
  ///
  /// In tr, this message translates to:
  /// **'At'**
  String get roomKickConfirm;

  /// No description provided for @roomConnecting.
  ///
  /// In tr, this message translates to:
  /// **'Odaya bağlanılıyor…'**
  String get roomConnecting;

  /// No description provided for @roomReconnect.
  ///
  /// In tr, this message translates to:
  /// **'Yeniden bağlan'**
  String get roomReconnect;

  /// No description provided for @roomLoadFailed.
  ///
  /// In tr, this message translates to:
  /// **'Oda yüklenemedi. Tekrar dene; sorun sürerse odadan çıkıp yeniden katıl.'**
  String get roomLoadFailed;

  /// No description provided for @roomGameFinished.
  ///
  /// In tr, this message translates to:
  /// **'Oyun bitti.'**
  String get roomGameFinished;

  /// No description provided for @gameTitle.
  ///
  /// In tr, this message translates to:
  /// **'Oyun'**
  String get gameTitle;

  /// No description provided for @gameLeaveMessage.
  ///
  /// In tr, this message translates to:
  /// **'Oyun sürerken ayrılırsan elenmiş sayılırsın ve tüm kartların açılır.'**
  String get gameLeaveMessage;

  /// No description provided for @gamePhaseDealing.
  ///
  /// In tr, this message translates to:
  /// **'Kartlar dağıtılıyor'**
  String get gamePhaseDealing;

  /// No description provided for @gamePhaseScenario.
  ///
  /// In tr, this message translates to:
  /// **'Senaryo'**
  String get gamePhaseScenario;

  /// No description provided for @gamePhaseReveal.
  ///
  /// In tr, this message translates to:
  /// **'Kart açma'**
  String get gamePhaseReveal;

  /// No description provided for @gamePhaseDiscussion.
  ///
  /// In tr, this message translates to:
  /// **'Tartışma'**
  String get gamePhaseDiscussion;

  /// No description provided for @gamePhaseVoting.
  ///
  /// In tr, this message translates to:
  /// **'Oylama'**
  String get gamePhaseVoting;

  /// No description provided for @gamePhaseTieBreak.
  ///
  /// In tr, this message translates to:
  /// **'Eşitlik'**
  String get gamePhaseTieBreak;

  /// No description provided for @gamePhaseElimination.
  ///
  /// In tr, this message translates to:
  /// **'Eleme'**
  String get gamePhaseElimination;

  /// No description provided for @gamePhaseFinished.
  ///
  /// In tr, this message translates to:
  /// **'Oyun bitti'**
  String get gamePhaseFinished;

  /// No description provided for @gameRound.
  ///
  /// In tr, this message translates to:
  /// **'Tur {round}/{total}'**
  String gameRound(int round, int total);

  /// No description provided for @gameCountdownLabel.
  ///
  /// In tr, this message translates to:
  /// **'Kalan süre {time}'**
  String gameCountdownLabel(String time);

  /// No description provided for @gameShowScenario.
  ///
  /// In tr, this message translates to:
  /// **'Senaryoyu göster'**
  String get gameShowScenario;

  /// No description provided for @gameScenarioDisaster.
  ///
  /// In tr, this message translates to:
  /// **'Ne oldu?'**
  String get gameScenarioDisaster;

  /// No description provided for @gameScenarioDuration.
  ///
  /// In tr, this message translates to:
  /// **'Sığınakta kalış süresi: {duration}'**
  String gameScenarioDuration(String duration);

  /// No description provided for @gameScenarioBunker.
  ///
  /// In tr, this message translates to:
  /// **'Sığınakta neler var?'**
  String get gameScenarioBunker;

  /// No description provided for @gameScenarioOutside.
  ///
  /// In tr, this message translates to:
  /// **'Dışarıda'**
  String get gameScenarioOutside;

  /// No description provided for @gameIntroSkip.
  ///
  /// In tr, this message translates to:
  /// **'Geç'**
  String get gameIntroSkip;

  /// No description provided for @gameIntroMute.
  ///
  /// In tr, this message translates to:
  /// **'Sesi kapat'**
  String get gameIntroMute;

  /// No description provided for @gameIntroUnmute.
  ///
  /// In tr, this message translates to:
  /// **'Sesi aç'**
  String get gameIntroUnmute;

  /// Senaryo tanıtımı altyazısı; satırlar (\n) anlatımın dört bölümüdür ve narration.ogg ile aynı olmalıdır.
  ///
  /// In tr, this message translates to:
  /// **'Son füze düştüğünde, gökyüzü karardı.\nKül bulutları güneşi yuttu… Dünya buz kesti; rüzgâr artık radyasyon taşıyor.\nÖnünüzde tek bir kapı var. İçeride üç yıl yetecek erzak… ama herkese yetecek yer yok.\nKapı kapanmadan karar verin… Kim içeri girecek?'**
  String get introNarrationNuclearWinter;

  /// No description provided for @introNarrationRedCough.
  ///
  /// In tr, this message translates to:
  /// **'Önce bir öksürük duyuldu. Sonra bir tane daha…\nAteş, kızıl bir sis gibi şehirden şehre yayıldı. Bilinen bir tedavisi yok. Sokaklar boşaldı, kapılar kilitlendi.\nSığınağın filtreleri temiz hava veriyor; on sekiz ay dayanmanız gerek. Ama içeride hastalığa da, şüpheye de yer yok.\nBirbirinizin gözlerine bakın… Kim içeri girecek?'**
  String get introNarrationRedCough;

  /// No description provided for @introNarrationMeteor.
  ///
  /// In tr, this message translates to:
  /// **'O gece gökyüzü bir anda aydınlandı.\nDev bir göktaşı okyanusa çarptı. Yükselen toz güneşi kapattı, asit yağmurları başladı.\nSığınakta temiz su ve koca bir kütüphane var. Ama yakıt sınırlı… ve iki yıl, uzun bir süre.\nHerkese yer yok. Karar verin… Kim içeri girecek?'**
  String get introNarrationMeteor;

  /// No description provided for @introNarrationSilentMachines.
  ///
  /// In tr, this message translates to:
  /// **'Önce telefonlar sustu. Sonra ekranlar kendi kendine açıldı.\nKontrolden çıkan ağ, sinyal yayan her şeyi avlıyor. Gökyüzünde dronlar dolaşıyor.\nSığınakta tek bir kural var: Elektronik yok. Bir yıl boyunca mum ışığında, sessizce bekleyeceksiniz.\nAma herkes için yer yok… Fısıltıyla sorun: Kim içeri girecek?'**
  String get introNarrationSilentMachines;

  /// No description provided for @introNarrationSolarStorm.
  ///
  /// In tr, this message translates to:
  /// **'Gökyüzü yeşil ve mor ışıklarla yandı.\nArdından şebekeler çöktü, uydular sustu, şehirler karanlığa gömüldü. Yüzeydeki radyasyon aylarca dinmeyecek.\nSığınağın kalkanı sağlam. Ama erzak dar, güneş panelleri hasarlı… Sekiz ay boyunca birbirinize güvenmek zorundasınız.\nPeki… Kim içeri girecek?'**
  String get introNarrationSolarStorm;

  /// No description provided for @gameScenarioCapacity.
  ///
  /// In tr, this message translates to:
  /// **'Sığınağa yalnızca {count} kişi girebilir.'**
  String gameScenarioCapacity(int count);

  /// No description provided for @gameMyCards.
  ///
  /// In tr, this message translates to:
  /// **'Kartlarım'**
  String get gameMyCards;

  /// No description provided for @gameCardOpen.
  ///
  /// In tr, this message translates to:
  /// **'Açık'**
  String get gameCardOpen;

  /// No description provided for @gameCardClosed.
  ///
  /// In tr, this message translates to:
  /// **'Kapalı'**
  String get gameCardClosed;

  /// No description provided for @gameNoCards.
  ///
  /// In tr, this message translates to:
  /// **'Bu oyunda kartın yok.'**
  String get gameNoCards;

  /// No description provided for @gameSpectatorBanner.
  ///
  /// In tr, this message translates to:
  /// **'İzleyicisin. Eşitlik olursa kararı izleyiciler verir.'**
  String get gameSpectatorBanner;

  /// No description provided for @gameRevealMine.
  ///
  /// In tr, this message translates to:
  /// **'Sıra sende. Bir kartını aç.'**
  String get gameRevealMine;

  /// No description provided for @gameRevealOther.
  ///
  /// In tr, this message translates to:
  /// **'{name} kartını açıyor…'**
  String gameRevealOther(String name);

  /// No description provided for @gameDiscussionInfo.
  ///
  /// In tr, this message translates to:
  /// **'İkna etme zamanı.'**
  String get gameDiscussionInfo;

  /// No description provided for @gameDiscussionChatHint.
  ///
  /// In tr, this message translates to:
  /// **'Neden sığınağa seni almalılar? Sohbette anlat, kartlarını savun, diğerlerini ikna et.'**
  String get gameDiscussionChatHint;

  /// No description provided for @gameDiscussionOpenChat.
  ///
  /// In tr, this message translates to:
  /// **'Sohbete yaz'**
  String get gameDiscussionOpenChat;

  /// No description provided for @gameReadyLocked.
  ///
  /// In tr, this message translates to:
  /// **'Hazırım ({seconds})'**
  String gameReadyLocked(int seconds);

  /// No description provided for @gameReadyProgress.
  ///
  /// In tr, this message translates to:
  /// **'{done}/{total} hazır'**
  String gameReadyProgress(int done, int total);

  /// No description provided for @gameVotingInfo.
  ///
  /// In tr, this message translates to:
  /// **'Oylama sürüyor. Oylar gizli.'**
  String get gameVotingInfo;

  /// No description provided for @gameVoteProgress.
  ///
  /// In tr, this message translates to:
  /// **'{done}/{total} oy verildi'**
  String gameVoteProgress(int done, int total);

  /// No description provided for @gameTieBreakInfo.
  ///
  /// In tr, this message translates to:
  /// **'Eşitlik! İzleyiciler karar veriyor…'**
  String get gameTieBreakInfo;

  /// No description provided for @gameEliminated.
  ///
  /// In tr, this message translates to:
  /// **'{name} sığınağın dışında kaldı.'**
  String gameEliminated(String name);

  /// No description provided for @gameDealingInfo.
  ///
  /// In tr, this message translates to:
  /// **'Kartlar dağıtılıyor…'**
  String get gameDealingInfo;

  /// No description provided for @gameRevealRound1Hint.
  ///
  /// In tr, this message translates to:
  /// **'İlk turda yalnızca meslek kartını açabilirsin.'**
  String get gameRevealRound1Hint;

  /// No description provided for @gameRevealConfirmTitle.
  ///
  /// In tr, this message translates to:
  /// **'{category} kartını aç?'**
  String gameRevealConfirmTitle(String category);

  /// No description provided for @gameRevealConfirmMessage.
  ///
  /// In tr, this message translates to:
  /// **'“{text}” herkese açılacak. Açılan kart geri kapatılamaz.'**
  String gameRevealConfirmMessage(String text);

  /// No description provided for @gameRevealConfirm.
  ///
  /// In tr, this message translates to:
  /// **'Aç'**
  String get gameRevealConfirm;

  /// No description provided for @gameReadyButton.
  ///
  /// In tr, this message translates to:
  /// **'Hazırım'**
  String get gameReadyButton;

  /// No description provided for @gameReadyDone.
  ///
  /// In tr, this message translates to:
  /// **'Hazırsın. Diğerleri bekleniyor.'**
  String get gameReadyDone;

  /// No description provided for @gameVoteHint.
  ///
  /// In tr, this message translates to:
  /// **'Elemek istediğin oyuncuya dokun.'**
  String get gameVoteHint;

  /// No description provided for @gameVoteConfirmTitle.
  ///
  /// In tr, this message translates to:
  /// **'{name} için oy ver?'**
  String gameVoteConfirmTitle(String name);

  /// No description provided for @gameVoteConfirmMessage.
  ///
  /// In tr, this message translates to:
  /// **'Oyunu verdikten sonra değiştiremezsin.'**
  String get gameVoteConfirmMessage;

  /// No description provided for @gameVoteConfirm.
  ///
  /// In tr, this message translates to:
  /// **'Oy ver'**
  String get gameVoteConfirm;

  /// No description provided for @gameVoteDone.
  ///
  /// In tr, this message translates to:
  /// **'Oyunu verdin.'**
  String get gameVoteDone;

  /// No description provided for @gameTieBreakMine.
  ///
  /// In tr, this message translates to:
  /// **'Eşitlik! Kararı izleyiciler veriyor; sen de oy ver.'**
  String get gameTieBreakMine;

  /// No description provided for @gameVoteFor.
  ///
  /// In tr, this message translates to:
  /// **'{name} için oy ver'**
  String gameVoteFor(String name);

  /// No description provided for @gameMethodVote.
  ///
  /// In tr, this message translates to:
  /// **'Oylamayla elendi'**
  String get gameMethodVote;

  /// No description provided for @gameMethodSpectators.
  ///
  /// In tr, this message translates to:
  /// **'İzleyicilerin kararıyla elendi'**
  String get gameMethodSpectators;

  /// No description provided for @gameMethodLottery.
  ///
  /// In tr, this message translates to:
  /// **'Eşitlik kurayla bozuldu'**
  String get gameMethodLottery;

  /// No description provided for @gameMethodLeft.
  ///
  /// In tr, this message translates to:
  /// **'Oyundan ayrıldı'**
  String get gameMethodLeft;

  /// No description provided for @gameMethodShortVote.
  ///
  /// In tr, this message translates to:
  /// **'oylama'**
  String get gameMethodShortVote;

  /// No description provided for @gameMethodShortSpectators.
  ///
  /// In tr, this message translates to:
  /// **'izleyiciler'**
  String get gameMethodShortSpectators;

  /// No description provided for @gameMethodShortLottery.
  ///
  /// In tr, this message translates to:
  /// **'kura'**
  String get gameMethodShortLottery;

  /// No description provided for @gameMethodShortLeft.
  ///
  /// In tr, this message translates to:
  /// **'ayrıldı'**
  String get gameMethodShortLeft;

  /// No description provided for @gameVotesCount.
  ///
  /// In tr, this message translates to:
  /// **'{count} oy'**
  String gameVotesCount(int count);

  /// No description provided for @gameNoVotes.
  ///
  /// In tr, this message translates to:
  /// **'Kimse oy vermedi; herkes eşit sayıldı.'**
  String get gameNoVotes;

  /// No description provided for @gameLotteryAmong.
  ///
  /// In tr, this message translates to:
  /// **'Kura adayları: {names}'**
  String gameLotteryAmong(String names);

  /// No description provided for @gameEliminatedCards.
  ///
  /// In tr, this message translates to:
  /// **'Kartları'**
  String get gameEliminatedCards;

  /// No description provided for @gameDoorClosedTitle.
  ///
  /// In tr, this message translates to:
  /// **'Kapı kapandı.'**
  String get gameDoorClosedTitle;

  /// No description provided for @gameDoorClosedBody.
  ///
  /// In tr, this message translates to:
  /// **'Artık izleyicisin.'**
  String get gameDoorClosedBody;

  /// No description provided for @resultWinners.
  ///
  /// In tr, this message translates to:
  /// **'Sığınağa girenler'**
  String get resultWinners;

  /// No description provided for @resultLosers.
  ///
  /// In tr, this message translates to:
  /// **'Dışarıda kalanlar'**
  String get resultLosers;

  /// No description provided for @resultRounds.
  ///
  /// In tr, this message translates to:
  /// **'Tur tur'**
  String get resultRounds;

  /// No description provided for @resultRoundLine.
  ///
  /// In tr, this message translates to:
  /// **'Tur {round}: {name} ({method})'**
  String resultRoundLine(int round, String name, String method);

  /// No description provided for @resultRematch.
  ///
  /// In tr, this message translates to:
  /// **'Tekrar oyna'**
  String get resultRematch;

  /// No description provided for @resultAbandoned.
  ///
  /// In tr, this message translates to:
  /// **'Oyun erken bitti: oyunda yeterli oyuncu kalmadı.'**
  String get resultAbandoned;

  /// No description provided for @resultWaitingHost.
  ///
  /// In tr, this message translates to:
  /// **'Oda sahibi yeni bir oyun başlatabilir.'**
  String get resultWaitingHost;

  /// No description provided for @resultLoading.
  ///
  /// In tr, this message translates to:
  /// **'Sonuçlar yükleniyor…'**
  String get resultLoading;

  /// No description provided for @chatTitle.
  ///
  /// In tr, this message translates to:
  /// **'Sohbet'**
  String get chatTitle;

  /// No description provided for @chatOpen.
  ///
  /// In tr, this message translates to:
  /// **'Sohbet'**
  String get chatOpen;

  /// No description provided for @chatUnread.
  ///
  /// In tr, this message translates to:
  /// **'Sohbet, {count} okunmamış mesaj'**
  String chatUnread(int count);

  /// No description provided for @chatEmpty.
  ///
  /// In tr, this message translates to:
  /// **'Henüz mesaj yok. İlk sen yaz!'**
  String get chatEmpty;

  /// No description provided for @chatHint.
  ///
  /// In tr, this message translates to:
  /// **'Mesaj yaz…'**
  String get chatHint;

  /// No description provided for @chatSend.
  ///
  /// In tr, this message translates to:
  /// **'Gönder'**
  String get chatSend;

  /// No description provided for @chatSending.
  ///
  /// In tr, this message translates to:
  /// **'Gönderiliyor…'**
  String get chatSending;

  /// No description provided for @chatRetryHint.
  ///
  /// In tr, this message translates to:
  /// **'Tekrar denemek için dokun.'**
  String get chatRetryHint;

  /// No description provided for @chatSpectatorNote.
  ///
  /// In tr, this message translates to:
  /// **'İzleyiciler yazamaz.'**
  String get chatSpectatorNote;

  /// No description provided for @modReport.
  ///
  /// In tr, this message translates to:
  /// **'Raporla'**
  String get modReport;

  /// No description provided for @modBlock.
  ///
  /// In tr, this message translates to:
  /// **'Engelle'**
  String get modBlock;

  /// No description provided for @modReportTitle.
  ///
  /// In tr, this message translates to:
  /// **'{name} adlı oyuncuyu raporla'**
  String modReportTitle(String name);

  /// No description provided for @modReasonHarassment.
  ///
  /// In tr, this message translates to:
  /// **'Taciz veya zorbalık'**
  String get modReasonHarassment;

  /// No description provided for @modReasonHate.
  ///
  /// In tr, this message translates to:
  /// **'Nefret söylemi'**
  String get modReasonHate;

  /// No description provided for @modReasonSpam.
  ///
  /// In tr, this message translates to:
  /// **'Spam veya reklam'**
  String get modReasonSpam;

  /// No description provided for @modReasonCheating.
  ///
  /// In tr, this message translates to:
  /// **'Hile'**
  String get modReasonCheating;

  /// No description provided for @modReasonOther.
  ///
  /// In tr, this message translates to:
  /// **'Diğer'**
  String get modReasonOther;

  /// No description provided for @modReportNote.
  ///
  /// In tr, this message translates to:
  /// **'Not (isteğe bağlı)'**
  String get modReportNote;

  /// No description provided for @modReportSend.
  ///
  /// In tr, this message translates to:
  /// **'Gönder'**
  String get modReportSend;

  /// No description provided for @modReportSent.
  ///
  /// In tr, this message translates to:
  /// **'Raporun alındı. Teşekkürler.'**
  String get modReportSent;

  /// No description provided for @modBlockTitle.
  ///
  /// In tr, this message translates to:
  /// **'{name} engellensin mi?'**
  String modBlockTitle(String name);

  /// No description provided for @modBlockMessage.
  ///
  /// In tr, this message translates to:
  /// **'Mesajlarını artık görmezsin. Oyunu etkilemez; engeli Ayarlar\'dan kaldırabilirsin.'**
  String get modBlockMessage;

  /// No description provided for @modBlocked.
  ///
  /// In tr, this message translates to:
  /// **'{name} engellendi.'**
  String modBlocked(String name);

  /// No description provided for @blocksTitle.
  ///
  /// In tr, this message translates to:
  /// **'Engellenen oyuncular'**
  String get blocksTitle;

  /// No description provided for @blocksEmpty.
  ///
  /// In tr, this message translates to:
  /// **'Engellediğin kimse yok.'**
  String get blocksEmpty;

  /// No description provided for @blocksUnblock.
  ///
  /// In tr, this message translates to:
  /// **'Engeli kaldır'**
  String get blocksUnblock;

  /// No description provided for @blocksUnblocked.
  ///
  /// In tr, this message translates to:
  /// **'{name} artık engelli değil.'**
  String blocksUnblocked(String name);

  /// No description provided for @settingsTitle.
  ///
  /// In tr, this message translates to:
  /// **'Ayarlar'**
  String get settingsTitle;

  /// No description provided for @settingsNickname.
  ///
  /// In tr, this message translates to:
  /// **'Takma ad'**
  String get settingsNickname;

  /// No description provided for @settingsAccount.
  ///
  /// In tr, this message translates to:
  /// **'Hesap'**
  String get settingsAccount;

  /// No description provided for @settingsDeleteAccount.
  ///
  /// In tr, this message translates to:
  /// **'Hesabımı sil'**
  String get settingsDeleteAccount;

  /// No description provided for @settingsDeleteTitle.
  ///
  /// In tr, this message translates to:
  /// **'Hesabın silinsin mi?'**
  String get settingsDeleteTitle;

  /// No description provided for @settingsDeleteMessage.
  ///
  /// In tr, this message translates to:
  /// **'Takma adın ve oturumların silinir, mesajların kaldırılır. Bulunduğun odadan çıkarsın; oyun sürüyorsa elenmiş sayılırsın. Bu işlem geri alınamaz.'**
  String get settingsDeleteMessage;

  /// No description provided for @settingsDeleteContinue.
  ///
  /// In tr, this message translates to:
  /// **'Devam'**
  String get settingsDeleteContinue;

  /// No description provided for @settingsDeleteFinalTitle.
  ///
  /// In tr, this message translates to:
  /// **'Emin misin?'**
  String get settingsDeleteFinalTitle;

  /// No description provided for @settingsDeleteFinalMessage.
  ///
  /// In tr, this message translates to:
  /// **'Hesabın kalıcı olarak silinecek ve geri getirilemeyecek.'**
  String get settingsDeleteFinalMessage;

  /// No description provided for @settingsVersion.
  ///
  /// In tr, this message translates to:
  /// **'Sürüm {version}'**
  String settingsVersion(String version);

  /// No description provided for @legalPrivacy.
  ///
  /// In tr, this message translates to:
  /// **'Gizlilik politikası'**
  String get legalPrivacy;

  /// No description provided for @legalTerms.
  ///
  /// In tr, this message translates to:
  /// **'Kullanım koşulları'**
  String get legalTerms;

  /// No description provided for @legalOpenFailed.
  ///
  /// In tr, this message translates to:
  /// **'Bağlantı açılamadı.'**
  String get legalOpenFailed;

  /// No description provided for @welcomeLegalNote.
  ///
  /// In tr, this message translates to:
  /// **'Başlayarak Kullanım Koşulları\'nı kabul etmiş olursun. Uygulama 13 yaş ve üstü içindir.'**
  String get welcomeLegalNote;

  /// No description provided for @settingsLegal.
  ///
  /// In tr, this message translates to:
  /// **'Yasal'**
  String get settingsLegal;
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
