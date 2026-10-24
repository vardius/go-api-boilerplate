package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSecretHash(t *testing.T) {
	hash, err := hashSecret(t.Context(), "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=19456,t=2,p=1" || len(parts[4]) != 22 || len(parts[5]) != 43 {
		t.Fatalf("hash = %q", hash)
	}
	if again, _ := hashSecret(t.Context(), "s3cret"); again == hash {
		t.Fatal("two hashes of one secret are equal: no salt")
	}
	for secret, want := range map[string]bool{"s3cret": true, "s3cret ": false, "": false} {
		if ok, err := verifySecret(t.Context(), secret, hash); err != nil || ok != want {
			t.Errorf("verify %q = %v, %v; want %v", secret, ok, err, want)
		}
	}
}

func TestVerifySecretReadsTheCostsFromTheHash(t *testing.T) {
	saved := secretParams
	secretParams = argon2Params{memory: 64, time: 1, threads: 1}
	hash, err := hashSecret(t.Context(), "pw")
	secretParams = saved
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(hash, "$m=64,t=1,p=1$") {
		t.Fatalf("hash = %q", hash)
	}
	if ok, err := verifySecret(t.Context(), "pw", hash); err != nil || !ok {
		t.Fatalf("verify = %v, %v", ok, err)
	}
}

func TestVerifySecretRefusesWhatIsNotAHash(t *testing.T) {
	const salt, key = "c2FsdHNhbHRzYWx0c2FsdA", "a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2U"
	for _, hash := range []string{
		"",
		"s3cret",
		"$argon2i$v=19$m=64,t=1,p=1$" + salt + "$" + key,
		"$argon2id$v=16$m=64,t=1,p=1$" + salt + "$" + key,
		"$argon2id$v=x$m=64,t=1,p=1$" + salt + "$" + key,
		"$argon2id$v=19$m=64,t=0,p=1$" + salt + "$" + key,
		"$argon2id$v=19$m=64,t=1,p=0$" + salt + "$" + key,
		"$argon2id$v=19$m=64$" + salt + "$" + key,
		"$argon2id$v=19$m=64,t=1,p=1$!!$" + key,
		"$argon2id$v=19$m=64,t=1,p=1$" + salt + "$!!",
		"$argon2id$v=19$m=64,t=1,p=1$" + salt + "$",
		"x$argon2id$v=19$m=64,t=1,p=1$" + salt + "$" + key,
	} {
		if ok, err := verifySecret(t.Context(), "pw", hash); ok || !errors.Is(err, errBadHash) {
			t.Errorf("verify against %q = %v, %v; want errBadHash", hash, ok, err)
		}
	}
}

func TestHashingWaitsForASlot(t *testing.T) {
	for range cap(hashing) {
		hashing <- struct{}{}
	}
	defer func() {
		for range cap(hashing) {
			<-hashing
		}
	}()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := hashSecret(ctx, "pw"); !errors.Is(err, context.Canceled) {
		t.Fatalf("hash with every slot taken = %v, want context.Canceled", err)
	}
	if _, err := verifySecret(ctx, "pw", "$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5"); !errors.Is(err, context.Canceled) {
		t.Fatalf("verify with every slot taken = %v, want context.Canceled", err)
	}
}
