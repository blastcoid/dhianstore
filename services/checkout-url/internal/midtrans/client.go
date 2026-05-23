// Package midtrans wraps Midtrans Payment Link API calls.
//
// Spec verified via Context7 (/websites/midtrans_reference):
//   - Endpoint: POST {MIDTRANS_API_BASE}/v1/payment-links
//   - Auth: Basic base64(SERVER_KEY + ":")
//   - Response shape: { payment_url, order_id, payment_link_id, token, transaction_status }
//
// The client uses stdlib net/http with an injectable Transport so tests can
// stub responses without spinning up a server.
package midtrans

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/order"
)

const (
	// DefaultTimeout caps each individual HTTP attempt. Retries are independent.
	DefaultTimeout = 10 * time.Second

	endpointPath = "/v1/payment-links"
)

// retryDelays controls how long to sleep before attempt N (0-indexed).
// 3 entries = 1 initial attempt + 2 retries.
var retryDelays = []time.Duration{0, 500 * time.Millisecond, 1500 * time.Millisecond}

// Client posts payment-link requests to Midtrans with retry + timeout.
type Client struct {
	httpClient *http.Client
	apiBase    string
	serverKey  string
	logger     zerolog.Logger
}

// Response is the subset of the Midtrans response we surface to callers.
type Response struct {
	PaymentURL    string `json:"payment_url"`
	OrderID       string `json:"order_id"`
	PaymentLinkID string `json:"payment_link_id"`
	Token         string `json:"token"`
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

// New constructs a client. Pass a custom http.Client (for tests) via WithHTTPClient.
func New(cfg *config.Config, logger zerolog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		apiBase:    cfg.MidtransAPIBase,
		serverKey:  cfg.MidtransServerKey,
		logger:     logger,
	}
}

// WithHTTPClient swaps the underlying http.Client. Tests inject a client
// whose Transport is a stub RoundTripper.
func (c *Client) WithHTTPClient(hc *http.Client) *Client {
	c.httpClient = hc
	return c
}

// CreatePaymentLink posts the body and returns the parsed response.
// Retries: 2 additional attempts on 5xx / network errors with exponential
// backoff. 4xx is never retried (caller error).
func (c *Client) CreatePaymentLink(ctx context.Context, body order.Request) (Response, error) {
	url := strings.TrimRight(c.apiBase, "/") + endpointPath
	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, &Error{Message: "marshal body", Cause: err}
	}

	var lastErr *Error
	for attempt, delay := range retryDelays {
		if delay > 0 {
			c.logger.Warn().
				Int("attempt", attempt+1).
				Dur("delay", delay).
				Msg("midtrans retry")
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return Response{}, &Error{Message: "context cancelled during retry backoff", Cause: ctx.Err()}
			}
		}

		resp, err := c.attempt(ctx, url, payload)
		if err == nil {
			return resp, nil
		}
		// 4xx bubbles up immediately; 5xx and network errors retry.
		var me *Error
		if errors.As(err, &me) && me.StatusCode >= 400 && me.StatusCode < 500 {
			return Response{}, me
		}
		lastErr = me
	}
	if lastErr == nil {
		lastErr = &Error{Message: "midtrans request failed (no attempts recorded)"}
	}
	return Response{}, lastErr
}

func (c *Client) attempt(ctx context.Context, url string, payload []byte) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return Response{}, &Error{Message: "build request", Cause: err}
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || isTimeoutErr(err) {
			return Response{}, &Error{Message: "request timed out", Cause: err}
		}
		return Response{}, &Error{Message: "network error: " + err.Error(), Cause: err}
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return Response{}, &Error{
			Message:    "read response body",
			StatusCode: resp.StatusCode,
			Cause:      readErr,
		}
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		var parsed Response
		if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
			return Response{}, &Error{
				Message:      "midtrans returned non-JSON success body",
				StatusCode:   resp.StatusCode,
				ResponseBody: string(bodyBytes),
				Cause:        err,
			}
		}
		return parsed, nil
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return Response{}, &Error{
			Message:      fmt.Sprintf("rejected request: %d", resp.StatusCode),
			StatusCode:   resp.StatusCode,
			ResponseBody: string(bodyBytes),
		}
	default:
		return Response{}, &Error{
			Message:      fmt.Sprintf("server error: %d", resp.StatusCode),
			StatusCode:   resp.StatusCode,
			ResponseBody: string(bodyBytes),
		}
	}
}

func (c *Client) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.serverKey+":"))
}

// isTimeoutErr unwraps to detect net/url.Error{Timeout()=true}.
func isTimeoutErr(err error) bool {
	type timeoutErr interface{ Timeout() bool }
	var te timeoutErr
	return errors.As(err, &te) && te.Timeout()
}
