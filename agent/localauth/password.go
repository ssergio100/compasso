// Package localauth verifies the responsible person's local password and
// grants durable, idempotent bonus events.
package localauth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params records all work factors included in the verifier.
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params are suitable for the responsible person's local secret.
var DefaultArgon2Params = Argon2Params{
	Memory: 64 * 1024, Iterations: 3, Parallelism: 2, SaltLength: 16, KeyLength: 32,
}

// HashPassword creates a standard PHC-style Argon2id verifier. It is used by
// the future server when changing the local password and by tests in phase 6.
func HashPassword(password string, params Argon2Params) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}
	if err := validateParams(params); err != nil {
		return "", err
	}
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, params.Memory, params.Iterations, params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword compares a password with a stored Argon2id verifier.
func VerifyPassword(password, verifier string) (bool, error) {
	parts := strings.Split(verifier, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, errors.New("invalid Argon2id verifier format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errors.New("unsupported Argon2id version")
	}
	if parts[2] != fmt.Sprintf("v=%d", version) {
		return false, errors.New("invalid Argon2id version field")
	}
	params := Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return false, errors.New("invalid Argon2id parameters")
	}
	if parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", params.Memory, params.Iterations, params.Parallelism) {
		return false, errors.New("invalid Argon2id parameter field")
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false, errors.New("invalid Argon2id salt")
	}
	expected, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false, errors.New("invalid Argon2id hash")
	}
	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(expected))
	if err := validateParams(params); err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func validateParams(params Argon2Params) error {
	if params.Memory < 8*1024 || params.Memory > 256*1024 {
		return errors.New("Argon2id memory must be between 8 MiB and 256 MiB")
	}
	if params.Iterations == 0 || params.Iterations > 10 {
		return errors.New("Argon2id iterations must be between 1 and 10")
	}
	if params.Parallelism == 0 || params.Parallelism > 8 {
		return errors.New("Argon2id parallelism must be between 1 and 8")
	}
	if params.SaltLength < 8 || params.SaltLength > 64 || params.KeyLength < 16 || params.KeyLength > 64 {
		return errors.New("Argon2id salt or key length is outside safe limits")
	}
	return nil
}
