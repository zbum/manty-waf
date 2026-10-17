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
