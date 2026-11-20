package tunnel

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

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

func (tun *Tunnel) dialContext(ctx context.Context, resolver *socks5.DNSResolver, network, addr string) (net.Conn, error) {
	sshConn, err := tun.getSSHClient()
	if err != nil {
		return nil, err
	}

	requestedAddr := addr
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(host) == nil {
		_, ip, err := resolver.Resolve(ctx, host)
		if err != nil {
			return nil, err
		}
		addr = net.JoinHostPort(ip.String(), port)
	}

	if tun.Debug {
		log.Printf("[Connect] target=%s resolved=%s ssh=%s\n", requestedAddr, addr, sshConn.RemoteAddr())
	}
	dialCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	conn, err := sshConn.DialContext(dialCtx, network, addr)
	if err != nil && tun.Debug {
		log.Printf("[Connect] target=%s resolved=%s error=%v\n", requestedAddr, addr, err)
	}
	return conn, err
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
			Debug:    tun.Debug,
		}
		tun.sshClient = ssh.New(sshConf)
	}

	return tun.sshClient.Connect()
}

func (tun *Tunnel) Socks5Run(ready chan<- struct{}) error {
	if _, err := tun.getSSHClient(); err != nil {
		return err
	}
	resolver := &socks5.DNSResolver{
		Debug:  tun.Debug,
		Server: tun.DNSServer,
	}
	s5Conf := &socks5.Config{
		Debug:     tun.Debug,
		Address:   tun.LocalAddress,
		DNSServer: tun.DNSServer,
		Resolver:  resolver,
		Ready:     ready,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return tun.dialContext(ctx, resolver, network, addr)
		},
	}
	socks5Server := socks5.New(s5Conf)
	return socks5Server.Run()
}

func (tun *Tunnel) HTTPRun(ready chan<- struct{}) error {
	if _, err := tun.getSSHClient(); err != nil {
		return err
	}
	resolver := &socks5.DNSResolver{
		Debug:  tun.Debug,
		Server: tun.DNSServer,
	}
	httpConf := &http.Config{
		Address: tun.LocalAddress,
		Ready:   ready,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return tun.dialContext(ctx, resolver, network, addr)
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
