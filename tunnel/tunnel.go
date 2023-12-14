package tunnel

import (
	"context"
	"net"

	"socks5ssh/http"
	"socks5ssh/socks5"
	"socks5ssh/ssh"
)

type Tunnel struct {
	DNSServer      string
	LocalAddress   string
	RemoteAddress  string
	RemoteUser     string
	RemoteAuthData string
	RemoteAuthType int
}

func (tun *Tunnel) Socks5Run() error {
	sshConf := &ssh.Config{
		Address:  tun.RemoteAddress,
		User:     tun.RemoteUser,
		AuthData: tun.RemoteAuthData,
		AuthType: tun.RemoteAuthType,
	}

	sshClient := ssh.New(sshConf)
	sshConn, err := sshClient.Connect()
	if err != nil {
		return err
	}
	defer sshConn.Close()

	s5Conf := &socks5.Config{
		Address:   tun.LocalAddress,
		DNSServer: tun.DNSServer,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sshConn.Dial(network, addr)
		},
	}
	socks5Server := socks5.New(s5Conf)
	return socks5Server.Run()
}

func (tun *Tunnel) HTTPRun() error {
	sshConf := &ssh.Config{
		Address:  tun.RemoteAddress,
		User:     tun.RemoteUser,
		AuthData: tun.RemoteAuthData,
		AuthType: tun.RemoteAuthType,
	}

	sshClient := ssh.New(sshConf)
	sshConn, err := sshClient.Connect()
	if err != nil {
		return err
	}
	defer sshConn.Close()

	httpConf := &http.Config{
		Address: tun.LocalAddress,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sshConn.Dial(network, addr)
		},
	}
	httpServer := http.New(httpConf)
	return httpServer.Run()
}
