// Package kubernetes implements Kubernetes cluster discovery for the Gantry
// plugin system. It uses the Kubernetes REST API directly via net/http so
// there is no dependency on client-go.
package kubernetes

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// GetWorkload fetches deployment and pod info for an app label across the given namespaces.
func (c *Client) GetWorkload(appName string, namespaces []string) (*WorkloadInfo, error) {
	info := &WorkloadInfo{AppName: appName, Deployments: []DeploymentInfo{}, Pods: []PodInfo{}}
	selector := "app=" + url.QueryEscape(appName)

	for _, ns := range namespaces {
		// Deployments
		var depList DeploymentList
		depPath := fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments?labelSelector=%s", ns, selector)
		if err := c.get(depPath, &depList); err == nil {
			for _, dep := range depList.Items {
				info.Deployments = append(info.Deployments, DeploymentInfo{
					Name:              dep.Metadata.Name,
					Namespace:         dep.Metadata.Namespace,
					DesiredReplicas:   dep.Spec.Replicas,
					ReadyReplicas:     dep.Status.ReadyReplicas,
					AvailableReplicas: dep.Status.ReadyReplicas,
				})
			}
		}

		// Pods
		var podList PodList
		podPath := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=%s", ns, selector)
		if err := c.get(podPath, &podList); err == nil {
			for _, pod := range podList.Items {
				info.Pods = append(info.Pods, podToInfo(pod))
			}
		}
	}

	return info, nil
}

// StreamLogs streams log lines from a pod container to w as plain text (one line per write, flushed).
func (c *Client) StreamLogs(w http.ResponseWriter, namespace, pod, container string, tailLines int) {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log?container=%s&follow=true&tailLines=%d",
		namespace, pod, url.QueryEscape(container), tailLines)

	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	// No timeout for streaming connections.
	streamClient := &http.Client{Transport: c.httpClient.Transport}
	resp, err := streamClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, canFlush := w.(http.Flusher)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fmt.Fprintln(w, scanner.Text())
		if canFlush {
			flusher.Flush()
		}
	}
}

// podToInfo converts a raw Pod API response into a PodInfo summary.
func podToInfo(pod Pod) PodInfo {
	statusMap := make(map[string]ContainerStatus, len(pod.Status.ContainerStatuses))
	for _, cs := range pod.Status.ContainerStatuses {
		statusMap[cs.Name] = cs
	}

	var totalRestarts int32
	readyCount := 0
	containers := make([]ContainerInfo, 0, max(len(pod.Spec.Containers), 1))

	for _, c := range pod.Spec.Containers {
		ci := ContainerInfo{Name: c.Name, Image: c.Image, State: "unknown"}
		if cs, ok := statusMap[c.Name]; ok {
			ci.Ready = cs.Ready
			ci.Restarts = cs.RestartCount
			totalRestarts += cs.RestartCount
			switch {
			case cs.State.Running != nil:
				ci.State = "running"
			case cs.State.Waiting != nil:
				ci.State = "waiting"
				ci.Reason = cs.State.Waiting.Reason
			case cs.State.Terminated != nil:
				ci.State = "terminated"
				ci.Reason = cs.State.Terminated.Reason
			}
			if cs.Ready {
				readyCount++
			}
		}
		containers = append(containers, ci)
	}

	return PodInfo{
		Name:          pod.Metadata.Name,
		Namespace:     pod.Metadata.Namespace,
		Phase:         pod.Status.Phase,
		Ready:         len(pod.Spec.Containers) > 0 && readyCount == len(pod.Spec.Containers),
		TotalRestarts: totalRestarts,
		NodeName:      pod.Spec.NodeName,
		StartTime:     pod.Status.StartTime,
		Containers:    containers,
	}
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
