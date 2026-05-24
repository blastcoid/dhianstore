package checkout

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/blastcoid/dhianstore/services/checkout-url/config"
)

// BuildPaymentLinkRequest assembles the Midtrans payload from a parsed
// request, pre-fetched products, and config. Products MUST be supplied by
// the caller (typically via CatalogClient.FetchProducts) — this function
// does not perform any lookup itself.
//
// Returns *ProductNotFoundError if a requested item's productID is missing
// from the supplied products slice.
//
// Payment tunables (CustomerRequired, EnabledPayments, Expiry) are sourced
// from config so they can be adjusted per environment without code changes.
func BuildPaymentLinkRequest(req Request, products []Product, cfg *config.Config) (Payload, error) {
	byID := make(map[string]Product, len(products))
	for _, p := range products {
		byID[p.ID] = p
	}

	itemDetails := make([]ItemDetail, 0, len(req.Items))
	grossAmount := 0

	for _, it := range req.Items {
		prod, ok := byID[it.ProductID]
		if !ok {
			return Payload{}, &ProductNotFoundError{ProductID: it.ProductID}
		}
		if it.Qty <= 0 {
			return Payload{}, fmt.Errorf("build payload: invalid qty %d for %q", it.Qty, it.ProductID)
		}
		itemDetails = append(itemDetails, ItemDetail{
			ID:       prod.ID,
			Name:     prod.Name,
			Price:    prod.Price,
			Quantity: it.Qty,
		})
		grossAmount += prod.Price * it.Qty
	}

	return Payload{
		TransactionDetails: TransactionDetails{
			OrderID:     uuid.NewString(),
			GrossAmount: grossAmount,
		},
		CustomerRequired: cfg.CustomerRequired,
		ItemDetails:      itemDetails,
		EnabledPayments:  cfg.EnabledPayments,
		Expiry:           Expiry{Duration: cfg.ExpiryDuration, Unit: cfg.ExpiryUnit},
		CustomField1:     req.Coupon,
		CustomField2:     req.CartOrigin,
		CustomField3:     req.Fbclid,
	}, nil
}
