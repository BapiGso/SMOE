package mymiddleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BapiGso/SMOE/internal/moe/mymiddleware"
)

func TestIPExtractor(t *testing.T) {
	cases := []struct {
		name    string
		proxies []string
		remote  string
		xff     string
		want    string
	}{
		{"no_proxy_ignores_xff", nil, "203.0.113.9:4000", "1.2.3.4", "203.0.113.9"},
		{"trusted_proxy", []string{"127.0.0.1"}, "127.0.0.1:4000", "1.2.3.4", "1.2.3.4"},
		{"untrusted_peer_ignores_xff", []string{"127.0.0.1"}, "203.0.113.9:4000", "1.2.3.4", "203.0.113.9"},
		{"private_not_trusted_by_default", []string{"127.0.0.1"}, "10.0.0.2:4000", "1.2.3.4", "10.0.0.2"},
		{"cidr_chain_skips_spoofed_prefix", []string{"127.0.0.1", "173.245.48.0/20"}, "127.0.0.1:4000", "9.9.9.9, 1.2.3.4, 173.245.48.7", "1.2.3.4"},
		{"ipv6", []string{"::1"}, "[::1]:4000", "2001:db8::1", "2001:db8::1"},
		{"no_xff", []string{"127.0.0.1"}, "127.0.0.1:4000", "", "127.0.0.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			extract, err := mymiddleware.IPExtractor(tc.proxies)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if got := extract(req); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIPExtractorInvalid(t *testing.T) {
	if _, err := mymiddleware.IPExtractor([]string{"localhost"}); err == nil {
		t.Fatal("expected error for non-IP entry")
	}
}
