package certificatestore

import (
	"time"
)

func CertsDueForRenewal(store *Store, now time.Time) []ManagedCertificate {
	due := []ManagedCertificate{}
	for _, cert := range store.All() {
		if cert.RenewAt.Before(now) {
			due = append(due, cert)
		}
	}

	return due
}

// one month before
func renewAtFromExpiration(expires time.Time) time.Time {
	return expires.AddDate(0, -1, 0)
}
