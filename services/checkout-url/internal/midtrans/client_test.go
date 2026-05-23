package midtrans

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/order"
)

// roundTripperFunc lets a test treat a closure as an http.RoundTripper. The
// closure receives the outgoing request and must return a response (or err).
type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newTestClient(t *testing.T, rt http.RoundTripper) *Client {
	t.Helper()
	cfg := &config.Config{
		MidtransServerKey: "SB-Mid-server-testkey123",
		MidtransAPIBase:   "https://api.sandbox.midtrans.com",
	}
	logger := zerolog.New(io.Discard)
	c := New(cfg, logger)
	c.WithHTTPClient(&http.Client{Transport: rt, Timeout: 2 * time.Second})
	return c
}

func sampleBody() order.Request {
	return order.Request{
		TransactionDetails: order.TransactionDetails{OrderID: "test-order", GrossAmount: 90000},
	}
}

func sampleSuccessResponseBody() string {
	r := Response{
		PaymentURL:    "https://app.sandbox.midtrans.com/payment-links/abc",
		OrderID:       "test-order",
		PaymentLinkID: "plink_abc",
		Token:         "tok_abc",
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

func TestCreatePaymentLink_Success(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "POST", req.Method)
		require.Equal(t, "https://api.sandbox.midtrans.com/v1/payment-links", req.URL.String())
		return makeResponse(200, sampleSuccessResponseBody()), nil
	}))

	got, err := c.CreatePaymentLink(context.Background(), sampleBody())
	require.NoError(t, err)
	require.Equal(t, "https://app.sandbox.midtrans.com/payment-links/abc", got.PaymentURL)
	require.Equal(t, "plink_abc", got.PaymentLinkID)
	require.Equal(t, "tok_abc", got.Token)
}

func TestCreatePaymentLink_SendsBasicAuthHeader(t *testing.T) {
	t.Parallel()
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("SB-Mid-server-testkey123:"))
	c := newTestClient(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, expected, req.Header.Get("Authorization"))
		require.Equal(t, "application/json", req.Header.Get("Accept"))
		require.Equal(t, "application/json", req.Header.Get("Content-Type"))
		return makeResponse(200, sampleSuccessResponseBody()), nil
	}))
	_, err := c.CreatePaymentLink(context.Background(), sampleBody())
	require.NoError(t, err)
}

func TestCreatePaymentLink_SerializesBodyAsJSON(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		var decoded order.Request
		require.NoError(t, json.Unmarshal(body, &decoded))
		require.Equal(t, "test-order", decoded.TransactionDetails.OrderID)
		require.Equal(t, 90000, decoded.TransactionDetails.GrossAmount)
		return makeResponse(200, sampleSuccessResponseBody()), nil
	}))
	_, err := c.CreatePaymentLink(context.Background(), sampleBody())
	require.NoError(t, err)
}

func TestCreatePaymentLink_4xxNoRetry(t *testing.T) {
	t.Parallel()
	var calls int32
	c := newTestClient(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return makeResponse(400, `{"error_messages":["bad data"]}`), nil
	}))

	_, err := c.CreatePaymentLink(context.Background(), sampleBody())
	require.Error(t, err)
	var me *Error
	require.True(t, errors.As(err, &me))
	require.Equal(t, 400, me.StatusCode)
	require.Contains(t, me.ResponseBody, "bad data")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "4xx must not retry")
}

func TestCreatePaymentLink_5xxRetriesUntilSuccess(t *testing.T) {
	t.Parallel()
	var calls int32
	c := newTestClient(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return makeResponse(500, "server error"), nil
		}
		return makeResponse(200, sampleSuccessResponseBody()), nil
	}))

	got, err := c.CreatePaymentLink(context.Background(), sampleBody())
	require.NoError(t, err)
	require.Equal(t, "https://app.sandbox.midtrans.com/payment-links/abc", got.PaymentURL)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestCreatePaymentLink_5xxAllAttemptsFail(t *testing.T) {
	t.Parallel()
	var calls int32
	c := newTestClient(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return makeResponse(503, "still down"), nil
	}))

	_, err := c.CreatePaymentLink(context.Background(), sampleBody())
	require.Error(t, err)
	var me *Error
	require.True(t, errors.As(err, &me))
	require.Equal(t, 503, me.StatusCode)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestCreatePaymentLink_NetworkErrorRetries(t *testing.T) {
	t.Parallel()
	var calls int32
	c := newTestClient(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return nil, errors.New("ECONNRESET")
	}))

	_, err := c.CreatePaymentLink(context.Background(), sampleBody())
	require.Error(t, err)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}
