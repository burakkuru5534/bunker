// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Turkish (`tr`).
class AppLocalizationsTr extends AppLocalizations {
  AppLocalizationsTr([String locale = 'tr']) : super(locale);

  @override
  String get appTitle => 'BUNKER — Son Kalanlar';

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
  String get connectionOfflineInGame =>
      'Bağlantı yok. Kısa sürede dönmezsen oyundan çıkarılırsın.';

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

  @override
  String get roomLobbyTitle => 'Lobi';

  @override
  String get roomLeave => 'Odadan ayrıl';

  @override
  String get roomLeaveTitle => 'Odadan ayrılmak istiyor musun?';

  @override
  String get roomLeaveMessage =>
      'Lobiden ayrılırsan davet koduyla tekrar katılabilirsin.';

  @override
  String get roomLeaveConfirm => 'Ayrıl';

  @override
  String get roomInviteCode => 'Davet kodu';

  @override
  String get roomCopy => 'Kopyala';

  @override
  String get roomCopied => 'Davet kodu kopyalandı';

  @override
  String get roomCapacity => 'Sığınak kapasitesi';

  @override
  String get roomCapacityHelp => 'Oyun sonunda sığınağa girecek kişi sayısı';

  @override
  String get roomScenario => 'Felaket senaryosu';

  @override
  String get roomScenarioHelp =>
      'Oda sahibi seçer. Seçilmezse her oyunda rastgele gelir.';

  @override
  String get roomScenarioRandom => 'Rastgele';

  @override
  String get roomScenarioChoose => 'Senaryo seç';

  @override
  String get roomScenarioLoading => 'Senaryolar yükleniyor…';

  @override
  String get roomScenarioLoadFailed => 'Senaryolar yüklenemedi.';

  @override
  String roomCapacityValue(int count) {
    return '$count kişi';
  }

  @override
  String roomPlayers(int count, int max) {
    return 'Oyuncular ($count/$max)';
  }

  @override
  String get roomYou => 'sen';

  @override
  String get roomReady => 'Hazırım';

  @override
  String get roomStart => 'Başlat';

  @override
  String roomStartTooFew(int count, int min) {
    return 'Başlatmak için en az $min oyuncu gerekli ($count/$min)';
  }

  @override
  String roomStartTooMany(int max) {
    return 'En fazla $max oyuncuyla başlatılabilir';
  }

  @override
  String roomStartNotReady(int ready, int total) {
    return 'Herkesin hazır olması bekleniyor ($ready/$total)';
  }

  @override
  String get roomWaitingHost => 'Oda sahibinin oyunu başlatması bekleniyor.';

  @override
  String get roomKick => 'Odadan at';

  @override
  String roomKickTitle(String name) {
    return '$name odadan atılsın mı?';
  }

  @override
  String get roomKickMessage => 'Atılan oyuncu bu odaya tekrar katılamaz.';

  @override
  String get roomKickConfirm => 'At';

  @override
  String get roomConnecting => 'Odaya bağlanılıyor…';

  @override
  String get roomReconnect => 'Yeniden bağlan';

  @override
  String get roomLoadFailed =>
      'Oda yüklenemedi. Tekrar dene; sorun sürerse odadan çıkıp yeniden katıl.';

  @override
  String get roomGameFinished => 'Oyun bitti.';

  @override
  String get gameTitle => 'Oyun';

  @override
  String get gameLeaveMessage =>
      'Oyun sürerken ayrılırsan elenmiş sayılırsın ve tüm kartların açılır.';

  @override
  String get gamePhaseDealing => 'Kartlar dağıtılıyor';

  @override
  String get gamePhaseScenario => 'Senaryo';

  @override
  String get gamePhaseReveal => 'Kart açma';

  @override
  String get gamePhaseDiscussion => 'Tartışma';

  @override
  String get gamePhaseVoting => 'Oylama';

  @override
  String get gamePhaseTieBreak => 'Eşitlik';

  @override
  String get gamePhaseElimination => 'Eleme';

  @override
  String get gamePhaseFinished => 'Oyun bitti';

  @override
  String gameRound(int round, int total) {
    return 'Tur $round/$total';
  }

  @override
  String gameCountdownLabel(String time) {
    return 'Kalan süre $time';
  }

  @override
  String get gameShowScenario => 'Senaryoyu göster';

  @override
  String get gameScenarioDisaster => 'Ne oldu?';

  @override
  String gameScenarioDuration(String duration) {
    return 'Sığınakta kalış süresi: $duration';
  }

  @override
  String get gameScenarioBunker => 'Sığınakta neler var?';

  @override
  String get gameScenarioOutside => 'Dışarıda';

  @override
  String get gameIntroSkip => 'Geç';

  @override
  String get gameIntroMute => 'Sesi kapat';

  @override
  String get gameIntroUnmute => 'Sesi aç';

  @override
  String get introNarrationNuclearWinter =>
      'Son füze düştüğünde, gökyüzü karardı.\nKül bulutları güneşi yuttu… Dünya buz kesti; rüzgâr artık radyasyon taşıyor.\nÖnünüzde tek bir kapı var. İçeride üç yıl yetecek erzak… ama herkese yetecek yer yok.\nKapı kapanmadan karar verin… Kim içeri girecek?';

  @override
  String get introNarrationRedCough =>
      'Önce bir öksürük duyuldu. Sonra bir tane daha…\nAteş, kızıl bir sis gibi şehirden şehre yayıldı. Bilinen bir tedavisi yok. Sokaklar boşaldı, kapılar kilitlendi.\nSığınağın filtreleri temiz hava veriyor; on sekiz ay dayanmanız gerek. Ama içeride hastalığa da, şüpheye de yer yok.\nBirbirinizin gözlerine bakın… Kim içeri girecek?';

  @override
  String get introNarrationMeteor =>
      'O gece gökyüzü bir anda aydınlandı.\nDev bir göktaşı okyanusa çarptı. Yükselen toz güneşi kapattı, asit yağmurları başladı.\nSığınakta temiz su ve koca bir kütüphane var. Ama yakıt sınırlı… ve iki yıl, uzun bir süre.\nHerkese yer yok. Karar verin… Kim içeri girecek?';

  @override
  String get introNarrationSilentMachines =>
      'Önce telefonlar sustu. Sonra ekranlar kendi kendine açıldı.\nKontrolden çıkan ağ, sinyal yayan her şeyi avlıyor. Gökyüzünde dronlar dolaşıyor.\nSığınakta tek bir kural var: Elektronik yok. Bir yıl boyunca mum ışığında, sessizce bekleyeceksiniz.\nAma herkes için yer yok… Fısıltıyla sorun: Kim içeri girecek?';

  @override
  String get introNarrationSolarStorm =>
      'Gökyüzü yeşil ve mor ışıklarla yandı.\nArdından şebekeler çöktü, uydular sustu, şehirler karanlığa gömüldü. Yüzeydeki radyasyon aylarca dinmeyecek.\nSığınağın kalkanı sağlam. Ama erzak dar, güneş panelleri hasarlı… Sekiz ay boyunca birbirinize güvenmek zorundasınız.\nPeki… Kim içeri girecek?';

  @override
  String gameScenarioCapacity(int count) {
    return 'Sığınağa yalnızca $count kişi girebilir.';
  }

  @override
  String get gameMyCards => 'Kartlarım';

  @override
  String get gameCardOpen => 'Açık';

  @override
  String get gameCardClosed => 'Kapalı';

  @override
  String get gameNoCards => 'Bu oyunda kartın yok.';

  @override
  String get gameSpectatorBanner =>
      'İzleyicisin. Eşitlik olursa kararı izleyiciler verir.';

  @override
  String get gameRevealMine => 'Sıra sende. Bir kartını aç.';

  @override
  String gameRevealOther(String name) {
    return '$name kartını açıyor…';
  }

  @override
  String get gameDiscussionInfo => 'İkna etme zamanı.';

  @override
  String gameReadyProgress(int done, int total) {
    return '$done/$total hazır';
  }

  @override
  String get gameVotingInfo => 'Oylama sürüyor. Oylar gizli.';

  @override
  String gameVoteProgress(int done, int total) {
    return '$done/$total oy verildi';
  }

  @override
  String get gameTieBreakInfo => 'Eşitlik! İzleyiciler karar veriyor…';

  @override
  String gameEliminated(String name) {
    return '$name sığınağın dışında kaldı.';
  }

  @override
  String get gameDealingInfo => 'Kartlar dağıtılıyor…';

  @override
  String get gameRevealRound1Hint =>
      'İlk turda yalnızca meslek kartını açabilirsin.';

  @override
  String gameRevealConfirmTitle(String category) {
    return '$category kartını aç?';
  }

  @override
  String gameRevealConfirmMessage(String text) {
    return '“$text” herkese açılacak. Açılan kart geri kapatılamaz.';
  }

  @override
  String get gameRevealConfirm => 'Aç';

  @override
  String get gameReadyButton => 'Hazırım';

  @override
  String get gameReadyDone => 'Hazırsın. Diğerleri bekleniyor.';

  @override
  String get gameVoteHint => 'Elemek istediğin oyuncuya dokun.';

  @override
  String gameVoteConfirmTitle(String name) {
    return '$name için oy ver?';
  }

  @override
  String get gameVoteConfirmMessage => 'Oyunu verdikten sonra değiştiremezsin.';

  @override
  String get gameVoteConfirm => 'Oy ver';

  @override
  String get gameVoteDone => 'Oyunu verdin.';

  @override
  String get gameTieBreakMine =>
      'Eşitlik! Kararı izleyiciler veriyor; sen de oy ver.';

  @override
  String gameVoteFor(String name) {
    return '$name için oy ver';
  }

  @override
  String get gameMethodVote => 'Oylamayla elendi';

  @override
  String get gameMethodSpectators => 'İzleyicilerin kararıyla elendi';

  @override
  String get gameMethodLottery => 'Eşitlik kurayla bozuldu';

  @override
  String get gameMethodLeft => 'Oyundan ayrıldı';

  @override
  String get gameMethodShortVote => 'oylama';

  @override
  String get gameMethodShortSpectators => 'izleyiciler';

  @override
  String get gameMethodShortLottery => 'kura';

  @override
  String get gameMethodShortLeft => 'ayrıldı';

  @override
  String gameVotesCount(int count) {
    return '$count oy';
  }

  @override
  String get gameNoVotes => 'Kimse oy vermedi; herkes eşit sayıldı.';

  @override
  String gameLotteryAmong(String names) {
    return 'Kura adayları: $names';
  }

  @override
  String get gameEliminatedCards => 'Kartları';

  @override
  String get gameDoorClosedTitle => 'Kapı kapandı.';

  @override
  String get gameDoorClosedBody => 'Artık izleyicisin.';

  @override
  String get resultWinners => 'Sığınağa girenler';

  @override
  String get resultLosers => 'Dışarıda kalanlar';

  @override
  String get resultRounds => 'Tur tur';

  @override
  String resultRoundLine(int round, String name, String method) {
    return 'Tur $round: $name ($method)';
  }

  @override
  String get resultRematch => 'Tekrar oyna';

  @override
  String get resultAbandoned =>
      'Oyun erken bitti: oyunda yeterli oyuncu kalmadı.';

  @override
  String get resultWaitingHost => 'Oda sahibi yeni bir oyun başlatabilir.';

  @override
  String get resultLoading => 'Sonuçlar yükleniyor…';

  @override
  String get chatTitle => 'Sohbet';

  @override
  String get chatOpen => 'Sohbet';

  @override
  String chatUnread(int count) {
    return 'Sohbet, $count okunmamış mesaj';
  }

  @override
  String get chatEmpty => 'Henüz mesaj yok. İlk sen yaz!';

  @override
  String get chatHint => 'Mesaj yaz…';

  @override
  String get chatSend => 'Gönder';

  @override
  String get chatSending => 'Gönderiliyor…';

  @override
  String get chatRetryHint => 'Tekrar denemek için dokun.';

  @override
  String get chatSpectatorNote => 'İzleyiciler yazamaz.';

  @override
  String get modReport => 'Raporla';

  @override
  String get modBlock => 'Engelle';

  @override
  String modReportTitle(String name) {
    return '$name adlı oyuncuyu raporla';
  }

  @override
  String get modReasonHarassment => 'Taciz veya zorbalık';

  @override
  String get modReasonHate => 'Nefret söylemi';

  @override
  String get modReasonSpam => 'Spam veya reklam';

  @override
  String get modReasonCheating => 'Hile';

  @override
  String get modReasonOther => 'Diğer';

  @override
  String get modReportNote => 'Not (isteğe bağlı)';

  @override
  String get modReportSend => 'Gönder';

  @override
  String get modReportSent => 'Raporun alındı. Teşekkürler.';

  @override
  String modBlockTitle(String name) {
    return '$name engellensin mi?';
  }

  @override
  String get modBlockMessage =>
      'Mesajlarını artık görmezsin. Oyunu etkilemez; engeli Ayarlar\'dan kaldırabilirsin.';

  @override
  String modBlocked(String name) {
    return '$name engellendi.';
  }

  @override
  String get blocksTitle => 'Engellenen oyuncular';

  @override
  String get blocksEmpty => 'Engellediğin kimse yok.';

  @override
  String get blocksUnblock => 'Engeli kaldır';

  @override
  String blocksUnblocked(String name) {
    return '$name artık engelli değil.';
  }

  @override
  String get settingsTitle => 'Ayarlar';

  @override
  String get settingsNickname => 'Takma ad';

  @override
  String get settingsAccount => 'Hesap';

  @override
  String get settingsDeleteAccount => 'Hesabımı sil';

  @override
  String get settingsDeleteTitle => 'Hesabın silinsin mi?';

  @override
  String get settingsDeleteMessage =>
      'Takma adın ve oturumların silinir, mesajların kaldırılır. Bulunduğun odadan çıkarsın; oyun sürüyorsa elenmiş sayılırsın. Bu işlem geri alınamaz.';

  @override
  String get settingsDeleteContinue => 'Devam';

  @override
  String get settingsDeleteFinalTitle => 'Emin misin?';

  @override
  String get settingsDeleteFinalMessage =>
      'Hesabın kalıcı olarak silinecek ve geri getirilemeyecek.';

  @override
  String settingsVersion(String version) {
    return 'Sürüm $version';
  }

  @override
  String get legalPrivacy => 'Gizlilik politikası';

  @override
  String get legalTerms => 'Kullanım koşulları';

  @override
  String get legalOpenFailed => 'Bağlantı açılamadı.';

  @override
  String get welcomeLegalNote =>
      'Başlayarak Kullanım Koşulları\'nı kabul etmiş olursun. Uygulama 13 yaş ve üstü içindir.';

  @override
  String get settingsLegal => 'Yasal';
}
