package checkout

import (
	"fmt"
	"strings"
)

// ParseIDRPrice converts Meta Catalog's localized IDR price string to an
// integer amount in IDR (no minor units — IDR doesn't use cents).
//
// Catalog input formats observed in the wild:
//
//	"Rp460.000"   → 460000
//	"Rp 460.000"  → 460000
//	"Rp460000"    → 460000 (no thousand separators)
//
// Rules:
//   - Strip leading "Rp" (case-insensitive) and any surrounding whitespace.
//   - Remove all "." (Indonesian thousand separator). IDR has no decimals.
//   - Remaining must be digits-only and parse to a positive int.
//
// This is intentionally IDR-only; multi-currency support would need a
// currency-aware parser and is out of scope here.
func ParseIDRPrice(raw string) (int, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("parse price: empty input")
	}
	if !strings.HasPrefix(strings.ToLower(s), "rp") {
		return 0, fmt.Errorf("parse price: missing %q prefix in %q", "Rp", raw)
	}
	s = strings.TrimSpace(s[2:])
	s = strings.ReplaceAll(s, ".", "")

	if s == "" {
		return 0, fmt.Errorf("parse price: no digits after prefix in %q", raw)
	}

	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("parse price: non-digit %q in %q", r, raw)
		}
		n = n*10 + int(r-'0')
	}
	if n <= 0 {
		return 0, fmt.Errorf("parse price: must be > 0, got %d from %q", n, raw)
	}
	return n, nil
}
