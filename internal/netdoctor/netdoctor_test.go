package netdoctor

import (
	"strings"
	"testing"
)

func TestParseHosts(t *testing.T) {
	hosts := `# Copyright (c) 1993-2009 Microsoft Corp.
#	127.0.0.1       localhost
127.0.0.1 localhost
::1 localhost
10.0.0.5   intranet.example.com   wiki.example.com  # office
0.0.0.0 ads.example.net
127.0.0.1 myapp.test
not-an-ip whatever.com
`
	got := parseHosts(strings.NewReader(hosts))
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3 (localhost lines and junk left out): %+v", len(got), got)
	}
	if got[0].Kind != "redirect" || got[0].IP != "10.0.0.5" || len(got[0].Names) != 2 {
		t.Errorf("redirect = %+v", got[0])
	}
	if got[1].Kind != "block" || got[1].Names[0] != "ads.example.net" {
		t.Errorf("block = %+v", got[1])
	}
	if got[2].Kind != "local" || got[2].Names[0] != "myapp.test" {
		t.Errorf("local = %+v", got[2])
	}
}

func TestParseWinHTTP(t *testing.T) {
	direct := "\r\nCurrent WinHTTP proxy settings:\r\n\r\n    Direct access (no proxy server).\r\n"
	if got := parseWinHTTP(direct); got != "" {
		t.Errorf("direct = %q", got)
	}
	proxied := "Current WinHTTP proxy settings:\r\n\r\n    Proxy Server(s) :  proxy.corp.local:8080\r\n    Bypass List     :  <local>\r\n"
	if got := parseWinHTTP(proxied); got != "proxy.corp.local:8080" {
		t.Errorf("proxied = %q", got)
	}
}

func TestFirstProxy(t *testing.T) {
	cases := map[string]string{
		"proxy:8080":                          "proxy:8080",
		"http=proxy:80;https=proxy:443":       "proxy:80",
		"ftp=f:21;https=secure:443;socks=s:1": "secure:443",
	}
	for in, want := range cases {
		if got := firstProxy(in); got != want {
			t.Errorf("firstProxy(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAdapterStep(t *testing.T) {
	if s := adapterStep(nil); s.Status != "fail" || s.Fix == "" {
		t.Errorf("no adapter: %+v", s)
	}
	if s := adapterStep(&Adapter{Name: "Wi-Fi", Kind: "Wi-Fi", IPv4: "169.254.10.4"}); s.Status != "fail" {
		t.Errorf("self-assigned address should fail: %+v", s)
	}
	if s := adapterStep(&Adapter{Name: "Wi-Fi", Kind: "Wi-Fi", IPv4: "192.168.1.20"}); s.Status != "ok" {
		t.Errorf("normal address: %+v", s)
	}
}

func TestSiteBlockedByHosts(t *testing.T) {
	hosts := []HostsEntry{{IP: "0.0.0.0", Names: []string{"example.org"}, Kind: "block"}}
	s := siteStep(t.Context(), "https://example.org/page", Proxy{}, hosts)
	if s.Status != "fail" || !strings.Contains(s.Detail, "blocks example.org") || s.Name != "example.org" {
		t.Errorf("got %+v", s)
	}
}

func TestSummaryPicksFirstBreak(t *testing.T) {
	r := Report{Steps: []Step{
		{ID: "adapter", Status: "ok"},
		{ID: "router", Status: "warn"},
		{ID: "dns", Status: "fail"},
		{ID: "internet", Status: "fail"},
	}}
	finish(&r)
	if r.Healthy || !strings.Contains(r.Summary, "DNS") {
		t.Errorf("summary = %q, healthy = %v", r.Summary, r.Healthy)
	}

	ok := Report{Steps: []Step{{ID: "adapter", Status: "ok"}, {ID: "router", Status: "warn"}, {ID: "internet", Status: "ok"}}}
	finish(&ok)
	if !ok.Healthy || !strings.Contains(ok.Summary, "online") {
		t.Errorf("a router that ignores ping is not a problem on its own: %q", ok.Summary)
	}
}
