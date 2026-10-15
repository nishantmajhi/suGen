// Package randutil provides small cryptographically secure random helpers
// shared by the password and passphrase generators.
package randutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Intn returns a cryptographically secure random int in [0, n).
// It mirrors the semantics of PHP's random_int(0, n-1).
func Intn(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("randutil: n must be positive, got %d", n)
	}
	max := big.NewInt(int64(n))
	v, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, fmt.Errorf("randutil: failed to generate random number: %w", err)
	}
	return int(v.Int64()), nil
}

// Digit returns a random ASCII digit rune in the range '0'-'9'.
func Digit() (byte, error) {
	n, err := Intn(10)
	if err != nil {
		return 0, err
	}
	return byte('0' + n), nil
}

// Shuffle randomly permutes the bytes of b in place using a
// Fisher-Yates shuffle backed by a CSPRNG.
func Shuffle(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := Intn(i + 1)
		if err != nil {
			return err
		}
		b[i], b[j] = b[j], b[i]
	}
	return nil
}
