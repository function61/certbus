package certificatestore

import (
	"crypto/tls"
	"strings"
)

// store interfaces only have ByHostname(exactHostname), so use these resolvers to look up
// foo.example.com for when you want to support possibly finding *.example.com cert.
//
// internally looks up the hostname first, and if it's not found, then it looks up the wildcard variant.

func ByHostnameSupportingWildcard(hostname string, store ManagedCertificateByHostnameFinder) *ManagedCertificate {
	cert := store.ByHostname(hostname)
	if cert != nil {
		return cert
	}

	return store.ByHostname(wildcardVersionOfHostname(hostname))
}

func DecryptedByHostnameSupportingWildcard(hostname string, store *DecryptedStore) (*tls.Certificate, error) {
	cert, err := store.ByHostname(hostname)
	if cert != nil {
		return cert, err
	}

	return store.ByHostname(wildcardVersionOfHostname(hostname))
}

// "foobar.example.com" => "*.example.com"
func wildcardVersionOfHostname(hostname string) string {
	if hostname == "" {
		return ""
	}

	return "*." + strings.Join(strings.Split(hostname, ".")[1:], ".")
}
