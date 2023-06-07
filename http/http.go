package http

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

type Config struct {
	Address      string
	ReadTimeout  int
	WriteTimeout int
	DialContext  func(ctx context.Context, network, addr string) (net.Conn, error)
}

type Server struct {
	config *Config
}

func New(conf *Config) *Server {
	if conf.ReadTimeout == 0 {
		conf.ReadTimeout = 5
	}

	if conf.WriteTimeout == 0 {
		conf.WriteTimeout = 15
	}

	server := &Server{
		config: conf,
	}
	return server
}

func (serv *Server) httpConnect(resWriter http.ResponseWriter, req *http.Request, httpTransport *http.Transport) {
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

func (serv *Server) Run() error {
	httpTransport := &http.Transport{
		DialContext: serv.config.DialContext,
	}

	server := &http.Server{
		ReadTimeout:  time.Duration(serv.config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(serv.config.WriteTimeout) * time.Second,
		Addr:         serv.config.Address,
		Handler: http.HandlerFunc(func(resWriter http.ResponseWriter, req *http.Request) {
			serv.httpConnect(resWriter, req, httpTransport)
		}),
	}
	return server.ListenAndServe()
}
