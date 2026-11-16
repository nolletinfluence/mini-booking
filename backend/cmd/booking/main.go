package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"minibooking/backend/internal/booking"
	"minibooking/backend/internal/httpapi"
	"minibooking/backend/internal/sqlite"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("booking", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) != 1 || (args[0] != "migrate" && args[0] != "seed" && args[0] != "serve") {
		return errors.New("usage: booking migrate|seed|serve")
	}
	timeZone := env("BOOKING_TIME_ZONE", "Asia/Bishkek")
	if timeZone == "Local" {
		return errors.New("BOOKING_TIME_ZONE must name an IANA time zone")
	}
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return fmt.Errorf("load BOOKING_TIME_ZONE: %w", err)
	}
	store, err := sqlite.Open(ctx, env("BOOKING_DB_PATH", "./data/booking.db"))
	if err != nil {
		return err
	}
	defer store.Close()

	if args[0] == "migrate" {
		return store.Migrate(ctx)
	}
	if err := store.CheckSchema(ctx); err != nil {
		return err
	}
	if args[0] == "seed" {
		return store.SeedSlots(ctx, booking.WeekSlots(time.Now(), location))
	}
	service := booking.NewService(store, location, time.Now)
	return serve(ctx, env("BOOKING_HTTP_ADDR", ":8080"), httpapi.NewHandler(service))
}

func serve(ctx context.Context, addr string, handler http.Handler) error {
	server := &http.Server{
		Addr: addr, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	stopped := make(chan error, 1)
	go func() {
		slog.Info("booking API listening", "address", addr)
		stopped <- server.ListenAndServe()
	}()

	select {
	case err := <-stopped:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownErr := server.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			_ = server.Close()
		}
		if err := <-stopped; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return shutdownErr
	}
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
