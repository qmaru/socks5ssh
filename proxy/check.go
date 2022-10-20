package proxy

import (
	"net"
	"strconv"
	"strings"
)

func addrCheck(addr string) bool {
	ipRes := net.ParseIP(addr)
	if ipRes == nil {
		_, err := net.LookupHost(addr)
		return err == nil
	}
	return true
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
		addrAndPort := strings.Split(address, ":")
		addr := addrAndPort[0]
		port := addrAndPort[1]

		if !addrCheck(addr) {
			return "[Address] Addr Error: " + addr, false
		}

		if !portCheck(port) {
			return "[Address] Port Error (1-65535): " + port, false
		}

		return "", true
	}
	return "[Address] Format Error: Addr:Port", false
}
