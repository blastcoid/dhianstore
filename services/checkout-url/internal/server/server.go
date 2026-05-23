// Package server wires Fiber routes, middleware, and handlers into a
// runnable *fiber.App. It exposes a single constructor so tests can build
// the same app the production main uses.
package server

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/handler"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/midtrans"
)

// New builds a fully wired *fiber.App. The caller is responsible for calling
// app.Listen (and Shutdown).
func New(cfg *config.Config, log zerolog.Logger, client *midtrans.Client) *fiber.App {
	fiberCfg := fiber.Config{
		ErrorHandler: errorHandler(cfg, log),
	}
	if cfg.IsProduction() {
		// Trust proxies inside private network ranges (LB / ingress / sidecars).
		// If you deploy with a public-facing proxy, override TrustProxyConfig
		// with its specific IPs.
		fiberCfg.TrustProxy = true
		fiberCfg.ProxyHeader = fiber.HeaderXForwardedFor
		fiberCfg.TrustProxyConfig = fiber.TrustProxyConfig{Private: true}
	}

	app := fiber.New(fiberCfg)

	app.Use(recoverer.New())
	app.Use(requestid.New())
	app.Use(requestLogger(log))

	// Health stays outside the rate limiter so probes never get blocked.
	app.Get("/health", handler.Health)

	checkoutHandler := handler.NewCheckout(cfg, client, log)
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
