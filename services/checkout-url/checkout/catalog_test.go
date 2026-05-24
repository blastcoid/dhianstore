package checkout

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductNotFoundError_Error(t *testing.T) {
	t.Parallel()
	err := &ProductNotFoundError{ProductID: "zmis5llkew"}
	require.Contains(t, err.Error(), "zmis5llkew")
	require.Contains(t, err.Error(), "not found")
}

func TestProductNotFoundError_ErrorsAs(t *testing.T) {
	t.Parallel()
	// Round-trip through error interface — domain consumers detect via errors.As.
	var err error = &ProductNotFoundError{ProductID: "foo"}
	var target *ProductNotFoundError
	require.True(t, errors.As(err, &target))
	require.Equal(t, "foo", target.ProductID)
}
