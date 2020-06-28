package main

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/lucas-clemente/quic-go/http3"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(5 * time.Minute)
	return tc, nil
}

func ListenAndServe(addr string, tlsConfig *tls.Config, handler http.Handler) error {
	// Prepare listener transports
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer udpConn.Close()

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	tcpConn, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	defer tcpConn.Close()

	// Prepare TLS listener
	tlsConn := tls.NewListener(tcpConn, tlsConfig)
	defer tlsConn.Close()

	// Prepare servers
	httpServer := &http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
        ReadTimeout:  1 * time.Minute,
        WriteTimeout: 1 * time.Minute,
	}

	quicServer := &http3.Server{
		Server: httpServer,
	}

	// Prepare httpServer handler
	httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		quicServer.SetQuicHeaders(w.Header())
		handler.ServeHTTP(w, r)
	})

	// Create error handler channels
	hErr := make(chan error)
	qErr := make(chan error)
	go func() {
		hErr <- httpServer.Serve(tlsConn)
	}()
	go func() {
		qErr <- quicServer.Serve(udpConn)
	}()

	// Synchronise and wait for errors
	select {
		case err := <-hErr:
			quicServer.Close()
			return err
		case err := <-qErr:
			return err
	}
}

func ListenAndServeTLSKeyPair(addr string, cert tls.Certificate, handler http.Handler) error {
	if addr == "" {
		return errors.New("Invalid address string")
	}

	// Prepare TLS configuration
	config := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}
	config.NextProtos = []string{"h2", "http/1.1"}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = cert

	// Prepare HTTP/3 server
	return ListenAndServe(addr, config, handler)
}
