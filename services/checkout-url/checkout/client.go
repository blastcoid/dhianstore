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
