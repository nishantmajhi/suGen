// Command server runs the suGen HTTP API and serves its static frontend.
package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nishantmajhi/suGen/internal/handler"
	"github.com/nishantmajhi/suGen/internal/wordlist"
	"github.com/nishantmajhi/suGen/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	srv, err := newServer(listenAddr())
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("suGen listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("shutting down...")
	return srv.Shutdown(shutdownCtx)
}

// newServer wires the routes, the request log and the timeouts into a
// server listening on addr.
func newServer(addr string) (*http.Server, error) {
	staticFS, err := fs.Sub(web.FS, "static")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /password/", handler.NewPasswordHandler())
	mux.Handle("GET /passphrase/", handler.NewPassphraseHandler(
		wordlist.Default(),
		wordlist.Adjectives(),
		wordlist.Nouns(),
	))
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	return &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}, nil
}

// listenAddr returns ":<PORT>" using the PORT environment variable if set,
// defaulting to 3000, the port the PHP version was run on.
func listenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	return ":" + port
}

// logRequests logs the method, path and duration of each request. The
// query string, where the generation options are, is never logged. The
// path is logged still escaped, so a %0A in a URL cannot start a new line.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.EscapedPath(), time.Since(start))
	})
}
