package functions

import (
	"testing"

	"github.com/shirou/gopsutil/v4/net"
)

func TestHasRoutableAddress(t *testing.T) {
	cases := []struct {
		name string
		in   net.InterfaceStat
		want bool
	}{
		{"wifi with an address", net.InterfaceStat{Flags: []string{"up", "broadcast"}, Addrs: net.InterfaceAddrList{{Addr: "192.168.1.20/24"}}}, true},
		{"loopback is always up", net.InterfaceStat{Flags: []string{"up", "loopback"}, Addrs: net.InterfaceAddrList{{Addr: "127.0.0.1/8"}}}, false},
		{"self-assigned only", net.InterfaceStat{Flags: []string{"up"}, Addrs: net.InterfaceAddrList{{Addr: "169.254.10.4/16"}, {Addr: "fe80::1/64"}}}, false},
		{"down", net.InterfaceStat{Flags: []string{"broadcast"}, Addrs: net.InterfaceAddrList{{Addr: "10.0.0.5/8"}}}, false},
		{"ipv6 global", net.InterfaceStat{Flags: []string{"up"}, Addrs: net.InterfaceAddrList{{Addr: "2001:db8::5/64"}}}, true},
	}
	for _, c := range cases {
		if got := hasRoutableAddress(c.in); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}
