// Structure of data for all state changes
package cbdomain

import (
	"github.com/function61/eventhorizon/pkg/ehevent"
	"time"
)

var Types = ehevent.Allocators{
	"CertificateObtained": func() ehevent.Event { return &CertificateObtained{} },
	"CertificateRemoved":  func() ehevent.Event { return &CertificateRemoved{} },
	"ConfigUpdated":       func() ehevent.Event { return &ConfigUpdated{} },
}

// ------

type CertificateObtained struct {
	meta                     ehevent.EventMeta
	Id                       string
	Reason                   string // "new" | "renewal"
	Domains                  []string
	Expires                  time.Time
	CertPemBundle            string
	PrivateKeyDekFingerprint string // identity of the DEK that encrypted this private key
	PrivateKeyCiphertext     []byte
}

func (e *CertificateObtained) MetaType() string         { return "CertificateObtained" }
func (e *CertificateObtained) Meta() *ehevent.EventMeta { return &e.meta }

func NewCertificateObtained(
	id string,
	reason string,
	domains []string,
	expires time.Time,
	certPemBundle string,
	privateKeyDekFingerprint string,
	privateKeyCiphertext []byte,
	meta ehevent.EventMeta,
) *CertificateObtained {
	return &CertificateObtained{
		meta:                     meta,
		Id:                       id,
		Reason:                   reason,
		Domains:                  domains,
		Expires:                  expires,
		CertPemBundle:            certPemBundle,
		PrivateKeyDekFingerprint: privateKeyDekFingerprint,
		PrivateKeyCiphertext:     privateKeyCiphertext,
	}
}

// ------

type CertificateRemoved struct {
	meta ehevent.EventMeta
	Id   string
}

func (e *CertificateRemoved) MetaType() string         { return "CertificateRemoved" }
func (e *CertificateRemoved) Meta() *ehevent.EventMeta { return &e.meta }

func NewCertificateRemoved(
	id string,
	meta ehevent.EventMeta,
) *CertificateRemoved {
	return &CertificateRemoved{
		meta: meta,
		Id:   id,
	}
}

// ------

type ConfigUpdated struct {
	meta                           ehevent.EventMeta
	ConfigEncryptionKeyFingerprint string
	ConfigCiphertext               []byte
}

func (e *ConfigUpdated) MetaType() string         { return "ConfigUpdated" }
func (e *ConfigUpdated) Meta() *ehevent.EventMeta { return &e.meta }

func NewConfigUpdated(
	keyFingerprint string,
	configCiphertext []byte,
	meta ehevent.EventMeta,
) *ConfigUpdated {
	return &ConfigUpdated{
		meta:                           meta,
		ConfigEncryptionKeyFingerprint: keyFingerprint,
		ConfigCiphertext:               configCiphertext,
	}
}
