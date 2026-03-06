package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply entity configuration from a YAML file",
		Long: `Apply entity definitions from a YAML or JSON file to the Gantry server.
Supports multi-document YAML (separated by ---). If an entity already exists,
it will be updated; otherwise it will be created.

Example:
  gantry apply -f services.yaml
  gantry apply -f catalog/ -f teams.yaml`,
		RunE: runApply,
	}

	cmd.Flags().StringSliceP("file", "f", nil, "Path to YAML/JSON file(s)")
	cmd.Flags().String("server", "http://localhost:8080", "Gantry server URL")
	cmd.MarkFlagRequired("file")

	return cmd
}

// applyEntity is a minimal struct for parsing just enough to identify the entity.
type applyEntity struct {
	Kind       string            `yaml:"kind" json:"kind"`
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Metadata   applyEntityMeta   `yaml:"metadata" json:"metadata"`
	Spec       map[string]any    `yaml:"spec" json:"spec"`
}

type applyEntityMeta struct {
	Name        string            `yaml:"name" json:"name"`
	Namespace   string            `yaml:"namespace" json:"namespace"`
	Title       string            `yaml:"title" json:"title"`
	Description string            `yaml:"description" json:"description"`
	Owner       string            `yaml:"owner" json:"owner"`
	Tags        []string          `yaml:"tags" json:"tags"`
	Annotations map[string]string `yaml:"annotations" json:"annotations"`
	Labels      map[string]string `yaml:"labels" json:"labels"`
}

func runApply(cmd *cobra.Command, args []string) error {
	files, _ := cmd.Flags().GetStringSlice("file")
	server, _ := cmd.Flags().GetString("server")
	server = strings.TrimRight(server, "/")

	client := &http.Client{Timeout: 30 * time.Second}

	var totalCreated, totalUpdated, totalFailed int

	for _, file := range files {
		entities, err := parseEntityFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file, err)
			totalFailed++
			continue
		}

		for _, ent := range entities {
			if err := validateApplyEntity(&ent); err != nil {
				fmt.Fprintf(os.Stderr, "  INVALID  %s/%s: %v\n", ent.Kind, ent.Metadata.Name, err)
				totalFailed++
				continue
			}

			action, err := applyOne(client, server, &ent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  FAILED   %s/%s: %v\n", ent.Kind, ent.Metadata.Name, err)
				totalFailed++
				continue
			}

			switch action {
			case "created":
				fmt.Printf("  CREATED  %s/%s\n", ent.Kind, ent.Metadata.Name)
				totalCreated++
			case "updated":
				fmt.Printf("  UPDATED  %s/%s\n", ent.Kind, ent.Metadata.Name)
				totalUpdated++
			}
		}
	}

	fmt.Println()
	fmt.Printf("Results: %d created, %d updated, %d failed\n", totalCreated, totalUpdated, totalFailed)

	if totalFailed > 0 {
		return fmt.Errorf("%d entities failed to apply", totalFailed)
	}
	return nil
}

func parseEntityFile(path string) ([]applyEntity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		return parseJSON(data)
	case ".yaml", ".yml":
		return parseYAML(data)
	default:
		// Try YAML first, fall back to JSON.
		entities, err := parseYAML(data)
		if err != nil {
			return parseJSON(data)
		}
		return entities, nil
	}
}

func parseJSON(data []byte) ([]applyEntity, error) {
	// Try as array first.
	var entities []applyEntity
	if err := json.Unmarshal(data, &entities); err == nil {
		return entities, nil
	}

	// Try as single object.
	var ent applyEntity
	if err := json.Unmarshal(data, &ent); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return []applyEntity{ent}, nil
}

func parseYAML(data []byte) ([]applyEntity, error) {
	var entities []applyEntity

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var ent applyEntity
		err := decoder.Decode(&ent)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
		// Skip empty documents (e.g., trailing ---).
		if ent.Kind == "" && ent.Metadata.Name == "" {
			continue
		}
		entities = append(entities, ent)
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities found in file")
	}

	return entities, nil
}

func validateApplyEntity(ent *applyEntity) error {
	var errs []string
	if strings.TrimSpace(ent.Kind) == "" {
		errs = append(errs, "kind is required")
	}
	if strings.TrimSpace(ent.Metadata.Name) == "" {
		errs = append(errs, "metadata.name is required")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func applyOne(client *http.Client, server string, ent *applyEntity) (string, error) {
	body, err := json.Marshal(ent)
	if err != nil {
		return "", fmt.Errorf("marshaling entity: %w", err)
	}

	// First try to create (POST).
	url := fmt.Sprintf("%s/api/v1/entities", server)
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("connecting to server: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		return "created", nil
	}

	// If 409 Conflict, the entity already exists. Try to update (PUT).
	if resp.StatusCode == http.StatusConflict {
		ns := ent.Metadata.Namespace
		if ns == "" {
			ns = "default"
		}
		url = fmt.Sprintf("%s/api/v1/entities/%s/%s/%s", server, ent.Kind, ns, ent.Metadata.Name)
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("creating update request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			return "", fmt.Errorf("connecting to server: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return "updated", nil
		}
		return "", fmt.Errorf("update failed with status %d", resp.StatusCode)
	}

	// Read error body for details.
	return "", fmt.Errorf("server returned status %d", resp.StatusCode)
}
