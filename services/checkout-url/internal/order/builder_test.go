package order

import (
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/catalog"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/parser"
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

func TestBuild_SingleItem_ComputesGrossAmount(t *testing.T) {
	t.Parallel()
	got, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, defaultCfg())
	require.NoError(t, err)
	require.Equal(t, 90000, got.TransactionDetails.GrossAmount)
}

func TestBuild_MultiItem_ComputesGrossAmount(t *testing.T) {
	t.Parallel()
	got, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{
			{ProductID: "grw7y67xo5", Qty: 2},
			{ProductID: "zmis5llkew", Qty: 3},
		},
	}, defaultCfg())
	require.NoError(t, err)
	require.Equal(t, 90000*2+75000*3, got.TransactionDetails.GrossAmount)
}

func TestBuild_ItemDetailsMapping(t *testing.T) {
	t.Parallel()
	got, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{
			{ProductID: "grw7y67xo5", Qty: 2},
			{ProductID: "zmis5llkew", Qty: 1},
		},
	}, defaultCfg())
	require.NoError(t, err)
	require.Equal(t, []ItemDetail{
		{ID: "grw7y67xo5", Name: "Product A", Price: 90000, Quantity: 2},
		{ID: "zmis5llkew", Name: "Product B", Price: 75000, Quantity: 1},
	}, got.ItemDetails)
}

func TestBuild_OrderIDIsUUIDv4(t *testing.T) {
	t.Parallel()
	got, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, defaultCfg())
	require.NoError(t, err)
	require.True(t, uuidV4.MatchString(got.TransactionDetails.OrderID),
		"order_id %q must be UUID v4", got.TransactionDetails.OrderID)
}

func TestBuild_RespectsConfigTunables(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		CustomerRequired: false,
		EnabledPayments:  []string{"other_qris", "gopay"},
		ExpiryDuration:   30,
		ExpiryUnit:       "hours",
	}
	got, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, cfg)
	require.NoError(t, err)
	require.False(t, got.CustomerRequired)
	require.Equal(t, []string{"other_qris", "gopay"}, got.EnabledPayments)
	require.Equal(t, Expiry{Duration: 30, Unit: "hours"}, got.Expiry)
}

func TestBuild_CustomFieldsPassthrough(t *testing.T) {
	t.Parallel()
	got, err := BuildPaymentLinkRequest(parser.Parsed{
		Items:      []parser.Item{{ProductID: "grw7y67xo5", Qty: 1}},
		Coupon:     "ADHA2026",
		CartOrigin: "meta_shops",
		Fbclid:     "abc123",
	}, defaultCfg())
	require.NoError(t, err)
	require.Equal(t, "ADHA2026", got.CustomField1)
	require.Equal(t, "meta_shops", got.CustomField2)
	require.Equal(t, "abc123", got.CustomField3)
}

func TestBuild_OmitsCustomFieldsInJSONWhenEmpty(t *testing.T) {
	t.Parallel()
	got, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, defaultCfg())
	require.NoError(t, err)

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	asMap := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &asMap))
	require.NotContains(t, asMap, "custom_field1")
	require.NotContains(t, asMap, "custom_field2")
	require.NotContains(t, asMap, "custom_field3")
	// customer_details object must never be present — sandbox rejects it
	// when partial.
	require.NotContains(t, asMap, "customer_details")
}

func TestBuild_UnknownProductReturnsProductNotFoundError(t *testing.T) {
	t.Parallel()
	_, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{{ProductID: "unknown", Qty: 1}},
	}, defaultCfg())
	require.Error(t, err)
	var pnf *catalog.ProductNotFoundError
	require.True(t, errors.As(err, &pnf))
	require.Equal(t, "unknown", pnf.ProductID)
}

func TestBuild_GeneratesDifferentOrderIDPerCall(t *testing.T) {
	t.Parallel()
	a, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, defaultCfg())
	require.NoError(t, err)
	b, err := BuildPaymentLinkRequest(parser.Parsed{
		Items: []parser.Item{{ProductID: "grw7y67xo5", Qty: 1}},
	}, defaultCfg())
	require.NoError(t, err)
	require.NotEqual(t, a.TransactionDetails.OrderID, b.TransactionDetails.OrderID)
}
