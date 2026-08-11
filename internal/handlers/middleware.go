package handlers

import (
	"context"
	"net/http"
	"os"
	"strings"

	"servicedesk/internal/auth"
)

type contextKey string

const claimsContextKey contextKey = "claims"

// RequireAuth verifies the Authorization: Bearer <token> header and, if
// valid, attaches the parsed claims to the request context for downstream
// handlers (see ClaimsFromContext).
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")

		claims, err := auth.ParseToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole wraps a handler so only the listed roles may call it.
// Must be used after RequireAuth (either as an outer wrapper, or expect
// claims already present on the context).
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeError(w, http.StatusUnauthorized, "not authenticated")
				return
			}
			if !allowed[claims.Role] {
				writeError(w, http.StatusForbidden, "you do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ClaimsFromContext(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims
}

// Chain applies middlewares in the given order: Chain(h, A, B) == A(B(h)).
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// CORS allows the React frontend to call this API — localhost during
// development, plus any origins listed in CORS_ALLOWED_ORIGINS (a
// comma-separated env var) for production, e.g. a Vercel deployment URL.
func CORS(next http.Handler) http.Handler {
	allowed := allowedOriginsFromEnv()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin, allowed) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedOriginsFromEnv() map[string]bool {
	defaults := []string{
		"http://localhost:5173", "http://127.0.0.1:5173",
		"http://localhost:4173", "http://127.0.0.1:4173",
		"http://localhost:3000",
	}
	set := make(map[string]bool, len(defaults))
	for _, o := range defaults {
		set[o] = true
	}
	// CORS_ALLOWED_ORIGINS="https://myapp.vercel.app,https://myapp.com"
	if extra := os.Getenv("CORS_ALLOWED_ORIGINS"); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				set[o] = true
			}
		}
	}
	return set
}

func isAllowedOrigin(origin string, allowed map[string]bool) bool {
	return origin != "" && allowed[origin]
}
