package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"unihub-workshop/internal/model"
)

func TestStructuredLoggerRetainsIdentityAndQueueCorrelation(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	user := &model.User{ID: "user-uuid", StudentID: "demo00001", Role: model.RoleStudent, AuthVersion: 1}
	token, err := GenerateJWT("logging-test-secret", user.ID, user.Role)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Use(chimw.RequestID, StructuredLogger)
	router.Use(AuthMiddleware("logging-test-secret", func(context.Context, string) (*model.User, error) { return user, nil }))
	router.Post("/enqueue", func(w http.ResponseWriter, r *http.Request) {
		SetLogCorrelationID(r, "queue-request-uuid")
		w.WriteHeader(http.StatusAccepted)
	})
	request := httptest.NewRequest(http.MethodPost, "/enqueue", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Request-ID", "http-request-uuid")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	var fields map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	for key, expected := range map[string]string{"request_id": "http-request-uuid", "correlation_id": "queue-request-uuid", "user_id": user.ID, "student_id": user.StudentID} {
		if fields[key] != expected {
			t.Errorf("%s: got %v, want %s", key, fields[key], expected)
		}
	}
	if response.Header().Get("X-Request-ID") != "http-request-uuid" || fields["status"] != float64(http.StatusAccepted) {
		t.Fatalf("request metadata mismatch: headers=%v fields=%v", response.Header(), fields)
	}
}
