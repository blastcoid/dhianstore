package meta

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/blastcoid/dhianstore/services/checkout-url/checkout"
	"github.com/blastcoid/dhianstore/services/checkout-url/config"
)

// newServerClient wires a httptest.Server to a fresh meta.Client. Cleanup
// registered via t.Cleanup.
func newServerClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	cfg := &config.Config{
		MetaGraphAPIBase: ts.URL,
		MetaGraphVersion: "v25.0",
		MetaCatalogID:    "1017309634048260",
		MetaAccessToken:  "EAAtest-token",
	}
	return New(cfg)
}

func sampleSuccessBody() string {
	return `{
		"data": [
			{
				"name": "Gamis ceruty combi brukat 4D + hijab ceruty",
				"price": "Rp460.000",
				"currency": "IDR",
				"retailer_id": "zmis5llkew",
				"id": "35587458977568108"
			},
			{
				"name": "Gamis Bini Orang Maxy Dress",
				"price": "Rp325.000",
				"currency": "IDR",
				"retailer_id": "grw7y67xo5",
				"id": "36347222504876161"
			}
		]
	}`
}

func TestFetchProducts_Success(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v25.0/1017309634048260/products", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleSuccessBody()))
	})

	got, err := c.FetchProducts(context.Background(), []string{"zmis5llkew", "grw7y67xo5"})
	require.NoError(t, err)
	require.Equal(t, []checkout.Product{
		{ID: "zmis5llkew", Name: "Gamis ceruty combi brukat 4D + hijab ceruty", Price: 460000},
		{ID: "grw7y67xo5", Name: "Gamis Bini Orang Maxy Dress", Price: 325000},
	}, got)
}

func TestFetchProducts_PreservesInputOrder(t *testing.T) {
	t.Parallel()
	// Server returns products in reverse order; client must re-order to match
	// the input retailerIDs slice.
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"retailer_id":"grw7y67xo5","name":"B","price":"Rp325.000","currency":"IDR"},
				{"retailer_id":"zmis5llkew","name":"A","price":"Rp460.000","currency":"IDR"}
			]
		}`))
	})

	got, err := c.FetchProducts(context.Background(), []string{"zmis5llkew", "grw7y67xo5"})
	require.NoError(t, err)
	require.Equal(t, []string{"zmis5llkew", "grw7y67xo5"}, []string{got[0].ID, got[1].ID})
}

func TestFetchProducts_SendsBearerAuth(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer EAAtest-token", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleSuccessBody()))
	})

	_, err := c.FetchProducts(context.Background(), []string{"zmis5llkew", "grw7y67xo5"})
	require.NoError(t, err)
}

func TestFetchProducts_QueryParams(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		filter := q.Get("filter")
		require.NotEmpty(t, filter)
		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(filter), &decoded))
		require.Contains(t, decoded, "retailer_id")

		fields := q.Get("fields")
		require.Contains(t, fields, "retailer_id")
		require.Contains(t, fields, "name")
		require.Contains(t, fields, "price")
		require.Contains(t, fields, "currency")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleSuccessBody()))
	})

	_, err := c.FetchProducts(context.Background(), []string{"zmis5llkew", "grw7y67xo5"})
	require.NoError(t, err)
}

func TestFetchProducts_MissingRetailerID(t *testing.T) {
	t.Parallel()
	// Server returns only one of the two requested IDs.
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"retailer_id":"zmis5llkew","name":"A","price":"Rp460.000","currency":"IDR"}
			]
		}`))
	})

	_, err := c.FetchProducts(context.Background(), []string{"zmis5llkew", "grw7y67xo5"})
	require.Error(t, err)
	var pnf *checkout.ProductNotFoundError
	require.True(t, errors.As(err, &pnf))
	require.Equal(t, "grw7y67xo5", pnf.ProductID)
}

func TestFetchProducts_UnexpectedCurrency(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"retailer_id":"foo","name":"X","price":"$5.99","currency":"USD"}
			]
		}`))
	})

	_, err := c.FetchProducts(context.Background(), []string{"foo"})
	require.Error(t, err)
	var me *Error
	require.True(t, errors.As(err, &me))
	require.Contains(t, me.Message, "currency")
	require.Contains(t, me.Message, "USD")
}

func TestFetchProducts_PriceParseError(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"retailer_id":"foo","name":"X","price":"invalid","currency":"IDR"}
			]
		}`))
	})

	_, err := c.FetchProducts(context.Background(), []string{"foo"})
	require.Error(t, err)
	var me *Error
	require.True(t, errors.As(err, &me))
	require.Contains(t, me.Message, "price parse")
}

func TestFetchProducts_4xxResponse(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid filter"}}`))
	})

	_, err := c.FetchProducts(context.Background(), []string{"foo"})
	require.Error(t, err)
	var me *Error
	require.True(t, errors.As(err, &me))
	require.Equal(t, http.StatusBadRequest, me.StatusCode)
}

func TestFetchProducts_5xxResponse(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("graph api down"))
	})

	_, err := c.FetchProducts(context.Background(), []string{"foo"})
	require.Error(t, err)
	var me *Error
	require.True(t, errors.As(err, &me))
	require.Equal(t, http.StatusInternalServerError, me.StatusCode)
}

func TestFetchProducts_EmptyInput(t *testing.T) {
	t.Parallel()
	// Server should never be called for empty input.
	c := newServerClient(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("server must not be called for empty retailerIDs")
	})

	_, err := c.FetchProducts(context.Background(), nil)
	require.Error(t, err)
}

func TestFetchProducts_NonJSONSuccessBody(t *testing.T) {
	t.Parallel()
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>not json</html>"))
	})

	_, err := c.FetchProducts(context.Background(), []string{"foo"})
	require.Error(t, err)
	var me *Error
	require.True(t, errors.As(err, &me))
	require.Contains(t, me.Message, "non-JSON")
}
