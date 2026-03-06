// Package dispatcher executes self-service actions based on their configured type.
// Each action entity spec declares a "type" field (e.g. "webhook") and
// type-specific fields (e.g. "url"). The Manager picks the right dispatcher
// and runs it, updating the ActionRun's status and publishing events.
package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gantrydev/gantry/internal/db"
	"github.com/gantrydev/gantry/internal/entity"
	"github.com/gantrydev/gantry/internal/events"
)

// Manager holds shared HTTP client and event bus, and dispatches action runs
// to the appropriate backend based on the action entity's spec.type.
type Manager struct {
	DB     *db.DB
	Events *events.Bus
	client *http.Client
}

// New creates a new Manager.
func New(database *db.DB, eventBus *events.Bus) *Manager {
	return &Manager{
		DB:     database,
		Events: eventBus,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Dispatch executes a pending action run asynchronously.
// It updates run status (running → success/failed) and publishes events.
// Call from a goroutine; the caller should have already persisted the run as "pending".
func (m *Manager) Dispatch(action *entity.Entity, run *db.ActionRun) {
	ctx := context.Background()

	// Transition to running.
	run.Status = "running"
	if err := m.DB.UpdateActionRun(ctx, run); err == nil {
		m.publishRunEvent(run)
	}

	var execErr error
	var outputJSON string

	actionType, _ := action.Spec["type"].(string)
	switch actionType {
	case "webhook":
		outputJSON, execErr = m.runWebhook(action, run)
	default:
		// Unrecognized type — complete immediately with a note.
		outputJSON = fmt.Sprintf(`{"message":"action type %q has no executor; marked as succeeded","type":%q}`,
			actionType, actionType)
	}

	now := time.Now().UTC()
	run.CompletedAt = &now
	run.Outputs = outputJSON
	if execErr != nil {
		run.Status = "failed"
		run.Error = execErr.Error()
	} else {
		run.Status = "success"
	}

	_ = m.DB.UpdateActionRun(ctx, run)
	m.publishRunEvent(run)
}

// publishRunEvent emits an action.run.updated event over the event bus.
func (m *Manager) publishRunEvent(run *db.ActionRun) {
	m.Events.Publish(events.Event{
		Type: events.ActionRunUpdated,
		Data: map[string]any{
			"runId":      run.ID,
			"actionName": run.ActionName,
			"status":     run.Status,
		},
	})
}

// runWebhook POSTs the action inputs to spec.url and captures the response body.
func (m *Manager) runWebhook(action *entity.Entity, run *db.ActionRun) (string, error) {
	rawURL, _ := action.Spec["url"].(string)
	if rawURL == "" {
		return "", fmt.Errorf("webhook action %q has no url in spec", action.Metadata.Name)
	}

	method := "POST"
	if m, ok := action.Spec["method"].(string); ok && m != "" {
		method = m
	}

	// Parse stored inputs.
	var inputs map[string]any
	if run.Inputs != "" {
		_ = json.Unmarshal([]byte(run.Inputs), &inputs)
	}

	payload, _ := json.Marshal(map[string]any{"inputs": inputs})
	req, err := http.NewRequest(method, rawURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("building webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Gantry/1.0")

	// Apply custom headers from spec.headers (map[string]string).
	if headers, ok := action.Spec["headers"].(map[string]any); ok {
		for k, v := range headers {
			if vs, ok := v.(string); ok {
				req.Header.Set(k, vs)
			}
		}
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16)) // 64 KiB max
	outputJSON, _ := json.Marshal(map[string]any{
		"statusCode": resp.StatusCode,
		"body":       string(body),
	})

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(outputJSON), fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	return string(outputJSON), nil
}
