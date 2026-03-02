package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var templatePattern = regexp.MustCompile(`\{\{input\.([^}]+)\}\}`)

// ApplyTemplate replaces {{input.field}} placeholders with values from the input map.
func ApplyTemplate(template string, input map[string]any) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		// Extract field name from {{input.fieldName}}
		parts := templatePattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		fieldName := parts[1]
		val, ok := input[fieldName]
		if !ok {
			return ""
		}
		return fmt.Sprintf("%v", val)
	})
}

// HttpNode executes HTTP requests.
type HttpNode struct {
	URL            string
	Method         string
	Headers        map[string]string
	Body           string
	TimeoutSeconds int
}

// NewHttpNode creates an HttpNode from params.
func NewHttpNode(params map[string]any) *HttpNode {
	node := &HttpNode{
		Method:         "GET",
		TimeoutSeconds: 30,
		Headers:        make(map[string]string),
	}

	if url, ok := params["url"].(string); ok {
		node.URL = url
	}
	if method, ok := params["method"].(string); ok {
		node.Method = method
	}
	if body, ok := params["body"].(string); ok {
		node.Body = body
	}
	if timeout, ok := params["timeout_seconds"].(float64); ok {
		node.TimeoutSeconds = int(timeout)
	}
	if headers, ok := params["headers"].(map[string]any); ok {
		for k, v := range headers {
			if vStr, ok := v.(string); ok {
				node.Headers[k] = vStr
			}
		}
	}

	return node
}

// Execute performs the HTTP request with template substitution.
func (n *HttpNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	url := ApplyTemplate(n.URL, input)
	if url == "" {
		return nil, fmt.Errorf("missing required param: url")
	}

	body := ApplyTemplate(n.Body, input)

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, n.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range n.Headers {
		req.Header.Set(k, ApplyTemplate(v, input))
	}

	client := &http.Client{
		Timeout: time.Duration(n.TimeoutSeconds) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Flatten response headers to single values
	headers := make(map[string]any)
	for k, v := range resp.Header {
		if len(v) == 1 {
			headers[k] = v[0]
		} else {
			headers[k] = strings.Join(v, ", ")
		}
	}

	return map[string]any{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
		"headers":     headers,
	}, nil
}
