package io

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPModule provides HTTP client functions for go-json programs.
type HTTPModule struct {
	security *SecurityConfig
	config   map[string]any
}

// NewHTTPModule creates a new HTTP I/O module.
func NewHTTPModule(security *SecurityConfig) *HTTPModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &HTTPModule{security: security}
}

func (m *HTTPModule) Name() string { return "http" }

func (m *HTTPModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *HTTPModule) Functions() map[string]any {
	return map[string]any{
		"get":    m.httpGet,
		"post":   m.httpPost,
		"put":    m.httpPut,
		"patch":  m.httpPatch,
		"delete": m.httpDelete,
	}
}

func (m *HTTPModule) httpGet(params ...any) (any, error) {
	return m.doRequest("GET", params)
}

func (m *HTTPModule) httpPost(params ...any) (any, error) {
	return m.doRequest("POST", params)
}

func (m *HTTPModule) httpPut(params ...any) (any, error) {
	return m.doRequest("PUT", params)
}

func (m *HTTPModule) httpPatch(params ...any) (any, error) {
	return m.doRequest("PATCH", params)
}

func (m *HTTPModule) httpDelete(params ...any) (any, error) {
	return m.doRequest("DELETE", params)
}

func (m *HTTPModule) doRequest(method string, params []any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("http.%s: url is required", strings.ToLower(method))
	}

	rawURL, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("http.%s: url must be a string", strings.ToLower(method))
	}

	if err := m.security.ValidateHTTPRequest(rawURL); err != nil {
		return nil, err
	}

	opts := extractHTTPOpts(params)

	timeout := time.Duration(m.security.HTTP.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if t, ok := opts["timeout"]; ok {
		if tf, ok := toFloat64Val(t); ok {
			timeout = time.Duration(tf) * time.Millisecond
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var bodyReader io.Reader
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if body, ok := opts["body"]; ok {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("http.%s: cannot serialize body: %s", strings.ToLower(method), err.Error())
			}
			bodyReader = strings.NewReader(string(bodyBytes))
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("http.%s: %s", strings.ToLower(method), err.Error())
	}

	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if headers, ok := opts["headers"].(map[string]any); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	m.applyAuth(req, opts)

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects (max 10)")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http.%s: %s", strings.ToLower(method), err.Error())
	}
	defer resp.Body.Close()

	maxSize := m.security.HTTP.MaxResponseSize
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024
	}
	limitedReader := io.LimitReader(resp.Body, maxSize+1)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("http.%s: error reading response: %s", strings.ToLower(method), err.Error())
	}
	if int64(len(bodyBytes)) > maxSize {
		return nil, fmt.Errorf("http.%s: response exceeds max size (%d bytes)", strings.ToLower(method), maxSize)
	}

	respHeaders := make(map[string]any)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	var body any
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			body = string(bodyBytes)
		}
	} else {
		body = string(bodyBytes)
	}

	return map[string]any{
		"status":  resp.StatusCode,
		"headers": respHeaders,
		"body":    body,
	}, nil
}

func (m *HTTPModule) applyAuth(req *http.Request, opts map[string]any) {
	auth, ok := opts["auth"].(map[string]any)
	if !ok {
		return
	}

	authType, _ := auth["type"].(string)
	switch strings.ToLower(authType) {
	case "bearer":
		token, _ := auth["token"].(string)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case "basic":
		username, _ := auth["username"].(string)
		password, _ := auth["password"].(string)
		req.SetBasicAuth(username, password)
	default:
	}
}

func extractHTTPOpts(params []any) map[string]any {
	opts := make(map[string]any)
	for i := 1; i < len(params); i++ {
		if m, ok := params[i].(map[string]any); ok {
			for k, v := range m {
				opts[k] = v
			}
		}
	}
	return opts
}

func toFloat64Val(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
