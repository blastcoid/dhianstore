// Binary checkout-url bridges Meta Shops checkout redirects to Midtrans
// Payment Links. See services/checkout-url/README.md for an overview.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/httpapi"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/logger"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/meta"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/midtrans"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg)
	catalogClient := meta.New(cfg)
	paymentClient := midtrans.New(cfg)
	app := httpapi.NewApp(cfg, log, catalogClient, paymentClient)

	addr := fmt.Sprintf(":%d", cfg.Port)
	go func() {
		log.Info().
			Int("port", cfg.Port).
			Str("env", cfg.AppEnv).
			Str("api_base", cfg.MidtransAPIBase).
			Msg("checkout-url service listening")

		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("server stopped")
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	received := <-sig
	log.Info().Str("signal", received.String()).Msg("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
		os.Exit(1)
	}
}
