package httpapi

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP, isteğin istemci IP'sini döndürür (sıklık sınırı için).
//
// trustedHops = 0: doğrudan bağlantı (yerel) → RemoteAddr.
// trustedHops = n: önümüzde n güvenilir vekil var; X-Forwarded-For'un
// sağdan n'inci girdisi kullanılır. Soldaki girdileri istemci yazabilir;
// onlara güvenilmez. Başlık yoksa veya girdi bozuksa RemoteAddr'a düşer.
// Cloud Run'da varsayılan 1'dir; ilk dağıtımda doğrulanır (security.md §1).
func ClientIP(r *http.Request, trustedHops int) string {
	if trustedHops > 0 {
		var parts []string
		for _, h := range r.Header.Values("X-Forwarded-For") {
			for _, p := range strings.Split(h, ",") {
				if p = strings.TrimSpace(p); p != "" {
					parts = append(parts, p)
				}
			}
		}
		if i := len(parts) - trustedHops; i >= 0 && i < len(parts) {
			if ip := net.ParseIP(parts[i]); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}

// RateKey, IP'yi sıklık sınırı anahtarına çevirir: IPv4 olduğu gibi,
// IPv6 /64 ağına indirgenir (kullanıcı /64 içinde adres değiştirebilir).
func RateKey(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String() + "/64"
}
