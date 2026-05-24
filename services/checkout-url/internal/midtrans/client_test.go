package midtrans

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/checkout"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
)

// newServerClient wires a httptest.Server to a fresh midtrans.Client. The
// returned cleanup func must be called by the test.
//
// Fasthttp (Fiber's transport) talks HTTP/1.1 to a standard net/httptest
// server fine — no special adapter needed.
func newServerClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	cfg := &config.Config{
		MidtransServerKey: "SB-Mid-server-testkey123",
		MidtransAPIBase:   ts.URL,
	}
	return New(cfg)
}

func samplePayload() checkout.Payload {
	return checkout.Payload{
		TransactionDetails: checkout.TransactionDetails{OrderID: "test-order", GrossAmount: 90000},
	}
}

func successResponseBody() string {
	r := checkout.Response{
		PaymentURL:    "https://app.sandbox.midtrans.com/payment-links/abc",
		OrderID:       "test-order",
		PaymentLinkID: "plink_abc",
		Token:         "tok_abc",
	}
	b, _ := json.Marshal(r)
	return string(b)
}

func TestCreatePaymentLink_Success(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/payment-links", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successResponseBody()))
	})

	got, err := c.CreatePaymentLink(context.Background(), samplePayload())
	require.NoError(t, err)
	require.Equal(t, "https://app.sandbox.midtrans.com/payment-links/abc", got.PaymentURL)
	require.Equal(t, "plink_abc", got.PaymentLinkID)
	require.Equal(t, "tok_abc", got.Token)
}

func TestCreatePaymentLink_SendsBasicAuthAndHeaders(t *testing.T) {
	t.Parallel()
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("SB-Mid-server-testkey123:"))

	c := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, wantAuth, r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		_, _ = w.Write([]byte(successResponseBody()))
	})

	_, err := c.CreatePaymentLink(context.Background(), samplePayload())
	require.NoError(t, err)
}

func TestCreatePaymentLink_SerializesBodyAsJSON(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var decoded checkout.Payload
		require.NoError(t, json.Unmarshal(body, &decoded))
		require.Equal(t, "test-order", decoded.TransactionDetails.OrderID)
		require.Equal(t, 90000, decoded.TransactionDetails.GrossAmount)
		_, _ = w.Write([]byte(successResponseBody()))
	})

	_, err := c.CreatePaymentLink(context.Background(), samplePayload())
	require.NoError(t, err)
}

func TestCreatePaymentLink_4xxNoRetry(t *testing.T) {
	t.Parallel()
	var calls int32
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_messages":["bad data"]}`))
	})

	_, err := c.CreatePaymentLink(context.Background(), samplePayload())
	require.Error(t, err)

	var me *Error
	require.True(t, errors.As(err, &me))
	require.Equal(t, http.StatusBadRequest, me.StatusCode)
	require.Contains(t, me.ResponseBody, "bad data")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "4xx must not retry")
}

func TestCreatePaymentLink_5xxRetriesUntilSuccess(t *testing.T) {
	t.Parallel()
	var calls int32
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
			return
		}
		_, _ = w.Write([]byte(successResponseBody()))
	})

	got, err := c.CreatePaymentLink(context.Background(), samplePayload())
	require.NoError(t, err)
	require.Equal(t, "https://app.sandbox.midtrans.com/payment-links/abc", got.PaymentURL)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestCreatePaymentLink_5xxAllAttemptsFail(t *testing.T) {
	t.Parallel()
	var calls int32
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("still down"))
	})

	_, err := c.CreatePaymentLink(context.Background(), samplePayload())
	require.Error(t, err)

	var me *Error
	require.True(t, errors.As(err, &me))
	require.Equal(t, http.StatusServiceUnavailable, me.StatusCode)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestCreatePaymentLink_ContextCancelledDuringBackoff(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		// Always 5xx so we proceed into backoff between attempts.
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := c.CreatePaymentLink(ctx, samplePayload())
	require.Error(t, err)
	var me *Error
	require.True(t, errors.As(err, &me))
	require.Contains(t, me.Message, "cancelled")
}
