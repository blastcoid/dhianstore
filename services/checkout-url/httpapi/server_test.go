package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/blastcoid/dhianstore/services/checkout-url/checkout"
	"github.com/blastcoid/dhianstore/services/checkout-url/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/midtrans"
)

// mockClient records each CreatePaymentLink call and returns canned results
// from a stack. Thread-safe.
type mockClient struct {
	mu      sync.Mutex
	calls   []checkout.Payload
	respond func(call int) (checkout.Response, error)
}

func (m *mockClient) CreatePaymentLink(_ context.Context, p checkout.Payload) (checkout.Response, error) {
	m.mu.Lock()
	idx := len(m.calls)
	m.calls = append(m.calls, p)
	m.mu.Unlock()
	if m.respond == nil {
		return checkout.Response{}, nil
	}
	return m.respond(idx)
}

func (m *mockClient) Calls() []checkout.Payload {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]checkout.Payload, len(m.calls))
	copy(out, m.calls)
	return out
}

// alwaysSucceed returns a canned 200 response from Midtrans.
func alwaysSucceed() func(int) (checkout.Response, error) {
	return func(_ int) (checkout.Response, error) {
		return checkout.Response{
			PaymentURL:    "https://app.sandbox.midtrans.com/payment-links/test-link",
			OrderID:       "returned-order-id",
			PaymentLinkID: "plink_test",
			Token:         "tok_test",
		}, nil
	}
}

func newTestApp(t *testing.T, mock *mockClient, tune func(*config.Config)) *fiber.App {
	t.Helper()
	cfg := &config.Config{
		MidtransServerKey: "SB-Mid-server-testkey",
		MidtransAPIBase:   "https://api.sandbox.midtrans.com",
		CustomerRequired:  true,
		EnabledPayments:   []string{"other_qris"},
		ExpiryDuration:    15,
		ExpiryUnit:        "minutes",
		Port:              8080,
		LogLevel:          "fatal",
		AppEnv:            "test",
		RateLimitPerMin:   1000,
	}
	if tune != nil {
		tune(cfg)
	}
	log := zerolog.New(io.Discard)
	return NewApp(cfg, log, mock)
}

// doGet bumps default app.Test timeout to 10s (some tests exercise retry
// paths that need ≥2s for backoffs).
func doGet(t *testing.T, app *fiber.App, url string) *http.Response {
	t.Helper()
	resp, err := app.Test(
		httptest.NewRequest(http.MethodGet, url, nil),
		fiber.TestConfig{Timeout: 10 * time.Second},
	)
	require.NoError(t, err)
	return resp
}

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

func TestHealth_Returns200(t *testing.T) {
	t.Parallel()
	mock := &mockClient{respond: alwaysSucceed()}
	app := newTestApp(t, mock, nil)

	resp := doGet(t, app, "/health")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, map[string]any{"status": "ok"}, readJSON(t, resp))
	require.Empty(t, mock.Calls(), "/health must not call Midtrans")
}

func TestCheckout_HappySingleItem(t *testing.T) {
	t.Parallel()
	mock := &mockClient{respond: alwaysSucceed()}
	app := newTestApp(t, mock, nil)

	resp := doGet(t, app, "/checkout?products=grw7y67xo5%3A1")
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, "https://app.sandbox.midtrans.com/payment-links/test-link", resp.Header.Get("Location"))
	require.Len(t, mock.Calls(), 1)
}

func TestCheckout_HappyMultiItemPayload(t *testing.T) {
	t.Parallel()
	mock := &mockClient{respond: alwaysSucceed()}
	app := newTestApp(t, mock, nil)

	resp := doGet(t, app, "/checkout?products=grw7y67xo5%3A2%2Czmis5llkew%3A1&coupon=ADHA2026&cart_origin=meta_shops&fbclid=abc123")
	require.Equal(t, http.StatusFound, resp.StatusCode)

	calls := mock.Calls()
	require.Len(t, calls, 1)
	p := calls[0]

	require.Equal(t, 90000*2+75000*1, p.TransactionDetails.GrossAmount)
	require.Equal(t, []checkout.ItemDetail{
		{ID: "grw7y67xo5", Name: "Product A", Price: 90000, Quantity: 2},
		{ID: "zmis5llkew", Name: "Product B", Price: 75000, Quantity: 1},
	}, p.ItemDetails)
	require.True(t, p.CustomerRequired)
	require.Equal(t, []string{"other_qris"}, p.EnabledPayments)
	require.Equal(t, checkout.Expiry{Duration: 15, Unit: "minutes"}, p.Expiry)
	require.Equal(t, "ADHA2026", p.CustomField1)
	require.Equal(t, "meta_shops", p.CustomField2)
	require.Equal(t, "abc123", p.CustomField3)
}

func TestCheckout_MissingProducts_400(t *testing.T) {
	t.Parallel()
	mock := &mockClient{respond: alwaysSucceed()}
	app := newTestApp(t, mock, nil)

	resp := doGet(t, app, "/checkout")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := readJSON(t, resp)
	require.Equal(t, "invalid_query", body["error"])
	require.NotEmpty(t, body["request_id"])
	require.Empty(t, mock.Calls(), "must not call Midtrans on parse error")
}

func TestCheckout_UnknownProduct_400(t *testing.T) {
	t.Parallel()
	mock := &mockClient{respond: alwaysSucceed()}
	app := newTestApp(t, mock, nil)

	resp := doGet(t, app, "/checkout?products=nonexistent%3A1")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := readJSON(t, resp)
	require.Equal(t, "product_not_found", body["error"])
	require.Equal(t, "nonexistent", body["product_id"])
	require.NotEmpty(t, body["request_id"])
	require.Empty(t, mock.Calls())
}

func TestCheckout_Midtrans5xx_502(t *testing.T) {
	t.Parallel()
	mock := &mockClient{respond: func(_ int) (checkout.Response, error) {
		return checkout.Response{}, &midtrans.Error{
			Message: "server error", StatusCode: 500, ResponseBody: "down",
		}
	}}
	app := newTestApp(t, mock, nil)

	resp := doGet(t, app, "/checkout?products=grw7y67xo5%3A1")
	require.Equal(t, http.StatusBadGateway, resp.StatusCode)

	body := readJSON(t, resp)
	require.Equal(t, "payment_provider_error", body["error"])
	require.NotEmpty(t, body["request_id"])
}

func TestCheckout_Midtrans4xx_502(t *testing.T) {
	t.Parallel()
	mock := &mockClient{respond: func(_ int) (checkout.Response, error) {
		return checkout.Response{}, &midtrans.Error{
			Message: "rejected", StatusCode: 400, ResponseBody: `{"error_messages":["bad"]}`,
		}
	}}
	app := newTestApp(t, mock, nil)

	resp := doGet(t, app, "/checkout?products=grw7y67xo5%3A1")
	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestRateLimit_BlocksAfterMax(t *testing.T) {
	t.Parallel()
	var callCount int32
	mock := &mockClient{respond: func(_ int) (checkout.Response, error) {
		atomic.AddInt32(&callCount, 1)
		return checkout.Response{
			PaymentURL: "https://app.sandbox.midtrans.com/payment-links/test",
		}, nil
	}}
	app := newTestApp(t, mock, func(cfg *config.Config) { cfg.RateLimitPerMin = 2 })

	url := "/checkout?products=grw7y67xo5%3A1"
	r1 := doGet(t, app, url)
	r2 := doGet(t, app, url)
	r3 := doGet(t, app, url)

	require.Equal(t, http.StatusFound, r1.StatusCode)
	require.Equal(t, http.StatusFound, r2.StatusCode)
	require.Equal(t, http.StatusTooManyRequests, r3.StatusCode)

	body := readJSON(t, r3)
	require.Equal(t, "rate_limited", body["error"])
	require.NotEmpty(t, body["request_id"])
}

func TestRateLimit_DoesNotLimitHealth(t *testing.T) {
	t.Parallel()
	mock := &mockClient{respond: alwaysSucceed()}
	app := newTestApp(t, mock, func(cfg *config.Config) { cfg.RateLimitPerMin = 1 })

	for i := 0; i < 5; i++ {
		resp := doGet(t, app, "/health")
		require.Equal(t, http.StatusOK, resp.StatusCode, "iteration %d", i+1)
	}
}
