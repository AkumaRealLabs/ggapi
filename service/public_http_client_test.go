package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublicHTTPClientRejectsPrivateTargets(t *testing.T) {
	client := NewPublicHTTPClient(time.Second)

	for _, target := range []string{
		"http://127.0.0.1/releases/latest",
		"http://localhost/releases/latest",
		"https://[::1]/releases/latest",
		"http://169.254.169.254/latest/meta-data",
	} {
		t.Run(target, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, target, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)

			require.Error(t, err)
			require.Nil(t, resp)
			require.Contains(t, err.Error(), "private IP address not allowed")
		})
	}
}
