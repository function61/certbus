// Simple wrapper + JSON struct for encrypting/decrypting short values inside pkencryptedstream
package encryptedbox

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"io"

	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/gokit/pkencryptedstream"
)

type Box struct {
	KeyFingerprint string `json:"key_fingerprint"` // .. of the encryption key that encrypted this box
	Ciphertext     []byte `json:"ciphertext"`      // gokit/pkencryptedstream
}

func New(keyFingerprint string, ciphertext []byte) *Box {
	return &Box{keyFingerprint, ciphertext}
}

func Encrypt(plaintext []byte, pubKey *rsa.PublicKey) (*Box, error) {
	pubKeyFingerprint, err := cryptoutil.Sha256FingerprintForPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	ciphertext := &bytes.Buffer{}
	encrypt, err := pkencryptedstream.Writer(ciphertext, pubKey)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(encrypt, bytes.NewReader(plaintext)); err != nil {
		return nil, err
	}

	if err := encrypt.Close(); err != nil {
		return nil, err
	}

	return &Box{pubKeyFingerprint, ciphertext.Bytes()}, nil
}

func (e *Box) Decrypt(privKey *rsa.PrivateKey, fingerprint string) ([]byte, error) {
	if e.KeyFingerprint != fingerprint {
		return nil, fmt.Errorf(
			"box was encrypted with key fingerprint %s, tried to open with %s",
			e.KeyFingerprint,
			fingerprint)
	}

	plaintextReader, err := pkencryptedstream.Reader(bytes.NewReader(e.Ciphertext), privKey)
	if err != nil {
		return nil, err
	}

	plaintext := &bytes.Buffer{}
	if _, err := io.Copy(plaintext, plaintextReader); err != nil {
		return nil, err
	}

	return plaintext.Bytes(), nil
}

func (e *Box) DecryptNoFingerprint(privKey *rsa.PrivateKey) ([]byte, error) {
	return e.Decrypt(privKey, e.KeyFingerprint)
}
