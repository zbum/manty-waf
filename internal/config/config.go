package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Mode is where this process sits on the path to the application.
type Mode string

const (
	ModeReverseProxy Mode = "reverse-proxy"
	ModeBridge       Mode = "bridge"
	ModeAgent        Mode = "agent"
	ModeIngress      Mode = "ingress"
	ModeSidecar      Mode = "sidecar"
)

// Config is the selected deployment and the addresses that go with it.
type Config struct {
	Mode     Mode
	Listen   string
	Upstream *url.URL
	Routes   map[string]*url.URL
}

// Parse reads the deployment mode and the addresses for that mode.
// reverse-proxy, agent, and sidecar use one -upstream.
// ingress uses repeated -route host=http://upstream.
// bridge takes the original destination from the accepted connection.
func Parse(args []string) (Config, error) {
	fs := flag.NewFlagSet("manty-waf", flag.ContinueOnError)
	var usage strings.Builder
	fs.SetOutput(&usage)
	modeFlag := fs.String("mode", string(ModeReverseProxy), "배치 방식: reverse-proxy, bridge, agent, ingress, sidecar")
	listen := fs.String("listen", ":8080", "프록시가 받을 주소")
	upstream := fs.String("upstream", "", "보호 대상 원본 주소")
	var routes routeFlag
	fs.Var(&routes, "route", "ingress 경로. host=http://upstream 형식, 여러 번 쓸 수 있다")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Config{}, errors.New(strings.TrimSpace(usage.String()))
		}
		return Config{}, err
	}

	mode := Mode(strings.TrimSpace(*modeFlag))
	switch mode {
	case ModeReverseProxy, ModeBridge, ModeAgent, ModeIngress, ModeSidecar:
	default:
		return Config{}, fmt.Errorf("알 수 없는 배치 방식: %s", mode)
	}
	listenAddr := strings.TrimSpace(*listen)
	if listenAddr == "" {
		return Config{}, errors.New("수신 주소가 필요합니다")
	}
	parsedUpstream, err := parseUpstream(*upstream)
	if err != nil {
		return Config{}, err
	}
	cfg := Config{Mode: mode, Listen: listenAddr, Upstream: parsedUpstream, Routes: routes.routes}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.Mode != ModeIngress && len(cfg.Routes) > 0 {
		return errors.New("-route는 ingress 모드에서만 사용합니다")
	}
	switch cfg.Mode {
	case ModeReverseProxy:
		if cfg.Upstream == nil {
			return errors.New("업스트림 주소가 필요합니다")
		}
	case ModeAgent:
		if cfg.Upstream == nil {
			return errors.New("업스트림 주소가 필요합니다")
		}
		if !loopbackHost(listenHost(cfg.Listen)) {
			return errors.New("에이전트 모드는 루프백 주소로만 수신합니다")
		}
	case ModeSidecar:
		if cfg.Upstream == nil {
			return errors.New("업스트림 주소가 필요합니다")
		}
		if !loopbackHost(listenHost(cfg.Listen)) {
			return errors.New("사이드카 모드는 루프백 주소로만 수신합니다")
		}
		if !loopbackHost(cfg.Upstream.Hostname()) {
			return errors.New("사이드카 모드의 업스트림은 루프백 주소여야 합니다")
		}
	case ModeIngress:
		if cfg.Upstream != nil {
			return errors.New("ingress 모드는 -upstream 대신 -route를 사용합니다")
		}
		if len(cfg.Routes) == 0 {
			return errors.New("ingress 모드에는 경로가 하나 이상 필요합니다")
		}
	case ModeBridge:
		if cfg.Upstream != nil {
			return errors.New("브리지 모드는 업스트림을 고정하지 않습니다")
		}
	}
	return nil
}

func parseUpstream(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("업스트림 주소가 올바르지 않습니다: %s", raw)
	}
	return parsed, nil
}

func listenHost(listen string) string {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return listen
	}
	return host
}

func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type routeFlag struct {
	routes map[string]*url.URL
}

func (f *routeFlag) String() string { return "" }

func (f *routeFlag) Set(value string) error {
	host, raw, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(host) == "" {
		return fmt.Errorf("경로는 host=http://upstream 형식이어야 합니다: %s", value)
	}
	upstream, err := parseUpstream(raw)
	if err != nil || upstream == nil {
		if err == nil {
			err = fmt.Errorf("업스트림 주소가 올바르지 않습니다: %s", raw)
		}
		return err
	}
	key := canonicalHost(host)
	if key == "" {
		return fmt.Errorf("경로 호스트가 비어 있습니다: %s", value)
	}
	if _, exists := f.routes[key]; exists {
		return fmt.Errorf("경로 호스트가 중복되었습니다: %s", key)
	}
	if f.routes == nil {
		f.routes = map[string]*url.URL{}
	}
	f.routes[key] = upstream
	return nil
}

func canonicalHost(host string) string {
	host = strings.TrimSpace(host)
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	return strings.ToLower(host)
}
