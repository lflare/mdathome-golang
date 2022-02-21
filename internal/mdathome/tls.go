package mdathome

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/spacemonkeygo/tlshowdy"
	"github.com/spf13/viper"
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
	if err = tc.SetKeepAlive(true); err != nil {
		log.Warn(fmt.Sprintf("failed to SetKeepAlive(): %s", err))
		return
	}
	if err = tc.SetKeepAlivePeriod(1 * time.Minute); err != nil {
		log.Warn(fmt.Sprintf("failed to SetKeepAlivePeriod(): %s", err))
		return
	}

	// Check SNI if configured to do so
	if viper.GetBool("security.reject_invalid_sni") {
		// Set deadline to prevent connection leaks
		if err = tc.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Warn(fmt.Sprintf("failed to SetDeadline(): %s", err))
			return
		}

		// Peek into the ClientHello message
		clientHello, conn, e := tlshowdy.Peek(tc)

		// Clear deadline
		if err = tc.SetDeadline(time.Time{}); err != nil {
			log.Warn(fmt.Sprintf("failed to clear SetDeadline(): %s", err))
			return
		}

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

func listenAndServeTLSKeyPair(handler http.Handler) error {
	// Build address
	addr := ":" + strconv.Itoa(viper.GetInt("client.port"))

	// Build HTTP server configuration
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  time.Second * time.Duration(viper.GetDuration("performance.client_timeout_seconds")),
		WriteTimeout: time.Second * time.Duration(viper.GetDuration("performance.client_timeout_seconds")),
	}
	config := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}

	// Prepare certificates
	config.GetCertificate = certHandler.GetCertificate()

	// If allowing http2
	if viper.GetBool("performance.allow_http2") {
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
