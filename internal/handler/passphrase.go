package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/nishantmajhi/suGen/internal/generator"
	"github.com/nishantmajhi/suGen/internal/httputil"
)

// Supported values for the "style" query parameter.
const (
	styleWords         = "words"
	styleAdjectiveNoun = "adjectivenoun"
)

// PassphraseHandler serves GET /passphrase/, generating a passphrase from
// query-string options. Field names and defaults match the original API,
// plus an added "style" option for an adjective+noun passphrase style
// (e.g. "Daily Athlete 540").
type PassphraseHandler struct {
	words      generator.WordPicker
	adjectives generator.WordPicker
	nouns      generator.WordPicker
}

// NewPassphraseHandler constructs a PassphraseHandler backed by the given
// word pickers (typically wordlist.Default(), wordlist.Adjectives(), and
// wordlist.Nouns()).
func NewPassphraseHandler(words, adjectives, nouns generator.WordPicker) *PassphraseHandler {
	return &PassphraseHandler{words: words, adjectives: adjectives, nouns: nouns}
}

func (h *PassphraseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	style := strings.ToLower(httputil.QueryParam(r, "style", styleWords))
	capitalizeWords := httputil.QueryBool(r, "capitalizeWords", true)
	includeNumbers := httputil.QueryBool(r, "includeNumbers", false)

	var (
		passphrase string
		err        error
	)

	switch style {
	case styleAdjectiveNoun:
		passphrase, err = generator.GenerateAdjectiveNounPassphrase(generator.AdjectiveNounOptions{
			CapitalizeWords: capitalizeWords,
			IncludeNumbers:  includeNumbers,
			TotalWords:      httputil.QueryInt(r, "totalWords", generator.MinAdjectiveNounWords),
		}, h.adjectives, h.nouns)
	default:
		passphrase, err = generator.GeneratePassphrase(generator.PassphraseOptions{
			CapitalizeWords: capitalizeWords,
			IncludeNumbers:  includeNumbers,
			TotalWords:      httputil.QueryInt(r, "totalWords", 2),
		}, h.words)
	}

	if errors.Is(err, generator.ErrTooManyWords) {
		// Same shape as the password endpoint's refusals.
		httputil.JSON(w, http.StatusOK, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		log.Printf("passphrase handler: %v", err)
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"passphrase": passphrase})
}
