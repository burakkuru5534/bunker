package profanity

import "testing"

// Yakalanması gerekenler (sohbet metni → maskelenmiş).
var dirty = map[string]string{
	"siktir git":              "*** git",
	"SİKTİR":                  "***",
	"s.i.k.t.i.r lan":         "*** lan",
	"s i k t i r":             "***",
	"siiiiktir":               "***",
	"s1kt1r":                  "***",
	"$ikerim":                 "***",
	"sikeyim":                 "***", // "sikke" istisnası tek k'li yazımı kapsamamalı
	"orospu çocuğu":           "*** çocuğu",
	"oro$pu":                  "***",
	"amk ya":                  "*** ya",
	"AQ":                      "***",
	"oç":                      "***",
	"bu adam şerefsiz":        "bu adam ***",
	"serefsizin teki":         "*** teki", // Türkçe klavyesiz yazım
	"yavsak":                  "***",
	"piç":                     "***",
	"götünü":                  "***",
	"gotunu":                  "***",
	"yarrrak":                 "***",
	"amına koyim":             "*** koyim",
	"amcık":                   "***",
	"amcik":                   "***",
	"fuck you":                "*** you",
	"motherfucker":            "***",
	"F.U.C.K":                 "***",
	"what the sh1t":           "what the ***",
	"selam, siktir! nasılsın": "selam, *** nasılsın",
}

// Masum: değişmemeli ve takma adda reddedilmemeli.
var clean = []string{
	"sıkıcı bir oyun", "sıkıntı yok", "sık sık oynarım", "sıkıştık", "çok sıkı",
	"şikayet ediyorum", "şike var", "şık elbise", "sikke koleksiyonu", "siklet farkı",
	"siklamen çiçeği", "siklon geliyor", "Sikh dini", "sikinti yok", "sikici bir tur",
	"götürür müsün", "götürmek", "götür onu", "ama neden", "amaç ne", "amca", "aman",
	"amin", "Amin", "I am here", "amino asit", "Ankara", "klasik", "fıkra",
	"yarar", "yaratık", "yarasa", "class", "pass", "assassin", "piknik", "picasso",
	"dolu", "dolar", "dolap", "dolmuş", "taşıma", "taşınmak", "mekan", "mkdir",
	"ibrahim", "kahve", "pesimist", "gavur", "kaltabanlık yok", "sürtünme",
	"basit", "masa", "bitki", "shiitake", "ashley", "Dickens", "pussycat", "cocktail",
	"orman", "orospuçocuğu değil ama", // birleşik → aşağıda ayrıca maskelenir, bkz. test
}

func TestMaskDirty(t *testing.T) {
	for in, want := range dirty {
		if got := Default.Mask(in); got != want {
			t.Errorf("Mask(%q) = %q, beklenen %q", in, got, want)
		}
		if !Default.Contains(in) {
			t.Errorf("Contains(%q) = false", in)
		}
	}
}

func TestCleanUntouched(t *testing.T) {
	for _, in := range clean[:len(clean)-1] {
		if got := Default.Mask(in); got != in {
			t.Errorf("masum metin değişti: %q → %q", in, got)
		}
		if Default.Contains(in) {
			t.Errorf("masum metin küfür sayıldı: %q", in)
		}
	}
	// Bitişik yazılmış küfür kelime başında yakalanır.
	if got := Default.Mask("orospuçocuğu değil ama"); got != "*** değil ama" {
		t.Errorf("bitişik: %q", got)
	}
}

func TestNicknames(t *testing.T) {
	for _, n := range []string{"Siktirci", "s_i_k_t_i_r", "Ali_Kahpe", "0r0spu", "AMK_42", "Serefsiz 1"} {
		if !Default.Contains(n) {
			t.Errorf("takma ad geçti: %q", n)
		}
	}
	for _, n := range []string{"Sıkıcı Ali", "Şikayetçi", "Amca_42", "Götürücü", "Işık", "Gökçe_7", "Siklon"} {
		if Default.Contains(n) {
			t.Errorf("masum takma ad reddedildi: %q", n)
		}
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"SİKTİR": "siktir", "IŞIK": "ışık", "s.i-k": "sik", "siiik": "siiik", "@m!n": "amin", "": "",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, beklenen %q", in, got, want)
		}
	}
}

// Maskeleme metnin geri kalanını ve boşluklarını korur.
func TestMaskPreservesRest(t *testing.T) {
	in := "  çok   güzel   amk   oyun  "
	if got := Default.Mask(in); got != "  çok   güzel   ***   oyun  " {
		t.Fatalf("%q", got)
	}
}
