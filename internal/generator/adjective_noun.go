package generator

import "fmt"

// MinAdjectiveNounWords is the fewest words the adjective+noun style can
// produce: exactly one adjective followed by one noun. Any requested
// TotalWords below this is clamped up to it.
const MinAdjectiveNounWords = 2

// AdjectiveNounOptions controls how GenerateAdjectiveNounPassphrase builds
// its passphrase.
type AdjectiveNounOptions struct {
	// CapitalizeWords capitalizes the first letter of every word (e.g.
	// "Daily Athlete"). When false, words are left exactly as they
	// appear in the word list (e.g. "daily athlete").
	CapitalizeWords bool
	// IncludeNumbers appends a space followed by 1-4 random digits to
	// the end of the passphrase (e.g. "Daily Athlete 540").
	IncludeNumbers bool
	// TotalWords is how many words to combine in total. The final word
	// is always a noun; every word before it is an adjective. Values
	// below MinAdjectiveNounWords are clamped up to it, since the style
	// requires at least one adjective and one noun.
	TotalWords int
}

// GenerateAdjectiveNounPassphrase builds a memorable passphrase by
// combining one or more random adjectives with a single trailing noun,
// space-separated just like the classic multi-word style, optionally
// capitalized and/or suffixed with random digits.
//
// Given TotalWords, the first TotalWords-1 words are adjectives and the
// last word is always a noun. For example, with TotalWords=5 the result
// combines 4 adjectives and 1 noun, e.g. "Daily Brave Mighty Swift Athlete".
//
// Example outputs: "daily athlete", "Daily Athlete", "Daily Athlete 540".
func GenerateAdjectiveNounPassphrase(opts AdjectiveNounOptions, adjectives, nouns WordPicker) (string, error) {
	if opts.TotalWords > MaxPassphraseWords {
		return "", ErrTooManyWords
	}

	words := make([]string, max(opts.TotalWords, MinAdjectiveNounWords))
	last := len(words) - 1
	for i := range words {
		picker := adjectives
		if i == last {
			picker = nouns
		}
		word, err := picker.Random()
		if err != nil {
			return "", fmt.Errorf("generator: %w", err)
		}
		words[i] = word
	}

	return assemble(words, opts.CapitalizeWords, opts.IncludeNumbers)
}
