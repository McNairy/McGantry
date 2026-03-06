package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func describeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe KIND NAME",
		Short: "Describe an entity in detail",
		Long: `Show detailed information about a specific entity in a human-readable format.

Requires exactly two arguments: the entity kind and name.

Examples:
  gantry describe Service payments-api
  gantry describe API payments-rest-api
  gantry describe Team platform-engineering -n production`,
		Args: cobra.ExactArgs(2),
		RunE: runDescribe,
	}

	cmd.Flags().String("server", "http://localhost:8080", "Gantry server URL")
	cmd.Flags().StringP("namespace", "n", "", "Entity namespace (default: \"default\")")
	cmd.Flags().String("token", "", "Authentication token (or set GANTRY_TOKEN)")

	return cmd
}

func runDescribe(cmd *cobra.Command, args []string) error {
	kind := args[0]
	name := args[1]

	server, _ := cmd.Flags().GetString("server")
	server = strings.TrimRight(server, "/")
	namespace, _ := cmd.Flags().GetString("namespace")
	token := getToken(cmd)

	ns := namespace
	if ns == "" {
		ns = "default"
	}

	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf("%s/api/v1/entities/%s/%s/%s", server, kind, ns, name)

	resp, err := doRequest(client, http.MethodGet, url, token)
	if err != nil {
		return err
	}

	body, err := readBody(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%s/%s not found in namespace %q", kind, name, ns)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var ent getEntity
	if err := json.Unmarshal(body, &ent); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	printDescribe(&ent)
	return nil
}

// printDescribe renders an entity in a kubectl-describe style format.
func printDescribe(ent *getEntity) {
	ns := ent.Metadata.Namespace
	if ns == "" {
		ns = "default"
	}

	printField("Name", ent.Metadata.Name)
	printField("Kind", ent.Kind)
	printField("Namespace", ns)

	if ent.Metadata.Title != "" {
		printField("Title", ent.Metadata.Title)
	}
	if ent.Metadata.Description != "" {
		printField("Description", ent.Metadata.Description)
	}
	if ent.Metadata.Owner != "" {
		printField("Owner", ent.Metadata.Owner)
	}
	if len(ent.Metadata.Tags) > 0 {
		printField("Tags", "["+strings.Join(ent.Metadata.Tags, ", ")+"]")
	}
	if !ent.Metadata.CreatedAt.IsZero() {
		printField("Created", ent.Metadata.CreatedAt.Format(time.RFC3339))
	}
	if !ent.Metadata.UpdatedAt.IsZero() {
		printField("Updated", ent.Metadata.UpdatedAt.Format(time.RFC3339))
	}
	if ent.Metadata.CreatedBy != "" {
		printField("Created By", ent.Metadata.CreatedBy)
	}

	// Print labels if present.
	if len(ent.Metadata.Labels) > 0 {
		fmt.Println()
		fmt.Println("Labels:")
		for k, v := range ent.Metadata.Labels {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	// Print annotations if present.
	if len(ent.Metadata.Annotations) > 0 {
		fmt.Println()
		fmt.Println("Annotations:")
		for k, v := range ent.Metadata.Annotations {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	// Print spec fields.
	if len(ent.Spec) > 0 {
		fmt.Println()
		fmt.Println("Spec:")
		printSpecFields(ent.Spec, "  ")
	}
}

// printField prints a single "Key:  Value" line with consistent alignment.
func printField(label, value string) {
	fmt.Printf("%-14s%s\n", label+":", value)
}

// printSpecFields recursively prints spec fields with proper indentation.
// It handles nested maps, slices, and scalar values for a clean describe output.
func printSpecFields(m map[string]any, indent string) {
	for key, val := range m {
		switch v := val.(type) {
		case map[string]any:
			fmt.Printf("%s%s:\n", indent, formatKey(key))
			printSpecFields(v, indent+"  ")
		case []any:
			if len(v) == 0 {
				fmt.Printf("%s%s: []\n", indent, formatKey(key))
				continue
			}
			// Check if all items are simple scalars.
			allScalar := true
			for _, item := range v {
				if _, ok := item.(map[string]any); ok {
					allScalar = false
					break
				}
			}
			if allScalar {
				fmt.Printf("%s%s:\n", indent, formatKey(key))
				for _, item := range v {
					fmt.Printf("%s  - %v\n", indent, item)
				}
			} else {
				fmt.Printf("%s%s:\n", indent, formatKey(key))
				for _, item := range v {
					if nested, ok := item.(map[string]any); ok {
						first := true
						for nk, nv := range nested {
							if first {
								fmt.Printf("%s  - %s: %v\n", indent, formatKey(nk), formatScalar(nv))
								first = false
							} else {
								fmt.Printf("%s    %s: %v\n", indent, formatKey(nk), formatScalar(nv))
							}
						}
					} else {
						fmt.Printf("%s  - %v\n", indent, item)
					}
				}
			}
		default:
			fmt.Printf("%s%s: %v\n", indent, formatKey(key), formatScalar(val))
		}
	}
}

// formatKey converts snake_case keys to Title Case for display.
func formatKey(key string) string {
	key = strings.ReplaceAll(key, "_", " ")
	words := strings.Fields(key)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// formatScalar formats a scalar value for display.
func formatScalar(v any) string {
	if v == nil {
		return "<none>"
	}
	return fmt.Sprintf("%v", v)
}
