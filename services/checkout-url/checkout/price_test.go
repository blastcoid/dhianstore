package checkout

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIDRPrice_Valid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"standard with thousand separators", "Rp460.000", 460000},
		{"space after Rp", "Rp 460.000", 460000},
		{"no separators", "Rp460000", 460000},
		{"lower case rp", "rp325.000", 325000},
		{"trailing whitespace", "Rp460.000  ", 460000},
		{"leading whitespace", "  Rp460.000", 460000},
		{"single digit price", "Rp1", 1},
		{"large amount", "Rp10.000.000", 10000000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseIDRPrice(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseIDRPrice_Invalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		in        string
		wantMatch string
	}{
		{"empty", "", "empty"},
		{"whitespace only", "   ", "empty"},
		{"missing Rp prefix", "460.000", "Rp"},
		{"USD format", "$5.99", "Rp"},
		{"only prefix no digits", "Rp", "no digits"},
		{"prefix with only separators", "Rp...", "no digits"},
		{"contains letters", "Rp460abc", "non-digit"},
		{"contains comma decimal", "Rp460,5", "non-digit"},
		{"zero", "Rp0", "must be > 0"},
		{"explicit zero with separator", "Rp0.000", "must be > 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseIDRPrice(tt.in)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantMatch)
		})
	}
}
