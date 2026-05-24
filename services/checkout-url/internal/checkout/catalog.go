package checkout

import "fmt"

// Product is the minimal product representation the rest of the checkout
// flow needs. CatalogClient implementations populate it from upstream (Meta
// Catalog in production). Prices are integer IDR — Midtrans rejects decimals.
type Product struct {
	ID    string
	Name  string
	Price int
}

// ProductNotFoundError signals that a requested productID was not in the
// catalog response. Handlers map this to a 400 response so the buyer sees
// a clear "product unavailable" rather than a generic server error.
type ProductNotFoundError struct {
	ProductID string
}

func (e *ProductNotFoundError) Error() string {
	return fmt.Sprintf("product not found: %s", e.ProductID)
}
