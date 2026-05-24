package checkout

import "context"

// PaymentLinkClient creates a Midtrans payment link from an assembled Payload.
//
// The interface is defined here (consumer-defined per "accept interfaces,
// return structs") so concrete implementations in the midtrans package — and
// mocks in tests — can be swapped freely without touching this package.
type PaymentLinkClient interface {
	CreatePaymentLink(ctx context.Context, payload Payload) (Response, error)
}

// CatalogClient fetches product details (name, price, currency) from the
// upstream catalog (Meta Commerce Catalog in production, mock in tests) given
// a list of retailer IDs.
//
// Implementations MUST return products in the same order as input retailerIDs
// is not required; callers index by Product.ID. Implementations MUST return
// *ProductNotFoundError for any requested retailerID that has no match.
type CatalogClient interface {
	FetchProducts(ctx context.Context, retailerIDs []string) ([]Product, error)
}
