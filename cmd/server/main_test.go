package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	captureLog(t)
	srv, err := newServer(":0")
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %s: %v", url, err)
	}
	return resp, string(body)
}

// captureLog sends the standard logger to a buffer for the test's duration.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() { log.SetOutput(prevOut); log.SetFlags(prevFlags) })
	return &buf
}

func TestServer_ServesTheAPIAndThePage(t *testing.T) {
	ts := newTestServer(t)

	for path, field := range map[string]string{"/password/": "password", "/passphrase/": "passphrase"} {
		resp, body := get(t, ts.URL+path)
		var got map[string]string
		if err := json.Unmarshal([]byte(body), &got); err != nil || got[field] == "" {
			t.Fatalf("GET %s: expected a %q field, got %d %q", path, field, resp.StatusCode, body)
		}
		if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
			t.Errorf("GET %s: Cache-Control = %q, want no-store", path, cc)
		}
	}

	resp, body := get(t, ts.URL+"/")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "<title>suGen</title>") {
		t.Fatalf("GET /: expected the page, got %d", resp.StatusCode)
	}
	for _, asset := range []string{"/index.js", "/index.css", "/css/colors.css", "/js/password.js", "/js/passphrase.js", "/js/notification.js", "/js/copyToClipboard.js", "/assets/favicon.ico"} {
		if resp, _ := get(t, ts.URL+asset); resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status %d", asset, resp.StatusCode)
		}
	}
}

func TestServer_GeneratorsAreGetOnly(t *testing.T) {
	ts := newTestServer(t)
	for _, path := range []string{"/password/", "/passphrase/"} {
		resp, err := http.Post(ts.URL+path, "text/plain", nil)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		// The page route catches other methods and answers 404.
		if resp.StatusCode == http.StatusOK || strings.Contains(string(body), "pass") {
			t.Fatalf("POST %s: status %d, body %q", path, resp.StatusCode, body)
		}
	}
}

func TestServer_HasTimeouts(t *testing.T) {
	srv, err := newServer(":0")
	if err != nil {
		t.Fatal(err)
	}
	if srv.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout is not set")
	}
	// Without IdleTimeout (and with no ReadTimeout to fall back on) a
	// keep-alive connection is held open for as long as the client likes.
	if srv.IdleTimeout <= 0 {
		t.Error("IdleTimeout is not set")
	}
}

func TestLogRequests_OneRequestIsOneLine(t *testing.T) {
	// The path is logged after percent-decoding, so a %0A in a URL used to
	// let anyone write a line of their choosing into the log.
	buf := captureLog(t)
	ts := httptest.NewServer(logRequests(http.NotFoundHandler()))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/password/x%0A2026/10/01%2000:00:00%20GET%20/admin%20forged")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one log line, got %d: %q", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "GET /password/x%0A2026") {
		t.Fatalf("expected the escaped path, got %q", lines[0])
	}
}

func TestLogRequests_NeverLogsTheQuery(t *testing.T) {
	buf := captureLog(t)
	ts := httptest.NewServer(logRequests(http.NotFoundHandler()))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/password/?passwordLength=17&minimumDigits=4")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if strings.Contains(buf.String(), "passwordLength") || strings.Contains(buf.String(), "?") {
		t.Fatalf("the query string reached the log: %q", buf.String())
	}
}

func TestListenAddr(t *testing.T) {
	t.Setenv("PORT", "")
	if got := listenAddr(); got != ":3000" {
		t.Errorf("default: got %q, want :3000", got)
	}
	t.Setenv("PORT", "8081")
	if got := listenAddr(); got != ":8081" {
		t.Errorf("PORT=8081: got %q, want :8081", got)
	}
}
