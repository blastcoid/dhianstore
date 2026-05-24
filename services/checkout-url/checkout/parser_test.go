package checkout

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQuery_Happy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		query map[string]string
		want  CheckoutRequest
	}{
		{
			name:  "single item, no optionals",
			query: map[string]string{"products": "grw7y67xo5:1"},
			want: CheckoutRequest{
				Items: []Item{{ProductID: "grw7y67xo5", Qty: 1}},
			},
		},
		{
			name: "multi item with all optionals",
			query: map[string]string{
				"products":    "grw7y67xo5:3,zmis5llkew:2",
				"coupon":      "ADHA2026",
				"cart_origin": "meta_shops",
				"fbclid":      "IwZXh0bgNhZW0",
			},
			want: CheckoutRequest{
				Items: []Item{
					{ProductID: "grw7y67xo5", Qty: 3},
					{ProductID: "zmis5llkew", Qty: 2},
				},
				Coupon:     "ADHA2026",
				CartOrigin: "meta_shops",
				Fbclid:     "IwZXh0bgNhZW0",
			},
		},
		{
			name:  "raw RFC 3986-encoded products is decoded",
			query: map[string]string{"products": "grw7y67xo5%3A3%2Czmis5llkew%3A2"},
			want: CheckoutRequest{
				Items: []Item{
					{ProductID: "grw7y67xo5", Qty: 3},
					{ProductID: "zmis5llkew", Qty: 2},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseQuery(tt.query)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseQuery_TruncatesFbclid(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 300)
	got, err := ParseQuery(map[string]string{
		"products": "grw7y67xo5:1",
		"fbclid":   long,
	})
	require.NoError(t, err)
	require.Len(t, got.Fbclid, MaxCustomFieldLen)
}

func TestParseQuery_InvalidCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseQuery(tt.query)
			require.Error(t, err)
			var iq *InvalidQueryError
			require.True(t, errors.As(err, &iq), "want *InvalidQueryError")
			if tt.wantMatch != "" {
				require.Contains(t, err.Error(), tt.wantMatch)
			}
		})
	}
}
