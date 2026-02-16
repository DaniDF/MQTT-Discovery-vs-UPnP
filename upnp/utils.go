package upnp

import (
	"net"
	"strings"
)

func FindHeader(header string, headerName string) (result string, flagFind bool) {
	for _, h := range strings.Split(header, "\n") {
		if strings.Contains(h, headerName) {
			h, _ = strings.CutPrefix(h, headerName+":")
			flagFind = true
			result = strings.Trim(h, " \t\r\n")
		}
	}

	return result, flagFind
}

func GetLocalIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
