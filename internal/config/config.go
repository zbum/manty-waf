package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"
)

// Config is how the proxy sits in front of one upstream.
type Config struct {
	Listen   string
	Upstream *url.URL
}

// Parse reads -listen and -upstream. The upstream must be an absolute http or https URL.
func Parse(args []string) (Config, error) {
	fs := flag.NewFlagSet("manty-waf", flag.ContinueOnError)
	var usage strings.Builder
	fs.SetOutput(&usage)
	listen := fs.String("listen", ":8080", "프록시가 받을 주소")
	upstream := fs.String("upstream", "", "보호 대상 원본 주소")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Config{}, errors.New(strings.TrimSpace(usage.String()))
		}
		return Config{}, err
	}
	raw := strings.TrimSpace(*upstream)
	if raw == "" {
		return Config{}, errors.New("업스트림 주소가 필요합니다")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return Config{}, fmt.Errorf("업스트림 주소가 올바르지 않습니다: %s", raw)
	}
	listenAddr := strings.TrimSpace(*listen)
	if listenAddr == "" {
		return Config{}, errors.New("수신 주소가 필요합니다")
	}
	return Config{Listen: listenAddr, Upstream: parsed}, nil
}
