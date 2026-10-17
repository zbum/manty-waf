package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/zbum/manty-waf/internal/config"
	"github.com/zbum/manty-waf/internal/proxy"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           proxy.New(cfg.Upstream, proxy.AllowAll{}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("수신 %s, 업스트림 %s", cfg.Listen, cfg.Upstream)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
