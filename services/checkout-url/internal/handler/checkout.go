package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/midtrans"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/order"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/parser"
)

// Checkout wires parser → order builder → Midtrans client → 302 redirect.
type Checkout struct {
	cfg    *config.Config
	client *midtrans.Client
	logger zerolog.Logger
}

// NewCheckout constructs the handler with its dependencies.
func NewCheckout(cfg *config.Config, client *midtrans.Client, logger zerolog.Logger) *Checkout {
	return &Checkout{cfg: cfg, client: client, logger: logger}
}

// Handle processes GET /checkout. All errors propagate to Fiber's central
// ErrorHandler for response mapping.
func (h *Checkout) Handle(c fiber.Ctx) error {
	reqID := requestid.FromContext(c)
	log := h.logger.With().Str("request_id", reqID).Logger()

	parsed, err := parser.ParseCheckoutQuery(c.Queries())
	if err != nil {
		return err
	}
	log.Info().
		Int("item_count", len(parsed.Items)).
		Str("coupon", parsed.Coupon).
		Str("cart_origin", parsed.CartOrigin).
		Msg("checkout request received")

	body, err := order.BuildPaymentLinkRequest(parsed, h.cfg)
	if err != nil {
		return err
	}
	log.Info().
		Str("order_id", body.TransactionDetails.OrderID).
		Int("gross_amount", body.TransactionDetails.GrossAmount).
		Msg("order built")

	result, err := h.client.CreatePaymentLink(c.Context(), body)
	if err != nil {
		return err
	}
	log.Info().
		Str("order_id", result.OrderID).
		Str("payment_link_id", result.PaymentLinkID).
		Str("payment_url", result.PaymentURL).
		Msg("payment link created")

	return c.Redirect().Status(fiber.StatusFound).To(result.PaymentURL)
}
