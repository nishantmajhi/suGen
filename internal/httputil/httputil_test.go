package httputil_test

import (
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nishantmajhi/suGen/internal/httputil"
)

func req(rawQuery string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
}

func TestQueryParam(t *testing.T) {
	r := req("name=hello&empty=")
	if got := httputil.QueryParam(r, "name", "default"); got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
	if got := httputil.QueryParam(r, "empty", "default"); got != "" {
		t.Errorf("expected empty string for present-but-empty param, got %q", got)
	}
	if got := httputil.QueryParam(r, "missing", "default"); got != "default" {
		t.Errorf("expected default for missing param, got %q", got)
	}
}

func TestQueryBool(t *testing.T) {
	cases := []struct {
		query string
		key   string
		def   bool
		want  bool
	}{
		{"mixedCase=true", "mixedCase", false, true},
		{"mixedCase=TRUE", "mixedCase", false, true},
		{"mixedCase=false", "mixedCase", true, false},
		{"", "mixedCase", true, true},
		{"", "mixedCase", false, false},
		{"mixedCase=nonsense", "mixedCase", true, false},
	}
	for _, c := range cases {
		if got := httputil.QueryBool(req(c.query), c.key, c.def); got != c.want {
			t.Errorf("QueryBool(%q, %q, %v) = %v, want %v", c.query, c.key, c.def, got, c.want)
		}
	}
}

func TestQueryInt(t *testing.T) {
	cases := []struct {
		query string
		key   string
		def   int
		want  int
	}{
		{"passwordLength=16", "passwordLength", 12, 16},
		{"passwordLength=12abc", "passwordLength", 0, 12}, // PHP intval() semantics
		{"passwordLength=abc", "passwordLength", 0, 0},
		{"", "passwordLength", 12, 12},
		{"passwordLength=-5", "passwordLength", 0, -5},
	}
	for _, c := range cases {
		if got := httputil.QueryInt(req(c.query), c.key, c.def); got != c.want {
			t.Errorf("QueryInt(%q, %q, %v) = %v, want %v", c.query, c.key, c.def, got, c.want)
		}
	}
}

func TestQueryInt_TrimsWhitespaceLikePHPIntval(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	q := r.URL.Query()
	q.Set("passwordLength", "  42")
	r.URL.RawQuery = q.Encode()

	if got := httputil.QueryInt(r, "passwordLength", 0); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestQueryInt_SaturatesOnOverflowLikePHPIntval(t *testing.T) {
	// PHP's intval() clamps a numeric string that overflows to
	// PHP_INT_MAX / PHP_INT_MIN. This parser used to return 0, which turned
	// an absurdly long request into a zero-length one.
	cases := []struct {
		raw  string
		want int
	}{
		{"99999999999999999999", math.MaxInt},
		{"+99999999999999999999", math.MaxInt},
		{"-99999999999999999999", math.MinInt},
		{"9223372036854775807", math.MaxInt},
		{"-9223372036854775808", math.MinInt},
		{"99999999999999999999abc", math.MaxInt},
	}
	for _, c := range cases {
		if got := httputil.QueryInt(req("n="+url.QueryEscape(c.raw)), "n", 7); got != c.want {
			t.Errorf("QueryInt(%q) = %d, want %d", c.raw, got, c.want)
		}
	}
}

func TestJSON_ForbidsStorage(t *testing.T) {
	rec := httptest.NewRecorder()
	httputil.JSON(rec, http.StatusOK, map[string]string{"password": "x"})
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-store")
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
}
