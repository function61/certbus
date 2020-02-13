package certificatestore

import (
	"github.com/function61/gokit/assert"
	"testing"
	"time"
)

func TestCertsDueForRenewal(t *testing.T) {
	certs, t0 := setupCommon(t)

	certsToRenew := CertsDueForRenewal(certs, t0)
	assert.Assert(t, len(certsToRenew) == 1)
	assert.EqualString(t, certsToRenew[0].Id, "dummyCertId")

	// travel backwards in time until this cert is no longer considered as must renew
	assert.Assert(t, len(CertsDueForRenewal(certs, t0.AddDate(0, 0, -9))) == 1)
	assert.Assert(t, len(CertsDueForRenewal(certs, t0.AddDate(0, 0, -10))) == 0)
}

func TestRenewAtFromExpiration(t *testing.T) {
	assert.EqualString(
		t,
		renewAtFromExpiration(time.Date(2020, 1, 31, 16, 54, 0, 0, time.UTC)).Format(time.RFC3339),
		"2019-12-31T16:54:00Z")
}
