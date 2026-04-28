package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type httpBridge struct {
	client     tls_client.HttpClient
	cookieJars map[string]http.CookieJar
	mu         sync.Mutex
}

func newHTTPBridge() *httpBridge {
	jar := tls_client.NewCookieJar()
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithCookieJar(jar),
		tls_client.WithRandomTLSExtensionOrder(),
	)
	if err != nil {
		client, _ = tls_client.NewHttpClient(tls_client.NewNoopLogger())
	}
	return &httpBridge{
		client:     client,
		cookieJars: make(map[string]http.CookieJar),
	}
}

func (h *httpBridge) Get(url string, opts *HTTPOptions) (*HTTPResponse, error) {
	return h.doRequest("GET", url, opts)
}

func (h *httpBridge) Post(url string, opts *HTTPOptions) (*HTTPResponse, error) {
	return h.doRequest("POST", url, opts)
}

func (h *httpBridge) Put(url string, opts *HTTPOptions) (*HTTPResponse, error) {
	return h.doRequest("PUT", url, opts)
}

func (h *httpBridge) Patch(url string, opts *HTTPOptions) (*HTTPResponse, error) {
	return h.doRequest("PATCH", url, opts)
}

func (h *httpBridge) Delete(url string, opts *HTTPOptions) (*HTTPResponse, error) {
	return h.doRequest("DELETE", url, opts)
}

func (h *httpBridge) resolveClient(opts *HTTPOptions) (tls_client.HttpClient, error) {
	if opts == nil {
		return h.client, nil
	}

	needsNew := opts.Profile != "" || opts.Proxy != "" || opts.Timeout > 0 ||
		opts.CookieJar != "" || opts.FollowRedirects != nil || opts.InsecureSkipVerify

	if !needsNew {
		return h.client, nil
	}

	clientOpts := []tls_client.HttpClientOption{
		tls_client.WithRandomTLSExtensionOrder(),
	}

	if opts.Timeout > 0 {
		clientOpts = append(clientOpts, tls_client.WithTimeoutSeconds(opts.Timeout/1000))
	} else {
		clientOpts = append(clientOpts, tls_client.WithTimeoutSeconds(30))
	}

	if opts.Profile != "" {
		if profile, ok := profiles.MappedTLSClients[opts.Profile]; ok {
			clientOpts = append(clientOpts, tls_client.WithClientProfile(profile))
		} else {
			clientOpts = append(clientOpts, tls_client.WithClientProfile(profiles.Chrome_133))
		}
	} else {
		clientOpts = append(clientOpts, tls_client.WithClientProfile(profiles.Chrome_133))
	}

	if opts.CookieJar != "" {
		h.mu.Lock()
		jar, exists := h.cookieJars[opts.CookieJar]
		if !exists {
			jar = tls_client.NewCookieJar()
			h.cookieJars[opts.CookieJar] = jar
		}
		h.mu.Unlock()
		clientOpts = append(clientOpts, tls_client.WithCookieJar(jar))
	} else {
		clientOpts = append(clientOpts, tls_client.WithCookieJar(tls_client.NewCookieJar()))
	}

	if opts.FollowRedirects != nil && !*opts.FollowRedirects {
		clientOpts = append(clientOpts, tls_client.WithNotFollowRedirects())
	}

	if opts.InsecureSkipVerify {
		clientOpts = append(clientOpts, tls_client.WithInsecureSkipVerify())
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), clientOpts...)
	if err != nil {
		return nil, NewError(ErrHTTPError, fmt.Sprintf("failed to create HTTP client: %s", err))
	}

	if opts.Proxy != "" {
		if err := client.SetProxy(opts.Proxy); err != nil {
			return nil, NewError(ErrHTTPError, fmt.Sprintf("invalid proxy URL: %s", err))
		}
	}

	return client, nil
}

func (h *httpBridge) doRequest(method, url string, opts *HTTPOptions) (*HTTPResponse, error) {
	client, err := h.resolveClient(opts)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if opts != nil && opts.Body != nil {
		bodyBytes, marshalErr := json.Marshal(opts.Body)
		if marshalErr != nil {
			return nil, NewError(ErrHTTPError, fmt.Sprintf("failed to marshal body: %s", marshalErr))
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, NewError(ErrHTTPError, fmt.Sprintf("failed to create request: %s", err))
	}

	if opts != nil {
		if len(opts.HeaderOrder) > 0 {
			req.Header[http.HeaderOrderKey] = opts.HeaderOrder
		}
		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}
		if bodyReader != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, NewRetryableError(ErrHTTPTimeout, fmt.Sprintf("request failed: %s", err))
	}
	defer resp.Body.Close()

	const maxBodySize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxBodySize)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, NewRetryableError(ErrHTTPError, fmt.Sprintf("failed to read response: %s", err))
	}

	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	var body any
	if json.Valid(respBody) {
		json.Unmarshal(respBody, &body)
	} else {
		body = string(respBody)
	}

	return &HTTPResponse{
		Status:  resp.StatusCode,
		Headers: headers,
		Body:    body,
	}, nil
}
