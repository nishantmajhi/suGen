package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nishantmajhi/suGen/internal/generator"
	"github.com/nishantmajhi/suGen/internal/handler"
	"github.com/nishantmajhi/suGen/internal/wordlist"
)

func doRequest(t *testing.T, h http.Handler, target string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode JSON response: %v (body: %s)", err, rec.Body.String())
	}
	return body
}

func TestPasswordHandler_Defaults(t *testing.T) {
	h := handler.NewPasswordHandler()
	body := doRequest(t, h, "/password/")

	pw, ok := body["password"].(string)
	if !ok {
		t.Fatalf("expected a %q field in response, got %v", "password", body)
	}
	if len(pw) != 12 {
		t.Errorf("expected default length 12, got %d (%q)", len(pw), pw)
	}
}

func TestPasswordHandler_CustomOptions(t *testing.T) {
	h := handler.NewPasswordHandler()
	body := doRequest(t, h, "/password/?mixedCase=true&passwordLength=20&numbers=true&minimumDigits=3&specialCharacters=true&minimumSpecialCharacters=2&avoidAmbiguousCharacters=true")

	pw, ok := body["password"].(string)
	if !ok {
		t.Fatalf("expected a %q field in response, got %v", "password", body)
	}
	if len(pw) != 20 {
		t.Errorf("expected length 20, got %d (%q)", len(pw), pw)
	}
}

func TestPasswordHandler_TooShortReturnsErrorField(t *testing.T) {
	h := handler.NewPasswordHandler()
	body := doRequest(t, h, "/password/?passwordLength=4&minimumDigits=3&minimumSpecialCharacters=3")

	if _, ok := body["error"]; !ok {
		t.Fatalf("expected an %q field in response, got %v", "error", body)
	}
}

func TestPassphraseHandler_ClassicDefaults(t *testing.T) {
	h := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	body := doRequest(t, h, "/passphrase/")

	phrase, ok := body["passphrase"].(string)
	if !ok {
		t.Fatalf("expected a %q field in response, got %v", "passphrase", body)
	}
	words := strings.Fields(phrase)
	if len(words) != 2 {
		t.Errorf("expected 2 words by default, got %d (%q)", len(words), phrase)
	}
}

func TestPassphraseHandler_AdjectiveNounStyle(t *testing.T) {
	h := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	body := doRequest(t, h, "/passphrase/?style=adjectiveNoun&includeNumbers=true")

	phrase, ok := body["passphrase"].(string)
	if !ok {
		t.Fatalf("expected a %q field in response, got %v", "passphrase", body)
	}

	// Default totalWords=2, so with includeNumbers=true we expect
	// "Adjective Noun <digits>" - exactly 3 space-separated parts.
	parts := strings.Fields(phrase)
	if len(parts) != 3 {
		t.Fatalf("expected 3 space-separated parts (adjective, noun, digits), got %q", phrase)
	}
	digits := parts[2]

	if digits == "" || len(digits) > 4 {
		t.Errorf("expected 1-4 trailing digits, got %q in %q", digits, phrase)
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			t.Errorf("expected only digits in suffix, got %q in %q", digits, phrase)
		}
	}

	// Each word should start with an uppercase letter by default
	// (capitalizeWords defaults to true).
	for _, word := range parts[:2] {
		if word == "" || word[0] < 'A' || word[0] > 'Z' {
			t.Errorf("expected each word to start with an uppercase letter by default, got %q", word)
		}
	}
}

func TestPassphraseHandler_AdjectiveNounStyleNoCapitalize(t *testing.T) {
	h := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	body := doRequest(t, h, "/passphrase/?style=adjectiveNoun&capitalizeWords=false")

	phrase := body["passphrase"].(string)
	if phrase[0] < 'a' || phrase[0] > 'z' {
		t.Errorf("expected passphrase to start lowercase when capitalizeWords=false, got %q", phrase)
	}
}

func TestPassphraseHandler_AdjectiveNounStyleRespectsTotalWords(t *testing.T) {
	h := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	body := doRequest(t, h, "/passphrase/?style=adjectiveNoun&totalWords=5")

	phrase := body["passphrase"].(string)
	words := strings.Fields(phrase)
	if len(words) != 5 {
		t.Errorf("expected 5 space-separated words for totalWords=5, got %q", phrase)
	}
}

func TestPassphraseHandler_AdjectiveNounStyleClampsLowTotalWords(t *testing.T) {
	h := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	body := doRequest(t, h, "/passphrase/?style=adjectiveNoun&totalWords=1")

	// Should not error; totalWords=1 gets clamped up to the minimum
	// of 2 (one adjective + one noun) inside the generator.
	phrase, ok := body["passphrase"].(string)
	if !ok {
		t.Fatalf("expected a passphrase field even when totalWords is below the minimum, got %v", body)
	}
	if len(strings.Fields(phrase)) != 2 {
		t.Errorf("expected clamp to 2 words, got %q", phrase)
	}
}

// Ensure the generator's WordPicker interface is satisfied by *wordlist.List
// (compile-time check used by the handler wiring in cmd/server/main.go).
var _ generator.WordPicker = wordlist.Default()

func TestPasswordHandler_OversizedLengthReturnsErrorField(t *testing.T) {
	// One request for a trillion characters used to kill the server with a
	// fatal out-of-memory error. It must now be refused like any other
	// impossible request: 200 with an "error" field, the API's existing shape.
	h := handler.NewPasswordHandler()
	for _, q := range []string{"passwordLength=1025", "passwordLength=1000000000000"} {
		body := doRequest(t, h, "/password/?"+q)
		if _, ok := body["password"]; ok {
			t.Fatalf("%s: expected no password, got %v", q, body)
		}
		if body["error"] != generator.ErrPasswordTooLong.Error() {
			t.Fatalf("%s: expected error %q, got %v", q, generator.ErrPasswordTooLong, body)
		}
	}
}

func TestPassphraseHandler_OversizedWordCountReturnsErrorField(t *testing.T) {
	h := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	for _, q := range []string{"totalWords=65", "totalWords=1000000000000", "style=adjectiveNoun&totalWords=1000000000000"} {
		body := doRequest(t, h, "/passphrase/?"+q)
		if _, ok := body["passphrase"]; ok {
			t.Fatalf("%s: expected no passphrase, got %v", q, body)
		}
		if body["error"] != generator.ErrTooManyWords.Error() {
			t.Fatalf("%s: expected error %q, got %v", q, generator.ErrTooManyWords, body)
		}
	}
}

// The page sends every field on every change, including the minimum of a
// class that is switched off (its input is disabled, not omitted). These are
// the queries it sends for two settings the sliders and boxes allow.
func TestPasswordHandler_QueriesThePageSends(t *testing.T) {
	h := handler.NewPasswordHandler()
	const page = "/password/?mixedCase=true&passwordLength=8&avoidAmbiguousCharacters=true"

	// Numbers unticked, 6 special characters: there is room, and the
	// disabled "Minimum digits: 3" must not count.
	body := doRequest(t, h, page+"&numbers=false&minimumDigits=3&specialCharacters=true&minimumSpecialCharacters=6")
	if pw, _ := body["password"].(string); len(pw) != 8 {
		t.Fatalf("numbers off: expected an 8-character password, got %v", body)
	}

	// 3 digits and 5 special characters leave no room for the two letters.
	// This used to return a 10-character password under an 8-character
	// slider.
	body = doRequest(t, h, page+"&numbers=true&minimumDigits=3&specialCharacters=true&minimumSpecialCharacters=5")
	if body["error"] != generator.ErrPasswordTooShort.Error() {
		t.Fatalf("no room: expected %q, got %v", generator.ErrPasswordTooShort, body)
	}
}

func TestPassphraseHandler_NonPositiveWordCountStillGivesAWord(t *testing.T) {
	h := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	for _, q := range []string{"totalWords=0", "totalWords=-4", "totalWords=abc", "totalWords=", "totalWords=0&includeNumbers=true"} {
		body := doRequest(t, h, "/passphrase/?"+q)
		phrase, _ := body["passphrase"].(string)
		words := strings.Fields(phrase)
		if len(words) == 0 || !isLetters(words[0]) {
			t.Fatalf("%s: expected at least one word, got %v", q, body)
		}
	}
}

func isLetters(s string) bool {
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return s != ""
}

func TestHandlers_OverflowingNumbersAreTooLargeNotZero(t *testing.T) {
	// An overflowing number used to parse as 0: passwordLength gave a
	// two-character password and totalWords an empty passphrase.
	const huge = "99999999999999999999"
	body := doRequest(t, handler.NewPasswordHandler(), "/password/?passwordLength="+huge)
	if body["error"] != generator.ErrPasswordTooLong.Error() {
		t.Fatalf("passwordLength: expected %q, got %v", generator.ErrPasswordTooLong, body)
	}
	h := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	body = doRequest(t, h, "/passphrase/?totalWords="+huge)
	if body["error"] != generator.ErrTooManyWords.Error() {
		t.Fatalf("totalWords: expected %q, got %v", generator.ErrTooManyWords, body)
	}
}

func TestHandlers_ResponsesAreNeverStored(t *testing.T) {
	// A GET response with no Cache-Control is written to the browser's disk
	// cache and may be kept by any proxy in between. Every response here
	// carries a secret, or a refusal that echoes what was asked for.
	words := handler.NewPassphraseHandler(wordlist.Default(), wordlist.Adjectives(), wordlist.Nouns())
	cases := []struct {
		h      http.Handler
		target string
	}{
		{handler.NewPasswordHandler(), "/password/"},
		{handler.NewPasswordHandler(), "/password/?passwordLength=2000"},
		{words, "/passphrase/"},
		{words, "/passphrase/?style=adjectiveNoun"},
		{words, "/passphrase/?totalWords=2000"},
		{handler.NewPassphraseHandler(failingPicker{}, failingPicker{}, failingPicker{}), "/passphrase/"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		c.h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.target, nil))
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s (status %d): Cache-Control = %q, want %q", c.target, rec.Code, got, "no-store")
		}
	}
}

type failingPicker struct{}

func (failingPicker) Random() (string, error) { return "", errors.New("no entropy") }
