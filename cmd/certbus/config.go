package main

import (
	"bytes"
	"context"
	"crypto"
	"fmt"
	"github.com/function61/certbus/pkg/cbdomain"
	"github.com/function61/certbus/pkg/certbus"
	"github.com/function61/certbus/pkg/certificatestore"
	"github.com/function61/certbus/pkg/encryptedbox"
	"github.com/function61/eventhorizon/pkg/ehevent"
	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/gokit/jsonfile"
	"github.com/go-acme/lego/v3/registration"
	"io"
	"time"
)

type config struct {
	LetsEncrypt           letsEncryptAccount    `json:"lets_encrypt"`
	CloudflareCredentials cloudflareCredentials `json:"cloudflare_credentials"`
	KekPublicKey          string                `json:"kek_public_key"` // used to encrypt certs' private keys
}

func displayConfig(ctx context.Context, out io.Writer) error {
	certs, err := certbus.ResolveRealtimeState(ctx, tenantClient())
	if err != nil {
		return err
	}

	conf, err := decryptConfig(certs)
	if err != nil {
		return fmt.Errorf("decryptConfig: %w", err)
	}

	return jsonfile.Marshal(out, conf)
}

func updateConfig(ctx context.Context, confToValidate io.Reader) error {
	conf := &config{}
	if err := jsonfile.Unmarshal(confToValidate, conf, true); err != nil {
		return err
	}

	// re-marshal to JSON (so our input JSON effectively becomes validated)
	confAsJson := &bytes.Buffer{}
	if err := jsonfile.Marshal(confAsJson, conf); err != nil {
		return err
	}

	privKey, err := loadManagerPrivateKey()
	if err != nil {
		return err
	}

	confJsonEncrypted, err := encryptedbox.Encrypt(confAsJson.Bytes(), &privKey.PublicKey)
	if err != nil {
		return err
	}

	confEvent := cbdomain.NewConfigUpdated(
		confJsonEncrypted.KeyFingerprint,
		confJsonEncrypted.Ciphertext,
		ehevent.MetaSystemUser(time.Now()))

	tenantCtx := tenantClient()

	return tenantCtx.Client.Append(
		ctx,
		tenantCtx.Stream(certificatestore.Stream),
		[]string{ehevent.Serialize(confEvent)})
}

type cloudflareCredentials struct {
	Email  string `json:"email"`
	ApiKey string `json:"api_key"`
}

type letsEncryptAccount struct {
	Email        string                 `json:"email"`
	PrivateKey   string                 `json:"private_key"`
	Registration *registration.Resource `json:"registration"`
}

type letsEncryptAccountLego struct {
	letsEncryptAccount
	privateKeyParsed crypto.PrivateKey
}

func (l *letsEncryptAccount) ToLegoInterface() (*letsEncryptAccountLego, error) {
	key, err := cryptoutil.ParsePemEncodedPrivateKey([]byte(l.PrivateKey))
	if err != nil {
		return nil, err
	}

	return &letsEncryptAccountLego{
		letsEncryptAccount: *l,
		privateKeyParsed:   key,
	}, nil
}

func (l *letsEncryptAccountLego) GetEmail() string {
	return l.Email
}

func (u *letsEncryptAccountLego) GetPrivateKey() crypto.PrivateKey {
	return u.privateKeyParsed
}

func (l *letsEncryptAccountLego) GetRegistration() *registration.Resource {
	return l.Registration
}

func (l *letsEncryptAccountLego) SetRegistration(reg *registration.Resource) {
	l.Registration = reg
}
