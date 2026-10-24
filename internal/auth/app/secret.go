package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2Params are the argon2id costs a new client secret is hashed with:
// OWASP's first recommendation, 19 MiB and two passes on one thread.
// Verifying reads the costs from the stored hash, so raising them leaves
// existing secrets working.
type argon2Params struct {
	memory  uint32 // KiB
	time    uint32
	threads uint8
}

var secretParams = argon2Params{memory: 19 * 1024, time: 2, threads: 1}

const (
	saltBytes = 16
	keyBytes  = 32
)

// hashing bounds how many hashes run at once, each holding its memory
// cost, so a burst of token requests can't exhaust memory.
var hashing = make(chan struct{}, runtime.GOMAXPROCS(0))

func argon2id(ctx context.Context, secret string, salt []byte, p argon2Params, n uint32) ([]byte, error) {
	select {
	case hashing <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-hashing }()
	return argon2.IDKey([]byte(secret), salt, p.time, p.memory, p.threads, n), nil
}

// hashSecret returns secret's argon2id hash in the PHC string format,
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>.
func hashSecret(ctx context.Context, secret string) (string, error) {
	salt := make([]byte, saltBytes)
	_, _ = rand.Read(salt)
	key, err := argon2id(ctx, secret, salt, secretParams, keyBytes)
	if err != nil {
		return "", err
	}
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, secretParams.memory, secretParams.time, secretParams.threads,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

var errBadHash = errors.New("not an argon2id hash in PHC format")

// verifySecret reports whether secret matches hash, a hashSecret result,
// comparing in constant time.
func verifySecret(ctx context.Context, secret, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, errBadHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errBadHash
	}
	var p argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil || p.time == 0 || p.threads == 0 {
		return false, errBadHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errBadHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, errBadHash
	}
	got, err := argon2id(ctx, secret, salt, p, uint32(len(want)))
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
