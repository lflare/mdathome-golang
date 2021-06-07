package mdathome

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/spacemonkeygo/tlshowdy"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	// Accept TCP connection
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}

	// Configure connection
	err = tc.SetKeepAlive(true)
	if err != nil {
		return
	}
	err = tc.SetKeepAlivePeriod(1 * time.Minute)
	if err != nil {
		return
	}

	// Check SNI value if configured
	if clientSettings.RejectInvalidSNI {
		// Peek into the ClientHello message
		clientHello, conn, errs := tlshowdy.Peek(tc)
		if clientHello == nil || errs != nil {
			// Close connection and return for fast fail
			err := conn.Close()
			return conn, err
		}

		// Check to allow for both mangadex.network SNI and localhost SNI
		if clientHello.ServerName != clientHostname && clientHello.ServerName != "localhost" {
			// Log
			log.Warn(fmt.Sprintf("blocked unauthorised SNI request: %s", clientHello.ServerName))

			// Close connection and return for fast fail
			err := conn.Close()
			return conn, err
		}

		// Return connection
		return conn, nil
	}

	// Return connection
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

	// Prepare certificates
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = cert

	// If allowing http2
	if clientSettings.AllowHTTP2 {
		config.NextProtos = []string{"h2", "http/1.1"}
	} else {
		config.NextProtos = []string{"http/1.1"}
	}

	// Listen to only IPv4 interfaces
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return err
	}

	// Start TLS listeners
	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
	return server.Serve(tlsListener)
}
