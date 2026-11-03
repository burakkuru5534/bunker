package profanity_test

import (
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/profanity"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Filtre, bağlandığı arayüzlere uyar ve takma ad doğrulamasında
// profanity alan koduyla reddeder.
func TestWiring(t *testing.T) {
	var _ auth.ProfanityChecker = profanity.Default.Contains
	var _ room.TextFilter = profanity.Default

	_, err := auth.ValidateNickname("Siktirci", profanity.Default.Contains)
	e, ok := err.(*apperr.Error)
	if !ok || e.Code != apperr.NicknameInvalid || e.Fields["nickname"] != apperr.FieldProfanity {
		t.Fatalf("küfürlü takma ad: %v", err)
	}
	for _, n := range []string{"Sıkıcı Ali", "Amca_42", "Işık"} {
		if _, err := auth.ValidateNickname(n, profanity.Default.Contains); err != nil {
			t.Fatalf("%q reddedildi: %v", n, err)
		}
	}
	// Sohbet: temizleme sonrası maskeleme (servisteki sıra).
	text, err := room.CleanMessage("selam\namk   nasılsın")
	if err != nil || profanity.Default.Mask(text) != "selam *** nasılsın" {
		t.Fatalf("sohbet: %q %v", profanity.Default.Mask(text), err)
	}
}
