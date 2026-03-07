package argocd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go2engle/gantry/internal/entity"
)

// EntityStore is the subset of db.DB used by the sync operation.
type EntityStore interface {
	GetEntity(ctx context.Context, kind, namespace, name string) (*entity.Entity, error)
	CreateEntity(ctx context.Context, e *entity.Entity) error
	UpdateEntity(ctx context.Context, e *entity.Entity) error
}

// SyncResult summarises what happened during an ArgoCD sync run.
type SyncResult struct {
	Apps    int      `json:"apps"`
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Errors  []string `json:"errors,omitempty"`
}

// Sync discovers ArgoCD Applications across all configured instances and upserts
// them as Gantry Service entities.
//
// App grouping:
//   - Apps are grouped by the value of the label key specified in config["labelKey"]
//     (default: "name"). For example, apps "dxc-portal-api-dev", "dxc-portal-api-qa",
//     and "dxc-portal-api-uat" that all carry label name=dxc-portal-api will be
//     merged into a single Service entity named "dxc-portal-api".
//   - The annotation argocd.io/appNames accumulates all instance:appName pairs for
//     the entity so the UI can fetch live status for each one individually.
//
// Kubernetes correlation:
//   - When the k8s plugin is also active, ArgoCD annotations are merged onto the
//     existing k8s-created entity (or vice versa) via the same upsert mechanism.
func Sync(ctx context.Context, config map[string]any, store EntityStore) (*SyncResult, error) {
	instances := AllInstanceConfigs(config)
	if len(instances) == 0 {
		return nil, fmt.Errorf("argocd plugin: no instances configured")
	}

	labelKey, _ := config["labelKey"].(string)
	if labelKey == "" {
		labelKey = "name"
	}

	res := &SyncResult{}
	for _, instCfg := range instances {
		instanceName, _ := instCfg["instanceName"].(string)
		argoURL, _ := instCfg["argocdUrl"].(string)
		argoURL = strings.TrimRight(argoURL, "/")
		project, _ := instCfg["project"].(string)

		client, err := NewClient(instCfg)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("instance %s: %v", instanceName, err))
			continue
		}

		apps, err := client.ListApplications(project)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("instance %s: list applications: %v", instanceName, err))
			continue
		}

		for _, app := range apps {
			e := appToService(app, argoURL, instanceName, labelKey)
			if err := upsert(ctx, store, e, res); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("instance %s, app %s: %v", instanceName, app.Metadata.Name, err))
			}
			res.Apps++
		}
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Converters
// ---------------------------------------------------------------------------

// appToService converts an ArgoCD Application into a Gantry Service entity.
// The entity name is derived from the app label specified by labelKey (default
// "name"), falling back to the ArgoCD app name. This enables grouping multiple
// environment-specific apps (dev/qa/prod) under one Service entity.
func appToService(app Application, argoURL, instanceName, labelKey string) *entity.Entity {
	// Entity name: use the label value for grouping, fall back to the app name.
	entityName := app.Metadata.Name
	if labelKey != "" {
		if lv, ok := app.Metadata.Labels[labelKey]; ok && lv != "" {
			entityName = lv
		}
	}

	annotations := buildAnnotations(app, argoURL, instanceName)
	tags := labelsToTags(app.Metadata.Labels)
	tags = append(tags, "argocd")

	desc := fmt.Sprintf("ArgoCD Application %q", app.Metadata.Name)
	if app.Spec.Source.RepoURL != "" {
		desc += fmt.Sprintf(" — %s", app.Spec.Source.RepoURL)
	}
	desc += "."

	spec := map[string]any{
		"type": "backend",
	}

	// Populate spec.repoUrl if the source is a Git repo (not a Helm chart-only
	// reference) so the GitHub tab can pick it up automatically.
	if app.Spec.Source.RepoURL != "" && app.Spec.Source.Chart == "" {
		spec["repoUrl"] = app.Spec.Source.RepoURL
	}

	// Add the ArgoCD destination namespace to spec.deployedIn so it matches
	// what the k8s plugin writes — both plugins add Environment references to
	// the same list, which is deduplicated by mergeDeployedIn.
	if app.Spec.Destination.Namespace != "" {
		spec["deployedIn"] = []any{
			map[string]any{"kind": "Environment", "name": app.Spec.Destination.Namespace},
		}
	}

	return &entity.Entity{
		Kind:       "Service",
		APIVersion: "gantry.io/v1",
		Metadata: entity.EntityMetadata{
			Name:        entityName,
			Namespace:   "default",
			Title:       entityName,
			Description: desc,
			Annotations: annotations,
			Tags:        tags,
		},
		Spec: spec,
	}
}

// buildAnnotations produces ArgoCD-specific annotations for the entity.
// The key argocd.io/appNames stores a single "instance:appName" pair for this
// app; mergeAnnotations accumulates multiple such values as a CSV.
func buildAnnotations(app Application, argoURL, instanceName string) map[string]string {
	appRef := instanceName + ":" + app.Metadata.Name
	a := map[string]string{
		"argocd.io/appNames":     appRef,
		"argocd.io/namespace":    app.Metadata.Namespace,
		"argocd.io/syncStatus":   app.Status.Sync.Status,
		"argocd.io/healthStatus": app.Status.Health.Status,
		"argocd.io/project":      app.Spec.Project,
		"argocd.io/repoURL":      app.Spec.Source.RepoURL,
		"argocd.io/destServer":   app.Spec.Destination.Server,
	}
	if app.Spec.Source.TargetRevision != "" {
		a["argocd.io/targetRevision"] = app.Spec.Source.TargetRevision
	}
	if app.Spec.Destination.Namespace != "" {
		a["argocd.io/destNamespace"] = app.Spec.Destination.Namespace
	}
	if app.Status.Health.Message != "" {
		a["argocd.io/healthMessage"] = app.Status.Health.Message
	}
	if app.Status.Sync.Revision != "" {
		a["argocd.io/syncRevision"] = app.Status.Sync.Revision
	}
	return a
}

// ---------------------------------------------------------------------------
// Upsert helpers — shared with k8s plugin style
// ---------------------------------------------------------------------------

func upsert(ctx context.Context, store EntityStore, e *entity.Entity, res *SyncResult) error {
	existing, err := store.GetEntity(ctx, e.Kind, e.Metadata.Namespace, e.Metadata.Name)
	if err != nil && !errors.Is(err, entity.ErrEntityNotFound) {
		return fmt.Errorf("get entity: %w", err)
	}
	if existing == nil {
		e.Metadata.CreatedBy = "argocd-plugin"
		e.Metadata.CreatedAt = time.Now().UTC()
		if err := store.CreateEntity(ctx, e); err != nil {
			return fmt.Errorf("create entity: %w", err)
		}
		res.Created++
		return nil
	}

	// Merge: preserve user-set fields, update ArgoCD-sourced annotations.
	existing.Metadata.Annotations = mergeAnnotations(existing.Metadata.Annotations, e.Metadata.Annotations)
	existing.Metadata.Tags = mergeTags(existing.Metadata.Tags, e.Metadata.Tags)
	existing.Spec = mergeSpec(existing.Spec, e.Spec)
	if err := store.UpdateEntity(ctx, existing); err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			return store.CreateEntity(ctx, e)
		}
		return fmt.Errorf("update entity: %w", err)
	}
	res.Updated++
	return nil
}

// mergeAnnotations overwrites ArgoCD-sourced keys, leaving others intact.
// argocd.io/appNames is special: values are accumulated as a CSV so that
// multiple apps grouped under one entity each contribute their instance:appName.
func mergeAnnotations(existing, incoming map[string]string) map[string]string {
	if existing == nil {
		return incoming
	}
	for k, v := range incoming {
		if k == "argocd.io/appNames" {
			existing[k] = mergeCSV(existing[k], v)
		} else {
			existing[k] = v
		}
	}
	// Migrate legacy argocd.io/appName key to the new plural format.
	if old, ok := existing["argocd.io/appName"]; ok && old != "" {
		existing["argocd.io/appNames"] = mergeCSV(existing["argocd.io/appNames"], old)
		delete(existing, "argocd.io/appName")
	}
	return existing
}

// mergeCSV merges two comma-separated value strings, deduplicating entries.
func mergeCSV(a, b string) string {
	seen := make(map[string]bool)
	var parts []string
	for _, p := range strings.Split(a+","+b, ",") {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ",")
}

// mergeSpec merges incoming spec into existing, with special handling for
// spec.deployedIn so that environment refs from both plugins accumulate.
func mergeSpec(existing, incoming map[string]any) map[string]any {
	if existing == nil {
		return incoming
	}
	result := make(map[string]any, len(existing))
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range incoming {
		if k == "deployedIn" {
			result[k] = mergeDeployedIn(existing[k], v)
		} else if k == "repoUrl" {
			// Only set repoUrl if not already set — don't overwrite a
			// user-specified value with the ArgoCD git URL.
			if _, already := existing["repoUrl"]; !already {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}
	return result
}

// mergeDeployedIn deduplicates {kind, name} references from two deployedIn lists.
func mergeDeployedIn(existing, incoming any) []any {
	seen := make(map[string]bool)
	var result []any
	for _, list := range []any{existing, incoming} {
		items, ok := list.([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := fmt.Sprintf("%s/%s", m["kind"], m["name"])
			if !seen[key] {
				seen[key] = true
				result = append(result, item)
			}
		}
	}
	return result
}

func mergeTags(existing, incoming []string) []string {
	seen := make(map[string]bool, len(existing))
	for _, t := range existing {
		seen[t] = true
	}
	for _, t := range incoming {
		if !seen[t] {
			existing = append(existing, t)
		}
	}
	return existing
}

func labelsToTags(labels map[string]string) []string {
	var tags []string
	if app, ok := labels["app"]; ok && app != "" {
		tags = append(tags, app)
	}
	return tags
}
