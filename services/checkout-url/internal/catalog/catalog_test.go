package catalog

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookup_KnownIDs(t *testing.T) {
	t.Parallel()

	p, err := Lookup("grw7y67xo5")
	require.NoError(t, err)
	require.Equal(t, Product{ID: "grw7y67xo5", Name: "Product A", Price: 90000}, p)

	p, err = Lookup("zmis5llkew")
	require.NoError(t, err)
	require.Equal(t, Product{ID: "zmis5llkew", Name: "Product B", Price: 75000}, p)
}

func TestLookup_UnknownReturnsProductNotFoundError(t *testing.T) {
	t.Parallel()

	_, err := Lookup("nonexistent")
	require.Error(t, err)

	var pnf *ProductNotFoundError
	require.True(t, errors.As(err, &pnf), "error must wrap *ProductNotFoundError")
	require.Equal(t, "nonexistent", pnf.ProductID)
	require.Contains(t, err.Error(), "nonexistent")
}

func TestProducts_AllPricesArePositiveIntegers(t *testing.T) {
	t.Parallel()

	require.GreaterOrEqual(t, len(Products), 2, "catalog must have at least 2 products")
	for id, p := range Products {
		require.Positive(t, p.Price, "%s price must be > 0", id)
		require.Equal(t, id, p.ID, "%s: Product.ID must match map key", id)
	}
}
