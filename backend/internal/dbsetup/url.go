package dbsetup

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ConnParams, bağlantı adresinin parçalarıdır.
type ConnParams struct {
	Role     string
	Password string
	// ProjectRef, Supabase proje kimliği (20 küçük harf). Supavisor kullanıcı
	// adı <rol>.<ref> biçimindedir. Boşsa kullanıcı adı yalnızca roldür (yerel test).
	ProjectRef string
	Host       string
	Port       int
	Database   string
	SSLMode    string // verify-full (varsayılan) | require | disable
	CertFile   string // sslrootcert; verify-full için zorunlu
}

var (
	projectRefRe = regexp.MustCompile(`^[a-z]{20}$`)
	hostRe       = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
)

// Validate, parametreleri denetler. Supabase sunucusunda ref ve verify-full zorunlu.
func (p ConnParams) Validate() error {
	var errs []error
	if p.Role == "" || p.Password == "" {
		errs = append(errs, errors.New("rol ve şifre gerekli"))
	}
	if p.ProjectRef != "" && !projectRefRe.MatchString(p.ProjectRef) {
		errs = append(errs, fmt.Errorf("proje kimliği 20 küçük harf olmalı: %q", p.ProjectRef))
	}
	if net.ParseIP(p.Host) == nil && !hostRe.MatchString(p.Host) {
		errs = append(errs, fmt.Errorf("sunucu adı geçersiz: %q", p.Host))
	}
	if strings.HasSuffix(p.Host, ".supabase.com") || strings.HasSuffix(p.Host, ".supabase.co") {
		if p.ProjectRef == "" {
			errs = append(errs, errors.New("Supabase için -ref (proje kimliği) gerekli"))
		}
		if p.SSLMode != "verify-full" {
			errs = append(errs, errors.New("Supabase için sslmode=verify-full zorunlu"))
		}
	}
	if p.Port < 1 || p.Port > 65535 {
		errs = append(errs, fmt.Errorf("port geçersiz: %d", p.Port))
	}
	switch p.SSLMode {
	case "verify-full":
		if p.CertFile == "" {
			errs = append(errs, errors.New("verify-full için sertifika dosyası (-cert) gerekli"))
		}
	case "require", "disable":
	default:
		errs = append(errs, fmt.Errorf("sslmode geçersiz: %q", p.SSLMode))
	}
	return errors.Join(errs...)
}

// URL, pgx'in anlayacağı bağlantı adresini üretir. Sonuç gizlidir.
func (p ConnParams) URL() string {
	user := p.Role
	if p.ProjectRef != "" {
		user += "." + p.ProjectRef
	}
	q := url.Values{}
	q.Set("sslmode", p.SSLMode)
	if p.SSLMode == "verify-full" {
		// Eğik çizgiye çevrilir: Windows yolu da çalışır ve .env'de kaçış gerekmez.
		q.Set("sslrootcert", strings.ReplaceAll(p.CertFile, `\`, "/"))
	}
	db := p.Database
	if db == "" {
		db = "postgres"
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, p.Password),
		Host:     net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
		Path:     "/" + db,
		RawQuery: q.Encode(),
	}
	return u.String()
}

// CertInfo, CA sertifikasının özetidir (kullanıcının doğru dosyayı
// indirdiğini görmesi için).
type CertInfo struct {
	Subject  string
	NotAfter string
	SHA256   string
	IsCA     bool
}

// InspectCert, PEM dosyasındaki ilk sertifikayı okur.
func InspectCert(path string) (CertInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CertInfo{}, fmt.Errorf("sertifika okunamadı: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return CertInfo{}, errors.New("sertifika PEM biçiminde değil (-----BEGIN CERTIFICATE-----)")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CertInfo{}, fmt.Errorf("sertifika çözülemedi: %w", err)
	}
	sum := sha256.Sum256(cert.Raw)
	return CertInfo{
		Subject:  cert.Subject.String(),
		NotAfter: cert.NotAfter.UTC().Format("2006-01-02"),
		SHA256:   hex.EncodeToString(sum[:]),
		IsCA:     cert.IsCA,
	}, nil
}
