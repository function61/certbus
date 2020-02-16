package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/function61/certbus/pkg/cbdomain"
	"github.com/function61/certbus/pkg/certbus"
	"github.com/function61/certbus/pkg/certificatestore"
	"github.com/function61/certbus/pkg/encryptedbox"
	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/eventhorizon/pkg/ehreader"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/gokit/logex"
	"github.com/go-acme/lego/v3/certificate"
	"github.com/go-acme/lego/v3/lego"
	legolog "github.com/go-acme/lego/v3/log"
	"github.com/go-acme/lego/v3/providers/dns/cloudflare"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// FIXME
func init() {
	legolog.Logger = logex.Prefix("lego", logex.StandardLogger())
}

func inspect(ctx context.Context, id string) error {
	certs, err := certbus.ResolveRealtimeState(ctx, tenantClient(), nil)
	if err != nil {
		return err
	}

	cert := certs.ById(id)
	if cert == nil {
		return fmt.Errorf("cert not found: %s", id)
	}

	return jsonfile.Marshal(os.Stdout, cert)
}

func listRenewable(ctx context.Context, after time.Time, renewFirst bool) error {
	certs, err := certbus.ResolveRealtimeState(ctx, tenantClient(), nil)
	if err != nil {
		return err
	}

	for idx, cert := range certificatestore.CertsDueForRenewal(certs, after) {
		fmt.Printf("- %s %v\n", cert.RenewAt.Format(time.RFC3339), cert.Domains)

		if renewFirst && idx == 0 {
			if err := renewCertificate(ctx, cert); err != nil {
				return err
			}
		}
	}

	return nil
}

func list(ctx context.Context) error {
	certs, err := certbus.ResolveRealtimeState(ctx, tenantClient(), nil)
	if err != nil {
		return err
	}

	for _, cert := range certs.All() {
		fmt.Printf("%s %s\n", cert.Id, strings.Join(cert.Domains, ", "))
	}

	return nil
}

func remove(ctx context.Context, id string) error {
	tenantCtx := tenantClient()

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
	return tenantCtx.Client.AppendAt(
		ctx,
		certs.Version(),
		[]string{ehevent.Serialize(removed)})
}

func tenantClient() ehreader.TenantClient {
	client, err := ehreader.TenantConfigFromEnv()
	if err != nil {
		panic(err)
	}

	return client
}

func newBasicCertificate(ctx context.Context, domain string) error {
	return newCertificateInternal(ctx, []string{"www." + domain, domain}, newCertId(), "new")
}

func newSubdomainCertificate(ctx context.Context, domain string) error {
	return newCertificateInternal(ctx, []string{domain}, newCertId(), "new")
}

func newWildcardCertificate(ctx context.Context, domain string) error {
	return newCertificateInternal(ctx, []string{"*." + domain, domain}, newCertId(), "new")
}

func renewCertificate(ctx context.Context, expiringCert certificatestore.ManagedCertificate) error {
	return newCertificateInternal(ctx, expiringCert.Domains, expiringCert.Id, "renewal")
}

func newCertificateInternal(
	ctx context.Context,
	domains []string,
	certId string,
	reason string,
) error {
	tenantCtx := tenantClient()

	certs, err := certbus.ResolveRealtimeState(ctx, tenantCtx, nil)
	if err != nil {
		return err
	}

	conf, err := decryptConfig(certs)
	if err != nil {
		return fmt.Errorf("decryptConfig: %w", err)
	}

	legoClient, err := makeLegoClient(*conf)
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
	)
	if err != nil {
		return err
	}

	return tenantCtx.Client.Append(
		ctx,
		tenantCtx.Stream(certificatestore.Stream),
		[]string{ehevent.Serialize(obtained)})
}

func makeLegoClient(conf config) (*lego.Client, error) {
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

	// ugly hack, but one can only deliver these via ENV vars
	os.Setenv("CLOUDFLARE_EMAIL", conf.CloudflareCredentials.Email)
	os.Setenv("CLOUDFLARE_API_KEY", conf.CloudflareCredentials.ApiKey)

	cloudflareProvider, err := cloudflare.NewDNSProvider()
	if err != nil {
		return nil, err
	}

	if err := legoClient.Challenge.SetDNS01Provider(cloudflareProvider); err != nil {
		return nil, err
	}

	return legoClient, nil
}

func makeCertificateObtainedEvent(
	certId string,
	certAndPrivateKey certificate.Resource,
	domains []string,
	publicKey []byte,
	reason string,
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
		ehevent.MetaSystemUser(time.Now()),
	), nil
}

// CLI arguments beginning with dash are problematic (which base64 URL variant can produce),
// so we'll be nice guys and guarantee that the ID won't start with one.
func randomBase64UrlWithoutLeadingDash(length int) string {
	id := cryptorandombytes.Base64Url(length)

	if id[0] == '-' {
		// try again. the odds should exponentially decrease for recursion level to increase
		return randomBase64UrlWithoutLeadingDash(length)
	}

	return id
}

func newCertId() string {
	return randomBase64UrlWithoutLeadingDash(8)
}

func loadManagerPrivateKey() (*rsa.PrivateKey, error) {
	privKeyPem, err := ioutil.ReadFile("certbus-manager.key")
	if err != nil {
		return nil, err
	}

	return cryptoutil.ParsePemPkcs1EncodedRsaPrivateKey(privKeyPem)
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
