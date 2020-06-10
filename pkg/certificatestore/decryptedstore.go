package certificatestore

import (
	"crypto/rsa"
	"crypto/tls"
	"sync"

	"github.com/function61/eventhorizon/pkg/ehclient"
	"github.com/function61/gokit/cryptoutil"
)

type ManagedCertificateByHostnameFinder interface {
	ByHostname(string) *ManagedCertificate
}

type VersionedByHostnameFinder interface {
	ManagedCertificateByHostnameFinder
	Version() ehclient.Cursor
}

type DecryptedStore struct {
	encryptedStore VersionedByHostnameFinder
	cache          map[string]*tls.Certificate
	cacheVersion   ehclient.Cursor
	key            *rsa.PrivateKey
	keyFingerprint string
	mu             sync.Mutex
}

// wraps encrypted store and on-the-fly decrypts (and caches) with our DEK the cert's private keys
func NewDecryptedStore(est VersionedByHostnameFinder, privateKey string) (*DecryptedStore, error) {
	privKey, err := cryptoutil.ParsePemPkcs1EncodedRsaPrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}

	fingerprint, err := cryptoutil.Sha256FingerprintForPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &DecryptedStore{
		encryptedStore: est,
		cache:          map[string]*tls.Certificate{},
		cacheVersion:   est.Version(),
		key:            privKey,
		keyFingerprint: fingerprint,
	}, nil
}

// NOTE: cert can be nil even if error nil
func (d *DecryptedStore) ByHostname(hostname string) (*tls.Certificate, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// changes in encrypted store?
	if !d.cacheVersion.Equal(d.encryptedStore.Version()) {
		// discard all cache.
		// (not expecting cert changes so frequent as to have an overall effect)
		d.cache = map[string]*tls.Certificate{}
	}

	cached, found := d.cache[hostname]
	if !found {
		managedCert := d.encryptedStore.ByHostname(hostname)
		if managedCert == nil {
			return nil, nil
		}

		// our private key cannot decrypt this?
		if d.keyFingerprint != managedCert.Certificate.PrivateKeyEncrypted.KeyFingerprint {
			return nil, nil
		}

		certKey, err := managedCert.Certificate.PrivateKeyEncrypted.Decrypt(d.key, d.keyFingerprint)
		if err != nil {
			return nil, err
		}

		keypair, err := tls.X509KeyPair([]byte(managedCert.Certificate.CertPemBundle), certKey)
		if err != nil {
			return nil, err
		}

		cached = &keypair

		// sprinkle cache entries for all aliases so for ("*.example.com", "example.com") cert
		// we won't end up polluting cache with a.example.com, b.example.com, c.example.com, ..
		for _, domain := range managedCert.Domains {
			d.cache[domain] = cached
		}
	}

	return cached, nil
}
