package main

import (
	"fmt"
	"strings"
	"time"

	"net/http"

	"github.com/spf13/cobra"
)

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export all entities from the catalog",
		Long: `Export all entities from the Gantry catalog to stdout.

Outputs multi-document YAML by default (entities separated by ---).
Use --format json for a JSON array.

Examples:
  gantry export                          # all entities as YAML
  gantry export --format json            # all entities as JSON
  gantry export --namespace production   # filter by namespace
  gantry export > catalog.yaml           # save to file`,
		RunE: runExport,
	}

	cmd.Flags().String("format", "yaml", "Output format: yaml, json")
	cmd.Flags().String("server", "http://localhost:8080", "Gantry server URL")
	cmd.Flags().String("token", "", "Authentication token (or set GANTRY_TOKEN)")
	cmd.Flags().StringP("namespace", "n", "", "Filter by namespace")

	return cmd
}

func runExport(cmd *cobra.Command, _ []string) error {
	server, _ := cmd.Flags().GetString("server")
	server = strings.TrimRight(server, "/")
	format, _ := cmd.Flags().GetString("format")
	namespace, _ := cmd.Flags().GetString("namespace")
	token := getToken(cmd)

	client := &http.Client{Timeout: 30 * time.Second}

	url := server + "/api/v1/entities"
	if namespace != "" {
		url += "?namespace=" + namespace
	}

	resp, err := doRequest(client, http.MethodGet, url, token)
	if err != nil {
		return err
	}

	body, err := readBody(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	entities, err := parseEntityResponse(body)
	if err != nil {
		return err
	}

	if len(entities) == 0 {
		fmt.Println("# No entities found.")
		return nil
	}

	switch format {
	case "json":
		return printJSON(entities)
	default:
		// YAML: print each entity as a separate document separated by ---
		for i, e := range entities {
			if i > 0 {
				fmt.Println("---")
			}
			if err := printYAML(e); err != nil {
				return err
			}
		}
		return nil
	}
}
