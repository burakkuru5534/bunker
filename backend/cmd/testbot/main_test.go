package main

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	ok := [][]string{
		{"-code", "k7m2qx"},
		{"-create"},
		{"-create", "-bots", "9", "-start-at", "10"},
		{"-code", "K7M2QX", "-api", "http://192.168.1.20:8080"},
		{"-code", "K7M2QX", "-api", "http://10.0.2.2:8080"},
		{"-code", "K7M2QX", "-api", "https://bunker-api-test.example.app", "-allow-remote"},
	}
	for _, args := range ok {
		o, err := parse(args)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if o.code != "" && o.code != "K7M2QX" {
			t.Fatalf("kod büyütülmeli: %q", o.code)
		}
	}

	bad := map[string][]string{
		"kod/oluştur yok":   {},
		"ikisi birden":      {"-create", "-code", "K7M2QX"},
		"bot sayısı":        {"-create", "-bots", "0"},
		"bot üst sınır":     {"-create", "-bots", "10"},
		"start-at":          {"-create", "-start-at", "3"},
		"uzak adres":        {"-code", "K7M2QX", "-api", "https://api.example.com"},
		"genel IP":          {"-code", "K7M2QX", "-api", "http://8.8.8.8:8080"},
		"bozuk adres":       {"-code", "K7M2QX", "-api", "ftp://localhost"},
		"bilinmeyen bayrak": {"-x"},
	}
	for name, args := range bad {
		if _, err := parse(args); err == nil {
			t.Errorf("%s: hata bekleniyordu", name)
		}
	}
	if _, err := parse([]string{"-code", "K", "-api", "https://api.example.com"}); err == nil || !strings.Contains(err.Error(), "-allow-remote") {
		t.Fatalf("uzak adres açıklaması: %v", err)
	}
}
