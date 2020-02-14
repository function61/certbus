package certificatestore

import (
	"context"
	"encoding/base64"
	"github.com/function61/certbus/pkg/cbdomain"
	"github.com/function61/eventhorizon/pkg/ehclient"
	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/eventhorizon/pkg/ehreader"
	"github.com/function61/gokit/assert"
	"testing"
	"time"
)

func TestByHostname(t *testing.T) {
	certs, _ := setupCommon(t)

	assert.Assert(t, len(certs.All()) == 1)

	assert.EqualString(t, certs.ById("dummyCertId").Certificate.CertPemBundle, exampleCert)

	assert.Assert(t, certs.ById("notFound") == nil)

	// find by exact match
	assert.EqualString(t, certs.ByHostname("prod4.fn61.net").Id, "dummyCertId")

	// stores don't directly support wildcards
	assert.Assert(t, certs.ByHostname("foo.prod4.fn61.net") == nil)

	// but we have helper that does
	assert.EqualString(t, ByHostnameSupportingWildcard("foo.prod4.fn61.net", certs).Id, "dummyCertId")
	assert.EqualString(t, ByHostnameSupportingWildcard("prod4.fn61.net", certs).Id, "dummyCertId")

	// wildcard should not match sub-sub domain
	assert.Assert(t, ByHostnameSupportingWildcard("foo.bar.prod4.fn61.net", certs) == nil)

	// these should not panic
	assert.Assert(t, ByHostnameSupportingWildcard("foo", certs) == nil)
	assert.Assert(t, ByHostnameSupportingWildcard("", certs) == nil)
}

func TestCertificateRemoval(t *testing.T) {
	certs, t0 := setupCommon(t)

	assert.Assert(t, len(certs.All()) == 1)

	assert.Assert(t, certs.ByHostname("prod4.fn61.net") != nil)

	pumpEvents(t, certs, cbdomain.NewCertificateRemoved(
		"dummyCertId",
		ehevent.MetaSystemUser(t0)))

	assert.Assert(t, len(certs.All()) == 0)
	assert.Assert(t, certs.ByHostname("prod4.fn61.net") == nil)
}

func TestGetLatestEncryptedConfig(t *testing.T) {
	certs, _ := setupCommon(t)

	assert.EqualString(
		t,
		certs.GetLatestEncryptedConfig().ConfigEncryptionKeyFingerprint,
		"encryptionKeyFingerprint")
}

func setupCommon(t *testing.T) (*Store, time.Time) {
	certs := New(ehreader.TenantId("dummyTenant"), nil)

	t0 := time.Date(2020, 1, 31, 16, 54, 0, 0, time.UTC)

	// unwrap base64 to raw bytes
	exampleCertPrivateKeyEncryptedWithExampleKek, err := base64.StdEncoding.DecodeString(
		exampleCertPrivateKeyEncryptedWithExampleKekBase64)
	assert.Ok(t, err)

	// our dummy cert expires in 21 days, therefore its renewal day is ~9 days in the past
	// NOTE: dummy cert actual hostnames and hostnames are not the same (I was lazy)
	pumpEvents(t, certs,
		cbdomain.NewCertificateObtained(
			"dummyCertId",
			"new",
			[]string{"*.prod4.fn61.net", "prod4.fn61.net"},
			t0.AddDate(0, 0, 21),
			exampleCert,
			"SHA256:wupoCrsM0GYWNWLwcBEDZZSe4ToLaxcuCWAgOiTsFCA", // of exampleCertsKek
			exampleCertPrivateKeyEncryptedWithExampleKek,
			ehevent.MetaSystemUser(t0)),
		cbdomain.NewConfigUpdated(
			"encryptionKeyFingerprint",
			[]byte{0x00, 0x01}, // dummy data (not really encryptedbox as there really would be)
			ehevent.MetaSystemUser(t0)),
	)

	return certs, t0
}

func pumpEvents(t *testing.T, processor ehreader.EventsProcessor, events ...ehevent.Event) {
	gotCallback := false

	assert.Ok(t, processor.ProcessEvents(context.TODO(), func(
		versionInDb ehclient.Cursor,
		handleEvent func(ehevent.Event) error,
		commit func(ehclient.Cursor) error,
	) error {
		gotCallback = true

		for _, event := range events {
			if err := handleEvent(event); err != nil {
				return err
			}
		}

		return commit(versionInDb.Next())
	}))

	assert.Assert(t, gotCallback)
}

const (
	exampleCert = `-----BEGIN CERTIFICATE-----
MIIEHDCCAoSgAwIBAgIQbyK0y1bhFzShdP+Wh5gKsTANBgkqhkiG9w0BAQsFADBf
MR4wHAYDVQQKExVta2NlcnQgZGV2ZWxvcG1lbnQgQ0ExGjAYBgNVBAsMEXJvb3RA
MzU1YTY4YjM5MTI5MSEwHwYDVQQDDBhta2NlcnQgcm9vdEAzNTVhNjhiMzkxMjkw
HhcNMTkwNjAxMDAwMDAwWhcNMzAwMjA4MTg0NDM2WjBFMScwJQYDVQQKEx5ta2Nl
cnQgZGV2ZWxvcG1lbnQgY2VydGlmaWNhdGUxGjAYBgNVBAsMEXJvb3RAMzU1YTY4
YjM5MTI5MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvPcc75ddkjI8
4VibjPczJ+lPtgxT5KY/v+uY9dzzc1ApkYj0Ny58RjIJb3VJl7Z53kJFBrATmesu
p3uTWZzxCiP5zO+bekDEXmFWGKt3wvE5Qwu+t9RFeUPqWkjKeyhQmjo2mmCzZ3L/
dhcVj+UwqtJcDl16B2u0YxNNbdvJGjoJ7teYAGH6pKmmld17z5+Bcqsi16IKk/3q
4JNDkHcBMMhR2P6oRwqVDuDDyaCABGKw3P9izY4GPwJZCkNMnVGJKvOp+eBqTEe9
0WjwdGg5TuB4/rWKOUGZHGhd9bGpOcgzcrmjDzOjbs3T/fjlwNWWSY/sLld6aMQg
aggT2+0WVQIDAQABo24wbDAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYIKwYB
BQUHAwEwDAYDVR0TAQH/BAIwADAfBgNVHSMEGDAWgBTU5eodMALrQqetKOuVAR2l
UxEImDAWBgNVHREEDzANggtleGFtcGxlLm9yZzANBgkqhkiG9w0BAQsFAAOCAYEA
PBs1EUmGs8pwkHzWCik1GPC3NK6eSNb5zyuyPoeXTourjDgKeSgz9xnujUernum4
MtWbr7Jg0VUN373UBT5c7Ty+8e0l5ODyWiXQUZ+YlHunPK8vlKe+pvBkltwCW7De
zsmtu/96n1MVU/5EKpntpf5F4IlR4JC+x6egt5XVDXd1fFKtYGwzPa6PvaqYDqq5
DZBsurnPjcIiH5ozc+7rVg9noVCWEMpZktQq/9kgqgbwmaAYk3CPOZOLEQw8tSHh
nI90+hE7La1jEyqwjKFwE0zR0etmoJYSLLAAZCPNL0WphEpqoxLkLm6qNmiRG8pe
OCeqDwSJgmgtbJ+66Rx3Q+OGZ4LvFcYE/Yu33sT39WenQDOKFKxlg1PTo2Ec0sBZ
mZ/xgSajyEYzK9eazqo0z2zYsnxQkrQOVYvpCMuoQGW+biWrWc667yC3T4zyb480
swNhxeADXfBfXzZdY1wYDD7Bmr4SP7F18ktS2/aNf6HweIA3+YYFfLsLnvFO/w+j
-----END CERTIFICATE-----
`
	exampleCertPrivateKeyEncryptedWithExampleKekBase64 = `
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
LV5TsvF9fwT/8BL4aCUYdRQG4adDJhB5Qu7vfn4=`
)
