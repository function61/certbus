package certificatestore

import (
	"context"
	"log"
	"sync"

	"github.com/function61/certbus/pkg/cbdomain"
	"github.com/function61/certbus/pkg/encryptedbox"
	"github.com/function61/eventhorizon/pkg/ehclient"
	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/eventhorizon/pkg/ehreader"
	"github.com/function61/gokit/logex"
)

const (
	Stream = "/certbus"
)

// "aggregate"
type Store struct {
	certificates []*ManagedCertificate
	byHostname   map[string]*ManagedCertificate
	latestConfig *cbdomain.ConfigUpdated
	version      ehclient.Cursor
	mu           sync.Mutex
	logl         *logex.Leveled
}

func New(tenant ehreader.Tenant, logger *log.Logger) *Store {
	return &Store{
		certificates: []*ManagedCertificate{},
		byHostname:   map[string]*ManagedCertificate{},
		version:      ehclient.Beginning(tenant.Stream(Stream)),
		logl:         logex.Levels(logger),
	}
}

func (c *Store) GetEventTypes() ehevent.Allocators {
	return cbdomain.Types
}

func (c *Store) Version() ehclient.Cursor {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.version
}

func (c *Store) GetLatestEncryptedConfig() *cbdomain.ConfigUpdated {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.latestConfig
}

func (c *Store) ByHostname(hostname string) *ManagedCertificate {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.byHostname[hostname]
}

func (c *Store) ById(id string) *ManagedCertificate {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cert := range c.certificates {
		if cert.Id == id {
			return cert
		}
	}

	return nil
}

func (c *Store) All() []ManagedCertificate {
	c.mu.Lock()
	defer c.mu.Unlock()

	copied := []ManagedCertificate{}
	for _, cert := range c.certificates {
		copied = append(copied, *cert)
	}

	return copied
}

func (c *Store) ProcessEvents(_ context.Context, processAndCommit ehreader.EventProcessorHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return processAndCommit(
		c.version,
		func(ev ehevent.Event) error { return c.processEvent(ev) },
		func(version ehclient.Cursor) error {
			c.version = version
			return nil
		})
}

func (c *Store) processEvent(ev ehevent.Event) error {
	switch e := ev.(type) {
	case *cbdomain.CertificateObtained:
		c.logl.Info.Printf("CertificateObtained domains=%v", e.Domains)

		cert := &ManagedCertificate{
			Id:      e.Id,
			Domains: e.Domains,
			RenewAt: renewAtFromExpiration(e.Expires),
			Certificate: CertDetails{
				NotAfter:      e.Expires,
				CertPemBundle: e.CertPemBundle,
				PrivateKeyEncrypted: &encryptedbox.Box{
					KeyFingerprint: e.PrivateKeyDekFingerprint,
					Ciphertext:     e.PrivateKeyCiphertext,
				},
			},
		}

		// since we'll append the cert to a list, we don't want 2x CertificateObtained
		// events adding two items to the list. double is natural due to renewals
		c.removeCertById(cert.Id)

		for _, domain := range cert.Domains {
			c.byHostname[domain] = cert
		}

		c.certificates = append(c.certificates, cert)
	case *cbdomain.CertificateRemoved:
		c.logl.Info.Printf("CertificateRemoved id=%s", e.Id)

		c.removeCertById(e.Id)
	case *cbdomain.ConfigUpdated:
		c.logl.Info.Println("ConfigUpdated")

		c.latestConfig = e
	default:
		return ehreader.UnsupportedEventTypeErr(ev)
	}

	return nil
}

func (c *Store) removeCertById(id string) {
	for idx, cert := range c.certificates {
		if cert.Id != id {
			continue
		}

		c.certificates = append(c.certificates[:idx], c.certificates[idx+1:]...)

		for _, domain := range cert.Domains {
			delete(c.byHostname, domain)
		}

		break
	}
}
