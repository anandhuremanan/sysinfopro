//go:build windows

package netdoctor

import (
	"context"
	"errors"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	ifTypeEthernet = 6
	ifTypeWiFi     = 71
	gaaIncludeGW   = 0x0080
	operUp         = 1
)

func sockaddrIP(sa windows.SocketAddress) net.IP {
	if sa.Sockaddr == nil {
		return nil
	}
	switch sa.Sockaddr.Addr.Family {
	case windows.AF_INET:
		p := (*syscall.RawSockaddrInet4)(unsafe.Pointer(sa.Sockaddr))
		return net.IPv4(p.Addr[0], p.Addr[1], p.Addr[2], p.Addr[3])
	case windows.AF_INET6:
		p := (*syscall.RawSockaddrInet6)(unsafe.Pointer(sa.Sockaddr))
		ip := make(net.IP, 16)
		copy(ip, p.Addr[:])
		return ip
	}
	return nil
}

// primaryAdapter is the connected adapter with an IPv4 address and a default
// gateway, preferring a cable over Wi-Fi; nil when there is none
func primaryAdapter() *Adapter {
	size := uint32(16 * 1024)
	var buf []byte
	for i := 0; i < 3; i++ {
		buf = make([]byte, size)
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, gaaIncludeGW, 0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])), &size)
		if err == nil {
			break
		}
		if !errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			return nil
		}
	}

	var best *Adapter
	bestRank := -1
	for a := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])); a != nil; a = a.Next {
		if a.OperStatus != operUp {
			continue
		}
		ad := Adapter{Name: windows.UTF16PtrToString(a.FriendlyName), DNS: []string{}}
		switch a.IfType {
		case ifTypeEthernet:
			ad.Kind = "Ethernet"
		case ifTypeWiFi:
			ad.Kind = "Wi-Fi"
		default:
			ad.Kind = "Other"
		}
		for u := a.FirstUnicastAddress; u != nil; u = u.Next {
			if ip := sockaddrIP(u.Address); ip != nil && ip.To4() != nil {
				ad.IPv4 = ip.String()
				break
			}
		}
		for g := a.FirstGatewayAddress; g != nil; g = g.Next {
			if ip := sockaddrIP(g.Address); ip != nil && ip.To4() != nil {
				ad.Gateway = ip.String()
				break
			}
		}
		for d := a.FirstDnsServerAddress; d != nil; d = d.Next {
			if ip := sockaddrIP(d.Address); ip != nil && ip.To4() != nil {
				ad.DNS = append(ad.DNS, ip.String())
			}
		}
		if ad.IPv4 == "" {
			continue
		}
		// VPN and virtual adapters rank below real ones; a gateway matters most
		rank := 0
		if ad.Gateway != "" {
			rank += 4
		}
		if ad.Kind == "Ethernet" {
			rank += 2
		} else if ad.Kind == "Wi-Fi" {
			rank += 1
		}
		if strings.Contains(strings.ToLower(ad.Name), "vethernet") || strings.Contains(strings.ToLower(ad.Name), "virtual") {
			rank -= 3
		}
		if rank > bestRank {
			copied := ad
			best, bestRank = &copied, rank
		}
	}
	return best
}

var (
	iphlpapi       = windows.NewLazySystemDLL("iphlpapi.dll")
	icmpCreateFile = iphlpapi.NewProc("IcmpCreateFile")
	icmpClose      = iphlpapi.NewProc("IcmpCloseHandle")
	icmpSendEcho   = iphlpapi.NewProc("IcmpSendEcho")
)

// ping sends one ICMP echo through Windows' ICMP helper, which needs no
// administrator rights, and returns the round trip in milliseconds
func ping(address string, timeout time.Duration) (int64, error) {
	ip := net.ParseIP(address).To4()
	if ip == nil {
		return 0, errors.New("not an IPv4 address")
	}
	handle, _, err := icmpCreateFile.Call()
	if windows.Handle(handle) == windows.InvalidHandle {
		return 0, err
	}
	defer icmpClose.Call(handle)

	payload := []byte("sysinfopro")
	reply := make([]byte, 64+len(payload)+8)
	dest := uint32(ip[0]) | uint32(ip[1])<<8 | uint32(ip[2])<<16 | uint32(ip[3])<<24
	n, _, _ := icmpSendEcho.Call(handle, uintptr(dest),
		uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)), 0,
		uintptr(unsafe.Pointer(&reply[0])), uintptr(len(reply)), uintptr(timeout.Milliseconds()))
	if n == 0 {
		return 0, errors.New("no reply")
	}
	// ICMP_ECHO_REPLY: Address (4 bytes), Status (4), RoundTripTime (4)
	status := *(*uint32)(unsafe.Pointer(&reply[4]))
	if status != 0 {
		return 0, errors.New("no reply")
	}
	return int64(*(*uint32)(unsafe.Pointer(&reply[8]))), nil
}

func readProxy(ctx context.Context) Proxy {
	p := Proxy{}
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err == nil {
		enabled, _, _ := key.GetIntegerValue("ProxyEnable")
		p.Enabled = enabled == 1
		p.Server, _, _ = key.GetStringValue("ProxyServer")
		p.AutoConfig, _, _ = key.GetStringValue("AutoConfigURL")
		p.Bypass, _, _ = key.GetStringValue("ProxyOverride")
		key.Close()
	}
	if !p.Enabled {
		p.Server = ""
	}

	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, "netsh", "winhttp", "show", "proxy")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if out, err := cmd.Output(); err == nil {
		p.WinHTTP = parseWinHTTP(string(out))
	}
	return p
}
