package randutil_test

import (
	"testing"

	"github.com/nishantmajhi/suGen/internal/randutil"
)

func TestIntn_RangeIsRespected(t *testing.T) {
	for i := 0; i < 200; i++ {
		v, err := randutil.Intn(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v < 0 || v >= 10 {
			t.Fatalf("expected value in [0, 10), got %d", v)
		}
	}
}

func TestIntn_RejectsNonPositive(t *testing.T) {
	if _, err := randutil.Intn(0); err == nil {
		t.Error("expected error for n=0")
	}
	if _, err := randutil.Intn(-1); err == nil {
		t.Error("expected error for n=-1")
	}
}

func TestDigit_ReturnsAsciiDigit(t *testing.T) {
	for i := 0; i < 100; i++ {
		d, err := randutil.Digit()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d < '0' || d > '9' {
			t.Fatalf("expected ASCII digit, got byte %d (%q)", d, d)
		}
	}
}

func TestShuffle_PreservesContentsAndLength(t *testing.T) {
	original := []byte("abcdefghij")
	shuffled := make([]byte, len(original))
	copy(shuffled, original)

	if err := randutil.Shuffle(shuffled); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(shuffled) != len(original) {
		t.Fatalf("expected same length, got %d vs %d", len(shuffled), len(original))
	}

	counts := map[byte]int{}
	for _, b := range original {
		counts[b]++
	}
	for _, b := range shuffled {
		counts[b]--
	}
	for b, c := range counts {
		if c != 0 {
			t.Fatalf("shuffle changed the multiset of bytes: %q count off by %d", b, c)
		}
	}
}

// chiSquare returns Pearson's statistic for counts against a uniform
// expectation.
func chiSquare(counts []int) float64 {
	total := 0
	for _, c := range counts {
		total += c
	}
	want := float64(total) / float64(len(counts))
	var x2 float64
	for _, c := range counts {
		d := float64(c) - want
		x2 += d * d / want
	}
	return x2
}

// Critical value of chi-square with 5 degrees of freedom at p = 1e-6, so a
// fair source fails these checks about once in a million runs.
const chi2df5 = 30.86

func TestIntn_IsUniform(t *testing.T) {
	counts := make([]int, 6)
	for i := 0; i < 60000; i++ {
		v, err := randutil.Intn(6)
		if err != nil {
			t.Fatal(err)
		}
		counts[v]++
	}
	if x2 := chiSquare(counts); x2 > chi2df5 {
		t.Fatalf("Intn(6) counts %v: chi-square %.1f > %.1f", counts, x2, chi2df5)
	}
}

func TestShuffle_EveryPermutationIsEquallyLikely(t *testing.T) {
	// Six orders of three items. The classic off-by-one, j drawn from
	// [0, n) instead of [0, i], produces some orders twice as often as
	// others, and never fails a length or multiset check.
	index := map[string]int{"abc": 0, "acb": 1, "bac": 2, "bca": 3, "cab": 4, "cba": 5}
	counts := make([]int, 6)
	for i := 0; i < 60000; i++ {
		b := []byte("abc")
		if err := randutil.Shuffle(b); err != nil {
			t.Fatal(err)
		}
		counts[index[string(b)]]++
	}
	if x2 := chiSquare(counts); x2 > chi2df5 {
		t.Fatalf("permutation counts %v: chi-square %.1f > %.1f", counts, x2, chi2df5)
	}
}
