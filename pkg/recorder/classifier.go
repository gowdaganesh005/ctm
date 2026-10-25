package recorder

import (
	"strings"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/store"
)

type ManagedFieldEntry struct {
	Manager    string     `json:"manager"`
	Operation  string     `json:"operation"`
	APIVersion string     `json:"apiVersion"`
	Time       *time.Time `json:"time,omitempty"`
}

// ClassifyActor inspects the managedFields entries and resource metadata
// to determine whether the change was initiated by a Human, GitOps pipeline, or Controller.
func ClassifyActor(managedFields []ManagedFieldEntry, annotations map[string]string) (store.ActorType, string) {
	if len(managedFields) == 0 {
		// Fallback to checking annotations
		if annotations != nil {
			if _, ok := annotations["kubectl.kubernetes.io/last-applied-configuration"]; ok {
				return store.ActorHuman, "kubectl"
			}
			if _, ok := annotations["helm.sh/hook"]; ok || annotations["meta.helm.sh/release-name"] != "" {
				return store.ActorGitOps, "helm"
			}
			if annotations["argocd.argoproj.io/tracking-id"] != "" {
				return store.ActorGitOps, "argocd"
			}
		}
		return store.ActorController, "unknown-controller"
	}

	// Pick the latest entry by timestamp
	var latestEntry ManagedFieldEntry
	var latestTime time.Time
	for _, entry := range managedFields {
		if entry.Time != nil {
			if latestTime.IsZero() || entry.Time.After(latestTime) {
				latestTime = *entry.Time
				latestEntry = entry
			}
		} else if latestEntry.Manager == "" {
			latestEntry = entry
		}
	}

	if latestEntry.Manager == "" {
		latestEntry = managedFields[len(managedFields)-1]
	}

	mgr := strings.ToLower(latestEntry.Manager)

	// Check Human indicators
	if isHumanManager(mgr) {
		return store.ActorHuman, latestEntry.Manager
	}

	// Check GitOps indicators
	if isGitOpsManager(mgr) {
		return store.ActorGitOps, latestEntry.Manager
	}

	// Check Controller / System indicators
	return store.ActorController, latestEntry.Manager
}

func isHumanManager(mgr string) bool {
	humanPatterns := []string{
		"kubectl",
		"kubectl-edit",
		"kubectl-patch",
		"kubectl-client-side-apply",
		"kubectl-set",
		"kubectl-rollout",
		"kubectl-scale",
		"kubectl-replace",
		"k9s",
		"lens",
		"dashboard",
		"octant",
		"headlamp",
		"curl",
		"postman",
		"user",
		"admin",
	}
	for _, p := range humanPatterns {
		if strings.Contains(mgr, p) {
			return true
		}
	}
	return false
}

func isGitOpsManager(mgr string) bool {
	gitopsPatterns := []string{
		"helm",
		"argo",
		"argocd",
		"flux",
		"kustomize-controller",
		"helm-controller",
		"source-controller",
		"ci-cd",
		"gitlab",
		"github-actions",
		"tekton",
		"drone",
		"crossplane",
		"terraform",
		"pulumi",
		"pipeline",
	}
	for _, p := range gitopsPatterns {
		if strings.Contains(mgr, p) {
			return true
		}
	}
	return false
}
