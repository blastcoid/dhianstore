package checkout

import (
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/blastcoid/dhianstore/services/checkout-url/config"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func defaultCfg() *config.Config {
	return &config.Config{
		CustomerRequired: true,
		EnabledPayments:  []string{"other_qris"},
		ExpiryDuration:   15,
		ExpiryUnit:       "minutes",
	}
}

func TestBuildPaymentLinkRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		req              Request
		cfg              *config.Config
		wantGrossAmount  int
		wantItemDetails  []ItemDetail
		wantCustomFields [3]string
	}{
		{
			name: "single item",
			req: Request{
				Items: []Item{{ProductID: "grw7y67xo5", Qty: 1}},
			},
			cfg:             defaultCfg(),
			wantGrossAmount: 90000,
			wantItemDetails: []ItemDetail{
				{ID: "grw7y67xo5", Name: "Product A", Price: 90000, Quantity: 1},
			},
		},
		{
			name: "multi item, different qty",
			req: Request{
				Items: []Item{
					{ProductID: "grw7y67xo5", Qty: 2},
					{ProductID: "zmis5llkew", Qty: 3},
				},
			},
			cfg:             defaultCfg(),
			wantGrossAmount: 90000*2 + 75000*3,
			wantItemDetails: []ItemDetail{
				{ID: "grw7y67xo5", Name: "Product A", Price: 90000, Quantity: 2},
				{ID: "zmis5llkew", Name: "Product B", Price: 75000, Quantity: 3},
			},
		},
		{
			name: "custom fields passthrough",
			req: Request{
				Items:      []Item{{ProductID: "grw7y67xo5", Qty: 1}},
				Coupon:     "ADHA2026",
				CartOrigin: "meta_shops",
				Fbclid:     "abc123",
			},
			cfg:              defaultCfg(),
			wantGrossAmount:  90000,
			wantItemDetails:  []ItemDetail{{ID: "grw7y67xo5", Name: "Product A", Price: 90000, Quantity: 1}},
			wantCustomFields: [3]string{"ADHA2026", "meta_shops", "abc123"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := BuildPaymentLinkRequest(tt.req, tt.cfg)
			require.NoError(t, err)
			require.Equal(t, tt.wantGrossAmount, got.TransactionDetails.GrossAmount)
			require.Equal(t, tt.wantItemDetails, got.ItemDetails)
			require.True(t, uuidV4.MatchString(got.TransactionDetails.OrderID),
				"order_id must be UUID v4, got %q", got.TransactionDetails.OrderID)
			require.Equal(t, tt.cfg.CustomerRequired, got.CustomerRequired)
			require.Equal(t, tt.cfg.EnabledPayments, got.EnabledPayments)
			require.Equal(t, Expiry{Duration: tt.cfg.ExpiryDuration, Unit: tt.cfg.ExpiryUnit}, got.Expiry)
			require.Equal(t, tt.wantCustomFields[0], got.CustomField1)
			require.Equal(t, tt.wantCustomFields[1], got.CustomField2)
			require.Equal(t, tt.wantCustomFields[2], got.CustomField3)
		})
	}
}

func TestBuildPaymentLinkRequest_OmitsCustomFieldsInJSON(t *testing.T) {
	t.Parallel()
	got, err := BuildPaymentLinkRequest(Request{
		Items: []Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, defaultCfg())
	require.NoError(t, err)

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	asMap := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &asMap))

	require.NotContains(t, asMap, "custom_field1")
	require.NotContains(t, asMap, "custom_field2")
	require.NotContains(t, asMap, "custom_field3")
	// customer_details must NEVER appear — sandbox rejects empty/partial.
	require.NotContains(t, asMap, "customer_details")
}

func TestBuildPaymentLinkRequest_UnknownProduct(t *testing.T) {
	t.Parallel()
	_, err := BuildPaymentLinkRequest(Request{
		Items: []Item{{ProductID: "unknown", Qty: 1}},
	}, defaultCfg())
	require.Error(t, err)
	var pnf *ProductNotFoundError
	require.True(t, errors.As(err, &pnf))
	require.Equal(t, "unknown", pnf.ProductID)
}

func TestBuildPaymentLinkRequest_DifferentOrderIDPerCall(t *testing.T) {
	t.Parallel()
	a, err := BuildPaymentLinkRequest(Request{
		Items: []Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, defaultCfg())
	require.NoError(t, err)
	b, err := BuildPaymentLinkRequest(Request{
		Items: []Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, defaultCfg())
	require.NoError(t, err)
	require.NotEqual(t, a.TransactionDetails.OrderID, b.TransactionDetails.OrderID)
}
