package checkout

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// MaxCustomFieldLen is the per-field cap Midtrans enforces on custom_field1/2/3.
// fbclid often exceeds 255 chars so the parser truncates it here.
const MaxCustomFieldLen = 255

// InvalidQueryError signals a client-correctable problem in the query string.
// Handlers map this to a 400 response.
type InvalidQueryError struct {
	Msg string
}

func (e *InvalidQueryError) Error() string { return e.Msg }

func invalidQuery(format string, args ...any) *InvalidQueryError {
	return &InvalidQueryError{Msg: fmt.Sprintf(format, args...)}
}

// ParseQuery validates and decodes the query params from a Meta Shops checkout
// redirect. Fiber's c.Queries() already decodes once; we re-decode defensively
// when a '%' is present (e.g., when callers pass raw encoded strings in tests).
//
// Meta spec: https://developers.facebook.com/docs/commerce-platform/setup-checkout-url
// Format: products=<id>:<qty>,<id>:<qty>
func ParseQuery(q map[string]string) (Request, error) {
	items, err := parseProducts(q["products"])
	if err != nil {
		return Request{}, err
	}
	return Request{
		Items:      items,
		Coupon:     q["coupon"],
		CartOrigin: q["cart_origin"],
		Fbclid:     truncate(q["fbclid"], MaxCustomFieldLen),
	}, nil
}

func parseProducts(raw string) ([]Item, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, invalidQuery(`query param "products" is required`)
	}
	decoded := raw
	if strings.Contains(raw, "%") {
		d, err := url.QueryUnescape(raw)
		if err != nil {
			return nil, invalidQuery(`query param "products" is not valid URI-encoded`)
		}
		decoded = d
	}

	pairs := splitNonEmpty(decoded, ",")
	if len(pairs) == 0 {
		return nil, invalidQuery(`query param "products" is empty`)
	}

	items := make([]Item, 0, len(pairs))
	for _, pair := range pairs {
		id, qty, err := parsePair(pair)
		if err != nil {
			return nil, err
		}
		items = append(items, Item{ProductID: id, Qty: qty})
	}
	return items, nil
}

func parsePair(pair string) (string, int, error) {
	parts := strings.Split(pair, ":")
	if len(parts) != 2 {
		return "", 0, invalidQuery(`malformed product pair: %q (expected "id:qty")`, pair)
	}
	id, qtyStr := parts[0], parts[1]
	if id == "" {
		return "", 0, invalidQuery(`malformed product pair: missing id in %q`, pair)
	}
	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty < 1 {
		return "", 0, invalidQuery(
			`invalid qty for product %q: %q (must be integer >= 1)`, id, qtyStr,
		)
	}
	return id, qty, nil
}

func splitNonEmpty(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
