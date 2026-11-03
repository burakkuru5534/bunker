package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name   string
		remote string
		xff    []string
		hops   int
		want   string
	}{
		{"yerel", "10.0.0.5:4321", nil, 0, "10.0.0.5"},
		{"yerelde başlık yok sayılır", "10.0.0.5:4321", []string{"1.2.3.4"}, 0, "10.0.0.5"},
		{"tek vekil", "169.254.1.1:80", []string{"203.0.113.7"}, 1, "203.0.113.7"},
		{"sahte sol girdiler", "169.254.1.1:80", []string{"6.6.6.6, 7.7.7.7, 203.0.113.7"}, 1, "203.0.113.7"},
		{"çoklu başlık satırı", "169.254.1.1:80", []string{"6.6.6.6", "203.0.113.7"}, 1, "203.0.113.7"},
		{"iki vekil", "169.254.1.1:80", []string{"6.6.6.6, 203.0.113.7, 35.191.0.1"}, 2, "203.0.113.7"},
		{"başlık yok", "169.254.1.1:80", nil, 1, "169.254.1.1"},
		{"yetersiz girdi", "169.254.1.1:80", []string{"203.0.113.7"}, 2, "169.254.1.1"},
		{"bozuk girdi", "169.254.1.1:80", []string{"bozuk"}, 1, "169.254.1.1"},
		{"ipv6", "[2001:db8::1]:443", nil, 0, "2001:db8::1"},
		{"port yok", "192.0.2.9", nil, 0, "192.0.2.9"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.remote
		for _, v := range c.xff {
			r.Header.Add("X-Forwarded-For", v)
		}
		if got := ClientIP(r, c.hops); got != c.want {
			t.Errorf("%s: %q, beklenen %q", c.name, got, c.want)
		}
	}
}

func TestRateKey(t *testing.T) {
	if RateKey("203.0.113.7") != "203.0.113.7" || RateKey("::ffff:203.0.113.7") != "203.0.113.7" {
		t.Fatal("IPv4 olduğu gibi")
	}
	a, b := RateKey("2001:db8:1:2:aaaa::1"), RateKey("2001:db8:1:2:bbbb::9")
	if a != b || a != "2001:db8:1:2::/64" {
		t.Fatalf("aynı /64 aynı anahtar: %s %s", a, b)
	}
	if RateKey("2001:db8:1:3::1") == a {
		t.Fatal("farklı /64 farklı anahtar")
	}
	if RateKey("bozuk") != "bozuk" {
		t.Fatal("bozuk girdi olduğu gibi")
	}
}
