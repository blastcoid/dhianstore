package server

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/catalog"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/midtrans"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/parser"
)

// errorHandler maps domain errors to HTTP responses. It always logs the error
// with the request ID so support can correlate against structured logs.
//
// Production responses never leak the original error message for generic 500s.
func errorHandler(cfg *config.Config, log zerolog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		reqID := requestid.FromContext(c)

		log.Error().
			Err(err).
			Str("request_id", reqID).
			Str("original_url", c.OriginalURL()).
			Msg("request failed")

		var (
			invalidQuery *parser.InvalidQueryError
			productMiss  *catalog.ProductNotFoundError
			midtransErr  *midtrans.Error
		)

		switch {
		case errors.As(err, &invalidQuery):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":      "invalid_query",
				"message":    invalidQuery.Error(),
				"request_id": reqID,
			})
		case errors.As(err, &productMiss):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":      "product_not_found",
				"message":    productMiss.Error(),
				"product_id": productMiss.ProductID,
				"request_id": reqID,
			})
		case errors.As(err, &midtransErr):
			body := fiber.Map{
				"error":      "payment_provider_error",
				"request_id": reqID,
			}
			if cfg.IsDevelopment() {
				body["message"] = midtransErr.Error()
			} else {
				body["message"] = "failed to create payment link"
			}
			return c.Status(fiber.StatusBadGateway).JSON(body)
		default:
			body := fiber.Map{
				"error":      "internal_error",
				"request_id": reqID,
			}
			if cfg.IsDevelopment() {
				body["message"] = err.Error()
			}
			return c.Status(fiber.StatusInternalServerError).JSON(body)
		}
	}
}

// requestLogger logs one structured line per HTTP request, including the
// request ID, method, path, status, and elapsed time.
//
// Note: when a handler returns an error, Fiber invokes ErrorHandler AFTER
// this middleware's c.Next() returns — meaning c.Response().StatusCode() is
// still the default 200 at this point. To avoid logging misleading 200s for
// what becomes a 4xx/5xx response, we skip the request-completed log on
// error paths; errorHandler already logs with full context + request_id.
func requestLogger(log zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		if err != nil {
			return err
		}

		reqID := requestid.FromContext(c)
		status := c.Response().StatusCode()

		evt := log.Info()
		if status >= 500 {
			evt = log.Error()
		} else if status >= 400 {
			evt = log.Warn()
		}

		evt.
			Str("request_id", reqID).
			Str("method", c.Method()).
			Str("url", c.OriginalURL()).
			Int("status", status).
			Dur("duration", time.Since(start)).
			Msg("request completed")
		return nil
	}
}
