package socks5

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"sync"
	"time"

	socks5 "github.com/armon/go-socks5"
	"github.com/miekg/dns"
)

type Config struct {
	Debug     bool
	Address   string
	Dial      func(ctx context.Context, network, addr string) (net.Conn, error)
	DNSServer string
	Resolver  *DNSResolver
	Ready     chan<- struct{}
}

type Server struct {
	debug  bool
	config *Config
}

type dnsCache struct {
	ip        net.IP
	expiresAt time.Time
}

type DNSResolver struct {
	Debug  bool
	Server string
	cache  sync.Map
}

func New(conf *Config) *Server {
	server := &Server{
		debug:  conf.Debug,
		config: conf,
	}
	return server
}

func (d *DNSResolver) exchange(name string) (r *dns.Msg, rtt time.Duration, err error) {
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
	case "tls", "dot":
		client.Net = "tcp-tls"
		server = net.JoinHostPort(u.Host, "853")
	default:
		return nil, 0, fmt.Errorf("unsupported DNS scheme: %s", u.Scheme)
	}
	return client.Exchange(msg, server)
}

func (d *DNSResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	if cached, ok := d.cache.Load(name); ok {
		entry := cached.(dnsCache)
		if time.Now().Before(entry.expiresAt) {
			if d.Debug {
				log.Printf("[DNS] resolver=cache name=%s ip=%s\n", name, entry.ip.String())
			}
			return ctx, entry.ip, nil
		}
		d.cache.Delete(name)
	}

	if d.Server == "" || d.Server == "local" {
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", name)
		if err != nil {
			if d.Debug {
				log.Printf("[DNS] resolver=local name=%s error=%v\n", name, err)
			}
			return ctx, nil, err
		}
		if len(ips) == 0 {
			err := fmt.Errorf("no A record for %q", name)
			if d.Debug {
				log.Printf("[DNS] resolver=local name=%s error=%v\n", name, err)
			}
			return ctx, nil, err
		}

		ip := ips[0]
		d.cache.Store(name, dnsCache{
			ip:        ip,
			expiresAt: time.Now().Add(60 * time.Second),
		})
		if d.Debug {
			log.Printf("[DNS] resolver=local name=%s ip=%s\n", name, ip.String())
		}
		return ctx, ip, nil
	}

	response, _, err := d.exchange(name)
	if err != nil {
		if d.Debug {
			log.Printf("[DNS] resolver=%s name=%s error=%v\n", d.Server, name, err)
		}
		return ctx, nil, err
	}

	if len(response.Answer) < 1 {
		err := fmt.Errorf("no A record for %q", name)
		if d.Debug {
			log.Printf("[DNS] resolver=%s name=%s error=%v\n", d.Server, name, err)
		}
		return ctx, nil, err
	}

	for _, ans := range response.Answer {
		if a, ok := ans.(*dns.A); ok {
			ttl := time.Duration(a.Hdr.Ttl) * time.Second
			if ttl < 60*time.Second {
				ttl = 60 * time.Second
			}

			d.cache.Store(name, dnsCache{
				ip:        a.A,
				expiresAt: time.Now().Add(ttl),
			})

			if d.Debug {
				log.Printf("[DNS] resolver=%s name=%s ip=%s ttl=%ds\n", d.Server, name, a.A.String(), a.Hdr.Ttl)
			}
			return ctx, a.A, nil
		}
	}
	err = fmt.Errorf("no A record for %q", name)
	if d.Debug {
		log.Printf("[DNS] resolver=%s name=%s error=%v\n", d.Server, name, err)
	}
	return ctx, nil, err
}

func (serv *Server) Run() error {
	resolver := serv.config.Resolver
	if resolver == nil {
		resolver = &DNSResolver{
			Debug:  serv.debug,
			Server: serv.config.DNSServer,
		}
	}

	conf := &socks5.Config{
		Dial:     serv.config.Dial,
		Resolver: resolver,
	}
	serverSocks, err := socks5.New(conf)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", serv.config.Address)
	if err != nil {
		return err
	}
	if serv.config.Ready != nil {
		close(serv.config.Ready)
	}
	return serverSocks.Serve(listener)
}
