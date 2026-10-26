// Package netdoctor works out where an internet connection breaks: the network
// adapter, the router, DNS, the internet itself, or one particular site. It also
// reports the proxy settings and hosts-file entries that quietly change where
// traffic goes.
package netdoctor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Step is one link in the chain, checked in order
type Step struct {
	ID     string `json:"id"`     // adapter | router | dns | internet | site
	Name   string `json:"name"`   // shown to the user
	Status string `json:"status"` // ok | warn | fail | skip
	Detail string `json:"detail"` // what was found, in plain words
	Fix    string `json:"fix"`    // what to try, when it is not ok
	Ms     int64  `json:"ms"`     // how long the check took; 0 when not timed
}

type Adapter struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"` // Wi-Fi | Ethernet | Other
	IPv4    string   `json:"ipv4"`
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
}

type Proxy struct {
	Enabled    bool   `json:"enabled"`     // the proxy browsers use (Windows Internet settings)
	Server     string `json:"server"`      // host:port
	AutoConfig string `json:"auto_config"` // a PAC script address, if set
	Bypass     string `json:"bypass"`
	WinHTTP    string `json:"winhttp"` // the system-wide proxy services use, or "" for direct
}

type HostsEntry struct {
	IP    string   `json:"ip"`
	Names []string `json:"names"`
	// points a name somewhere other than this PC, or blocks it (0.0.0.0)
	Kind string `json:"kind"` // redirect | block | local
}

type Report struct {
	Steps   []Step       `json:"steps"`
	Adapter *Adapter     `json:"adapter"`
	Proxy   Proxy        `json:"proxy"`
	Hosts   []HostsEntry `json:"hosts"`
	// one line: where it breaks, or that all is well
	Summary string `json:"summary"`
	Healthy bool   `json:"healthy"`
}

// Windows' own connectivity check: the same address it uses to decide whether
// to show "No internet" in the taskbar
const (
	probeURL  = "http://www.msftconnecttest.com/connecttest.txt"
	probeText = "Microsoft Connect Test"
	dnsProbe  = "www.microsoft.com"
)

// Run checks every link in order. site is optional ("github.com" or a URL).
// onStep, when set, is called as each check finishes so the screen can follow along.
func Run(ctx context.Context, site string, onStep func(Step)) Report {
	r := Report{}
	add := func(s Step) {
		r.Steps = append(r.Steps, s)
		if onStep != nil {
			onStep(s)
		}
	}

	adapter := primaryAdapter()
	r.Adapter = adapter
	add(adapterStep(adapter))
	r.Proxy = readProxy(ctx)
	r.Hosts = readHosts()
	if adapter == nil {
		add(Step{ID: "router", Name: "Router", Status: "skip"})
		add(Step{ID: "dns", Name: "Name lookup (DNS)", Status: "skip"})
		add(Step{ID: "internet", Name: "Internet", Status: "skip"})
		if site != "" {
			add(Step{ID: "site", Name: siteName(site), Status: "skip"})
		}
		finish(&r)
		return r
	}

	add(routerStep(adapter.Gateway))
	add(dnsStep(ctx, adapter.DNS))
	add(internetStep(ctx, r.Proxy))
	if site != "" {
		add(siteStep(ctx, site, r.Proxy, r.Hosts))
	}
	finish(&r)
	return r
}

func adapterStep(a *Adapter) Step {
	s := Step{ID: "adapter", Name: "Network adapter"}
	if a == nil {
		s.Status = "fail"
		s.Detail = "No network adapter is connected with an address."
		s.Fix = "Turn Wi-Fi on and join a network, or plug in the network cable. Check that airplane mode is off."
		return s
	}
	s.Status = "ok"
	s.Detail = fmt.Sprintf("%s, address %s", a.Name, a.IPv4)
	if a.Kind != "Other" && !strings.EqualFold(a.Name, a.Kind) {
		s.Detail = fmt.Sprintf("%s (%s), address %s", a.Name, a.Kind, a.IPv4)
	}
	if strings.HasPrefix(a.IPv4, "169.254.") {
		s.Status = "fail"
		s.Detail = fmt.Sprintf("%s has no proper address (%s): the router did not give it one.", a.Name, a.IPv4)
		s.Fix = "Restart the router, then disconnect and reconnect. If it keeps happening, the router's DHCP may be off."
	}
	return s
}

func routerStep(gateway string) Step {
	s := Step{ID: "router", Name: "Router"}
	if gateway == "" {
		s.Status = "fail"
		s.Detail = "This network has no router (default gateway) set."
		s.Fix = "Reconnect to the network. If the address was typed in by hand, add the router's address as the default gateway."
		return s
	}
	ms, err := ping(gateway, 1500*time.Millisecond)
	if err != nil {
		// many routers and corporate networks simply ignore ping
		s.Status = "warn"
		s.Detail = fmt.Sprintf("The router (%s) did not answer a ping. Some routers ignore ping, so this alone may be fine.", gateway)
		s.Fix = "If the next checks fail too, restart the router."
		return s
	}
	s.Ms = ms
	s.Status = "ok"
	s.Detail = fmt.Sprintf("Router %s answered in %d ms", gateway, ms)
	if ms > 100 {
		s.Status = "warn"
		s.Detail = fmt.Sprintf("Router %s is slow to answer (%d ms). The Wi-Fi signal may be weak or the network busy.", gateway, ms)
		s.Fix = "Move closer to the router, or use a cable to compare."
	}
	return s
}

func resolverFor(server string) *net.Resolver {
	if server == "" {
		return net.DefaultResolver
	}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, "udp", net.JoinHostPort(server, "53"))
		},
	}
}

func dnsStep(ctx context.Context, servers []string) Step {
	s := Step{ID: "dns", Name: "Name lookup (DNS)"}
	lookupCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	began := time.Now()
	_, err := net.DefaultResolver.LookupHost(lookupCtx, dnsProbe)
	s.Ms = time.Since(began).Milliseconds()
	if err == nil {
		s.Status = "ok"
		s.Detail = fmt.Sprintf("Names are found in %d ms", s.Ms)
		if s.Ms > 300 {
			s.Status = "warn"
			s.Detail = fmt.Sprintf("Names are found, but slowly (%d ms). Pages will take longer to start loading.", s.Ms)
			s.Fix = "A public DNS server such as 1.1.1.1 or 8.8.8.8 is often faster than the router's."
		}
		return s
	}

	// the system lookup failed; see whether any configured server answers directly
	for _, server := range servers {
		c, cancel := context.WithTimeout(ctx, 3*time.Second)
		_, err := resolverFor(server).LookupHost(c, dnsProbe)
		cancel()
		if err == nil {
			s.Status = "warn"
			s.Detail = fmt.Sprintf("Windows could not look up names, but the DNS server %s answers directly.", server)
			s.Fix = "Reconnect to the network. If that does not help, restarting the PC clears Windows' DNS cache."
			return s
		}
	}
	s.Status = "fail"
	s.Detail = "Website names cannot be turned into addresses, so no site will load by name."
	s.Fix = "Restart the router. If that does not help, set a public DNS server (1.1.1.1 or 8.8.8.8) in the adapter's settings."
	return s
}

// clientFor returns an HTTP client that goes through the proxy browsers use, when one is set
func clientFor(p Proxy, timeout time.Duration) *http.Client {
	transport := &http.Transport{Proxy: nil}
	if p.Enabled && p.Server != "" {
		if u, err := url.Parse("http://" + firstProxy(p.Server)); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		// a captive portal (hotel or airport sign-in page) answers with a redirect
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// "http=proxy:80;https=proxy:443" or "proxy:8080" -> the address for http
func firstProxy(server string) string {
	if !strings.Contains(server, "=") {
		return server
	}
	for _, part := range strings.Split(server, ";") {
		if k, v, ok := strings.Cut(part, "="); ok && (strings.EqualFold(k, "http") || strings.EqualFold(k, "https")) {
			return v
		}
	}
	return server
}

func internetStep(ctx context.Context, p Proxy) Step {
	s := Step{ID: "internet", Name: "Internet"}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	began := time.Now()
	resp, err := clientFor(p, 6*time.Second).Do(req)
	s.Ms = time.Since(began).Milliseconds()
	if err != nil {
		s.Status = "fail"
		s.Detail = "The internet cannot be reached from this network."
		s.Fix = "Restart the router. If other devices are offline too, the problem is with the internet provider."
		if p.AutoConfig != "" && !p.Enabled {
			s.Status = "warn"
			s.Detail = "A direct connection is blocked, but this PC uses a proxy script (" + p.AutoConfig + ") that browsers follow."
			s.Fix = "If web pages load in the browser, this network only allows traffic through its proxy, which is normal at work."
		}
		return s
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode == http.StatusOK && strings.Contains(string(body), probeText) {
		s.Status = "ok"
		s.Detail = fmt.Sprintf("Connected to the internet (%d ms)", s.Ms)
		if p.Enabled {
			s.Detail += ", through the proxy " + firstProxy(p.Server)
		}
		return s
	}
	s.Status = "warn"
	s.Detail = "This network is showing its own page instead of the internet: usually a sign-in page (hotel, airport, café)."
	s.Fix = "Open a browser and visit any site to see the sign-in page."
	return s
}

func siteName(site string) string {
	host := site
	if u, err := url.Parse(site); err == nil && u.Host != "" {
		host = u.Host
	}
	return strings.TrimSuffix(host, "/")
}

func siteStep(ctx context.Context, site string, p Proxy, hosts []HostsEntry) Step {
	host := siteName(site)
	s := Step{ID: "site", Name: host}

	for _, h := range hosts {
		for _, n := range h.Names {
			if strings.EqualFold(n, host) && h.Kind != "local" {
				s.Status = "fail"
				s.Detail = fmt.Sprintf("The hosts file sends %s to %s, so it never reaches the real site.", host, h.IP)
				if h.Kind == "block" {
					s.Detail = fmt.Sprintf("The hosts file blocks %s (it points it at %s).", host, h.IP)
				}
				s.Fix = `Remove the line for it from C:\Windows\System32\drivers\etc\hosts (needs administrator rights).`
				return s
			}
		}
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	addrs, err := net.DefaultResolver.LookupHost(lookupCtx, host)
	cancel()
	if err != nil || len(addrs) == 0 {
		s.Status = "fail"
		s.Detail = fmt.Sprintf("%s could not be found. Check the spelling; the site may also be down or gone.", host)
		return s
	}

	target := "https://" + host + "/"
	if u, err := url.Parse(site); err == nil && u.Scheme != "" && u.Host != "" {
		target = site
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, target, nil)
	began := time.Now()
	resp, err := clientFor(p, 8*time.Second).Do(req)
	s.Ms = time.Since(began).Milliseconds()
	if err != nil {
		s.Status = "fail"
		s.Detail = fmt.Sprintf("%s was found (%s) but did not answer.", host, addrs[0])
		s.Fix = "The site may be down, or this network may block it. Try it on another network, such as a phone hotspot."
		return s
	}
	resp.Body.Close()
	switch {
	case resp.StatusCode < 400:
		s.Status = "ok"
		s.Detail = fmt.Sprintf("%s answered in %d ms", host, s.Ms)
	case resp.StatusCode == 403 || resp.StatusCode == 451:
		s.Status = "warn"
		s.Detail = fmt.Sprintf("%s answered, but refused the request (%d). It may block this network or region.", host, resp.StatusCode)
	case resp.StatusCode >= 500:
		s.Status = "warn"
		s.Detail = fmt.Sprintf("%s answered with a server error (%d): the problem is on the site's side.", host, resp.StatusCode)
	default:
		s.Status = "ok"
		s.Detail = fmt.Sprintf("%s answered (%d) in %d ms", host, resp.StatusCode, s.Ms)
	}
	return s
}

// finish writes the one-line summary: the first link that broke, or all clear
func finish(r *Report) {
	for _, s := range r.Steps {
		if s.Status == "fail" {
			r.Summary = summaryFor(s)
			return
		}
	}
	for _, s := range r.Steps {
		if s.Status == "warn" && s.ID != "router" {
			r.Summary = summaryFor(s)
			r.Healthy = true
			return
		}
	}
	r.Healthy = true
	r.Summary = "Everything checks out: this PC is online."
}

func summaryFor(s Step) string {
	switch s.ID {
	case "adapter":
		return "The problem is on this PC: it is not connected to a network."
	case "router":
		return "The problem is between this PC and the router."
	case "dns":
		return "The connection works, but website names cannot be looked up (DNS)."
	case "internet":
		if s.Status == "warn" {
			return "Connected, but this network wants something first: " + strings.ToLower(s.Detail[:1]) + s.Detail[1:]
		}
		return "The router is reachable, but the internet is not: the router or the provider is the problem."
	default:
		return "The connection works, but " + s.Name + " does not."
	}
}

// --- hosts file ---

func hostsPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "drivers", "etc", "hosts")
}

func readHosts() []HostsEntry {
	f, err := os.Open(hostsPath())
	if err != nil {
		return []HostsEntry{}
	}
	defer f.Close()
	return parseHosts(f)
}

// parseHosts lists the lines that change where names go. Plain localhost lines,
// which every PC has, are left out.
func parseHosts(r io.Reader) []HostsEntry {
	entries := []HostsEntry{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ip, names := fields[0], fields[1:]
		if net.ParseIP(ip) == nil {
			continue
		}
		onlyLocalhost := true
		for _, n := range names {
			if !strings.EqualFold(n, "localhost") {
				onlyLocalhost = false
			}
		}
		if onlyLocalhost {
			continue
		}
		kind := "redirect"
		switch {
		case ip == "0.0.0.0" || ip == "::":
			kind = "block"
		case strings.HasPrefix(ip, "127.") || ip == "::1":
			kind = "local"
		}
		entries = append(entries, HostsEntry{IP: ip, Names: names, Kind: kind})
	}
	return entries
}

// parseWinHTTP reads `netsh winhttp show proxy`: "" for a direct connection,
// otherwise the proxy server
func parseWinHTTP(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "direct access") {
			return ""
		}
		if k, v, ok := strings.Cut(line, ":"); ok && strings.Contains(strings.ToLower(k), "proxy server") {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
