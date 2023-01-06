package proxy

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

func addrCheck(addr string) error {
	ipRes := net.ParseIP(addr)
	if ipRes == nil {
		_, err := net.LookupHost(addr)
		return err
	}
	return nil
}

func portCheck(port string) error {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	if portInt > 0 && portInt < 65536 {
		return nil
	}
	return errors.New("wrong port range: 1-65535")
}

func RunningCheck(address string) bool {
	_, err := net.Dial("tcp", address)
	return err == nil
}

func AddressChecker(address string) error {
	if strings.Contains(address, ":") {
		addrAndPort := strings.Split(address, ":")
		addr := addrAndPort[0]
		port := addrAndPort[1]

		err := addrCheck(addr)
		if err != nil {
			return err
		}
		err = portCheck(port)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("address error: <host>:<port>")
}
