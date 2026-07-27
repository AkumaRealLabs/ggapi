package service

import (
	"net"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
)

type publicHTTPRoundTripper struct {
	protection *common.SSRFProtection
	transport  *http.Transport
}

func (t *publicHTTPRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.protection.ValidateURL(req.URL.String()); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(req)
}

func (t *publicHTTPRoundTripper) CloseIdleConnections() {
	t.transport.CloseIdleConnections()
}

// NewPublicHTTPClient creates an outbound client that always rejects private,
// loopback, link-local, and reserved destinations at URL validation and dial time.
// It intentionally ignores the configurable fetch allowlist and outbound proxy.
func NewPublicHTTPClient(timeout time.Duration) *http.Client {
	protection := &common.SSRFProtection{
		AllowPrivateIp:         false,
		DomainFilterMode:       false,
		IpFilterMode:           false,
		AllowedPorts:           []int{80, 443},
		ApplyIPFilterForDomain: true,
	}
	netDialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	protectedDialer := &protectedFetchDialer{
		resolver:    net.DefaultResolver,
		dialContext: netDialer.DialContext,
		getProtection: func() (*common.SSRFProtection, bool, error) {
			return protection, true, nil
		},
	}
	transport := &http.Transport{
		DialContext:           protectedDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          common.RelayMaxIdleConns,
		MaxIdleConnsPerHost:   common.RelayMaxIdleConnsPerHost,
		IdleConnTimeout:       time.Duration(common.RelayIdleConnTimeout) * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Transport: &publicHTTPRoundTripper{
			protection: protection,
			transport:  transport,
		},
		Timeout: timeout,
	}
}
