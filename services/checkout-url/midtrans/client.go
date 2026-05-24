// Package midtrans wraps Midtrans Payment Link API calls. It implements
// checkout.PaymentLinkClient, so consumers depend on the domain interface
// rather than this concrete type.
//
// Spec verified via Context7 (/websites/midtrans_reference):
//   - Endpoint: POST {MIDTRANS_API_BASE}/v1/payment-links
//   - Auth: Basic base64(SERVER_KEY + ":")
//   - Response shape: { payment_url, order_id, payment_link_id, token, transaction_status }
//
// HTTP transport uses Fiber's fasthttp-based client for connection-pool reuse.
// JSON encode/decode uses bytedance/sonic for high-throughput marshaling.
package midtrans

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3/client"

	"github.com/blastcoid/dhianstore/services/checkout-url/checkout"
	"github.com/blastcoid/dhianstore/services/checkout-url/config"
)

const (
	// DefaultTimeout caps each individual HTTP attempt. Retries are independent.
	DefaultTimeout = 10 * time.Second

	endpointPath = "/v1/payment-links"
)

// retryDelays controls the sleep before attempt N (0-indexed). Length determines
// total attempts (1 initial + 2 retries here).
var retryDelays = []time.Duration{0, 500 * time.Millisecond, 1500 * time.Millisecond}

// Client posts payment-link requests to Midtrans with retry + timeout.
type Client struct {
	httpClient *client.Client
	apiBase    string
	authHeader string
}

// Error wraps non-success outcomes (4xx, 5xx after retries, network, timeout).
// StatusCode is 0 for non-HTTP failures.
type Error struct {
	Message      string
	StatusCode   int
	ResponseBody string
	Cause        error
}

func (e *Error) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("midtrans: %s (status=%d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("midtrans: %s", e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

// New constructs a client. Tests substitute the upstream URL via cfg.MidtransAPIBase
// (e.g., pointing at httptest.NewServer) — no transport-level injection needed.
func New(cfg *config.Config) *Client {
	cc := client.New().SetTimeout(DefaultTimeout)
	return &Client{
		httpClient: cc,
		apiBase:    strings.TrimRight(cfg.MidtransAPIBase, "/"),
		authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.MidtransServerKey+":")),
	}
}

// CreatePaymentLink posts the payload and returns the parsed response.
// Retry policy: 2 additional attempts on 5xx and network errors with
// exponential backoff. 4xx is never retried — that is a caller bug Midtrans
// won't accept regardless of retry.
func (c *Client) CreatePaymentLink(ctx context.Context, payload checkout.Payload) (checkout.Response, error) {
	body, err := sonic.Marshal(payload)
	if err != nil {
		return checkout.Response{}, &Error{Message: "marshal payload", Cause: err}
	}

	url := c.apiBase + endpointPath
	var lastErr *Error

	for _, delay := range retryDelays {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return checkout.Response{}, &Error{Message: "context cancelled during backoff", Cause: ctx.Err()}
			}
		}

		resp, err := c.attempt(ctx, url, body)
		if err == nil {
			return resp, nil
		}

		// 4xx bubbles up immediately; 5xx and network errors retry.
		var me *Error
		if errors.As(err, &me) && me.StatusCode >= 400 && me.StatusCode < 500 {
			return checkout.Response{}, me
		}
		lastErr = me
	}

	if lastErr == nil {
		lastErr = &Error{Message: "request failed (no attempts recorded)"}
	}
	return checkout.Response{}, lastErr
}

func (c *Client) attempt(ctx context.Context, url string, body []byte) (checkout.Response, error) {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", c.authHeader).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetRawBody(body).
		Post(url)
	if err != nil {
		return checkout.Response{}, &Error{Message: "network: " + err.Error(), Cause: err}
	}
	defer resp.Close()

	status := resp.StatusCode()
	bodyBytes := resp.Body()

	switch {
	case status >= 200 && status < 300:
		var parsed checkout.Response
		if err := sonic.Unmarshal(bodyBytes, &parsed); err != nil {
			return checkout.Response{}, &Error{
				Message:      "non-JSON success body",
				StatusCode:   status,
				ResponseBody: string(bodyBytes),
				Cause:        err,
			}
		}
		return parsed, nil
	case status >= 400 && status < 500:
		return checkout.Response{}, &Error{
			Message:      fmt.Sprintf("rejected: %d", status),
			StatusCode:   status,
			ResponseBody: string(bodyBytes),
		}
	default:
		return checkout.Response{}, &Error{
			Message:      fmt.Sprintf("server error: %d", status),
			StatusCode:   status,
			ResponseBody: string(bodyBytes),
		}
	}
}
