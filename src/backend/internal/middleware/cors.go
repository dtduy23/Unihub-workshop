package middleware

import (
	"net/http"
	"strings"
)

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(origins string) func(http.Handler) http.Handler {
	allowedMap := make(map[string]struct{})
	allowAll := false

	for _, o := range strings.Split(origins, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed == "*" {
			allowAll = true
		} else if trimmed != "" {
			allowedMap[trimmed] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Luôn thêm Vary: Origin để các tầng Cache/Proxy/CDN không lưu lẫn response giữa các domain
			w.Header().Add("Vary", "Origin")

			_, inMap := allowedMap[origin]
			isAllowed := (allowAll && origin != "") || inMap

			if isAllowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Idempotency-Key")
				w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, Retry-After, X-Total-Count")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			// Xử lý Preflight Request (OPTIONS)
			if r.Method == http.MethodOptions {
				if origin != "" && !isAllowed {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
