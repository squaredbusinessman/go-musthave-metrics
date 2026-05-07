package agent

import (
	"net"
	"sync"
)

const headerXRealIP = "X-Real-IP"

var (
	hostIPOnce sync.Once
	hostIP     string
)

// HostIP возвращает IP-адрес хоста агента для заголовка X-Real-IP.
func HostIP() string {
	hostIPOnce.Do(func() {
		hostIP = discoverHostIP()
	})
	return hostIP
}

func discoverHostIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	var firstIPv6 string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ip := ipFromAddr(addr)
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			if ipv4 := ip.To4(); ipv4 != nil {
				return ipv4.String()
			}
			if firstIPv6 == "" {
				firstIPv6 = ip.String()
			}
		}
	}

	if firstIPv6 != "" {
		return firstIPv6
	}

	return "127.0.0.1"
}

func ipFromAddr(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}
