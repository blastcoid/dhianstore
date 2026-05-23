package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/midtrans"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/order"
)

// roundTripperFunc adapts a closure to http.RoundTripper.
type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func successResponseBody() string {
	r := midtrans.Response{
		PaymentURL:    "https://app.sandbox.midtrans.com/payment-links/test-link",
		OrderID:       "returned-order-id",
		PaymentLinkID: "plink_test",
		Token:         "tok_test",
	}
	b, _ := json.Marshal(r)
	return string(b)
}

func makeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": {"application/json"}},
	}
}

// newTestApp constructs a wired *fiber.App with the supplied RoundTripper.
// fn applies after defaults so individual tests can tweak fields like
// RateLimitPerMin without copy-pasting the whole config.
func newTestApp(t *testing.T, rt http.RoundTripper, fn func(*config.Config)) *fiber.App {
	t.Helper()
	cfg := &config.Config{
		MidtransServerKey:  "SB-Mid-server-testkey123",
		MidtransClientKey:  "SB-Mid-client-testkey123",
		MidtransMerchantID: "Gtest1234",
		MidtransAPIBase:    "https://api.sandbox.midtrans.com",
		CustomerRequired:   true,
		EnabledPayments:    []string{"other_qris"},
		ExpiryDuration:     15,
		ExpiryUnit:         "minutes",
		Port:               8080,
		LogLevel:           "fatal",
		AppEnv:             "test",
		RateLimitPerMin:    1000,
	}
	if fn != nil {
		fn(cfg)
	}
	logger := zerolog.New(io.Discard)
	client := midtrans.New(cfg, logger).WithHTTPClient(&http.Client{
		Transport: rt,
		Timeout:   2 * time.Second,
	})
	return New(cfg, logger, client)
}

// doGet issues a GET request through app.Test and returns the response.
//
// Fiber's app.Test defaults to a 1s timeout, which is too short for tests
// that exercise the Midtrans client's retry path (3 attempts with backoffs
// 0+500ms+1500ms = ≥2s). We bump it to 10s to cover the worst case.
func doGet(t *testing.T, app *fiber.App, url string) *http.Response {
	t.Helper()
	resp, err := app.Test(
		httptest.NewRequest(http.MethodGet, url, nil),
		fiber.TestConfig{Timeout: 10 * time.Second},
	)
	require.NoError(t, err)
	return resp
}

// readJSON decodes the body as JSON into a map.
func readJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if len(body) == 0 {
		return nil
	}
	m := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

// alwaysOK returns a Midtrans 200 every call. Each invocation captures the
// outgoing request into a slice for later assertion.
func alwaysOK(captured *[]*http.Request) http.RoundTripper {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// Drain and re-attach body so the test can re-read.
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))
		// Store the request with a copy of the body for assertions.
		stored := req.Clone(req.Context())
		stored.Body = io.NopCloser(bytes.NewReader(body))
		*captured = append(*captured, stored)
		return makeResponse(200, successResponseBody()), nil
	})
}

func TestHealth_Returns200(t *testing.T) {
	t.Parallel()
	captured := []*http.Request{}
	a := newTestApp(t, alwaysOK(&captured), nil)
	resp := doGet(t, a, "/health")
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, map[string]any{"status": "ok"}, readJSON(t, resp))
	require.Empty(t, captured, "/health must not call Midtrans")
}

func TestCheckout_HappySingleItem_Redirects302(t *testing.T) {
	t.Parallel()
	captured := []*http.Request{}
	a := newTestApp(t, alwaysOK(&captured), nil)
	resp := doGet(t, a, "/checkout?products=grw7y67xo5%3A1")
	require.Equal(t, 302, resp.StatusCode)
	require.Equal(t, "https://app.sandbox.midtrans.com/payment-links/test-link", resp.Header.Get("Location"))
	require.Len(t, captured, 1)
}

func TestCheckout_HappyMultiItem_SendsCorrectMidtransBody(t *testing.T) {
	t.Parallel()
	captured := []*http.Request{}
	a := newTestApp(t, alwaysOK(&captured), nil)

	resp := doGet(t, a, "/checkout?products=grw7y67xo5%3A2%2Czmis5llkew%3A1&coupon=ADHA2026&cart_origin=meta_shops&fbclid=abc123")
	require.Equal(t, 302, resp.StatusCode)
	require.Len(t, captured, 1)

	out := captured[0]
	require.Equal(t, "https://api.sandbox.midtrans.com/v1/payment-links", out.URL.String())

	body, _ := io.ReadAll(out.Body)
	var got order.Request
	require.NoError(t, json.Unmarshal(body, &got))

	require.Equal(t, 90000*2+75000*1, got.TransactionDetails.GrossAmount)
	require.Equal(t, []order.ItemDetail{
		{ID: "grw7y67xo5", Name: "Product A", Price: 90000, Quantity: 2},
		{ID: "zmis5llkew", Name: "Product B", Price: 75000, Quantity: 1},
	}, got.ItemDetails)
	require.True(t, got.CustomerRequired)
	require.Equal(t, []string{"other_qris"}, got.EnabledPayments)
	require.Equal(t, order.Expiry{Duration: 15, Unit: "minutes"}, got.Expiry)
	require.Equal(t, "ADHA2026", got.CustomField1)
	require.Equal(t, "meta_shops", got.CustomField2)
	require.Equal(t, "abc123", got.CustomField3)

	// customer_details must NOT appear in serialized JSON (sandbox rejects).
	asMap := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &asMap))
	require.NotContains(t, asMap, "customer_details")
}

func TestCheckout_SendsBasicAuthHeader(t *testing.T) {
	t.Parallel()
	captured := []*http.Request{}
	a := newTestApp(t, alwaysOK(&captured), nil)
	resp := doGet(t, a, "/checkout?products=grw7y67xo5%3A1")
	require.Equal(t, 302, resp.StatusCode)
	require.Len(t, captured, 1)

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("SB-Mid-server-testkey123:"))
	require.Equal(t, expectedAuth, captured[0].Header.Get("Authorization"))
}

func TestCheckout_MissingProducts_400(t *testing.T) {
	t.Parallel()
	captured := []*http.Request{}
	a := newTestApp(t, alwaysOK(&captured), nil)
	resp := doGet(t, a, "/checkout")
	require.Equal(t, 400, resp.StatusCode)
	body := readJSON(t, resp)
	require.Equal(t, "invalid_query", body["error"])
	require.NotEmpty(t, body["request_id"])
	require.Empty(t, captured, "must not call Midtrans on parse error")
}

func TestCheckout_UnknownProduct_400(t *testing.T) {
	t.Parallel()
	captured := []*http.Request{}
	a := newTestApp(t, alwaysOK(&captured), nil)
	resp := doGet(t, a, "/checkout?products=nonexistent%3A1")
	require.Equal(t, 400, resp.StatusCode)
	body := readJSON(t, resp)
	require.Equal(t, "product_not_found", body["error"])
	require.Equal(t, "nonexistent", body["product_id"])
	require.NotEmpty(t, body["request_id"])
	require.Empty(t, captured)
}

func TestCheckout_Midtrans5xxAllAttempts_502(t *testing.T) {
	t.Parallel()
	var calls int32
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return makeResponse(500, "server error"), nil
	})
	a := newTestApp(t, rt, nil)
	resp := doGet(t, a, "/checkout?products=grw7y67xo5%3A1")
	require.Equal(t, 502, resp.StatusCode)
	body := readJSON(t, resp)
	require.Equal(t, "payment_provider_error", body["error"])
	require.NotEmpty(t, body["request_id"])
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestCheckout_Midtrans4xx_502_NoRetry(t *testing.T) {
	t.Parallel()
	var calls int32
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return makeResponse(400, `{"error_messages":["bad"]}`), nil
	})
	a := newTestApp(t, rt, nil)
	resp := doGet(t, a, "/checkout?products=grw7y67xo5%3A1")
	require.Equal(t, 502, resp.StatusCode)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestRateLimit_429AfterMax(t *testing.T) {
	t.Parallel()
	captured := []*http.Request{}
	a := newTestApp(t, alwaysOK(&captured), func(cfg *config.Config) {
		cfg.RateLimitPerMin = 2
	})

	url := "/checkout?products=grw7y67xo5%3A1"
	r1 := doGet(t, a, url)
	r2 := doGet(t, a, url)
	r3 := doGet(t, a, url)

	require.Equal(t, 302, r1.StatusCode)
	require.Equal(t, 302, r2.StatusCode)
	require.Equal(t, 429, r3.StatusCode)
	body := readJSON(t, r3)
	require.Equal(t, "rate_limited", body["error"])
	require.NotEmpty(t, body["request_id"])
}

func TestRateLimit_DoesNotLimitHealth(t *testing.T) {
	t.Parallel()
	captured := []*http.Request{}
	a := newTestApp(t, alwaysOK(&captured), func(cfg *config.Config) {
		cfg.RateLimitPerMin = 1
	})
	for i := 0; i < 5; i++ {
		resp := doGet(t, a, "/health")
		require.Equal(t, 200, resp.StatusCode, "request %d", i+1)
	}
}
