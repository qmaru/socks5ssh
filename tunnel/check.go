package tunnel

import (
	"errors"
	"net"
	"strconv"
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

func AddressChecker(address string) error {
	addr, port, err := net.SplitHostPort(address)
	if err != nil {
		return errors.New("address error: <host>:<port>")
	}
	if err := addrCheck(addr); err != nil {
		return err
	}
	return portCheck(port)
}
