package mdathome

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}

	err = tc.SetKeepAlive(true)
	if err != nil {
		return
	}

	err = tc.SetKeepAlivePeriod(1 * time.Minute)
	if err != nil {
		return
	}

	return tc, nil
}

func listenAndServeTLSKeyPair(addr string, allowHTTP2 bool, cert tls.Certificate, handler http.Handler) error {
	if addr == "" {
		return errors.New("invalid address string")
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 1 * time.Minute,
	}
	config := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = cert

	if allowHTTP2 {
		config.NextProtos = []string{"h2", "http/1.1"}
	} else {
		config.NextProtos = []string{"http/1.1"}
	}

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
	return server.Serve(tlsListener)
}
