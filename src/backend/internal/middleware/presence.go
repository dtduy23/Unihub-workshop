package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"unihub-workshop/internal/presence"

	"github.com/go-chi/chi/v5"
)

// PresenceMiddleware records active client presence (IP or User ID)
// into Redis HyperLogLog for real-time traffic ramp-up detection and autoscaling.
type PresenceMiddleware struct {
	tracker *presence.PresenceTracker
}

// NewPresenceMiddleware creates a new PresenceMiddleware instance.
func NewPresenceMiddleware(tracker *presence.PresenceTracker) *PresenceMiddleware {
	return &PresenceMiddleware{tracker: tracker}
}

// Handler intercepts HTTP requests to track visitor presence.
// It executes asynchronously in the background so it adds 0ms latency to the request.
func (pm *PresenceMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract Client Identifier (UserID if logged in, X-Device-ID if provided, otherwise Client IP)
		clientID := ""
		if uid, ok := r.Context().Value(UserIDKey).(string); ok && uid != "" {
			clientID = uid
		} else if devID := r.Header.Get("X-Device-ID"); devID != "" {
			clientID = devID
		} else {
			clientID = extractClientIP(r)
		}

		// 2. Extract Workshop ID (if viewing or waiting for a specific workshop)
		workshopID := extractWorkshopID(r)

		// 3. Record presence asynchronously to avoid blocking the HTTP request pipeline
		if pm.tracker != nil && clientID != "" {
			go func(cID, wID string) {
				_ = pm.tracker.RecordPresence(context.Background(), cID, wID)
			}(clientID, workshopID)
		}

		next.ServeHTTP(w, r)
	})
}

// extractClientIP extracts client IP address from proxy headers or RemoteAddr.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (standard behind Ingress / Load Balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// Fallback to RemoteAddr (strip port if present)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

// extractWorkshopID attempts to extract workshop ID from URL path or query params.
func extractWorkshopID(r *http.Request) string {
	// 1. Chi URL Param (if available in route context)
	if wID := chi.URLParam(r, "id"); wID != "" {
		return wID
	}
	if wID := chi.URLParam(r, "workshopId"); wID != "" {
		return wID
	}

	// 2. Query param (e.g. ?workshop_id=... or ?workshopId=...)
	if qID := r.URL.Query().Get("workshop_id"); qID != "" {
		return qID
	}
	if qID := r.URL.Query().Get("workshopId"); qID != "" {
		return qID
	}

	// 3. Simple path inspection fallback
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, part := range pathParts {
		if (part == "workshops" || part == "waiting-room") && i+1 < len(pathParts) {
			candidate := pathParts[i+1]
			if candidate != "" && candidate != "presence" && candidate != "list" {
				return candidate
			}
		}
	}

	return ""
}
