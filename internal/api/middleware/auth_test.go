package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devaldrete/dotbrain/internal/api/middleware"
	"github.com/gin-gonic/gin"
)

func newTestRouter(apiKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Health bypasses auth — registered outside the auth group
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// All other routes require auth
	v1 := r.Group("/api/v1")
	if apiKey != "" {
		v1.Use(middleware.APIKeyAuth(apiKey))
	}
	v1.GET("/workflows", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	})

	return r
}

// TestAuthMiddleware_MissingHeader verifies that a request with no
// Authorization header is rejected with 401.
func TestAuthMiddleware_MissingHeader(t *testing.T) {
	router := newTestRouter("secret-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/workflows", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestAuthMiddleware_WrongKey verifies that a request with an incorrect
// API key is rejected with 401.
func TestAuthMiddleware_WrongKey(t *testing.T) {
	router := newTestRouter("secret-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/workflows", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestAuthMiddleware_CorrectKey verifies that a request with the correct
// API key passes through to the handler.
func TestAuthMiddleware_CorrectKey(t *testing.T) {
	router := newTestRouter("secret-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/workflows", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// TestAuthMiddleware_MalformedHeader verifies that a malformed Authorization
// header (not "Bearer <token>") is rejected with 401.
func TestAuthMiddleware_MalformedHeader(t *testing.T) {
	router := newTestRouter("secret-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/workflows", nil)
	req.Header.Set("Authorization", "secret-key") // missing "Bearer " prefix
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestAuthMiddleware_HealthBypass verifies that the health endpoint returns
// 200 without any Authorization header.
func TestAuthMiddleware_HealthBypass(t *testing.T) {
	router := newTestRouter("secret-key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: health must bypass auth", w.Code)
	}
}

// TestAuthMiddleware_DisabledWhenNoKey verifies that when no API key is
// configured, all requests pass through without any auth check.
func TestAuthMiddleware_DisabledWhenNoKey(t *testing.T) {
	router := newTestRouter("") // empty key = auth disabled

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/workflows", nil)
	// No Authorization header — should still succeed
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with auth disabled, got %d", w.Code)
	}
}
