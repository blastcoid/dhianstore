// Package parser converts the raw query string from Meta Shops checkout
// redirects into a typed Parsed struct. All validation lives here so the
// handler can assume valid input downstream.
//
// Meta spec: https://developers.facebook.com/docs/commerce-platform/setup-checkout-url
// Input format: products=<id>:<qty>,<id>:<qty> (RFC 3986-encoded in the URL).
package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// MaxCustomFieldLen is the per-field cap Midtrans enforces on custom_field1/2/3.
// fbclid often exceeds 255 chars so we truncate it here.
const MaxCustomFieldLen = 255

// Item is one line in the cart.
type Item struct {
	ProductID string
	Qty       int
}

// Parsed is the validated, typed form of the incoming query.
type Parsed struct {
	Items      []Item
	Coupon     string
	CartOrigin string
	Fbclid     string
}

// InvalidQueryError signals a client-correctable problem in the query string.
// Handlers map this to a 400 response.
type InvalidQueryError struct {
	Msg string
}

func (e *InvalidQueryError) Error() string { return e.Msg }

func newInvalidQuery(format string, args ...any) *InvalidQueryError {
	return &InvalidQueryError{Msg: fmt.Sprintf(format, args...)}
}

// ParseCheckoutQuery validates and decodes the relevant query params.
//
// Fiber's c.Queries() already decodes once, but callers (including tests) may
// pass raw encoded strings, so we re-decode defensively when a '%' is present.
func ParseCheckoutQuery(q map[string]string) (Parsed, error) {
	items, err := parseProducts(q["products"])
	if err != nil {
		return Parsed{}, err
	}
	return Parsed{
		Items:      items,
		Coupon:     nonEmpty(q["coupon"]),
		CartOrigin: nonEmpty(q["cart_origin"]),
		Fbclid:     truncate(nonEmpty(q["fbclid"]), MaxCustomFieldLen),
	}, nil
}

func parseProducts(raw string) ([]Item, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, newInvalidQuery(`query param "products" is required`)
	}
	decoded := raw
	if strings.Contains(raw, "%") {
		d, err := url.QueryUnescape(raw)
		if err != nil {
			return nil, newInvalidQuery(`query param "products" is not valid URI-encoded`)
		}
		decoded = d
	}

	pairs := splitNonEmpty(decoded, ",")
	if len(pairs) == 0 {
		return nil, newInvalidQuery(`query param "products" is empty`)
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
		return "", 0, newInvalidQuery(`malformed product pair: %q (expected "id:qty")`, pair)
	}
	id, qtyStr := parts[0], parts[1]
	if id == "" {
		return "", 0, newInvalidQuery(`malformed product pair: missing id in %q`, pair)
	}
	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty < 1 {
		return "", 0, newInvalidQuery(
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

func nonEmpty(s string) string {
	if s == "" {
		return ""
	}
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
