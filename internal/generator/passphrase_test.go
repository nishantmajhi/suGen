package generator_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nishantmajhi/suGen/internal/generator"
)

// fixedPicker is a WordPicker that always returns the same word, useful
// for deterministic assertions about spacing/capitalization/counts.
type fixedPicker struct {
	word string
}

func (f fixedPicker) Random() (string, error) {
	return f.word, nil
}

func TestGeneratePassphrase_WordCountAndSpacing(t *testing.T) {
	opts := generator.PassphraseOptions{
		CapitalizeWords: true,
		IncludeNumbers:  false,
		TotalWords:      3,
	}
	phrase, err := generator.GeneratePassphrase(opts, fixedPicker{word: "apple"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Apple Apple Apple"
	if phrase != want {
		t.Fatalf("expected %q, got %q", want, phrase)
	}
}

func TestGeneratePassphrase_NoCapitalizeKeepsWordsLowercase(t *testing.T) {
	opts := generator.PassphraseOptions{
		CapitalizeWords: false,
		IncludeNumbers:  false,
		TotalWords:      2,
	}
	phrase, err := generator.GeneratePassphrase(opts, fixedPicker{word: "banana"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phrase != "banana banana" {
		t.Fatalf("expected lowercase words, got %q", phrase)
	}
}

func TestGeneratePassphrase_IncludeNumbersAppendsTrailingDigits(t *testing.T) {
	opts := generator.PassphraseOptions{
		CapitalizeWords: true,
		IncludeNumbers:  true,
		TotalWords:      2,
	}
	phrase, err := generator.GeneratePassphrase(opts, fixedPicker{word: "cherry"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parts := strings.SplitN(phrase, " ", 3)
	if len(parts) != 3 {
		t.Fatalf("expected 'Word Word digits', got %q", phrase)
	}
	digits := parts[2]
	if len(digits) < 1 || len(digits) > 4 {
		t.Fatalf("expected 1-4 trailing digits, got %q (len %d)", digits, len(digits))
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			t.Fatalf("expected only digits in suffix, got %q", digits)
		}
	}
}

func TestGeneratePassphrase_SingleWord(t *testing.T) {
	opts := generator.PassphraseOptions{
		CapitalizeWords: true,
		IncludeNumbers:  false,
		TotalWords:      1,
	}
	phrase, err := generator.GeneratePassphrase(opts, fixedPicker{word: "date"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phrase != "Date" {
		t.Fatalf("expected %q, got %q", "Date", phrase)
	}
}

func TestGeneratePassphrase_RejectsWordCountAboveMaximum(t *testing.T) {
	calls := 0
	counting := pickerFunc(func() (string, error) { calls++; return "word", nil })
	for _, n := range []int{generator.MaxPassphraseWords + 1, 1 << 40} {
		_, err := generator.GeneratePassphrase(generator.PassphraseOptions{TotalWords: n}, counting)
		if !errors.Is(err, generator.ErrTooManyWords) {
			t.Fatalf("TotalWords=%d: expected ErrTooManyWords, got %v", n, err)
		}
	}
	if calls != 0 {
		t.Fatalf("expected the limit to be checked before any word is drawn, got %d draws", calls)
	}
}

func TestGeneratePassphrase_AcceptsWordCountAtMaximum(t *testing.T) {
	phrase, err := generator.GeneratePassphrase(
		generator.PassphraseOptions{TotalWords: generator.MaxPassphraseWords}, fixedPicker{word: "w"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := len(strings.Fields(phrase)); n != generator.MaxPassphraseWords {
		t.Fatalf("expected %d words, got %d", generator.MaxPassphraseWords, n)
	}
}

func TestGeneratePassphrase_NeverEmpty(t *testing.T) {
	// A word count of zero or less used to return "" as a successful
	// passphrase, or only the digits with IncludeNumbers. It is now raised
	// to one word, as the adjective+noun style raises its count to two.
	for _, n := range []int{0, -1, -100} {
		phrase, err := generator.GeneratePassphrase(
			generator.PassphraseOptions{CapitalizeWords: true, TotalWords: n}, fixedPicker{word: "kiwi"})
		if err != nil || phrase != "Kiwi" {
			t.Fatalf("TotalWords=%d: expected %q, got %q, %v", n, "Kiwi", phrase, err)
		}

		phrase, err = generator.GeneratePassphrase(
			generator.PassphraseOptions{CapitalizeWords: true, IncludeNumbers: true, TotalWords: n}, fixedPicker{word: "kiwi"})
		if err != nil || !strings.HasPrefix(phrase, "Kiwi ") {
			t.Fatalf("TotalWords=%d with numbers: expected %q followed by digits, got %q, %v", n, "Kiwi ", phrase, err)
		}
	}
}
