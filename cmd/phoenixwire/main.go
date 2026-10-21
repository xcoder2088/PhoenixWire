package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/xcoder2088/PhoenixWire/internal/web"
)

func main() {
	addr := os.Getenv("PHOENIXWIRE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8788"
	}
	s := &http.Server{Addr: addr, Handler: web.NewHandler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	log.Printf("Phoenix Wire preview: http://%s (payments and email disabled)", addr)
	log.Fatal(s.ListenAndServe())
}
