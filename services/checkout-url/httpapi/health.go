// Package httpapi wires the checkout domain into Fiber HTTP routes and
// middleware. It owns transport concerns: routing, error mapping, request
// logging, rate limiting.
package httpapi

import "github.com/gofiber/fiber/v3"

// Health is the liveness probe — never touches external dependencies so it
// stays green even when Midtrans is down. That is observed separately via
// /checkout error rates.
func Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}
