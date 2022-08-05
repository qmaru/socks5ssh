package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/armon/go-socks5"
	"golang.org/x/crypto/ssh"
)

type SSHProxy struct{}

func (proxy *SSHProxy) sshTunnel(sshAddress, sshUser, sshPassword string) (*ssh.Client, error) {
	sshConf := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.Password(sshPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshConn, err := ssh.Dial("tcp", sshAddress, sshConf)
	if err != nil {
		return nil, err
	}
	return sshConn, nil
}

func (proxy *SSHProxy) Socks5Run(localAddress, sshAddress, sshUser, sshPassword string) error {
	sshConn, err := proxy.sshTunnel(sshAddress, sshUser, sshPassword)
	if err != nil {
		return err
	}
	defer sshConn.Close()

	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sshConn.Dial(network, addr)
		},
	}

	serverSocks, err := socks5.New(conf)
	if err != nil {
		return err
	}

	err = serverSocks.ListenAndServe("tcp", localAddress)
	if err != nil {
		return err
	}
	return nil
}

func (proxy *SSHProxy) httpConnect(resWriter http.ResponseWriter, req *http.Request, httpTransport *http.Transport) {
	resp, err := httpTransport.RoundTrip(req)
	if err != nil {
		http.Error(resWriter, err.Error(), http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			resWriter.Header().Add(k, v)
		}
	}

	resWriter.WriteHeader(resp.StatusCode)
	io.Copy(resWriter, resp.Body)
}

func (proxy *SSHProxy) HTTPRun(localAddress, sshAddress, sshUser, sshPassword string) error {
	sshConn, err := proxy.sshTunnel(sshAddress, sshUser, sshPassword)
	if err != nil {
		return err
	}
	defer sshConn.Close()

	httpTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sshConn.Dial(network, addr)
		},
	}

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         localAddress,
		Handler: http.HandlerFunc(func(resWriter http.ResponseWriter, req *http.Request) {
			proxy.httpConnect(resWriter, req, httpTransport)
		}),
	}
	err = server.ListenAndServe()
	if err != nil {
		return err
	}
	return nil
}
