package profanity

// Kelime listeleri (SAFE-013). Bilinçli olarak ölçülü tutulur: amaç ağır
// küfür ve hakaretleri yakalamak, gündelik konuşmayı engellememektir.
// Hafif sözler ("salak", "aptal") listede yoktur.
//
// Kelimeler doğru Türkçe yazımıyla girilir (ş, ç, ğ, ı, ö, ü); yükleme
// sırasında Normalize'dan geçer (küçük harf, tekrar eden harf teke iner).
// Eşleşme tek yönlüdür: metindeki düz harf listedeki Türkçe harfin yerine
// geçebilir ("serefsiz" ~ "şerefsiz"), tersi geçemez ("şikayet" ≁ "sik").
//
// Yeni kelime eklerken: önce filter_test.go'daki masum örneklere bir karşılık
// ekleyin; kelime başı eşleşmesi (prefixes) Türkçe eklerle birçok masum
// kelimeyi yakalayabilir.

// prefixes: kelime bu kökle başlıyorsa küfür (ekli halleri yakalar).
var prefixes = []string{
	// Türkçe
	"sik", "orospu", "orosbu", "yarrak", "dalyarak", "amcık", "amına", "amınakoy",
	"kahpe", "pezevenk", "gavat", "ibne", "yavşak", "şerefsiz", "kaltak",
	"sürtük", "puşt", "godoş", "kancık", "götveren", "götlek", "taşak", "taşşak",
	// İngilizce
	"fuck", "motherfuck", "shit", "bitch", "cunt", "whore", "slut", "bastard", "nigger", "nigga", "faggot",
}

// exact: yalnızca kelimenin tamamı eşleşirse küfür (kısaltmalar ve kökü
// masum kelimelere de ön ek olan sözcükler). "am" ve "amın" bilinçli olarak
// yok: İngilizce "I am" ve düz yazılmış "amin" ile çakışır.
var exact = []string{
	"amk", "amq", "aq", "mk", "oç", "sg", "piç", "piçler", "piçin",
	"göt", "götü", "götün", "götünü", "götüne", "götten", "götler",
	"amını", "yarak", "dick", "cock", "pussy", "ass", "asshole",
}

// allow: kelime bu önekle başlıyorsa yukarıdaki kökler uygulanmaz (masum
// kelimeler). Harf harf eşleşir (tekrarlar dahil): "sikke" yalnızca çift k
// ile; tek k'li "sike…" küfür olarak kalır. Bilinen sınır: istisnayı bilerek
// taklit eden yazım (ör. "sikke" ile başlayıp küfre devam eden) yakalanmaz;
// bunu çözmek Türkçe ek çözümlemesi gerektirir. Raporlama bu boşluğu kapatır.
var allow = []string{
	"sikke", "siklet", "siklamen", "siklon", "sikloid", "siklik", "sikh",
	// Türkçe klavyesi olmayanın düz yazdığı yaygın masum kelimeler
	// (sıkıntı, sıkıcı).
	"sikint", "sikici",
	"shitake", "shiitake",
}
