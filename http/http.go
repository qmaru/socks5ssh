package http

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

var bufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

type Config struct {
	Address      string
	ReadTimeout  int
	WriteTimeout int
	DialContext  func(ctx context.Context, network, addr string) (net.Conn, error)
	Ready        chan<- struct{}
}

type Server struct {
	config *Config
}

func copyWithBuffer(dst io.Writer, src io.Reader) {
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	io.CopyBuffer(dst, src, *bufPtr)
}

func copyWithCloseWrite(dst io.Writer, src io.Reader) {
	copyWithBuffer(dst, src)
	if closeWriter, ok := dst.(interface{ CloseWrite() error }); ok {
		_ = closeWriter.CloseWrite()
	}
}

func New(conf *Config) *Server {
	server := &Server{
		config: conf,
	}
	return server
}

func (serv *Server) httpRequest(resWriter http.ResponseWriter, req *http.Request, httpTransport *http.Transport) {
	forwardReq := req.Clone(req.Context())
	forwardReq.RequestURI = ""
	if forwardReq.URL.Scheme == "" {
		forwardReq.URL.Scheme = "http"
	}
	if forwardReq.URL.Host == "" {
		forwardReq.URL.Host = req.Host
	}
	forwardReq.Host = ""
	forwardReq.Header.Del("Proxy-Connection")
	forwardReq.Header.Del("Proxy-Authorization")
	forwardReq.Header.Del("Proxy-Authenticate")

	resp, err := httpTransport.RoundTrip(forwardReq)
	if err != nil {
		http.Error(resWriter, err.Error(), http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			resWriter.Header().Add(k, v)
		}
	}

	resWriter.WriteHeader(resp.StatusCode)
	copyWithBuffer(resWriter, resp.Body)
}

func (serv *Server) httpConnect(resWriter http.ResponseWriter, req *http.Request) {
	target := req.Host
	if target == "" {
		target = req.URL.Host
	}
	if target == "" {
		http.Error(resWriter, "missing CONNECT target", http.StatusBadRequest)
		return
	}

	remoteConn, err := serv.config.DialContext(req.Context(), "tcp", target)
	if err != nil {
		http.Error(resWriter, err.Error(), http.StatusBadGateway)
		return
	}
	defer remoteConn.Close()
	if req.ProtoMajor >= 2 {
		serv.http2Connect(resWriter, req, remoteConn)
		return
	}

	hijacker, ok := resWriter.(http.Hijacker)
	if !ok {
		http.Error(resWriter, "HTTP hijacking is not supported", http.StatusInternalServerError)
		return
	}
	clientConn, clientRW, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer clientConn.Close()

	if _, err := fmt.Fprint(clientRW, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}
	if err := clientRW.Flush(); err != nil {
		return
	}

	done := make(chan struct{}, 2)
	go func() {
		copyWithCloseWrite(remoteConn, clientRW)
		done <- struct{}{}
	}()
	go func() {
		copyWithCloseWrite(clientConn, remoteConn)
		done <- struct{}{}
	}()
	<-done
	<-done
}

func (serv *Server) http2Connect(resWriter http.ResponseWriter, req *http.Request, remoteConn net.Conn) {
	resWriter.WriteHeader(http.StatusOK)
	if flusher, ok := resWriter.(http.Flusher); ok {
		flusher.Flush()
	}

	done := make(chan bool, 2)
	go func() {
		copyWithCloseWrite(remoteConn, req.Body)
		done <- true
	}()
	go func() {
		copyWithBuffer(resWriter, remoteConn)
		done <- false
	}()
	select {
	case clientDone := <-done:
		if clientDone {
			select {
			case <-done:
			case <-req.Context().Done():
			}
		} else {
			_ = req.Body.Close()
		}
	case <-req.Context().Done():
	}
}

func (serv *Server) Run() error {
	httpTransport := &http.Transport{
		DialContext:       serv.config.DialContext,
		ForceAttemptHTTP2: true,
	}
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	server := &http.Server{
		ReadTimeout:  time.Duration(serv.config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(serv.config.WriteTimeout) * time.Second,
		Protocols:    &protocols,
		Addr:         serv.config.Address,
		Handler: http.HandlerFunc(func(resWriter http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodConnect {
				serv.httpConnect(resWriter, req)
				return
			}
			serv.httpRequest(resWriter, req, httpTransport)
		}),
	}

	listener, err := net.Listen("tcp", serv.config.Address)
	if err != nil {
		return err
	}
	if serv.config.Ready != nil {
		close(serv.config.Ready)
	}
	return server.Serve(listener)
}
