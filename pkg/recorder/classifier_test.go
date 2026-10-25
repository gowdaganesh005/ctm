package recorder

import (
	"testing"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/store"
)

func TestClassifyActor(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-5 * time.Minute)

	tests := []struct {
		name          string
		managedFields []ManagedFieldEntry
		annotations   map[string]string
		wantActor     store.ActorType
		wantManager   string
	}{
		{
			name: "Human kubectl-edit",
			managedFields: []ManagedFieldEntry{
				{Manager: "argocd-controller", Time: &earlier},
				{Manager: "kubectl-edit", Time: &now},
			},
			wantActor:   store.ActorHuman,
			wantManager: "kubectl-edit",
		},
		{
			name: "GitOps ArgoCD Controller",
			managedFields: []ManagedFieldEntry{
				{Manager: "argocd-application-controller", Time: &now},
			},
			wantActor:   store.ActorGitOps,
			wantManager: "argocd-application-controller",
		},
		{
			name: "GitOps Helm",
			managedFields: []ManagedFieldEntry{
				{Manager: "helm", Time: &now},
			},
			wantActor:   store.ActorGitOps,
			wantManager: "helm",
		},
		{
			name: "System Controller kubelet",
			managedFields: []ManagedFieldEntry{
				{Manager: "kubelet", Time: &now},
			},
			wantActor:   store.ActorController,
			wantManager: "kubelet",
		},
		{
			name: "Fallback annotation kubectl",
			managedFields: nil,
			annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
			},
			wantActor:   store.ActorHuman,
			wantManager: "kubectl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotActor, gotManager := ClassifyActor(tt.managedFields, tt.annotations)
			if gotActor != tt.wantActor {
				t.Errorf("got actor %v, want %v", gotActor, tt.wantActor)
			}
			if gotManager != tt.wantManager {
				t.Errorf("got manager %v, want %v", gotManager, tt.wantManager)
			}
		})
	}
}
