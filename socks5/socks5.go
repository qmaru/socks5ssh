package socks5

import (
	"context"
	"net"
	"net/url"
	"time"

	socks5 "github.com/armon/go-socks5"
	"github.com/miekg/dns"
)

type Config struct {
	Address   string
	Dial      func(ctx context.Context, network, addr string) (net.Conn, error)
	DNSServer string
}

type Server struct {
	config *Config
}

func New(conf *Config) *Server {
	server := &Server{
		config: conf,
	}
	return server
}

type DNSResolver struct {
	Server string
}

func (d DNSResolver) exchange(name string) (r *dns.Msg, rtt time.Duration, err error) {
	u, err := url.Parse(d.Server)
	if err != nil {
		return nil, 0, err
	}

	client := dns.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(name), dns.TypeA)
	server := ""

	switch u.Scheme {
	case "", "udp":
		if u.Scheme == "" {
			u.Host = d.Server
		}
		client.Net = "udp"
		server = net.JoinHostPort(u.Host, "53")
	case "tcp":
		client.Net = "tcp"
		server = net.JoinHostPort(u.Host, "53")
	case "tls":
		client.Net = "tcp-tls"
		server = net.JoinHostPort(u.Host, "853")
	}
	return client.Exchange(msg, server)
}

func (d DNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	response, _, err := d.exchange(name)
	if err != nil {
		return ctx, nil, err
	}

	if len(response.Answer) < 1 {
		return ctx, nil, err
	}

	for _, ans := range response.Answer {
		if a, ok := ans.(*dns.A); ok {
			return ctx, a.A, err
		}
	}
	return ctx, nil, nil
}

func (serv *Server) Run() error {
	resolver := DNSResolver{
		Server: serv.config.DNSServer,
	}

	conf := &socks5.Config{
		Dial:     serv.config.Dial,
		Resolver: resolver,
	}
	serverSocks, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return serverSocks.ListenAndServe("tcp", serv.config.Address)
}
