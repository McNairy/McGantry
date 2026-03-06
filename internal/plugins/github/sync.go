package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/go2engle/gantry/internal/entity"
)

// EntityStore is the subset of db.DB used by the enrich operation.
type EntityStore interface {
	ListEntities(ctx context.Context, kind, namespace string) ([]*entity.Entity, error)
	UpdateEntity(ctx context.Context, e *entity.Entity) error
}

// Sync scans all catalog entities that have a github.com repoUrl in their spec
// and enriches their metadata annotations with live data from the GitHub API.
// It does not create new entities — it only augments ones that already exist.
func Sync(ctx context.Context, config map[string]any, store EntityStore) (*SyncResult, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}

	entities, err := store.ListEntities(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("list entities: %w", err)
	}

	res := &SyncResult{Scanned: len(entities)}

	for _, e := range entities {
		repoURL, _ := e.Spec["repoUrl"].(string)
		if repoURL == "" || !strings.Contains(repoURL, "github.com") {
			continue
		}

		owner, repo, err := ParseGitHubURL(repoURL)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s %s/%s: %v", e.Kind, e.Metadata.Namespace, e.Metadata.Name, err))
			continue
		}

		repoInfo, err := client.GetRepo(owner, repo)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s %s/%s: %v", e.Kind, e.Metadata.Namespace, e.Metadata.Name, err))
			continue
		}

		enrichEntity(e, owner, repo, repoInfo)

		if err := store.UpdateEntity(ctx, e); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s %s/%s: update failed: %v", e.Kind, e.Metadata.Namespace, e.Metadata.Name, err))
			continue
		}
		res.Enriched++
	}

	return res, nil
}

// enrichEntity writes GitHub-sourced annotations onto an existing entity.
// It only sets annotation values — it never overwrites user-edited spec fields.
func enrichEntity(e *entity.Entity, owner, repo string, r *Repository) {
	if e.Metadata.Annotations == nil {
		e.Metadata.Annotations = make(map[string]string)
	}
	e.Metadata.Annotations["github.com/owner"] = owner
	e.Metadata.Annotations["github.com/repo"] = repo
	e.Metadata.Annotations["github.com/full-name"] = r.FullName
	e.Metadata.Annotations["github.com/default-branch"] = r.DefaultBranch
	if r.Language != "" {
		e.Metadata.Annotations["github.com/language"] = r.Language
	}
	if len(r.Topics) > 0 {
		e.Metadata.Annotations["github.com/topics"] = strings.Join(r.Topics, ",")
	}
	if r.Archived {
		e.Metadata.Annotations["github.com/archived"] = "true"
	} else {
		delete(e.Metadata.Annotations, "github.com/archived")
	}
}
