package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"unihub-workshop/internal/presence"
)

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "203.0.113.195",
		},
		{
			name:       "X-Forwarded-For multiple IPs (first is client)",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "203.0.113.195",
		},
		{
			name:       "X-Real-IP fallback",
			headers:    map[string]string{"X-Real-IP": "198.51.100.1"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "198.51.100.1",
		},
		{
			name:       "RemoteAddr fallback with port",
			headers:    map[string]string{},
			remoteAddr: "192.0.2.1:54321",
			expectedIP: "192.0.2.1",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "192.0.2.1",
			expectedIP: "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/workshops", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := extractClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("extractClientIP() = %v, want %v", ip, tt.expectedIP)
			}
		})
	}
}

func TestExtractWorkshopID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		expectedID string
	}{
		{
			name:       "Standard workshop path",
			path:       "/api/v1/workshops/ws-uuid-1234",
			expectedID: "ws-uuid-1234",
		},
		{
			name:       "Waiting room path",
			path:       "/api/v1/registrations/waiting-room/ws-uuid-5678",
			expectedID: "ws-uuid-5678",
		},
		{
			name:       "Query param path",
			path:       "/api/v1/registrations?workshop_id=ws-uuid-9999",
			expectedID: "ws-uuid-9999",
		},
		{
			name:       "No workshop ID in path",
			path:       "/api/v1/auth/login",
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			id := extractWorkshopID(req)
			if id != tt.expectedID {
				t.Errorf("extractWorkshopID(%s) = %v, want %v", tt.path, id, tt.expectedID)
			}
		})
	}
}

func TestPresenceMiddleware_Execution(t *testing.T) {
	tracker := presence.NewPresenceTracker(nil, 3)
	pm := NewPresenceMiddleware(tracker)

	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	mw := pm.Handler(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/workshops/ws-101", nil)
	req.RemoteAddr = "192.168.1.100:8080"
	// Set authenticated user context
	ctx := context.WithValue(req.Context(), UserIDKey, "user-student-001")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Fatalf("expected nextHandler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
