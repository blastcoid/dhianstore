// Package checkout holds the domain types, business logic, and consumer-defined
// interfaces for the checkout flow. It depends only on stdlib + google/uuid;
// concrete implementations of its interfaces live in sibling packages.
package checkout

// Item is one line in the parsed cart.
type Item struct {
	ProductID string
	Qty       int
}

// CheckoutRequest is the validated, typed form of the incoming query string
// from a Meta Shops checkout redirect.
type CheckoutRequest struct {
	Items      []Item
	Coupon     string
	CartOrigin string
	Fbclid     string
}

// Payload is the JSON body sent to Midtrans POST /v1/payment-links.
//
// Note: we intentionally do NOT include a CustomerDetails field — sandbox
// rejects empty/partial customer_details with 400 ("Please fill in at least
// either email, phone or name in customer_details object"). CustomerRequired
// alone is enough to surface the buyer-info form.
type Payload struct {
	TransactionDetails TransactionDetails `json:"transaction_details"`
	CustomerRequired   bool               `json:"customer_required"`
	ItemDetails        []ItemDetail       `json:"item_details"`
	EnabledPayments    []string           `json:"enabled_payments"`
	Expiry             Expiry             `json:"expiry"`
	CustomField1       string             `json:"custom_field1,omitempty"`
	CustomField2       string             `json:"custom_field2,omitempty"`
	CustomField3       string             `json:"custom_field3,omitempty"`
}

// TransactionDetails carries order_id and gross_amount.
type TransactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int    `json:"gross_amount"`
}

// ItemDetail is one line rendered on the Midtrans payment page.
type ItemDetail struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}

// Expiry uses plural units ("minutes" | "hours" | "days") — Midtrans rejects
// singular forms.
type Expiry struct {
	Duration int    `json:"duration"`
	Unit     string `json:"unit"`
}

// Response is the subset of the Midtrans payment-link response we surface to
// callers. JSON tags map to the API's snake_case fields.
type Response struct {
	PaymentURL    string `json:"payment_url"`
	OrderID       string `json:"order_id"`
	PaymentLinkID string `json:"payment_link_id"`
	Token         string `json:"token"`
}
