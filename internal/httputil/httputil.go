// Package httputil provides small helpers for reading query parameters and
// writing JSON responses, matching the semantics of the original PHP
// helperFunctions.php (getParam + jsonResponse).
package httputil

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// QueryParam returns the raw string value of key from the request's query
// string, or def if the key is absent. Unlike url.Values.Get, this treats
// a present-but-empty value as present (matching PHP's isset()), only
// falling back to def when the key doesn't appear at all.
func QueryParam(r *http.Request, key, def string) string {
	values := r.URL.Query()
	if v, ok := values[key]; ok && len(v) > 0 {
		return v[0]
	}
	return def
}

// QueryBool returns true if the query parameter's value, compared
// case-insensitively, equals "true". Matches the PHP endpoints' use of
// strtolower(getParam(...)) === 'true'.
func QueryBool(r *http.Request, key string, def bool) bool {
	defStr := "false"
	if def {
		defStr = "true"
	}
	return strings.ToLower(QueryParam(r, key, defStr)) == "true"
}

// QueryInt returns the query parameter parsed the way PHP's intval()
// would: leading whitespace and an optional sign are consumed, then as
// many leading digits as possible are parsed; anything non-numeric
// (including a totally non-numeric string) yields 0. Falls back to def
// only when the key is absent.
func QueryInt(r *http.Request, key string, def int) int {
	raw := QueryParam(r, key, "")
	if raw == "" {
		if _, present := r.URL.Query()[key]; !present {
			return def
		}
	}
	return phpIntval(raw)
}

// phpIntval mimics PHP's intval() for base-10 strings.
func phpIntval(s string) int {
	s = strings.TrimSpace(s)
	i := 0
	sign := 1
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if s[i] == '-' {
			sign = -1
		}
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if start == i {
		return 0
	}
	// Parse with the sign attached, so the most negative int is in range.
	n, err := strconv.Atoi(s[:i])
	if err != nil {
		// Only a range error is possible here, and intval() saturates.
		if sign < 0 {
			return math.MinInt
		}
		return math.MaxInt
	}
	return n
}

// JSON writes data as a JSON response with the given status code,
// matching the Content-Type set by the original jsonResponse().
//
// Every JSON response this service sends carries a freshly generated
// secret, or a refusal that echoes what was asked for, so none may be
// stored: not in the browser's disk cache, and not by a proxy.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
