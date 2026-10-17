package config

import (
	"strings"
	"testing"
)

func TestParseRequiresHTTPUpstream(t *testing.T) {
	cfg, err := Parse([]string{"-listen", "127.0.0.1:18080", "-upstream", "http://app.internal:8081"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:18080" {
		t.Fatalf("listen = %q", cfg.Listen)
	}
	if cfg.Upstream.String() != "http://app.internal:8081" {
		t.Fatalf("upstream = %s", cfg.Upstream)
	}
	if cfg.Mode != ModeReverseProxy {
		t.Fatalf("mode = %s", cfg.Mode)
	}
}

func TestParseHelpPrintsFlags(t *testing.T) {
	_, err := Parse([]string{"-h"})
	if err == nil || !strings.Contains(err.Error(), "-upstream") {
		t.Fatalf("help = %v", err)
	}
}

func TestParseRejectsMissingOrRelativeUpstream(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Fatal("missing upstream was accepted")
	}
	if _, err := Parse([]string{"-upstream", "/app"}); err == nil {
		t.Fatal("relative upstream was accepted")
	}
	if _, err := Parse([]string{"-upstream", "ftp://app.internal"}); err == nil {
		t.Fatal("non-http upstream was accepted")
	}
}

func TestParseSelectsDeploymentMode(t *testing.T) {
	agent, err := Parse([]string{"-mode", "agent", "-listen", "127.0.0.1:8080", "-upstream", "http://10.0.0.8:8081"})
	if err != nil {
		t.Fatal(err)
	}
	if agent.Mode != ModeAgent {
		t.Fatalf("mode = %s", agent.Mode)
	}
	if _, err := Parse([]string{"-mode", "agent", "-listen", ":8080", "-upstream", "http://10.0.0.8:8081"}); err == nil {
		t.Fatal("agent mode accepted a non-loopback listen address")
	}

	sidecar, err := Parse([]string{"-mode", "sidecar", "-listen", "localhost:8080", "-upstream", "http://127.0.0.1:8081"})
	if err != nil {
		t.Fatal(err)
	}
	if sidecar.Mode != ModeSidecar {
		t.Fatalf("mode = %s", sidecar.Mode)
	}
	if _, err := Parse([]string{"-mode", "sidecar", "-listen", "127.0.0.1:8080", "-upstream", "http://app.internal:8081"}); err == nil {
		t.Fatal("sidecar mode accepted a non-loopback upstream")
	}

	ingress, err := Parse([]string{"-mode", "ingress", "-route", "Shop.Example=http://10.0.0.2:8080", "-route", "pay.example=http://10.0.0.3:8080"})
	if err != nil {
		t.Fatal(err)
	}
	if ingress.Routes["shop.example"].Host != "10.0.0.2:8080" || ingress.Routes["pay.example"] == nil {
		t.Fatalf("routes = %#v", ingress.Routes)
	}
	if _, err := Parse([]string{"-mode", "ingress", "-upstream", "http://10.0.0.2:8080"}); err == nil {
		t.Fatal("ingress mode accepted -upstream")
	}

	if _, err := Parse([]string{"-mode", "bridge"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse([]string{"-mode", "bridge", "-upstream", "http://10.0.0.2:8080"}); err == nil {
		t.Fatal("bridge mode accepted -upstream")
	}
	if _, err := Parse([]string{"-mode", "tap"}); err == nil {
		t.Fatal("unknown mode was accepted")
	}
}
