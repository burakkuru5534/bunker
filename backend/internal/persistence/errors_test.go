package persistence

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestUniqueViolation(t *testing.T) {
	err := fmt.Errorf("sarılı: %w", &pgconn.PgError{Code: "23505", ConstraintName: "rooms_invite_code_open_key"})
	if !UniqueViolation(err, "rooms_invite_code_open_key") || !UniqueViolation(err, "") {
		t.Fatal("tekil ihlal tanınmalı")
	}
	if UniqueViolation(err, "başka") || UniqueViolation(&pgconn.PgError{Code: "23503"}, "") || UniqueViolation(errors.New("x"), "") {
		t.Fatal("yanlış eşleşme")
	}
}
