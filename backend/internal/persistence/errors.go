package persistence

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// UniqueViolation, err belirtilen tekil kısıtın (veya constraint boşsa
// herhangi bir tekil kısıtın) ihlali mi? Modüller veritabanındaki kural
// ihlallerini (ör. "tek aktif oda") alan hatasına bu yolla çevirir.
func UniqueViolation(err error, constraint string) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505" && (constraint == "" || pg.ConstraintName == constraint)
}
