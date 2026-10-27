package functions

import (
	"net"
	"time"
)

// IsConnectedToInternet is true when the internet answers: Google's public DNS
// first, then the host Windows itself uses to decide "No internet" (many work
// networks block outside DNS servers, which alone would read as offline)
func IsConnectedToInternet() bool {
	timeout := 2 * time.Second
	for _, addr := range []string{"8.8.8.8:53", "www.msftconnecttest.com:80"} {
		if conn, err := net.DialTimeout("tcp", addr, timeout); err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}
