package proxy

import (
	"net"
	"strconv"
	"strings"
)

func ipCheck(ip string) bool {
	a := net.ParseIP(ip)
	return a != nil
}

func portCheck(port string) bool {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	if portInt > 0 && portInt < 65536 {
		return true
	}
	return false
}

func AddressChecker(address string) (string, bool) {
	if strings.Contains(address, ":") {
		ipAndPort := strings.Split(address, ":")
		ip := ipAndPort[0]
		port := ipAndPort[1]

		if !ipCheck(ip) {
			return "[Address] IP Error: " + ip, false
		}

		if !portCheck(port) {
			return "[Address] Port Error (1-65535): " + port, false
		}

		return "", true
	}
	return "[Address] Format Error: IP:Port", false
}
