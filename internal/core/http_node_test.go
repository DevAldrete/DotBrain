package core_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devaldrete/dotbrain/internal/core"
)

func TestApplyTemplate_SubstitutesInputFields(t *testing.T) {
	input := map[string]any{
		"name": "Alice",
		"id":   42.0,
	}

	result := core.ApplyTemplate("Hello {{input.name}}, your ID is {{input.id}}", input)

	expected := "Hello Alice, your ID is 42"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestApplyTemplate_NoPlaceholders(t *testing.T) {
	result := core.ApplyTemplate("no placeholders here", map[string]any{})
	if result != "no placeholders here" {
		t.Errorf("expected unchanged string, got %q", result)
	}
}

func TestApplyTemplate_MissingField(t *testing.T) {
	result := core.ApplyTemplate("Hello {{input.missing}}", map[string]any{})
	if result != "Hello " {
		t.Errorf("expected empty substitution, got %q", result)
	}
}

func TestHttpNode_Execute_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("X-Custom", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"greeting": "hello"}`))
	}))
	defer server.Close()

	node := core.NewHttpNode(map[string]any{
		"url":    server.URL,
		"method": "GET",
	})

	result, err := node.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status_code"] != 200.0 && result["status_code"] != 200 {
		t.Errorf("expected status_code 200, got %v", result["status_code"])
	}

	body, ok := result["body"].(string)
	if !ok {
		t.Fatalf("expected body to be string, got %T", result["body"])
	}
	if body != `{"greeting": "hello"}` {
		t.Errorf("expected body %q, got %q", `{"greeting": "hello"}`, body)
	}

	headers, ok := result["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers to be map, got %T", result["headers"])
	}
	if headers["X-Custom"] != "test-value" {
		t.Errorf("expected X-Custom header, got %v", headers["X-Custom"])
	}
}

func TestHttpNode_Execute_POST_WithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		var body map[string]any
		json.Unmarshal(bodyBytes, &body)

		if body["name"] != "Alice" {
			t.Errorf("expected name Alice in body, got %v", body["name"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"created": true}`))
	}))
	defer server.Close()

	node := core.NewHttpNode(map[string]any{
		"url":    server.URL,
		"method": "POST",
		"body":   `{"name": "{{input.name}}"}`,
		"headers": map[string]any{
			"Content-Type": "application/json",
		},
	})

	result, err := node.Execute(context.Background(), map[string]any{
		"name": "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status_code"] != 201 {
		t.Errorf("expected status_code 201, got %v", result["status_code"])
	}
}

func TestHttpNode_Execute_TemplateSubstitutionInURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/42" {
			t.Errorf("expected path /users/42, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`ok`))
	}))
	defer server.Close()

	node := core.NewHttpNode(map[string]any{
		"url":    server.URL + "/users/{{input.user_id}}",
		"method": "GET",
	})

	result, err := node.Execute(context.Background(), map[string]any{
		"user_id": 42.0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status_code"] != 200 {
		t.Errorf("expected status_code 200, got %v", result["status_code"])
	}
}

func TestHttpNode_Execute_Non2xxPassesThrough(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	node := core.NewHttpNode(map[string]any{
		"url":    server.URL,
		"method": "GET",
	})

	result, err := node.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("non-2xx should not return error, got: %v", err)
	}

	if result["status_code"] != 404 {
		t.Errorf("expected status_code 404, got %v", result["status_code"])
	}
}

func TestHttpNode_Execute_MissingURL(t *testing.T) {
	node := core.NewHttpNode(map[string]any{
		"method": "GET",
	})

	_, err := node.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestHttpNode_Execute_DefaultMethodIsGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected default GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`ok`))
	}))
	defer server.Close()

	node := core.NewHttpNode(map[string]any{
		"url": server.URL,
	})

	_, err := node.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
