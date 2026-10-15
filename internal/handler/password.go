package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/nishantmajhi/suGen/internal/generator"
	"github.com/nishantmajhi/suGen/internal/httputil"
)

// PasswordHandler serves GET /password/, generating a password from
// query-string options. Field names and defaults match the original API.
type PasswordHandler struct{}

// NewPasswordHandler constructs a PasswordHandler.
func NewPasswordHandler() *PasswordHandler {
	return &PasswordHandler{}
}

func (h *PasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	opts := generator.PasswordOptions{
		MixedCase:                httputil.QueryBool(r, "mixedCase", true),
		Length:                   httputil.QueryInt(r, "passwordLength", 12),
		Numbers:                  httputil.QueryBool(r, "numbers", true),
		MinimumDigits:            httputil.QueryInt(r, "minimumDigits", 0),
		SpecialCharacters:        httputil.QueryBool(r, "specialCharacters", true),
		MinimumSpecialCharacters: httputil.QueryInt(r, "minimumSpecialCharacters", 0),
		AvoidAmbiguousCharacters: httputil.QueryBool(r, "avoidAmbiguousCharacters", true),
	}

	password, err := generator.GeneratePassword(opts)
	if err != nil {
		if errors.Is(err, generator.ErrPasswordTooShort) || errors.Is(err, generator.ErrPasswordTooLong) {
			// The original PHP endpoint reports this as a 200 with an
			// "error" field, so we preserve that for API compatibility.
			httputil.JSON(w, http.StatusOK, map[string]string{"error": err.Error()})
			return
		}
		log.Printf("password handler: %v", err)
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"password": password})
}
