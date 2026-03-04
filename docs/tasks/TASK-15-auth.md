# TASK-15 — Authentication Middleware

**Phase:** 8 — Security  
**Priority:** High (required before any network-accessible deployment)  
**Depends on:** nothing  
**Files affected:** `internal/api/router.go`, new `internal/api/middleware/auth.go`, `cmd/dotbrain/main.go`, `.env.example`

---

## Problem

Every API endpoint is publicly accessible with zero authentication. Anyone who can reach the server can:

- Create and trigger arbitrary workflows
- Read all execution history including `node_executions.input_data` (which contains LLM prompts, API payloads, and any sensitive trigger data)
- Consume API keys stored in node params

This is not acceptable for any deployment beyond `localhost`.

---

## Goal

Implement API key authentication via a Gin middleware. A static API key is configured via environment variable. All `/api/v1` routes require a valid `Authorization: Bearer <key>` header. Infrastructure endpoints (`/health`, `/readiness`) remain unauthenticated.

This is the minimum viable gate. More sophisticated auth (JWT, OAuth, multi-user) can be layered on top later.

---

## Implementation

### Middleware

```go
// internal/api/middleware/auth.go

package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

// APIKeyAuth returns a Gin middleware that validates the Authorization header.
// The expected token is passed at construction time (read from env by the caller).
func APIKeyAuth(expectedKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Skip auth for unauthenticated paths
        // (router registers these outside the authenticated group, so this
        //  middleware won't be reached for them — but defense in depth)

        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "missing Authorization header",
            })
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "Authorization header must be in the format: Bearer <token>",
            })
            return
        }

        if parts[1] != expectedKey {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "invalid API key",
            })
            return
        }

        c.Next()
    }
}
```

### Router Integration

Auth is applied to the `/api/v1` group but **not** to `/health` and `/readiness`, which must remain accessible to Kubernetes probes without credentials:

```go
func (a *API) NewRouter(apiKey string) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(gin.Logger())

    // Unauthenticated infrastructure endpoints
    r.GET("/api/v1/health", a.healthCheckHandler)
    r.GET("/api/v1/readiness", a.readinessHandler)

    // All other routes require auth
    v1 := r.Group("/api/v1")
    if apiKey != "" {
        v1.Use(middleware.APIKeyAuth(apiKey))
    }
    // ... register workflow and run routes on v1
}
```

When `apiKey` is empty (not configured), auth middleware is skipped entirely — preserving the current behavior for local development without requiring env configuration.

### Environment Variable

```bash
# .env.example
API_KEY=your-secret-key-here
```

Read in `main.go`:

```go
apiKey := os.Getenv("API_KEY")
if apiKey == "" {
    slog.Warn("API_KEY not set — authentication is disabled")
}
router := api.NewRouter(apiKey)
```

---

## Frontend Integration

The SvelteKit frontend's `web/src/lib/api.ts` must include the key in every request. The key is injected at build/deploy time via an environment variable:

```ts
// web/src/lib/api.ts
const API_KEY = import.meta.env.VITE_API_KEY ?? '';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
        headers: {
            'Content-Type': 'application/json',
            ...(API_KEY ? { 'Authorization': `Bearer ${API_KEY}` } : {}),
            ...options?.headers,
        },
        ...options
    });
    // ...
}
```

```bash
# web/.env
VITE_API_KEY=your-secret-key-here
```

---

## Acceptance Criteria

- [ ] Requests to `/api/v1/workflows` without an `Authorization` header return 401
- [ ] Requests with `Authorization: Bearer <wrong-key>` return 401
- [ ] Requests with `Authorization: Bearer <correct-key>` succeed normally
- [ ] `GET /api/v1/health` and `GET /api/v1/readiness` return 200 without any auth header
- [ ] When `API_KEY` env var is empty, all endpoints work without auth (dev mode)
- [ ] A startup log line indicates whether auth is enabled or disabled
- [ ] `go test ./internal/api/...` passes with auth middleware tests
- [ ] Existing tests that do not set an auth header must be updated to include the key, OR the test API is initialized without a key

---

## TDD Approach

```go
// TestAuthMiddleware_MissingHeader — assert 401
func TestAuthMiddleware_MissingHeader(t *testing.T) { ... }

// TestAuthMiddleware_WrongKey — assert 401
func TestAuthMiddleware_WrongKey(t *testing.T) { ... }

// TestAuthMiddleware_CorrectKey — assert request passes through
func TestAuthMiddleware_CorrectKey(t *testing.T) { ... }

// TestAuthMiddleware_HealthBypass — health endpoint returns 200 without header
func TestAuthMiddleware_HealthBypass(t *testing.T) { ... }

// TestAuthMiddleware_DisabledWhenNoKey — no key configured → all requests pass
func TestAuthMiddleware_DisabledWhenNoKey(t *testing.T) { ... }
```

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes; existing tests updated where needed
- `.env.example` documents `API_KEY`
- `docs/core/api.md` updated: note that all endpoints (except health/readiness) require `Authorization: Bearer <key>`
