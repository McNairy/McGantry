package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// getEntity mirrors the server-side Entity model for JSON deserialization.
type getEntity struct {
	Kind       string         `json:"kind"`
	APIVersion string         `json:"apiVersion"`
	Metadata   getEntityMeta  `json:"metadata"`
	Spec       map[string]any `json:"spec,omitempty"`
}

type getEntityMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty"`
	UpdatedAt   time.Time         `json:"updatedAt,omitempty"`
	CreatedBy   string            `json:"createdBy,omitempty"`
}

// getEntityList mirrors the server-side EntityList response.
type getEntityList struct {
	Items      []getEntity `json:"items"`
	TotalCount int         `json:"totalCount"`
}

func getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [KIND] [NAME]",
		Short: "List or get entities from the catalog",
		Long: `List or get entities from the Gantry catalog.

With no arguments, shows a summary of all entity kinds with counts.
With one argument (kind), lists all entities of that kind in a table.
With two arguments (kind, name), shows the full entity definition.

Examples:
  gantry get                        # summary of all kinds
  gantry get Service                # list all services
  gantry get Service payments-api   # get a specific service
  gantry get Service -o yaml        # list services as YAML
  gantry get Service payments-api -o json`,
		Args: cobra.MaximumNArgs(2),
		RunE: runGet,
	}

	cmd.Flags().StringP("output", "o", "table", "Output format: table, yaml, json")
	cmd.Flags().String("server", "http://localhost:8080", "Gantry server URL")
	cmd.Flags().StringP("namespace", "n", "", "Filter by namespace")
	cmd.Flags().String("token", "", "Authentication token (or set GANTRY_TOKEN)")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	server, _ := cmd.Flags().GetString("server")
	server = strings.TrimRight(server, "/")
	output, _ := cmd.Flags().GetString("output")
	namespace, _ := cmd.Flags().GetString("namespace")
	token := getToken(cmd)

	client := &http.Client{Timeout: 30 * time.Second}

	switch len(args) {
	case 0:
		return getSummary(client, server, namespace, token, output)
	case 1:
		return getList(client, server, args[0], namespace, token, output)
	case 2:
		return getOne(client, server, args[0], args[1], namespace, token, output)
	default:
		return fmt.Errorf("too many arguments")
	}
}

// getToken reads the auth token from --token flag or GANTRY_TOKEN env var.
func getToken(cmd *cobra.Command) string {
	token, _ := cmd.Flags().GetString("token")
	if token != "" {
		return token
	}
	return os.Getenv("GANTRY_TOKEN")
}

// doRequest creates and executes an authenticated HTTP request.
func doRequest(client *http.Client, method, url, token string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to server: %w", err)
	}
	return resp, nil
}

// readBody reads the full response body and closes it.
func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	return body, nil
}

// getSummary fetches all entities and prints a summary table grouped by kind.
func getSummary(client *http.Client, server, namespace, token, output string) error {
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

	if output == "json" {
		return printJSON(entities)
	}
	if output == "yaml" {
		return printYAML(entities)
	}

	// Group by kind and show counts.
	kindCounts := make(map[string]int)
	for _, e := range entities {
		kindCounts[e.Kind]++
	}

	if len(kindCounts) == 0 {
		fmt.Println("No entities found.")
		return nil
	}

	// Sort kinds alphabetically for stable output.
	kinds := make([]string, 0, len(kindCounts))
	for k := range kindCounts {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "KIND\tCOUNT")
	for _, k := range kinds {
		fmt.Fprintf(w, "%s\t%d\n", k, kindCounts[k])
	}
	return w.Flush()
}

// getList fetches and displays entities of a specific kind.
func getList(client *http.Client, server, kind, namespace, token, output string) error {
	url := server + "/api/v1/entities/" + kind
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

	if output == "json" {
		return printJSON(entities)
	}
	if output == "yaml" {
		return printYAML(entities)
	}

	if len(entities) == 0 {
		fmt.Printf("No %s entities found.\n", kind)
		return nil
	}

	printEntityTable(entities)
	return nil
}

// getOne fetches and displays a single entity.
func getOne(client *http.Client, server, kind, name, namespace, token, output string) error {
	ns := namespace
	if ns == "" {
		ns = "default"
	}

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

	if output == "json" {
		return printJSON(ent)
	}
	// For table or yaml output on a single entity, show YAML for readability.
	return printYAML(ent)
}

// parseEntityResponse tries to parse the response as an EntityList first,
// then falls back to a bare array of entities.
func parseEntityResponse(body []byte) ([]getEntity, error) {
	// Try EntityList format: {"items": [...], "totalCount": N}
	var list getEntityList
	if err := json.Unmarshal(body, &list); err == nil && list.Items != nil {
		return list.Items, nil
	}

	// Fall back to a bare JSON array.
	var entities []getEntity
	if err := json.Unmarshal(body, &entities); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return entities, nil
}

// printEntityTable renders entities in a kubectl-style table.
func printEntityTable(entities []getEntity) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tNAMESPACE\tOWNER\tTAGS\tAGE")
	for _, e := range entities {
		ns := e.Metadata.Namespace
		if ns == "" {
			ns = "default"
		}
		tags := strings.Join(e.Metadata.Tags, ", ")
		age := formatAge(e.Metadata.CreatedAt)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.Metadata.Name,
			ns,
			e.Metadata.Owner,
			tags,
			age,
		)
	}
	w.Flush()
}

// formatAge converts a timestamp to a human-readable relative age string.
func formatAge(t time.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}

	d := time.Since(t)
	if d < 0 {
		return "0s"
	}

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/(24*365)))
	}
}

// printJSON renders v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

// printYAML renders v as YAML to stdout.
// It round-trips through JSON to respect json struct tags for field naming,
// then marshals to YAML for clean human-readable output.
func printYAML(v any) error {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encoding: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return fmt.Errorf("encoding: %w", err)
	}

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	defer enc.Close()
	if err := enc.Encode(generic); err != nil {
		return fmt.Errorf("encoding YAML: %w", err)
	}
	return nil
}
