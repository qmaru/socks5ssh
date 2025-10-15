package tunnel

import (
	"context"
	"log"
	"net"
	"sync"

	"socks5ssh/http"
	"socks5ssh/socks5"
	"socks5ssh/ssh"

	gossh "golang.org/x/crypto/ssh"
)

type Tunnel struct {
	Debug          bool
	DNSServer      string
	LocalAddress   string
	RemoteAddress  string
	RemoteUser     string
	RemoteAuthData string
	RemoteAuthType int
	sshClient      *ssh.Client
	sshMux         sync.Mutex
}

func (tun *Tunnel) getSSHClient() (*gossh.Client, error) {
	tun.sshMux.Lock()
	defer tun.sshMux.Unlock()

	if tun.sshClient == nil {
		sshConf := &ssh.Config{
			Address:  tun.RemoteAddress,
			User:     tun.RemoteUser,
			AuthData: tun.RemoteAuthData,
			AuthType: tun.RemoteAuthType,
		}
		tun.sshClient = ssh.New(sshConf)
	}

	return tun.sshClient.Connect()
}

func (tun *Tunnel) Socks5Run() error {
	s5Conf := &socks5.Config{
		Debug:     tun.Debug,
		Address:   tun.LocalAddress,
		DNSServer: tun.DNSServer,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			sshConn, err := tun.getSSHClient()
			if err != nil {
				return nil, err
			}
			if tun.Debug {
				log.Printf("[Connect] %s -> %s\n", sshConn.LocalAddr().String(), sshConn.RemoteAddr().String())
				log.Printf("[Connect] %s -> %s\n", sshConn.LocalAddr().String(), addr)
			}
			return sshConn.Dial(network, addr)
		},
	}
	socks5Server := socks5.New(s5Conf)
	return socks5Server.Run()
}

func (tun *Tunnel) HTTPRun() error {
	httpConf := &http.Config{
		Address: tun.LocalAddress,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			sshConn, err := tun.getSSHClient()
			if err != nil {
				return nil, err
			}
			if tun.Debug {
				log.Printf("[Connect] %s -> %s -> %s\n", sshConn.LocalAddr(), sshConn.RemoteAddr(), addr)
			}
			return sshConn.Dial(network, addr)
		},
	}
	httpServer := http.New(httpConf)
	return httpServer.Run()
}

func (tun *Tunnel) Close() error {
	tun.sshMux.Lock()
	defer tun.sshMux.Unlock()

	if tun.sshClient != nil {
		return tun.sshClient.Close()
	}
	return nil
}
