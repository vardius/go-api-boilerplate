// Package keys holds the auth service's signing keys and signs its tokens.
//
// The service signs with one ES256 key and publishes its public half, with
// the public keys of the keys it signed with before, as a JWKS, so tokens
// signed before a rotation keep verifying until they expire. Every key's
// kid is its RFC 7638 thumbprint, so a kid names exactly one key.
//
// A Signer issues RFC 9068 access tokens with those keys, and a
// ServiceToken is the short-lived token the service presents to the user
// service to redeem login codes. That token's scope, login-codes:redeem, is
// the one scope Signer.Sign refuses: nothing else can ask for it.
package keys

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
)

// Config says where the keys are. The service embeds it without a prefix.
type Config struct {
	SigningKeyFile  string `env:"SIGNING_KEY_FILE" desc:"Path of the ES256 (P-256) private key tokens are signed with, as a PKCS#8 PEM. Required unless APP_ENV is development, which then signs with a key generated at start."`
	PreviousKeysDir string `env:"PREVIOUS_KEYS_DIR" desc:"Directory of public keys (*.pem, PKIX) the service signed with before, still published in the JWKS so their tokens verify until they expire."`
}

// Keys is the signing key and every public key the JWKS publishes.
type Keys struct {
	signing *ecdsa.PrivateKey
	kid     string
	jwks    []byte
}

// Load reads the keys cfg names. Without a SigningKeyFile it generates an
// ephemeral key and logs a warning when ephemeral is allowed, as in
// development, and fails otherwise: tokens signed with a key that dies with
// the process stop verifying at the next restart.
func Load(cfg Config, ephemeral bool, log *slog.Logger) (*Keys, error) {
	var signing *ecdsa.PrivateKey
	switch {
	case cfg.SigningKeyFile != "":
		key, err := readPrivateKey(cfg.SigningKeyFile)
		if err != nil {
			return nil, err
		}
		signing = key
	case ephemeral:
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("keys: generate a signing key: %w", err)
		}
		log.Warn("no SIGNING_KEY_FILE: signing with a key generated at start; tokens stop verifying when the process restarts")
		signing = key
	default:
		return nil, errors.New("keys: SIGNING_KEY_FILE is required outside development")
	}

	var previous []*ecdsa.PublicKey
	if cfg.PreviousKeysDir != "" {
		keys, err := readPublicKeys(cfg.PreviousKeysDir)
		if err != nil {
			return nil, err
		}
		previous = keys
	}
	return New(signing, previous...)
}

// New returns Keys that sign with signing and also publish previous.
func New(signing *ecdsa.PrivateKey, previous ...*ecdsa.PublicKey) (*Keys, error) {
	if signing == nil || signing.Curve != elliptic.P256() {
		return nil, errors.New("keys: the signing key must be a P-256 key")
	}
	type jwk struct {
		Kty string `json:"kty"`
		Crv string `json:"crv"`
		X   string `json:"x"`
		Y   string `json:"y"`
		Kid string `json:"kid"`
		Use string `json:"use"`
		Alg string `json:"alg"`
	}
	var set struct {
		Keys []jwk `json:"keys"`
	}
	var kid string
	for i, pub := range append([]*ecdsa.PublicKey{&signing.PublicKey}, previous...) {
		x, y, err := coordinates(pub)
		if err != nil {
			return nil, err
		}
		k := jwk{Kty: "EC", Crv: "P-256", X: x, Y: y, Kid: thumbprint(x, y), Use: "sig", Alg: "ES256"}
		if i == 0 {
			kid = k.Kid
		}
		// A previous key that is the signing key, or listed twice, is
		// published once.
		if slices.ContainsFunc(set.Keys, func(o jwk) bool { return o.Kid == k.Kid }) {
			continue
		}
		set.Keys = append(set.Keys, k)
	}
	body, err := json.Marshal(set)
	if err != nil {
		return nil, fmt.Errorf("keys: encode the jwks: %w", err)
	}
	return &Keys{signing: signing, kid: kid, jwks: body}, nil
}

// KID returns the kid of the signing key.
func (k *Keys) KID() string { return k.kid }

// JWKS serves the public keys as a JSON Web Key Set. Verifiers refetch it
// when they meet a kid they don't know, so a short cache lifetime is enough
// to spare the service while letting a rotation show up within a minute.
func (k *Keys) JWKS() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = w.Write(k.jwks)
	})
}

func readPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("keys: read the signing key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("keys: %s is not a PKCS#8 PEM (-----BEGIN PRIVATE KEY-----)", path)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("keys: parse the signing key %s: %w", path, err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || key.Curve != elliptic.P256() {
		return nil, fmt.Errorf("keys: %s is not a P-256 key, which ES256 needs", path)
	}
	return key, nil
}

// readPublicKeys reads every *.pem in dir, in name order.
func readPublicKeys(dir string) ([]*ecdsa.PublicKey, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("keys: previous keys: %w", err)
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.pem"))
	if err != nil {
		return nil, fmt.Errorf("keys: previous keys: %w", err)
	}
	keys := make([]*ecdsa.PublicKey, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("keys: read a previous key: %w", err)
		}
		block, _ := pem.Decode(data)
		if block == nil || block.Type != "PUBLIC KEY" {
			return nil, fmt.Errorf("keys: previous key %s is not a public key PEM (-----BEGIN PUBLIC KEY-----)", path)
		}
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("keys: parse the previous key %s: %w", path, err)
		}
		key, ok := parsed.(*ecdsa.PublicKey)
		if !ok || key.Curve != elliptic.P256() {
			return nil, fmt.Errorf("keys: previous key %s is not a P-256 key", path)
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// coordinates returns a P-256 key's x and y, base64url-encoded at their
// full 32 bytes as RFC 7518 §6.2.1 requires.
func coordinates(pub *ecdsa.PublicKey) (x, y string, err error) {
	if pub == nil || pub.Curve != elliptic.P256() {
		return "", "", errors.New("keys: every key must be a P-256 key")
	}
	b, err := pub.Bytes() // 0x04 || X || Y
	if err != nil {
		return "", "", fmt.Errorf("keys: encode a public key: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[1:33]), base64.RawURLEncoding.EncodeToString(b[33:]), nil
}

// thumbprint is the RFC 7638 thumbprint of an EC key: the SHA-256 of its
// required members, crv, kty, x and y, in that order with no whitespace.
func thumbprint(x, y string) string {
	var b bytes.Buffer
	b.WriteString(`{"crv":"P-256","kty":"EC","x":"`)
	b.WriteString(x)
	b.WriteString(`","y":"`)
	b.WriteString(y)
	b.WriteString(`"}`)
	sum := sha256.Sum256(b.Bytes())
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// jwksTransport answers every request with the JWKS, so the service's own
// authn.Verifier reads the keys from memory rather than over the network.
type jwksTransport struct {
	jwks []byte
}

func (t jwksTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		_ = r.Body.Close()
	}
	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": {"application/json"}},
		Body:          io.NopCloser(bytes.NewReader(t.jwks)),
		ContentLength: int64(len(t.jwks)),
		Request:       r,
	}, nil
}
