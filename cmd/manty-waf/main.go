package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/zbum/manty-waf/internal/config"
	"github.com/zbum/manty-waf/internal/originaldst"
	"github.com/zbum/manty-waf/internal/proxy"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	handler := handlerFor(cfg)
	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if cfg.Mode == config.ModeBridge {
		if !originaldst.Supported() {
			log.Printf("이 운영체제에서는 투명 브리지의 원래 목적지를 읽지 못합니다")
		}
		server.ConnContext = bridgeDestination
	}
	log.Printf("모드 %s, 수신 %s", cfg.Mode, cfg.Listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func handlerFor(cfg config.Config) http.Handler {
	switch cfg.Mode {
	case config.ModeIngress:
		return proxy.NewIngress(cfg.Routes, proxy.AllowAll{})
	case config.ModeBridge:
		return proxy.NewBridge(proxy.AllowAll{})
	default:
		return proxy.New(cfg.Upstream, proxy.AllowAll{})
	}
}

func bridgeDestination(ctx context.Context, conn net.Conn) context.Context {
	host, port, err := originaldst.Detect(conn)
	if err != nil {
		return ctx
	}
	upstream := &url.URL{Scheme: "http", Host: net.JoinHostPort(host, strconv.Itoa(port))}
	return proxy.WithDestination(ctx, upstream)
}
