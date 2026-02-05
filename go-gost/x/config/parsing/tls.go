package parsing

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gost/core/logger"
	"github.com/go-gost/x/config"
	tls_util "github.com/go-gost/x/internal/util/tls"
)

var (
	defaultTLSConfig atomic.Value
	deviceID         string
	deviceIDOnce     sync.Once
)

func getDeviceID() string {
	deviceIDOnce.Do(func() {
		idFile := "device.id"
		if data, err := os.ReadFile(idFile); err == nil && len(data) > 0 {
			deviceID = string(data)
			return
		}
		
		deviceID = generateDeviceID()
		
		if err := os.WriteFile(idFile, []byte(deviceID), 0644); err != nil {
			if log := logger.Default(); log != nil {
				log.Warnf("Failed to save device ID: %v", err)
			}
		}
	})
	return deviceID
}

func generateDeviceID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		randomBytes = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	}
	
	uniqueStr := fmt.Sprintf("%s-%d-%s", hostname, time.Now().UnixNano(), hex.EncodeToString(randomBytes))
	
	hash := sha256.New()
	hash.Write([]byte(uniqueStr))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

func generateDisguisedDomain(deviceID string) string {
	domains := []string{
		"com", "net", "org", "top", "cc", "info", "biz", "co", "me", "io",
		"us", "cn", "uk", "de", "fr", "jp", "ru", "br", "au", "ca",
	}
	
	prefixes := []string{
		"api", "app", "web", "www", "cdn", "static", "assets", "media", "img", "imgcdn",
		"secure", "ssl", "tls", "https", "gateway", "proxy", "service", "portal", "hub",
		"cloud", "server", "node", "host", "site", "page", "view", "load", "cache",
	}
	
	seed := deviceID[:8]
	
	seedInt := int64(0)
	for _, char := range seed {
		var val int64
		if char >= '0' && char <= '9' {
			val = int64(char - '0')
		} else if char >= 'a' && char <= 'f' {
			val = int64(char - 'a' + 10)
		}
		seedInt = seedInt*16 + val
	}
	
	domainIndex := seedInt % int64(len(domains))
	domain := domains[domainIndex]
	
	prefixIndex := (seedInt / int64(len(domains))) % int64(len(prefixes))
	prefix := prefixes[prefixIndex]
	
	randomSuffix := fmt.Sprintf("%d", seedInt%9999+1000)
	
	return fmt.Sprintf("%s.%s.%s", prefix, randomSuffix, domain)
}

func generateDisguisedOrganization(deviceID string) string {
	companyPrefixes := []string{
		"Cloud", "Digital", "Secure", "Global", "Advanced", "Enterprise", "Professional",
		"Tech", "Data", "Network", "System", "Service", "Solution", "Platform",
		"Smart", "Fast", "Reliable", "Modern", "Innovative", "Dynamic",
	}
	
	companySuffixes := []string{
		"Technologies", "Systems", "Solutions", "Services", "Corporation", "Corp",
		"Inc", "Ltd", "LLC", "Group", "International", "Global", "Enterprises",
		"Software", "Networks", "Security", "Communications", "Infrastructure",
	}
	
	seed := deviceID[:8]
	
	seedInt := int64(0)
	for _, char := range seed {
		var val int64
		if char >= '0' && char <= '9' {
			val = int64(char - '0')
		} else if char >= 'a' && char <= 'f' {
			val = int64(char - 'a' + 10)
		}
		seedInt = seedInt*16 + val
	}
	
	prefixIndex := seedInt % int64(len(companyPrefixes))
	suffixIndex := (seedInt / int64(len(companyPrefixes))) % int64(len(companySuffixes))
	
	prefix := companyPrefixes[prefixIndex]
	suffix := companySuffixes[suffixIndex]
	
	randomSuffix := fmt.Sprintf("%d", seedInt%999+100)
	
	return fmt.Sprintf("%s %s %s", prefix, randomSuffix, suffix)
}

func DefaultTLSConfig() *tls.Config {
	v, _ := defaultTLSConfig.Load().(*tls.Config)
	return v
}

func SetDefaultTLSConfig(cfg *tls.Config) {
	defaultTLSConfig.Store(cfg)
}

func BuildDefaultTLSConfig(cfg *config.TLSConfig) (*tls.Config, error) {
	if cfg == nil {
		cfg = &config.TLSConfig{
			CertFile: "cert.pem",
			KeyFile:  "key.pem",
			CAFile:   "ca.pem",
		}
	}

	tlsConfig, err := tls_util.LoadDefaultConfig(cfg.CertFile, cfg.KeyFile, cfg.CAFile)
	if err != nil {
		cert, err := genCertificate(cfg.Validity, cfg.Organization, cfg.CommonName)
		if err != nil {
			return nil, err
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		if log := logger.Default(); log != nil {
			log.Debug("load global TLS certificate files failed, use random generated certificate")
		}
	} else {
		if log := logger.Default(); log != nil {
			log.Debug("load global TLS certificate files OK")
		}
	}

	return tlsConfig, nil
}

func genCertificate(validity time.Duration, org string, cn string) (cert tls.Certificate, err error) {
	rawCert, rawKey, err := generateKeyPair(validity, org, cn)
	if err != nil {
		return
	}
	return tls.X509KeyPair(rawCert, rawKey)
}

func generateKeyPair(validity time.Duration, org string, cn string) (rawCert, rawKey []byte, err error) {

	var priv crypto.PrivateKey
	priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return
	}

	if validity <= 0 {
		validity = time.Hour * 24 * 365 // one year
	}
	if org == "" {
		deviceID := getDeviceID()
		org = generateDisguisedOrganization(deviceID)
	}
	if cn == "" {
		deviceID := getDeviceID()
		cn = generateDisguisedDomain(deviceID)
	}

	validFor := validity
	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{org},
			CommonName:   cn,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}
	if _, isRSA := priv.(*rsa.PrivateKey); isRSA {
		template.KeyUsage |= x509.KeyUsageKeyEncipherment
	}

	template.DNSNames = append(template.DNSNames, cn)
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		return
	}

	rawCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return
	}
	rawKey = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	return
}

func publicKey(priv crypto.PrivateKey) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}
