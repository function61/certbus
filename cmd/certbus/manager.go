package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/function61/certbus/pkg/cbdomain"
	"github.com/function61/certbus/pkg/certbus"
	"github.com/function61/certbus/pkg/certificatestore"
	"github.com/function61/certbus/pkg/encryptedbox"
	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/eventhorizon/pkg/ehreader"
	"github.com/function61/gokit/aws/s3facade"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/gokit/logex"
	"github.com/function61/lambda-alertmanager/pkg/alertmanagerclient"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	legolog "github.com/go-acme/lego/v4/log"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/scylladb/termtables"
)

// FIXME
func init() {
	legolog.Logger = logex.Prefix("lego", logex.StandardLogger())
}

func inspect(ctx context.Context, id string) error {
	certs, err := certbus.ResolveRealtimeState(ctx, readTenantCtx(), nil)
	if err != nil {
		return err
	}

	cert := certs.ById(id)
	if cert == nil {
		return fmt.Errorf("cert not found: %s", id)
	}

	return jsonfile.Marshal(os.Stdout, cert)
}

func renew(ctx context.Context, id string) error {
	certs, err := certbus.ResolveRealtimeState(ctx, readTenantCtx(), nil)
	if err != nil {
		return err
	}

	cert := certs.ById(id)
	if cert == nil {
		return fmt.Errorf("cert not found: %s", id)
	}

	return renewCertificate(ctx, *cert)
}

func listRenewable(ctx context.Context, after time.Time, renewFirst bool) error {
	tenantCtx := readTenantCtx()

	certs, err := certbus.ResolveRealtimeState(ctx, tenantCtx, nil)
	if err != nil {
		return err
	}

	tbl := termtables.CreateTable()
	tbl.AddHeaders("Id", "RenewAt", "Domains")

	for idx, cert := range certificatestore.CertsDueForRenewal(certs, after) {
		tbl.AddRow(
			cert.Id,
			cert.RenewAt.Format(time.RFC3339),
			strings.Join(cert.Domains, ", "))

		if renewFirst && idx == 0 {
			if err := renewCertificate(ctx, cert); err != nil {
				return err
			}
		}
	}

	if renewFirst {
		conf, err := decryptConfig(certs)
		if err != nil {
			return fmt.Errorf("decryptConfig: %w", err)
		}

		if conf.AlertManagerBaseurl != "" {
			if err := alertmanagerclient.New(conf.AlertManagerBaseurl).DeadMansSwitchCheckin(
				ctx,
				"CertBus "+tenantCtx.Stream(certificatestore.Stream),
				48*time.Hour,
			); err != nil {
				return err
			}
		}
	}

	fmt.Println(tbl.Render())

	return nil
}

func list(ctx context.Context) error {
	certs, err := certbus.ResolveRealtimeState(ctx, readTenantCtx(), nil)
	if err != nil {
		return err
	}

	tbl := termtables.CreateTable()
	tbl.AddHeaders("Id", "Expires", "Challenge", "Domains")

	for _, cert := range certs.All() {
		tbl.AddRow(
			cert.Id,
			cert.Certificate.NotAfter.Format(time.RFC3339),
			cert.ChallengeType,
			strings.Join(cert.Domains, ", "))
	}

	fmt.Println(tbl.Render())

	return nil
}

func remove(ctx context.Context, id string) error {
	tenantCtx := readTenantCtx()

	certs, err := certbus.ResolveRealtimeState(ctx, tenantCtx, nil)
	if err != nil {
		return err
	}

	if certs.ById(id) == nil {
		return fmt.Errorf("cert to remove not found by id: %s", id)
	}

	removed := cbdomain.NewCertificateRemoved(
		id,
		ehevent.MetaSystemUser(time.Now()))

	// this uses optimistic locking
	// TODO: retry logic
	_, err = tenantCtx.Client.AppendAfter(
		ctx,
		certs.Version(),
		[]string{ehevent.Serialize(removed)})
	return err
}

func readTenantCtx() ehreader.TenantCtx {
	client, err := ehreader.TenantCtxFrom(ehreader.ConfigFromEnv)
	if err != nil {
		panic(err)
	}

	return *client
}

func newBasicCertificate(ctx context.Context, domain string, challengeType challenge.Type) error {
	return newCertificateInternal(
		ctx,
		[]string{"www." + domain, domain},
		newCertId(),
		"new",
		challengeType)
}

func newSubdomainCertificate(ctx context.Context, domain string, challengeType challenge.Type) error {
	return newCertificateInternal(
		ctx,
		[]string{domain},
		newCertId(),
		"new",
		challengeType)
}

func newWildcardCertificate(ctx context.Context, domain string, challengeType challenge.Type) error {
	return newCertificateInternal(
		ctx,
		[]string{"*." + domain, domain},
		newCertId(),
		"new",
		challengeType)
}

func renewCertificate(ctx context.Context, expiringCert certificatestore.ManagedCertificate) error {
	// we need to renew the cert using the same challenge type that we used before with this certificate
	challengeType, err := func() (challenge.Type, error) {
		switch expiringCert.ChallengeType {
		case challenge.DNS01.String(), "": // old events didn't record challenge type
			return challenge.DNS01, nil
		case challenge.HTTP01.String():
			return challenge.HTTP01, nil
		default:
			return "", fmt.Errorf("unknown challengeType: %s", expiringCert.ChallengeType)
		}
	}()
	if err != nil {
		return err
	}

	return newCertificateInternal(
		ctx,
		expiringCert.Domains,
		expiringCert.Id,
		"renewal",
		challengeType)
}

func newCertificateInternal(
	ctx context.Context,
	domains []string,
	certId string,
	reason string,
	challengeType challenge.Type,
) error {
	tenantCtx := readTenantCtx()

	certs, err := certbus.ResolveRealtimeState(ctx, tenantCtx, nil)
	if err != nil {
		return err
	}

	conf, err := decryptConfig(certs)
	if err != nil {
		return fmt.Errorf("decryptConfig: %w", err)
	}

	legoClient, err := makeLegoClient(*conf, challengeType)
	if err != nil {
		return err
	}

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	resp, err := legoClient.Certificate.Obtain(request)
	if err != nil {
		return err
	}

	obtained, err := makeCertificateObtainedEvent(
		certId,
		*resp,
		domains,
		[]byte(conf.KekPublicKey),
		reason,
		challengeType,
	)
	if err != nil {
		return err
	}

	_, err = tenantCtx.Client.Append(
		ctx,
		tenantCtx.Stream(certificatestore.Stream),
		[]string{ehevent.Serialize(obtained)})
	return err
}

func makeLegoClient(conf config, challengeType challenge.Type) (*lego.Client, error) {
	adapter, err := conf.LetsEncrypt.ToLegoInterface()
	if err != nil {
		return nil, err
	}

	legoClient, err := lego.NewClient(lego.NewConfig(adapter))
	if err != nil {
		return nil, err
	}

	if adapter.GetRegistration() == nil {
		// one could be obtained with:
		//     legoClient.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		return nil, errors.New("LetsEncrypt registration empty")
	}

	switch challengeType {
	case challenge.DNS01:
		cfConf := cloudflare.NewDefaultConfig() // sets important fields (like TTL)
		cfConf.AuthEmail = conf.CloudflareCredentials.Email
		cfConf.AuthKey = conf.CloudflareCredentials.ApiKey

		cloudflareProvider, err := cloudflare.NewDNSProviderConfig(cfConf)
		if err != nil {
			return nil, err
		}

		if err := legoClient.Challenge.SetDNS01Provider(cloudflareProvider); err != nil {
			return nil, err
		}

		return legoClient, nil
	case challenge.HTTP01:
		if conf.AcmeHTTP01Challenges == nil {
			return nil, errors.New("cannot use HTTP-01 due to missing configuration")
		}

		validationsBucket, err := s3facade.Bucket(
			conf.AcmeHTTP01Challenges.Bucket,
			nil, // use AWS-SDK built-in credentials resolving so this works with Lambda roles
			conf.AcmeHTTP01Challenges.Region)
		if err != nil {
			return nil, err
		}

		if err := legoClient.Challenge.SetHTTP01Provider(&bucketChallengeUploader{validationsBucket}); err != nil {
			return nil, err
		}

		return legoClient, nil
	default:
		return nil, fmt.Errorf("unimplemented challenge: %s", challengeType)
	}
}

func makeCertificateObtainedEvent(
	certId string,
	certAndPrivateKey certificate.Resource,
	domains []string,
	publicKey []byte,
	reason string,
	challengeType challenge.Type,
) (*cbdomain.CertificateObtained, error) {
	certParsed, err := cryptoutil.ParsePemX509Certificate(certAndPrivateKey.Certificate)
	if err != nil {
		return nil, err
	}

	pubKey, err := cryptoutil.ParsePemPkcs1EncodedRsaPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	privateKeyEncrypted, err := encryptedbox.Encrypt(certAndPrivateKey.PrivateKey, pubKey)
	if err != nil {
		return nil, err
	}

	return cbdomain.NewCertificateObtained(
		certId,
		reason,
		domains,
		certParsed.NotAfter,
		string(certAndPrivateKey.Certificate),
		privateKeyEncrypted.KeyFingerprint,
		privateKeyEncrypted.Ciphertext,
		challengeType.String(),
		ehevent.MetaSystemUser(time.Now()),
	), nil
}

func newCertId() string {
	return cryptorandombytes.Base64UrlWithoutLeadingDash(8)
}

func loadManagerPrivateKey() (*rsa.PrivateKey, error) {
	// cannot just pass this base64 encoded because we're hitting Lambda limits:
	//   https://twitter.com/joonas_fi/status/1235122048340357120
	fromEnvVar := strings.ReplaceAll(os.Getenv("CERTBUS_MANAGER_KEY"), `\n`, "\n")
	if fromEnvVar != "" {
		return cryptoutil.ParsePemPkcs1EncodedRsaPrivateKey([]byte(fromEnvVar))
	} else {
		privKeyPem, err := ioutil.ReadFile("certbus-manager.key")
		if err != nil {
			return nil, err
		}

		return cryptoutil.ParsePemPkcs1EncodedRsaPrivateKey(privKeyPem)
	}
}

func decryptConfig(certs *certificatestore.Store) (*config, error) {
	privKey, err := loadManagerPrivateKey()
	if err != nil {
		return nil, err
	}

	encryptedConf := certs.GetLatestEncryptedConfig()
	if encryptedConf == nil {
		return nil, errors.New("no config found")
	}

	plaintextJson, err := encryptedbox.New(
		encryptedConf.ConfigEncryptionKeyFingerprint,
		encryptedConf.ConfigCiphertext).DecryptNoFingerprint(privKey)
	if err != nil {
		return nil, err
	}

	conf := &config{}
	return conf, jsonfile.Unmarshal(bytes.NewReader(plaintextJson), conf, true)
}
