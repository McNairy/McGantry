package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// actionRun mirrors the server-side ActionRun model for JSON deserialization.
type actionRun struct {
	ID          string         `json:"id"`
	ActionName  string         `json:"actionName"`
	Status      string         `json:"status"`
	Inputs      map[string]any `json:"inputs,omitempty"`
	Outputs     map[string]any `json:"outputs,omitempty"`
	TriggeredBy string         `json:"triggeredBy"`
	StartedAt   string         `json:"startedAt,omitempty"`
	CompletedAt string         `json:"completedAt,omitempty"`
	Error       string         `json:"error,omitempty"`
}

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <action>",
		Short: "Execute an action",
		Long: `Execute a Gantry action by name.

Pass inputs as key=value pairs with --input (repeatable) or as a JSON object
with --inputs. If both are provided, --inputs is used as a base and --input
flags overlay it.

Examples:
  gantry run deploy-service
  gantry run deploy-service --input service=payments --input env=production
  gantry run deploy-service --inputs '{"service":"payments","env":"production"}'
  gantry run deploy-service --wait --timeout 120`,
		Args: cobra.ExactArgs(1),
		RunE: runAction,
	}

	cmd.Flags().String("server", "http://localhost:8080", "Gantry server URL")
	cmd.Flags().String("token", "", "Authentication token (or set GANTRY_TOKEN)")
	cmd.Flags().StringArray("input", nil, "Input as key=value (repeatable)")
	cmd.Flags().String("inputs", "", "Inputs as a JSON object")
	cmd.Flags().Bool("wait", false, "Wait for the action run to complete")
	cmd.Flags().Int("timeout", 60, "Timeout in seconds when --wait is set")

	return cmd
}

func runAction(cmd *cobra.Command, args []string) error {
	name := args[0]
	server, _ := cmd.Flags().GetString("server")
	server = strings.TrimRight(server, "/")
	token := getToken(cmd)
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Build inputs map.
	inputs, err := buildInputs(cmd)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// POST to execute the action.
	run, err := executeAction(client, server, token, name, inputs)
	if err != nil {
		return err
	}

	fmt.Printf("Action run started: %s (id: %s)\n", run.ActionName, run.ID)
	fmt.Printf("Status: %s\n", run.Status)

	if !wait {
		return nil
	}

	// Poll until done or timeout.
	fmt.Printf("Waiting for completion (timeout: %ds)...\n", timeout)
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		run, err = getActionRun(client, server, token, name, run.ID)
		if err != nil {
			return fmt.Errorf("polling run status: %w", err)
		}

		if run.Status != "pending" && run.Status != "running" {
			break
		}
	}

	// Print final status.
	fmt.Printf("\nFinal status: %s\n", run.Status)
	if run.Error != "" {
		fmt.Printf("Error: %s\n", run.Error)
	}
	if len(run.Outputs) > 0 {
		fmt.Println("Outputs:")
		for k, v := range run.Outputs {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
	if run.CompletedAt != "" {
		fmt.Printf("Completed at: %s\n", run.CompletedAt)
	}

	if run.Status == "failed" {
		return fmt.Errorf("action run failed")
	}
	return nil
}

// buildInputs constructs the inputs map from --inputs JSON and --input key=value flags.
func buildInputs(cmd *cobra.Command) (map[string]any, error) {
	inputs := map[string]any{}

	// Parse --inputs JSON base.
	inputsJSON, _ := cmd.Flags().GetString("inputs")
	if inputsJSON != "" {
		if err := json.Unmarshal([]byte(inputsJSON), &inputs); err != nil {
			return nil, fmt.Errorf("invalid --inputs JSON: %w", err)
		}
	}

	// Overlay --input key=value pairs.
	kvs, _ := cmd.Flags().GetStringArray("input")
	for _, kv := range kvs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --input %q: expected key=value", kv)
		}
		inputs[parts[0]] = parts[1]
	}

	return inputs, nil
}

// executeAction POSTs to /api/v1/actions/{name}/execute and returns the run.
func executeAction(client *http.Client, server, token, name string, inputs map[string]any) (*actionRun, error) {
	payload := map[string]any{"inputs": inputs}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling inputs: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/actions/%s/execute", server, name)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to server: %w", err)
	}

	respBody, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("action %q not found", name)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var run actionRun
	if err := json.Unmarshal(respBody, &run); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &run, nil
}

// getActionRun fetches a run's current status.
func getActionRun(client *http.Client, server, token, actionName, runID string) (*actionRun, error) {
	url := fmt.Sprintf("%s/api/v1/actions/%s/runs/%s", server, actionName, runID)
	resp, err := doRequest(client, http.MethodGet, url, token)
	if err != nil {
		return nil, err
	}

	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var run actionRun
	if err := json.Unmarshal(body, &run); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &run, nil
}
