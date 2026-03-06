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

// Sync discovers ArgoCD Applications and upserts them as Gantry Service entities.
//
// Correlation with Kubernetes:
//   - ArgoCD Application names typically match the `app` label used by the
//     Kubernetes deployment, which is also the entity name created by the k8s
//     plugin. When both plugins are active, this function merges ArgoCD
//     annotations onto the existing k8s-created Service entity (or creates a
//     new one if none exists yet).
//   - spec.deployedIn is augmented with the ArgoCD destination namespace so
//     the Kubernetes tab can look up the right pods.
func Sync(ctx context.Context, config map[string]any, store EntityStore) (*SyncResult, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}

	project, _ := config["project"].(string)
	argoURL, _ := config["argocdUrl"].(string)
	argoURL = strings.TrimRight(argoURL, "/")

	apps, err := client.ListApplications(project)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}

	res := &SyncResult{}
	for _, app := range apps {
		e := appToService(app, argoURL)
		if err := upsert(ctx, store, e, res); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("app %s: %v", app.Metadata.Name, err))
		}
		res.Apps++
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Converters
// ---------------------------------------------------------------------------

// appToService converts an ArgoCD Application into a Gantry Service entity.
// The entity name is the ArgoCD app name, matching the `app` label convention
// used by the Kubernetes plugin — enabling cross-plugin correlation.
func appToService(app Application, argoURL string) *entity.Entity {
	annotations := buildAnnotations(app, argoURL)
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
			Name:        app.Metadata.Name,
			Namespace:   "default",
			Title:       app.Metadata.Name,
			Description: desc,
			Annotations: annotations,
			Tags:        tags,
		},
		Spec: spec,
	}
}

func buildAnnotations(app Application, argoURL string) map[string]string {
	a := map[string]string{
		"argocd.io/appName":      app.Metadata.Name,
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
	// ArgoCD server UI deep-link for the application.
	if argoURL != "" {
		a["argocd.io/appURL"] = argoURL + "/applications/" + app.Metadata.Name
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
func mergeAnnotations(existing, incoming map[string]string) map[string]string {
	if existing == nil {
		return incoming
	}
	for k, v := range incoming {
		existing[k] = v
	}
	return existing
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
