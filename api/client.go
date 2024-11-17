package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jorenkoyen/conter/version"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	base *url.URL
	http *http.Client
}

// NewClientFromEnv will create a new API client for the current environment it's running in.
func NewClientFromEnv() *Client {
	return &Client{
		base: &url.URL{Scheme: "http", Host: "127.0.0.1:6640"},
		http: http.DefaultClient,
	}
}

// NewClient creates a new API client with the specified base URL and HTTP client.
func NewClient(base *url.URL, http *http.Client) *Client {
	return &Client{
		base: base,
		http: http,
	}
}

// TODO: enhance error parsing ...
// checkResponseError will check the response of the HTTP server if it contains any error.
func checkResponseError(resp *http.Response, body []byte) error {
	if resp.StatusCode < http.StatusBadRequest {
		return nil
	}

	apiError := StatusError{StatusCode: resp.StatusCode}

	err := json.Unmarshal(body, &apiError)
	if err != nil {
		// Use the full body as the message if we fail to decode a response.
		apiError.ErrorMessage = string(body)
	}

	return apiError
}

// do will execute the HTTP request.
func (c *Client) do(ctx context.Context, method, path string, reqBody io.Reader, respData any) error {
	endpoint := c.base.JoinPath(path)
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.UserAgent())

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	output, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err = checkResponseError(res, output); err != nil {
		return err
	}

	if len(output) > 0 && respData != nil {
		if err = json.Unmarshal(output, respData); err != nil {
			return err
		}
	}
	return nil
}

// CertificateList will return the current certificates known in the system.
func (c *Client) CertificateList(ctx context.Context) ([]Certificate, error) {
	var list []Certificate
	if err := c.do(ctx, http.MethodGet, "/api/certificates", nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// CertificateInspect will return the detailed information about the certificate for the given domain.
func (c *Client) CertificateInspect(ctx context.Context, domain string) (*Certificate, error) {
	var cert Certificate
	if err := c.do(ctx, http.MethodGet, "/api/certificates/"+domain, nil, &cert); err != nil {
		return nil, err
	}
	return &cert, nil
}

// CertificateRenew will renew an existing certificate for the given domain.
func (c *Client) CertificateRenew(ctx context.Context, domain string) error {
	endpoint := fmt.Sprintf("/api/certificates/%s/renew", domain)
	return c.do(ctx, http.MethodPost, endpoint, nil, nil)
}
