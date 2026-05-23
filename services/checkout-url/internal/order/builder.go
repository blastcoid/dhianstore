// Package order composes the Midtrans Payment Link request body from a
// parsed query. Pure logic: no I/O, no network.
//
// Behavior notes (verified against Midtrans sandbox, not just docs):
//   - We DO NOT serialise a customer_details object. The sandbox rejects an
//     empty or partial customer_details with 400 ("Please fill in at least
//     either email, phone or name in customer_details object"). Setting
//     CustomerRequired=true alone is enough — Midtrans still renders the
//     buyer-info form on the payment page.
//   - Payment tunables (CustomerRequired, EnabledPayments, Expiry) live in
//     config so they can be adjusted per environment without code changes.
//   - custom_field1/2/3 max 255 chars each (parser already truncates fbclid).
package order

import (
	"github.com/google/uuid"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/catalog"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/parser"
)

// Request is the JSON body sent to POST /v1/payment-links.
type Request struct {
	TransactionDetails TransactionDetails `json:"transaction_details"`
	CustomerRequired   bool               `json:"customer_required"`
	ItemDetails        []ItemDetail       `json:"item_details"`
	EnabledPayments    []string           `json:"enabled_payments"`
	Expiry             Expiry             `json:"expiry"`
	CustomField1       string             `json:"custom_field1,omitempty"`
	CustomField2       string             `json:"custom_field2,omitempty"`
	CustomField3       string             `json:"custom_field3,omitempty"`
}

// TransactionDetails captures order_id and gross_amount.
type TransactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int    `json:"gross_amount"`
}

// ItemDetail is one line shown on the Midtrans payment page.
type ItemDetail struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}

// Expiry uses plural units ("minutes", "hours", "days") — singular is rejected.
type Expiry struct {
	Duration int    `json:"duration"`
	Unit     string `json:"unit"`
}

// BuildPaymentLinkRequest expands parsed items via the catalog, sums the
// gross amount, generates a fresh UUID v4 order_id, and assembles the body.
// Returns *catalog.ProductNotFoundError if any productID is unknown.
func BuildPaymentLinkRequest(p parser.Parsed, cfg *config.Config) (Request, error) {
	itemDetails := make([]ItemDetail, 0, len(p.Items))
	grossAmount := 0

	for _, it := range p.Items {
		prod, err := catalog.Lookup(it.ProductID)
		if err != nil {
			return Request{}, err
		}
		itemDetails = append(itemDetails, ItemDetail{
			ID:       prod.ID,
			Name:     prod.Name,
			Price:    prod.Price,
			Quantity: it.Qty,
		})
		grossAmount += prod.Price * it.Qty
	}

	return Request{
		TransactionDetails: TransactionDetails{
			OrderID:     uuid.NewString(),
			GrossAmount: grossAmount,
		},
		CustomerRequired: cfg.CustomerRequired,
		ItemDetails:      itemDetails,
		EnabledPayments:  cfg.EnabledPayments,
		Expiry:           Expiry{Duration: cfg.ExpiryDuration, Unit: cfg.ExpiryUnit},
		CustomField1:     p.Coupon,
		CustomField2:     p.CartOrigin,
		CustomField3:     p.Fbclid,
	}, nil
}
