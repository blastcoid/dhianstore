package httpapi

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"

	"github.com/blastcoid/dhianstore/services/checkout-url/checkout"
	"github.com/blastcoid/dhianstore/services/checkout-url/config"
)

// CheckoutHandler wires the checkout flow: parse query → build payload →
// create payment link → 302 redirect.
type CheckoutHandler struct {
	cfg    *config.Config
	client checkout.PaymentLinkClient
	logger zerolog.Logger
}

// NewCheckoutHandler constructs the handler with its dependencies.
func NewCheckoutHandler(cfg *config.Config, c checkout.PaymentLinkClient, l zerolog.Logger) *CheckoutHandler {
	return &CheckoutHandler{cfg: cfg, client: c, logger: l}
}

// Handle processes GET /checkout. All errors propagate to Fiber's central
// ErrorHandler for response mapping.
func (h *CheckoutHandler) Handle(c fiber.Ctx) error {
	reqID := requestid.FromContext(c)
	log := h.logger.With().Str("request_id", reqID).Logger()

	req, err := checkout.ParseQuery(c.Queries())
	if err != nil {
		return err
	}
	log.Info().
		Int("item_count", len(req.Items)).
		Str("coupon", req.Coupon).
		Str("cart_origin", req.CartOrigin).
		Msg("checkout request received")

	payload, err := checkout.BuildPaymentLinkRequest(req, h.cfg)
	if err != nil {
		return err
	}
	log.Info().
		Str("order_id", payload.TransactionDetails.OrderID).
		Int("gross_amount", payload.TransactionDetails.GrossAmount).
		Msg("payload built")

	result, err := h.client.CreatePaymentLink(c.Context(), payload)
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
