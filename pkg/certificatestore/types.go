// Store provides certificates with private keys still encrypted (= unusable)
// DecryptedStore requires KEK (private key) to decrypt the cert private keys
package certificatestore

import (
	"time"

	"github.com/function61/certbus/pkg/encryptedbox"
)

type ManagedCertificate struct {
	Id            string      `json:"id"`
	Domains       []string    `json:"domains"` // when wildcard: ["*.domain", "domain"]
	RenewAt       time.Time   `json:"renew_at"`
	Certificate   CertDetails `json:"certificate"`
	ChallengeType string      `json:"challenge_type"`
}

type CertDetails struct {
	NotAfter            time.Time         `json:"not_after"`
	CertPemBundle       string            `json:"cert_pem_bundle"` // "bundle" = contains intermediate cert
	PrivateKeyEncrypted *encryptedbox.Box `json:"private_key_encrypted"`
}
