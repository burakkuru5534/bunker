package httpapi

import "testing"

func TestParseVersion(t *testing.T) {
	v, err := ParseVersion("1.12.3")
	if err != nil || v != (Version{1, 12, 3}) || v.String() != "1.12.3" {
		t.Fatalf("ParseVersion = %v, %v", v, err)
	}
	for _, bad := range []string{"", "1", "1.2", "1.2.3.4", "1.2.x", "-1.2.3", "01.2.3", "1.2.3+4", "1.2.3-beta", " 1.2.3", "1..3", "1234567.0.0"} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("%q reddedilmeli", bad)
		}
	}
}

func TestVersionLess(t *testing.T) {
	cases := []struct {
		a, b Version
		less bool
	}{
		{Version{0, 1, 0}, Version{0, 2, 0}, true},
		{Version{0, 2, 0}, Version{0, 2, 0}, false},
		{Version{0, 10, 0}, Version{0, 9, 9}, false},
		{Version{1, 0, 0}, Version{0, 99, 99}, false},
		{Version{1, 2, 3}, Version{1, 2, 4}, true},
	}
	for _, c := range cases {
		if c.a.Less(c.b) != c.less {
			t.Errorf("%v < %v = %v bekleniyordu", c.a, c.b, c.less)
		}
	}
}
