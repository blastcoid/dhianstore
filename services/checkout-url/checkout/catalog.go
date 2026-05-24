package checkout

import "fmt"

// Product describes one orderable item. Prices are integer IDR (Midtrans
// rejects decimals).
type Product struct {
	ID    string
	Name  string
	Price int
}

// Products is the source-of-truth catalog keyed by the merchant product ID
// that arrives in the Meta Shops checkout URL. MVP scope: hardcoded; swap to
// a DB-backed lookup behind the same Lookup() signature later.
var Products = map[string]Product{
	"grw7y67xo5": {ID: "grw7y67xo5", Name: "Product A", Price: 90000},
	"zmis5llkew": {ID: "zmis5llkew", Name: "Product B", Price: 75000},
}

// ProductNotFoundError signals that a requested productID is not in Products.
// Handlers map this to a 400 response.
type ProductNotFoundError struct {
	ProductID string
}

func (e *ProductNotFoundError) Error() string {
	return fmt.Sprintf("product not found: %s", e.ProductID)
}

// Lookup returns the product matching id or a *ProductNotFoundError.
func Lookup(id string) (Product, error) {
	p, ok := Products[id]
	if !ok {
		return Product{}, &ProductNotFoundError{ProductID: id}
	}
	return p, nil
}
