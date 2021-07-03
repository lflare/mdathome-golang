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
		log.Warn(fmt.Sprintf("failed to AcceptTCP(): %s", err))
		return
	}

	// Configure connection
	err = tc.SetKeepAlive(true)
	if err != nil {
		log.Warn(fmt.Sprintf("failed to SetKeepAlive(): %s", err))
		return
	}
	err = tc.SetKeepAlivePeriod(1 * time.Minute)
	if err != nil {
		log.Warn(fmt.Sprintf("failed to SetKeepAlivePeriod(): %s", err))
		return
	}

	// Check SNI if configured to do so
	if clientSettings.RejectInvalidSNI {
		// Peek into the ClientHello message
		clientHello, conn, e := tlshowdy.Peek(tc)

		// Check ClientHello SNI for both mangadex.network or localhost domain
		if clientHello != nil && (clientHello.ServerName == clientHostname || clientHello.ServerName == "localhost") {
			return conn, nil
		}

		// If no ClientHello, or if error is present
		if e != nil {
			log.Warn(fmt.Sprintf("failed to peek into TLS body: %s", e))
		} else if clientHello == nil {
			log.Warn(fmt.Sprintf("failed to extract ClientHello: %s", e))
		}

		// Close connection and return for fast fail
		if conn != nil {
			conn.Close()
			return conn, nil
		} else {
			return tc, nil
		}

	}

	// Return default connection
	return tc, nil
}

func listenAndServeTLSKeyPair(addr string, allowHTTP2 bool, cert tls.Certificate, handler http.Handler) error {
	if addr == "" {
		return errors.New("invalid address string")
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  1 * time.Minute,
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
