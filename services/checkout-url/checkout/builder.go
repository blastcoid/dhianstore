package checkout

import (
	"github.com/google/uuid"

	"github.com/blastcoid/dhianstore/services/checkout-url/config"
)

// BuildPaymentLinkRequest expands a parsed request via the catalog, sums the
// gross amount, generates a fresh UUID v4 order_id, and assembles the Midtrans
// payload. Returns *ProductNotFoundError if any productID is unknown.
//
// Payment tunables (CustomerRequired, EnabledPayments, Expiry) are sourced
// from config so they can be adjusted per environment without code changes.
func BuildPaymentLinkRequest(req Request, cfg *config.Config) (Payload, error) {
	itemDetails := make([]ItemDetail, 0, len(req.Items))
	grossAmount := 0

	for _, it := range req.Items {
		prod, err := Lookup(it.ProductID)
		if err != nil {
			return Payload{}, err
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
