package httpapi

import (
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"

	"github.com/blastcoid/dhianstore/services/checkout-url/checkout"
	"github.com/blastcoid/dhianstore/services/checkout-url/config"
)

// NewApp builds a fully wired *fiber.App. The caller is responsible for
// calling app.Listen and ShutdownWithContext.
func NewApp(cfg *config.Config, log zerolog.Logger, client checkout.PaymentLinkClient) *fiber.App {
	fcfg := fiber.Config{
		ErrorHandler: errorHandler(cfg, log),
		// Sonic for JSON marshal/unmarshal — Fiber-recommended fastest
		// alternative; configured at app level so c.JSON and BodyParser use it
		// across all handlers.
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
	}
	if cfg.IsProduction() {
		// Trust proxies inside private network ranges (LB / ingress / sidecars).
		fcfg.TrustProxy = true
		fcfg.ProxyHeader = fiber.HeaderXForwardedFor
		fcfg.TrustProxyConfig = fiber.TrustProxyConfig{Private: true}
	}

	app := fiber.New(fcfg)

	app.Use(recoverer.New())
	app.Use(requestid.New())
	app.Use(requestLogger(log))

	// Health stays outside the rate limiter so probes never get blocked.
	app.Get("/health", Health)

	checkoutHandler := NewCheckoutHandler(cfg, client, log)
	app.Get("/checkout",
		limiter.New(limiter.Config{
			Max:        cfg.RateLimitPerMin,
			Expiration: time.Minute,
			KeyGenerator: func(c fiber.Ctx) string {
				return c.IP()
			},
			LimitReached: func(c fiber.Ctx) error {
				reqID := requestid.FromContext(c)
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error":               "rate_limited",
					"message":             "too many requests",
					"retry_after_seconds": 60,
					"request_id":          reqID,
				})
			},
		}),
		checkoutHandler.Handle,
	)

	return app
}
