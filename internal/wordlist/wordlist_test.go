package wordlist_test

import (
	"testing"

	"github.com/nishantmajhi/suGen/internal/wordlist"
)

func TestDefault_HasManyWords(t *testing.T) {
	list := wordlist.Default()
	if list.Len() < 1000 {
		t.Fatalf("expected a large dictionary, got only %d words", list.Len())
	}
}

func TestAdjectives_NotEmpty(t *testing.T) {
	list := wordlist.Adjectives()
	if list.Len() == 0 {
		t.Fatal("expected a non-empty adjectives list")
	}
}

func TestNouns_NotEmpty(t *testing.T) {
	list := wordlist.Nouns()
	if list.Len() == 0 {
		t.Fatal("expected a non-empty nouns list")
	}
}

func TestRandom_ReturnsWordFromList(t *testing.T) {
	list := wordlist.Adjectives()
	word, err := list.Random()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if word == "" {
		t.Fatal("expected a non-empty word")
	}
}

func TestRandom_ProducesVariety(t *testing.T) {
	list := wordlist.Nouns()
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		word, err := list.Random()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		seen[word] = true
	}
	if len(seen) < 5 {
		t.Fatalf("expected varied results across 50 draws, got only %d distinct words", len(seen))
	}
}
