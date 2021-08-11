package mdathome

import (
	"crypto/tls"
	"sync"
)

type certificateHandler struct {
	certMu sync.RWMutex
	cert   *tls.Certificate
}

func NewCertificateReloader(cert tls.Certificate) *certificateHandler {
	result := &certificateHandler{}
	result.cert = &cert
	return result
}

func (ch *certificateHandler) updateCertificate(cert tls.Certificate) error {
	ch.certMu.Lock()
	defer ch.certMu.Unlock()
	ch.cert = &cert
	return nil
}

func (ch *certificateHandler) GetCertificate() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		ch.certMu.RLock()
		defer ch.certMu.RUnlock()
		return ch.cert, nil
	}
}
