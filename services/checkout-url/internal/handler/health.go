// Package handler contains the Fiber HTTP handlers.
package handler

import "github.com/gofiber/fiber/v3"

// Health is the liveness probe — never touches external dependencies so it
// stays green even when Midtrans is down. That is a separate concern,
// observed via /checkout error rates.
func Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}
