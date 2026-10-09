package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	dto "github.com/prometheus/client_model/go"
	"unihub-workshop/internal/metrics"
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

func TestObservabilityRecordsCommittedHTTPStatus(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		handler http.HandlerFunc
	}{
		{"implicit OK", http.StatusOK, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
			w.WriteHeader(http.StatusInternalServerError)
		}},
		{"first final header", http.StatusAccepted, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			w.WriteHeader(http.StatusInternalServerError)
		}},
		{"empty response", http.StatusOK, func(w http.ResponseWriter, r *http.Request) {}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			previous := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
			t.Cleanup(func() { slog.SetDefault(previous) })
			path := "/observability/" + strconv.Itoa(test.status) + "/" + test.name
			counter := metrics.RequestsTotal.WithLabelValues(http.MethodGet, path, strconv.Itoa(test.status))
			counterValue := func() float64 {
				var metric dto.Metric
				if err := counter.Write(&metric); err != nil {
					t.Fatal(err)
				}
				return metric.GetCounter().GetValue()
			}
			before := counterValue()
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "http://example.test/status", nil)
			request.URL.Path = path
			StructuredLogger(MetricsMiddleware(test.handler)).ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("HTTP status: got %d, want %d", response.Code, test.status)
			}
			var fields map[string]interface{}
			if err := json.Unmarshal(output.Bytes(), &fields); err != nil {
				t.Fatal(err)
			}
			if fields["status"] != float64(test.status) {
				t.Errorf("logged status %v differs from HTTP status %d", fields["status"], test.status)
			}
			if actual := counterValue() - before; actual != 1 {
				t.Errorf("expected one request with committed HTTP status, got %v", actual)
			}
		})
	}
}

func TestObservabilityPreservesResponseFlushing(t *testing.T) {
	response := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("stream"))
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("wrapped response cannot flush: %v", err)
		}
	})
	StructuredLogger(MetricsMiddleware(handler)).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if !response.Flushed {
		t.Error("response was not flushed")
	}
}
