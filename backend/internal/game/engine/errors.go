package engine

import "github.com/burakkuru5534/bunker/backend/internal/apperr"

// Kural hataları istemciye apperr kodlarıyla gider (api.md §2).
func invalidPhase() error { return apperr.New(apperr.InvalidPhase, "eylem bu aşamada yapılamaz") }
