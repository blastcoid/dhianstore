package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCheckoutQuery_SingleItem(t *testing.T) {
	t.Parallel()
	got, err := ParseCheckoutQuery(map[string]string{"products": "grw7y67xo5:1"})
	require.NoError(t, err)
	require.Equal(t, []Item{{ProductID: "grw7y67xo5", Qty: 1}}, got.Items)
	require.Empty(t, got.Coupon)
	require.Empty(t, got.CartOrigin)
	require.Empty(t, got.Fbclid)
}

func TestParseCheckoutQuery_MultiItemWithOptionalFields(t *testing.T) {
	t.Parallel()
	got, err := ParseCheckoutQuery(map[string]string{
		"products":    "grw7y67xo5:3,zmis5llkew:2",
		"coupon":      "ADHA2026",
		"cart_origin": "meta_shops",
		"fbclid":      "IwZXh0bgNhZW0",
	})
	require.NoError(t, err)
	require.Equal(t, []Item{
		{ProductID: "grw7y67xo5", Qty: 3},
		{ProductID: "zmis5llkew", Qty: 2},
	}, got.Items)
	require.Equal(t, "ADHA2026", got.Coupon)
	require.Equal(t, "meta_shops", got.CartOrigin)
	require.Equal(t, "IwZXh0bgNhZW0", got.Fbclid)
}

func TestParseCheckoutQuery_DecodesRawRFC3986(t *testing.T) {
	t.Parallel()
	got, err := ParseCheckoutQuery(map[string]string{
		"products": "grw7y67xo5%3A3%2Czmis5llkew%3A2",
	})
	require.NoError(t, err)
	require.Equal(t, []Item{
		{ProductID: "grw7y67xo5", Qty: 3},
		{ProductID: "zmis5llkew", Qty: 2},
	}, got.Items)
}

func TestParseCheckoutQuery_TruncatesFbclidTo255(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 300)
	got, err := ParseCheckoutQuery(map[string]string{
		"products": "grw7y67xo5:1",
		"fbclid":   long,
	})
	require.NoError(t, err)
	require.Len(t, got.Fbclid, 255)
}

func TestParseCheckoutQuery_InvalidCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		query     map[string]string
		wantMatch string
	}{
		{"missing products", map[string]string{}, "products"},
		{"empty products", map[string]string{"products": ""}, "products"},
		{"no colon", map[string]string{"products": "grw7y67xo5"}, "malformed"},
		{"missing id", map[string]string{"products": ":3"}, "missing id"},
		{"qty zero", map[string]string{"products": "grw7y67xo5:0"}, "invalid qty"},
		{"qty negative", map[string]string{"products": "grw7y67xo5:-1"}, "invalid qty"},
		{"qty non-numeric", map[string]string{"products": "grw7y67xo5:foo"}, "invalid qty"},
		{"qty decimal", map[string]string{"products": "grw7y67xo5:1.5"}, "invalid qty"},
		{"bad URI encoding", map[string]string{"products": "%E0%A4%A"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseCheckoutQuery(tc.query)
			require.Error(t, err)
			var iq *InvalidQueryError
			require.True(t, errors.As(err, &iq), "want *InvalidQueryError")
			if tc.wantMatch != "" {
				require.Contains(t, err.Error(), tc.wantMatch)
			}
		})
	}
}

func TestParseCheckoutQuery_EmptyOptionalsBecomeEmptyStrings(t *testing.T) {
	t.Parallel()
	got, err := ParseCheckoutQuery(map[string]string{
		"products":    "grw7y67xo5:1",
		"coupon":      "",
		"cart_origin": "",
		"fbclid":      "",
	})
	require.NoError(t, err)
	require.Empty(t, got.Coupon)
	require.Empty(t, got.CartOrigin)
	require.Empty(t, got.Fbclid)
}
