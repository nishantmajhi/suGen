package generator

import (
	"fmt"
	"strings"

	"github.com/nishantmajhi/suGen/internal/randutil"
)

// MinPassphraseWords is the fewest words GeneratePassphrase produces. Any
// requested TotalWords below this is raised to it, so the result is never
// empty.
const MinPassphraseWords = 1

// MaxPassphraseWords is the most words either passphrase style will
// combine. The count arrives from an unauthenticated query string, so it has
// to be bounded before any word is drawn.
const MaxPassphraseWords = 64

// ErrTooManyWords is returned when the requested word count exceeds
// MaxPassphraseWords.
var ErrTooManyWords = fmt.Errorf("number of words exceeds the maximum of %d", MaxPassphraseWords)

// PassphraseOptions controls how GeneratePassphrase builds a passphrase.
// Field semantics and defaults intentionally mirror the original PHP
// endpoint.
type PassphraseOptions struct {
	CapitalizeWords bool
	IncludeNumbers  bool
	TotalWords      int
}

// WordPicker returns a single random word. *wordlist.List satisfies this
// interface, but any implementation can be substituted (useful for tests).
type WordPicker interface {
	Random() (string, error)
}

// GeneratePassphrase builds a space-separated passphrase of TotalWords
// words, optionally capitalized and suffixed with 1-4 random digits.
// Values of TotalWords below MinPassphraseWords are raised to it.
func GeneratePassphrase(opts PassphraseOptions, picker WordPicker) (string, error) {
	if opts.TotalWords > MaxPassphraseWords {
		return "", ErrTooManyWords
	}

	words := make([]string, max(opts.TotalWords, MinPassphraseWords))
	for i := range words {
		word, err := picker.Random()
		if err != nil {
			return "", fmt.Errorf("generator: %w", err)
		}
		words[i] = word
	}

	return assemble(words, opts.CapitalizeWords, opts.IncludeNumbers)
}

// assemble joins words with spaces, capitalizing each if asked, and with
// includeNumbers appends a space and 1-4 random digits. Both passphrase
// styles finish this way.
func assemble(words []string, capitalizeWords, includeNumbers bool) (string, error) {
	if capitalizeWords {
		for i, w := range words {
			words[i] = capitalize(w)
		}
	}
	phrase := strings.Join(words, " ")

	if !includeNumbers {
		return phrase, nil
	}

	numDigits, err := randutil.Intn(4) // yields 0-3
	if err != nil {
		return "", fmt.Errorf("generator: %w", err)
	}
	numDigits++ // shift to 1-4, matching PHP's random_int(1, 4)

	suffix := make([]byte, numDigits)
	for i := range suffix {
		suffix[i], err = randutil.Digit()
		if err != nil {
			return "", fmt.Errorf("generator: %w", err)
		}
	}
	return phrase + " " + string(suffix), nil
}

// capitalize uppercases only the first byte of s, leaving the rest
// untouched -- matching PHP's ucfirst().
func capitalize(s string) string {
	if s == "" {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-'a'+'A') + s[1:]
	}
	return s
}
