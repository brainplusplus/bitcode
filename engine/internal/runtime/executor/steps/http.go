package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
)

type HTTPHandler struct{}

func (h *HTTPHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	method := step.Method
	if method == "" {
		method = "GET"
	}

	url := interpolate(step.URL, execCtx)

	var bodyReader io.Reader
	if step.Body != nil {
		bodyBytes, err := json.Marshal(step.Body)
		if err != nil {
			return fmt.Errorf("failed to marshal HTTP body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range step.Headers {
		req.Header.Set(k, interpolate(v, execCtx))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read HTTP response: %w", err)
	}

	var result any
	if err := json.Unmarshal(respBody, &result); err != nil {
		result = string(respBody)
	}

	varName := step.Into
	if varName == "" {
		varName = "http_result"
	}
	execCtx.Variables[varName] = result
	execCtx.Result = result
	return nil
}
