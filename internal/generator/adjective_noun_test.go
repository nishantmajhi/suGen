package generator_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nishantmajhi/suGen/internal/generator"
)

func TestGenerateAdjectiveNounPassphrase_Capitalized(t *testing.T) {
	phrase, err := generator.GenerateAdjectiveNounPassphrase(
		generator.AdjectiveNounOptions{CapitalizeWords: true, IncludeNumbers: false, TotalWords: 2},
		fixedPicker{word: "daily"},
		fixedPicker{word: "athlete"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phrase != "Daily Athlete" {
		t.Fatalf("expected %q, got %q", "Daily Athlete", phrase)
	}
}

func TestGenerateAdjectiveNounPassphrase_NotCapitalized(t *testing.T) {
	phrase, err := generator.GenerateAdjectiveNounPassphrase(
		generator.AdjectiveNounOptions{CapitalizeWords: false, IncludeNumbers: false, TotalWords: 2},
		fixedPicker{word: "daily"},
		fixedPicker{word: "athlete"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phrase != "daily athlete" {
		t.Fatalf("expected %q, got %q", "daily athlete", phrase)
	}
}

func TestGenerateAdjectiveNounPassphrase_WithNumbers(t *testing.T) {
	phrase, err := generator.GenerateAdjectiveNounPassphrase(
		generator.AdjectiveNounOptions{CapitalizeWords: true, IncludeNumbers: true, TotalWords: 2},
		fixedPicker{word: "daily"},
		fixedPicker{word: "athlete"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const wantPrefix = "Daily Athlete "
	if len(phrase) <= len(wantPrefix) {
		t.Fatalf("expected trailing digits after %q, got %q", wantPrefix, phrase)
	}
	if phrase[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("expected prefix %q, got %q", wantPrefix, phrase)
	}
	digits := phrase[len(wantPrefix):]
	if len(digits) < 1 || len(digits) > 4 {
		t.Fatalf("expected 1-4 trailing digits, got %q (len %d)", digits, len(digits))
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			t.Fatalf("expected only digits in suffix, got %q", digits)
		}
	}
}

func TestGenerateAdjectiveNounPassphrase_MultipleAdjectivesBeforeNoun(t *testing.T) {
	// With TotalWords=5, the generator should call adjectives.Random()
	// 4 times and nouns.Random() once, joining 4 adjectives followed by
	// exactly 1 noun with spaces.
	adjSequence := []string{"daily", "brave", "mighty", "swift"}
	adjIdx := 0
	adjPicker := pickerFunc(func() (string, error) {
		w := adjSequence[adjIdx]
		adjIdx++
		return w, nil
	})
	nounPicker := fixedPicker{word: "athlete"}

	phrase, err := generator.GenerateAdjectiveNounPassphrase(
		generator.AdjectiveNounOptions{CapitalizeWords: true, IncludeNumbers: false, TotalWords: 5},
		adjPicker, nounPicker,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Daily Brave Mighty Swift Athlete"
	if phrase != want {
		t.Fatalf("expected %q, got %q", want, phrase)
	}
	if adjIdx != 4 {
		t.Fatalf("expected adjectives.Random() to be called 4 times, got %d", adjIdx)
	}
	if len(strings.Fields(phrase)) != 5 {
		t.Fatalf("expected 5 space-separated words, got %q", phrase)
	}
}

func TestGenerateAdjectiveNounPassphrase_ClampsBelowMinimum(t *testing.T) {
	// TotalWords=1 (or 0, or negative) should be clamped up to
	// MinAdjectiveNounWords (2): exactly 1 adjective + 1 noun.
	for _, tw := range []int{1, 0, -3} {
		phrase, err := generator.GenerateAdjectiveNounPassphrase(
			generator.AdjectiveNounOptions{CapitalizeWords: true, TotalWords: tw},
			fixedPicker{word: "daily"},
			fixedPicker{word: "athlete"},
		)
		if err != nil {
			t.Fatalf("unexpected error for TotalWords=%d: %v", tw, err)
		}
		if phrase != "Daily Athlete" {
			t.Fatalf("TotalWords=%d: expected clamp to 2 words (%q), got %q", tw, "Daily Athlete", phrase)
		}
	}
}

func TestGenerateAdjectiveNounPassphrase_UsesRealWordLists(t *testing.T) {
	adj := fixedPicker{word: "brave"}
	noun := fixedPicker{word: "falcon"}
	phrase, err := generator.GenerateAdjectiveNounPassphrase(
		generator.AdjectiveNounOptions{CapitalizeWords: true, TotalWords: 2},
		adj, noun,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phrase != "Brave Falcon" {
		t.Fatalf("expected adjective followed by noun separated by a space, got %q", phrase)
	}
}

// pickerFunc adapts a plain function to the generator.WordPicker
// interface, letting tests return a different word on each call.
type pickerFunc func() (string, error)

func (f pickerFunc) Random() (string, error) {
	return f()
}

func TestGenerateAdjectiveNounPassphrase_RejectsWordCountAboveMaximum(t *testing.T) {
	for _, n := range []int{generator.MaxPassphraseWords + 1, 1 << 40} {
		_, err := generator.GenerateAdjectiveNounPassphrase(
			generator.AdjectiveNounOptions{TotalWords: n}, fixedPicker{word: "daily"}, fixedPicker{word: "athlete"})
		if !errors.Is(err, generator.ErrTooManyWords) {
			t.Fatalf("TotalWords=%d: expected ErrTooManyWords, got %v", n, err)
		}
	}
}
