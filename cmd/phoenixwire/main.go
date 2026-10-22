package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xcoder2088/PhoenixWire/internal/web"
	"github.com/xcoder2088/PhoenixWire/internal/wire"
)

func main() {
	addr := os.Getenv("PHOENIXWIRE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8788"
	}
	api, closeServices, err := wire.FromEnv()
	if err != nil {
		log.Fatal("Phoenix Wire configuration unavailable: ", err)
	}
	defer closeServices()
	s := &http.Server{Addr: addr, Handler: web.NewHandler(api), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	log.Printf("Phoenix Wire: http://%s (integration configured: %t)", addr, api != nil)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- s.ListenAndServe() }()
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Print("Phoenix Wire HTTP server stopped: ", err)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdown)
	}
}
