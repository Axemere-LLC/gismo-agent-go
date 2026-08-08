package agent

import (
	"crypto/subtle"
	"net/http"
)

// BearerAuth wraps handler, rejecting any request whose Authorization
// header isn't exactly "Bearer <key>" with 401. This is the referee-facing
// half of the outbound key described in gismo-agent-hosting's
// registering-your-agent.md#the-outbound-key — NewHandler's output has no
// such check on its own, since a template with no key configured has
// nothing to compare against.
func BearerAuth(key string, handler http.Handler) http.Handler {
	want := []byte("Bearer " + key)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		// key == "" must never authorize: without this, an unset/empty
		// secret would make "Bearer " (no token) itself a valid credential.
		if key == "" || len(got) != len(want) || subtle.ConstantTimeCompare(got, want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
