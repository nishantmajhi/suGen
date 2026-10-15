// Package wordlist provides access to curated word lists (a general
// dictionary, plus adjectives and nouns) used to build passphrases.
package wordlist

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"

	"github.com/nishantmajhi/suGen/internal/randutil"
)

//go:embed data/words.txt
var rawWords []byte

//go:embed data/adjectives.txt
var rawAdjectives []byte

//go:embed data/nouns.txt
var rawNouns []byte

// List holds an in-memory word list and can produce cryptographically
// random selections from it.
type List struct {
	words []string
}

// Package-wide singletons, loaded once at package init.
var (
	defaultList    *List
	adjectivesList *List
	nounsList      *List
)

func init() {
	defaultList = mustLoad(rawWords, "words")
	adjectivesList = mustLoad(rawAdjectives, "adjectives")
	nounsList = mustLoad(rawNouns, "nouns")
}

func mustLoad(data []byte, name string) *List {
	words, err := parse(data)
	if err != nil {
		panic(fmt.Sprintf("wordlist: failed to parse embedded %s list: %v", name, err))
	}
	return &List{words: words}
}

func parse(data []byte) ([]string, error) {
	var words []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Words can be long-ish but never anywhere near the default 64KB
	// token limit, so the default buffer size is fine.
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		words = append(words, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("word list is empty")
	}
	return words, nil
}

// Default returns the package-wide general dictionary word list, used by
// the classic multi-word passphrase style.
func Default() *List {
	return defaultList
}

// Adjectives returns the package-wide adjectives list, used by the
// "adjectiveNoun" passphrase style.
func Adjectives() *List {
	return adjectivesList
}

// Nouns returns the package-wide nouns list, used by the "adjectiveNoun"
// passphrase style.
func Nouns() *List {
	return nounsList
}

// Random returns a single random word from the list, chosen using a
// cryptographically secure random source.
func (l *List) Random() (string, error) {
	idx, err := randutil.Intn(len(l.words))
	if err != nil {
		return "", fmt.Errorf("wordlist: %w", err)
	}
	return l.words[idx], nil
}

// Len returns the number of words available.
func (l *List) Len() int {
	return len(l.words)
}
