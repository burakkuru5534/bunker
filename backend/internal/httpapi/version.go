package httpapi

import (
	"fmt"
	"strconv"
	"strings"
)

// Version, MAJOR.MINOR.PATCH biçiminde uygulama sürümü.
type Version struct{ Major, Minor, Patch int }

// ParseVersion, "1.2.3" biçimini çözer. Ön ek veya ek (ör. "+5") kabul edilmez.
func ParseVersion(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("sürüm %q: MAJOR.MINOR.PATCH bekleniyor", s)
	}
	var n [3]int
	for i, p := range parts {
		if p == "" || len(p) > 6 {
			return Version{}, fmt.Errorf("sürüm %q geçersiz", s)
		}
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 || (len(p) > 1 && p[0] == '0') {
			return Version{}, fmt.Errorf("sürüm %q geçersiz", s)
		}
		n[i] = v
	}
	return Version{n[0], n[1], n[2]}, nil
}

// Less, v < o mu?
func (v Version) Less(o Version) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	return v.Patch < o.Patch
}

func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }
