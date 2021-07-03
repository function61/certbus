package certificatestore

import (
	"testing"

	"github.com/function61/certbus/pkg/cbdomain"
	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/gokit/assert"
	"github.com/function61/gokit/cryptoutil"
)

func TestDecryptedStore(t *testing.T) {
	certs, t0 := setupCommon(t)

	// count calls to the backing store, so we can validate cache hits/misses
	byHostnameCalls := &backingStoreCountingAdapter{certs, 0}

	assert.Assert(t, byHostnameCalls.calls == 0)

	// this provides access to encrypted private keys in "certs" with our decryption key
	decryptedStore, err := NewDecryptedStore(byHostnameCalls, exampleCertsKek)
	assert.Ok(t, err)

	cert, err := DecryptedByHostnameSupportingWildcard("prod4.fn61.net", decryptedStore)
	assert.Ok(t, err)
	assert.Assert(t, cert != nil)

	assert.Assert(t, byHostnameCalls.calls == 1)

	// test same again for the cached path
	cert, err = decryptedStore.ByHostname("prod4.fn61.net")
	assert.Ok(t, err)
	assert.Assert(t, cert != nil)

	assert.Assert(t, byHostnameCalls.calls == 1)

	// cache misses (that still find cert via wildcard lookup) still increase calls ..
	cert, err = DecryptedByHostnameSupportingWildcard("bar.prod4.fn61.net", decryptedStore)
	assert.Ok(t, err)
	//nolint:staticcheck (false positive)
	assert.Assert(t, cert != nil)

	assert.Assert(t, byHostnameCalls.calls == 2)

	// .. even when wildcard entry is in cache
	_, _ = DecryptedByHostnameSupportingWildcard("bar.prod4.fn61.net", decryptedStore)
	assert.Assert(t, byHostnameCalls.calls == 3)
	_, _ = DecryptedByHostnameSupportingWildcard("bar.prod4.fn61.net", decryptedStore)
	assert.Assert(t, byHostnameCalls.calls == 4)

	//nolint:staticcheck
	pubKey, err := cryptoutil.PublicKeyFromPrivateKey(cert.PrivateKey)
	assert.Ok(t, err)

	fingerprint, err := cryptoutil.Sha256FingerprintForPublicKey(pubKey)
	assert.Ok(t, err)

	assert.EqualString(t, fingerprint, "SHA256:hK9fnUGCB7IdZctrNoS86xbS0RNX+e5/aLDFDrqfpb4")

	// test cache eviction
	pumpEvents(t, certs, cbdomain.NewCertificateRemoved(
		"dummyCertId",
		ehevent.MetaSystemUser(t0)))

	cert, err = DecryptedByHostnameSupportingWildcard("foo.prod4.fn61.net", decryptedStore)
	assert.Ok(t, err)
	assert.Assert(t, cert == nil)

	assert.Assert(t, byHostnameCalls.calls == 6)
}

func TestDecryptedStoreWithWrongKek(t *testing.T) {
	certs, _ := setupCommon(t)

	withCorrectKek, err := NewDecryptedStore(certs, exampleCertsKek)
	assert.Ok(t, err)
	withWrongKek, err := NewDecryptedStore(certs, unrelatedKek)
	assert.Ok(t, err)

	canDecryptCertPrivateKey := func(store *DecryptedStore) bool {
		cert, err := DecryptedByHostnameSupportingWildcard("foobar.prod4.fn61.net", store)
		assert.Ok(t, err)
		return cert != nil
	}

	assert.Assert(t, canDecryptCertPrivateKey(withCorrectKek))
	assert.Assert(t, !canDecryptCertPrivateKey(withWrongKek))
}

type backingStoreCountingAdapter struct {
	VersionedByHostnameFinder
	calls int
}

func (b *backingStoreCountingAdapter) ByHostname(hostname string) *ManagedCertificate {
	b.calls++

	return b.VersionedByHostnameFinder.ByHostname(hostname)
}

const (
	exampleCertsKek = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA2F/JLxSjS0orro9m0pGWHZWPFnljJXCt3T3EOmuKz2eZtV3f
WQn2z10Xr5XjT+AV5/8z/OGldTJSX760lm9rRNSasLglUS1l6x+5CqJIIO8hA+6U
1rqC49wnSjzeEit8gIvJ+BG2jZckya2DwS9moYiJu3vMx4gDs9cPHSuyJ+d7D+xh
wOQRJpymlsY4WMMBJ8uEMnQuKS37Zc/mp3GvRi9+pUPRxno1zxQD1+uUINJ7Mb0Y
dQ7S22+sKfgfmuExGoZzQKkxcylFZXXJjfAOuP8YYEGp0VEfAYwJjAGYjdTaKDjI
s4I5scEy5hNgWLhZ2pytg8YOiRuiG/fC3a3ClQIDAQABAoIBAAsHzwzMY4q6DEII
43gGnf2CG1pM8+X7uZFWzcMgqmHqaSaa04EJhgCKQWPdI0p2JQe/tdnFcxbnatWg
tjoZEgHfSMeLi7N4ugJjip5lKYIsTqWRqxrLRVLybTpWogeRGfa/qZsw4/qR4vk5
FEdr8DJ58HOTWxws7etkIkwdZyariUieKRsz+9eF9iUlvvIChxW7CMC57Y/Vnex2
13aka3pCX3HxBweGMeJR7I/Wye/4IPeHBpaQksDlkeqZ1vAnMBCJC5sVwKGpwlgZ
LLpckmx1uEBN8QZMcrLj2wcOVtaB6SeVSpiz9qdXErt9FttMr4zKzyeLOzgtM09+
oJArtL0CgYEA3BvsCuJ4Oo0j4JPADBX46vQ+UbSMPcQRH46LVkMR3lIb1ThBwd0o
REEhAR6lhLz2GEbBUD8a5Ffz/oyH2O8BbXAryReeWzGQKCfgqjxNwwNs1RtpfkfQ
H+8IY15lZ37oKIgUXDu7XvOQtZ5IbFv03bOr/oATl3HhjliftRTK8F8CgYEA+6f0
3e29NZL5TAS32zYv1fiKyOwxK1Y76O/1as8yXoCKJRKOSS8H0S4H8B9PRaiCvdHt
mQq5hR6ytVdFGHfCoW+TR2HepZF8nBBVsPCHVzNMyYMMU1zzA0K+p97jsJJuIyTl
p7LPj+9TgDnIx/m71YkRVFdq1xGtgwtMZxEfIYsCgYEAgCDo6PUoU70xc1vO4bow
qmT/mgRl1ta5uQr7ZX2peyeE+DvFW5roA8N9+O7kHz74au1VPuddOitQ547azZdj
11cCxg6vqhpR9m4wRCjSg4EM64kHgfE/4Db/RQkAMp0Xe/CrGX3T9tQGGxNGyX1G
L4CV0JKx1OkACiLg5UJzWDUCgYEA5YLmZci+uS+TwWrEK16d/e0w1dHjffylounF
z2WsMFfWpbzom4ITBQmQH8TOTV9D7c6ZfOw1Cl1W6t/umkQO86CIl5+AqUuoc8TK
Ahc7t6GHtHiaMyUgVKb4rq0uxwik/dRWxrzjZAgHBXitzwPJ9ROPBHa9b+wlbNBP
G+iXlcUCgYBNUJiPMeLVAqTX+rUQml2cRhG6Dd35G7vqmiAL2qt/1iroUI2ps5j9
4E6vbOHb9lobY2Uuoz9DO5+PWsneQ5YYy5lg3ES3XV5r5z/jNEIMcGVmTLnonz88
qeoWKIZkLi5Aif+hRjsq0vtr7Ke5kYA2OCibzbho/0lGs8/ZDUielg==
-----END RSA PRIVATE KEY-----
`
	unrelatedKek = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyYoF9i9CnsyX/TclUphfTBj8TIb+36H8VfCQi7yG7YOf9f9m
BlocKbsEkgutC1ZhldmwtPVBxAbV2aKswBhWO2NmhpiJZBVariynNFkzCR6tQIBE
N+HVNI7fvv+GFGoaOBJVUZrqHBlYjqY/BuelHrJote7sYpUSKtKhhLz3svXNDHWc
++X5rsCBtZQH5i3vg+0QOkHTtKMHyiW6KEf7N4HjXZCu+1IZDPC89o+jSVGyRszT
aemh5kyw8E3ztiUCUYxt3xBdRsZPYEsbMIB5jYLkQTA5cFLDoc6WQdSb59afqszx
x5RwprmZBanWYXbEhSuWWjSPTPu0DqchkzxVIQIDAQABAoIBAG1nN4VEcm2xsnAK
l4AWpuSwS4VfYswTKt+cD1tLpBMa+KKZWdDo6ZDdrMV7ARy+b4rg+UPCP0kiTMQv
wockrureMrGt7CcgUHFsW/fW1BWHSZVSC7YqKYq2ZE9Sdn5uen0ltprt9Vf7ik8l
f+FHriLTxnO8lyWMtqf4XyWnTu8d948idIKVJWA9kvzyD/F4LeGu6GFAPb5QwsrL
I6oNUArfB/ELKIq3BqpbPWOSlv6TtMNvAS6FaOMUIydTeVURm5F5OQf2mFnzpLit
zQyNMqqC93ZIwNeWadP0b7CRJZ1i1k0APtMHZqTFZKGnZ990hqJboQHKIjPnhytK
/ThaAAECgYEA/GDQuYvc/ww2n/6pMWGRZGJB5+qZMCeEWO7T6nPKGl5zv9idzUfF
M9eAHzJ+qy4BxRNCH0Hy5YubhWnrrXKTZ7u/BkbGjn4gITIDu0MhepVPlPocJMb0
VFt/NHppPQBAaCy6jhxEftgL1uvXCxUmuPLrazHlm3suU7QRb2ezBQECgYEAzG5v
nOHGfVShHZDjCAAIt2MWcoOFpTsnxUhAQ7j5kxLb65BTH0F0tRKn4eBGMzgw/TYM
MjAXQ+u5eO1wQiw7puHfD0daGM+WA34bH0M+piVZk9SxA61J7kAMO5N88RPaO/vd
sCbr0t0oE+kJAtu83EJjmwdMyzEBZQ3VIYm4sCECgYEA5fwr+Qn1iA5PMRnWgQOS
hNHtkTP+CR3Zw1lQkFSYFdOA05DIrKr8kDOPs95GBCRWxIq6NNXaTUgdn0RY2qSQ
o3U5rLSOeIeDK/zx3ZJdTeIGtZH+V51eRgljMCVlBYvXJZetIZes65Jhp6cfPiA2
O1BTLEo6HKfyHaD4SndLcgECgYEAjLZ/UOb/LwlvlOBDxR/w3/nuW4g4F5FuQJcI
1RSfhSJ4Cd7fuCXf5TsgH5O1/k9xOPlYz7rWaMP6eEhG+uVjce0LEoM0ett4EJNe
q9gnaUlQLTc7WKKQvtOLF/7fAzl8/8jPwQ4pSI09pubCcxs5FgsEcJNHwpzKrvok
d99KJsECgYBQ0/XPqbYKfM5QIS+J0B6FIcrQG7Zf2tI0qeP54GMGGcviEMKdQyxY
ZVqP5LHLvBXk5W0enr/eZuPdMOpaAF4yZ1tepkIEmbsOSiFvthzwwmxJrC8jfJO6
5guMtPm+COXQjizDqszWKr77jfk/aH1jh+BDukIMgr+RK9IX5qJKAg==
-----END RSA PRIVATE KEY-----
`
)
