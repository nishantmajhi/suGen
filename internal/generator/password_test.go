package generator_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/nishantmajhi/suGen/internal/generator"
)

func TestGeneratePassword_DefaultLikeOptions(t *testing.T) {
	opts := generator.PasswordOptions{
		MixedCase:                true,
		Length:                   12,
		Numbers:                  true,
		SpecialCharacters:        true,
		AvoidAmbiguousCharacters: true,
	}
	pw, err := generator.GeneratePassword(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) != opts.Length {
		t.Fatalf("expected length %d, got %d (%q)", opts.Length, len(pw), pw)
	}
}

func TestGeneratePassword_TooShortReturnsSentinelError(t *testing.T) {
	opts := generator.PasswordOptions{
		Length:                   4,
		Numbers:                  true,
		MinimumDigits:            3,
		SpecialCharacters:        true,
		MinimumSpecialCharacters: 3,
	}
	_, err := generator.GeneratePassword(opts)
	if !errors.Is(err, generator.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestGeneratePassword_HonorsMinimumCounts(t *testing.T) {
	opts := generator.PasswordOptions{
		MixedCase:                true,
		Length:                   20,
		Numbers:                  true,
		MinimumDigits:            5,
		SpecialCharacters:        true,
		MinimumSpecialCharacters: 4,
		AvoidAmbiguousCharacters: true,
	}
	pw, err := generator.GeneratePassword(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) != opts.Length {
		t.Fatalf("expected length %d, got %d (%q)", opts.Length, len(pw), pw)
	}

	var digits, specials int
	for _, c := range pw {
		if c >= '0' && c <= '9' {
			digits++
		}
		if strings.ContainsRune(generator.SpecialChars, c) {
			specials++
		}
	}
	if digits < opts.MinimumDigits {
		t.Errorf("expected at least %d digits, got %d in %q", opts.MinimumDigits, digits, pw)
	}
	if specials < opts.MinimumSpecialCharacters {
		t.Errorf("expected at least %d special chars, got %d in %q", opts.MinimumSpecialCharacters, specials, pw)
	}
}

func TestGeneratePassword_AvoidsAmbiguousCharacters(t *testing.T) {
	opts := generator.PasswordOptions{
		MixedCase:                true,
		Length:                   200,
		Numbers:                  true,
		SpecialCharacters:        false,
		AvoidAmbiguousCharacters: true,
	}
	pw, err := generator.GeneratePassword(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range pw {
		if generator.AmbiguousChars[byte(c)] {
			t.Fatalf("password contains ambiguous character %q: %s", c, pw)
		}
	}
}

func TestGeneratePassword_NoMixedCaseUsesOnlyLowercase(t *testing.T) {
	opts := generator.PasswordOptions{
		MixedCase: false,
		Length:    10,
		Numbers:   false,
	}
	pw, err := generator.GeneratePassword(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) != 10 {
		t.Fatalf("expected length 10, got %d (%q)", len(pw), pw)
	}
	for _, c := range pw {
		if !strings.ContainsRune(generator.LowerCaseChars, c) {
			t.Fatalf("expected only lowercase chars, found %q in %q", c, pw)
		}
	}
}

func TestGeneratePassword_IsRandomAcrossCalls(t *testing.T) {
	opts := generator.PasswordOptions{
		MixedCase:         true,
		Length:            16,
		Numbers:           true,
		SpecialCharacters: true,
	}
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		pw, err := generator.GeneratePassword(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		seen[pw] = true
	}
	if len(seen) < 15 {
		t.Fatalf("expected mostly unique passwords across 20 generations, got %d unique", len(seen))
	}
}

func TestGeneratePassword_RejectsLengthAboveMaximum(t *testing.T) {
	// Before the cap, the length went straight into make([]byte, 0, n): a
	// request for a trillion characters was a fatal out-of-memory error that
	// took the whole server down, not a panic net/http could recover from.
	for _, length := range []int{generator.MaxPasswordLength + 1, 1 << 40} {
		_, err := generator.GeneratePassword(generator.PasswordOptions{Length: length})
		if !errors.Is(err, generator.ErrPasswordTooLong) {
			t.Fatalf("Length=%d: expected ErrPasswordTooLong, got %v", length, err)
		}
	}
}

func TestGeneratePassword_AcceptsLengthAtMaximum(t *testing.T) {
	pw, err := generator.GeneratePassword(generator.PasswordOptions{
		MixedCase: true, Length: generator.MaxPasswordLength, Numbers: true, SpecialCharacters: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) != generator.MaxPasswordLength {
		t.Fatalf("expected length %d, got %d", generator.MaxPasswordLength, len(pw))
	}
}

// TestGeneratePassword_LengthIsExactOrRefused walks every combination of
// options over small lengths and minimums. Each request must either produce
// a password of exactly the requested length that meets every enabled
// minimum, or be refused with ErrPasswordTooShort, and it must be refused
// exactly when the guaranteed characters cannot fit: one lowercase letter,
// one uppercase letter with MixedCase, and the minimums of the classes that
// are switched on.
func TestGeneratePassword_LengthIsExactOrRefused(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		for _, numbers := range []bool{false, true} {
			for _, specials := range []bool{false, true} {
				for _, avoid := range []bool{false, true} {
					for minD := -2; minD <= 5; minD++ {
						for minS := -2; minS <= 5; minS++ {
							for length := -1; length <= 12; length++ {
								opts := generator.PasswordOptions{
									MixedCase: mixed, Length: length,
									Numbers: numbers, MinimumDigits: minD,
									SpecialCharacters: specials, MinimumSpecialCharacters: minS,
									AvoidAmbiguousCharacters: avoid,
								}
								checkPassword(t, opts)
							}
						}
					}
				}
			}
		}
	}
}

func checkPassword(t *testing.T, opts generator.PasswordOptions) {
	t.Helper()
	required := 1
	if opts.MixedCase {
		required++
	}
	wantDigits, wantSpecials := 0, 0
	if opts.Numbers {
		wantDigits = max(opts.MinimumDigits, 0)
	}
	if opts.SpecialCharacters {
		wantSpecials = max(opts.MinimumSpecialCharacters, 0)
	}
	required += wantDigits + wantSpecials

	pw, err := generator.GeneratePassword(opts)
	if opts.Length < required {
		if !errors.Is(err, generator.ErrPasswordTooShort) {
			t.Fatalf("%+v: needs %d characters, expected ErrPasswordTooShort, got %q, %v", opts, required, pw, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("%+v: unexpected error: %v", opts, err)
	}
	if len(pw) != opts.Length {
		t.Fatalf("%+v: expected length %d, got %d (%q)", opts, opts.Length, len(pw), pw)
	}
	var lower, upper, digits, specials int
	for _, c := range pw {
		switch {
		case strings.ContainsRune(generator.LowerCaseChars, c):
			lower++
		case strings.ContainsRune(generator.UpperCaseChars, c):
			upper++
		case strings.ContainsRune(generator.DigitChars, c):
			digits++
		case strings.ContainsRune(generator.SpecialChars, c):
			specials++
		default:
			t.Fatalf("%+v: unexpected character %q in %q", opts, c, pw)
		}
		if opts.AvoidAmbiguousCharacters && generator.AmbiguousChars[byte(c)] {
			t.Fatalf("%+v: ambiguous character %q in %q", opts, c, pw)
		}
	}
	if lower < 1 || (opts.MixedCase && upper < 1) || (!opts.MixedCase && upper > 0) {
		t.Fatalf("%+v: wrong letters in %q", opts, pw)
	}
	if digits < wantDigits || (!opts.Numbers && digits > 0) {
		t.Fatalf("%+v: wrong digits in %q", opts, pw)
	}
	if specials < wantSpecials || (!opts.SpecialCharacters && specials > 0) {
		t.Fatalf("%+v: wrong special characters in %q", opts, pw)
	}
}

func TestGeneratePassword_SwitchedOffMinimumDoesNotCount(t *testing.T) {
	// The page disables "Minimum digits" when "Numbers" is unticked, but
	// still sends its value. With 8 characters, 6 special characters and
	// numbers off, the 3 digits that will never be generated used to be
	// counted against the length, and the request was refused.
	pw, err := generator.GeneratePassword(generator.PasswordOptions{
		MixedCase: true, Length: 8,
		Numbers: false, MinimumDigits: 3,
		SpecialCharacters: true, MinimumSpecialCharacters: 6,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) != 8 {
		t.Fatalf("expected length 8, got %q", pw)
	}
}

func TestGeneratePassword_LettersCountTowardsTheMinimum(t *testing.T) {
	// 8 characters with at least 3 digits and 5 special characters leaves
	// no room for the lowercase and uppercase letter the generator always
	// includes. The original returned 10 characters here.
	pw, err := generator.GeneratePassword(generator.PasswordOptions{
		MixedCase: true, Length: 8,
		Numbers: true, MinimumDigits: 3,
		SpecialCharacters: true, MinimumSpecialCharacters: 5,
	})
	if !errors.Is(err, generator.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %q, %v", pw, err)
	}
}

func TestGeneratePassword_NegativeMinimumsAreZero(t *testing.T) {
	pw, err := generator.GeneratePassword(generator.PasswordOptions{
		Length: 6, Numbers: true, MinimumDigits: -10,
		SpecialCharacters: true, MinimumSpecialCharacters: -10,
	})
	if err != nil || len(pw) != 6 {
		t.Fatalf("expected a 6-character password, got %q, %v", pw, err)
	}

	// A negative minimum used to lower the bar enough for a negative
	// length to pass the check and reach make() as a negative capacity,
	// which panics.
	_, err = generator.GeneratePassword(generator.PasswordOptions{
		Length: -5, Numbers: true, MinimumDigits: -10,
	})
	if !errors.Is(err, generator.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort for a negative length, got %v", err)
	}
}

func TestGeneratePassword_HugeMinimumsAreRefused(t *testing.T) {
	// Two minimums of MaxInt add up to a negative number; that must not
	// slip past the length check.
	_, err := generator.GeneratePassword(generator.PasswordOptions{
		Length: 12, Numbers: true, MinimumDigits: math.MaxInt,
		SpecialCharacters: true, MinimumSpecialCharacters: math.MaxInt,
	})
	if !errors.Is(err, generator.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestGeneratePassword_GuaranteedCharactersCanBeAnywhere(t *testing.T) {
	// The guaranteed lowercase letter, uppercase letter and digit are
	// placed first and then shuffled. Three characters and nothing else, so
	// each position should hold the digit a third of the time.
	opts := generator.PasswordOptions{MixedCase: true, Length: 3, Numbers: true, MinimumDigits: 1}
	counts := make([]float64, 3)
	const n = 30000
	for i := 0; i < n; i++ {
		pw, err := generator.GeneratePassword(opts)
		if err != nil {
			t.Fatal(err)
		}
		counts[strings.IndexAny(pw, generator.DigitChars)]++
	}
	// Chi-square, 2 degrees of freedom, p = 1e-6.
	var x2 float64
	for _, c := range counts {
		d := c - n/3
		x2 += d * d / (n / 3)
	}
	if x2 > 27.63 {
		t.Fatalf("digit position counts %v: chi-square %.1f > 27.63", counts, x2)
	}
}
