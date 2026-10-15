package generator

import (
	"errors"
	"fmt"

	"github.com/nishantmajhi/suGen/internal/randutil"
)

// Exported charsets, useful both internally and for callers/tests that
// need to reason about what characters a generated password can contain.
const (
	LowerCaseChars = "abcdefghijklmnopqrstuvwxyz"
	UpperCaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	DigitChars     = "0123456789"
	SpecialChars   = "~!@#$%^&-=_+?"
)

// AmbiguousChars is the set of characters considered easy to misread
// (uppercase I, lowercase l, uppercase O, and zero) and therefore
// filtered out when AvoidAmbiguousCharacters is set.
var AmbiguousChars = map[byte]bool{
	'I': true,
	'l': true,
	'O': true,
	'0': true,
}

// MaxPasswordLength is the longest password GeneratePassword will build.
// The length arrives from an unauthenticated query string, so it has to be
// bounded before anything is allocated for it.
const MaxPasswordLength = 1024

// ErrPasswordTooShort is returned when the requested password length
// cannot satisfy the requested minimum digit/special character counts.
var ErrPasswordTooShort = errors.New("password length too short for the specified minimum requirements")

// ErrPasswordTooLong is returned when the requested length exceeds
// MaxPasswordLength.
var ErrPasswordTooLong = fmt.Errorf("password length exceeds the maximum of %d", MaxPasswordLength)

// PasswordOptions controls how GeneratePassword builds a password. Field
// semantics and defaults intentionally mirror the original PHP endpoint.
type PasswordOptions struct {
	MixedCase                bool
	Length                   int
	Numbers                  bool
	MinimumDigits            int
	SpecialCharacters        bool
	MinimumSpecialCharacters int
	AvoidAmbiguousCharacters bool
}

// GeneratePassword produces a random password according to opts, using a
// cryptographically secure random source throughout.
func GeneratePassword(opts PasswordOptions) (string, error) {
	if opts.Length > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	// Every password gets one lowercase letter, and one uppercase letter
	// with MixedCase, before anything else. A minimum only applies when its
	// character class is switched on, and a negative one means none.
	letters := 1
	if opts.MixedCase {
		letters = 2
	}
	minDigits := 0
	if opts.Numbers {
		minDigits = max(opts.MinimumDigits, 0)
	}
	minSpecials := 0
	if opts.SpecialCharacters {
		minSpecials = max(opts.MinimumSpecialCharacters, 0)
	}
	// Subtract rather than add: two huge minimums would overflow a sum.
	if opts.Length < letters ||
		opts.Length-letters < minDigits ||
		opts.Length-letters-minDigits < minSpecials {
		return "", ErrPasswordTooShort
	}

	lowerSource := LowerCaseChars
	upperSource := UpperCaseChars
	digitSource := DigitChars
	if opts.AvoidAmbiguousCharacters {
		lowerSource = string(filterAmbiguous([]byte(lowerSource)))
		upperSource = string(filterAmbiguous([]byte(upperSource)))
		digitSource = string(filterAmbiguous([]byte(digitSource)))
	}

	seed := []byte(lowerSource)

	sprout := make([]byte, 0, opts.Length)
	lowerByte, err := randomByte(lowerSource)
	if err != nil {
		return "", err
	}
	sprout = append(sprout, lowerByte)

	if opts.MixedCase {
		seed = append(seed, upperSource...)
		upperByte, err := randomByte(upperSource)
		if err != nil {
			return "", err
		}
		sprout = append(sprout, upperByte)
	}
	if opts.Numbers {
		seed = append(seed, DigitChars...)
	}
	if opts.SpecialCharacters {
		seed = append(seed, SpecialChars...)
	}
	if opts.AvoidAmbiguousCharacters {
		seed = filterAmbiguous(seed)
	}

	nmCoat, err := randomChars(digitSource, minDigits)
	if err != nil {
		return "", err
	}
	spCoat, err := randomChars(SpecialChars, minSpecials)
	if err != nil {
		return "", err
	}

	coreLength := opts.Length - letters - minDigits - minSpecials

	core, err := randomChars(string(seed), coreLength)
	if err != nil {
		return "", err
	}

	sprout = append(sprout, []byte(nmCoat)...)
	sprout = append(sprout, []byte(spCoat)...)
	sprout = append(sprout, []byte(core)...)

	if err := randutil.Shuffle(sprout); err != nil {
		return "", fmt.Errorf("generator: failed to shuffle password: %w", err)
	}

	return string(sprout), nil
}

func filterAmbiguous(seed []byte) []byte {
	filtered := make([]byte, 0, len(seed))
	for _, c := range seed {
		if !AmbiguousChars[c] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func randomByte(charset string) (byte, error) {
	idx, err := randutil.Intn(len(charset))
	if err != nil {
		return 0, fmt.Errorf("generator: %w", err)
	}
	return charset[idx], nil
}

// randomChars returns n characters picked (with replacement) at random
// from charset. It returns an empty string for n <= 0, matching the
// original PHP loop's behavior for non-positive lengths.
func randomChars(charset string, n int) (string, error) {
	if n <= 0 || len(charset) == 0 {
		return "", nil
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		idx, err := randutil.Intn(len(charset))
		if err != nil {
			return "", fmt.Errorf("generator: %w", err)
		}
		out[i] = charset[idx]
	}
	return string(out), nil
}
