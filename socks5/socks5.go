package socks5

import (
	"context"
	"net"

	socks5 "github.com/armon/go-socks5"
)

type Config struct {
	Address string
	Dial    func(ctx context.Context, network, addr string) (net.Conn, error)
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

func (serv *Server) Run() error {
	conf := &socks5.Config{
		Dial: serv.config.Dial,
	}
	serverSocks, err := socks5.New(conf)
	if err != nil {
		return err
	}
	return serverSocks.ListenAndServe("tcp", serv.config.Address)
}
