package checkout

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		productID string
		want      Product
		wantErrAs error
	}{
		{
			name:      "known product A",
			productID: "grw7y67xo5",
			want:      Product{ID: "grw7y67xo5", Name: "Product A", Price: 90000},
		},
		{
			name:      "known product B",
			productID: "zmis5llkew",
			want:      Product{ID: "zmis5llkew", Name: "Product B", Price: 75000},
		},
		{
			name:      "unknown id returns ProductNotFoundError",
			productID: "nonexistent",
			wantErrAs: &ProductNotFoundError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Lookup(tt.productID)
			if tt.wantErrAs != nil {
				require.Error(t, err)
				var target *ProductNotFoundError
				require.True(t, errors.As(err, &target))
				require.Equal(t, tt.productID, target.ProductID)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestProducts_Invariants(t *testing.T) {
	t.Parallel()
	require.GreaterOrEqual(t, len(Products), 2, "catalog must have at least 2 products")
	for id, p := range Products {
		require.Positive(t, p.Price, "%s price must be > 0", id)
		require.Equal(t, id, p.ID, "%s: Product.ID must match map key", id)
	}
}
