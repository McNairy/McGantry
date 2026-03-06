// Package kubernetes implements Kubernetes cluster discovery for the Gantry
// plugin system. It uses the Kubernetes REST API directly via net/http so
// there is no dependency on client-go.
package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a minimal Kubernetes REST API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a client from plugin config values.
//
//   config["clusterUrl"] — required, e.g. "https://192.168.1.1:6443"
//   config["token"]      — required, service account bearer token
//   config["caData"]     — optional, base64-encoded PEM cluster CA certificate
func NewClient(config map[string]any) (*Client, error) {
	clusterURL, _ := config["clusterUrl"].(string)
	if clusterURL == "" {
		return nil, fmt.Errorf("kubernetes plugin: clusterUrl is required")
	}
	token, _ := config["token"].(string)
	if token == "" {
		return nil, fmt.Errorf("kubernetes plugin: token is required")
	}

	tlsCfg := &tls.Config{InsecureSkipVerify: false} //nolint:gosec

	if caDataB64, _ := config["caData"].(string); caDataB64 != "" {
		caDER, err := base64.StdEncoding.DecodeString(caDataB64)
		if err != nil {
			return nil, fmt.Errorf("kubernetes plugin: decode caData: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caDER) {
			return nil, fmt.Errorf("kubernetes plugin: failed to parse CA certificate")
		}
		tlsCfg.RootCAs = pool
	} else {
		// Skip TLS verification when no CA is provided; common in dev clusters.
		tlsCfg.InsecureSkipVerify = true //nolint:gosec
	}

	return &Client{
		baseURL: clusterURL,
		token:   token,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: &http.Transport{TLSClientConfig: tlsCfg},
		},
	}, nil
}

// get performs a GET request to the Kubernetes API and decodes the JSON response.
func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("k8s request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("k8s %s: HTTP %d: %s", path, resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}
