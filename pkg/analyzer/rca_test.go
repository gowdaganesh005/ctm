package analyzer

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/store"
)

func TestInspectPodForIncidents(t *testing.T) {
	// Pod with CrashLoopBackOff and OOMKilled
	podJSON := `{
		"metadata": {"name": "api-gateway", "namespace": "prod"},
		"status": {
			"phase": "Running",
			"containerStatuses": [
				{
					"name": "gateway",
					"restartCount": 5,
					"state": {
						"waiting": {
							"reason": "CrashLoopBackOff",
							"message": "back-off restarting failed container"
						}
					},
					"lastState": {
						"terminated": {
							"reason": "OOMKilled",
							"exitCode": 137
						}
					}
				}
			]
		}
	}`

	incidents := InspectPodForIncidents(podJSON)
	if len(incidents) == 0 {
		t.Fatalf("expected incidents detected, got 0")
	}

	foundCrash := false
	for _, inc := range incidents {
		if inc.ResourceName == "api-gateway" && inc.Namespace == "prod" {
			foundCrash = true
		}
	}
	if !foundCrash {
		t.Errorf("expected incident on api-gateway in prod")
	}
}

func TestInspectNodeForIncidents(t *testing.T) {
	nodeJSON := `{
		"metadata": {"name": "worker-pool-node-3"},
		"status": {
			"conditions": [
				{
					"type": "Ready",
					"status": "False",
					"reason": "KubeletNotReady",
					"message": "container runtime network not ready"
				}
			]
		}
	}`

	incidents := InspectNodeForIncidents(nodeJSON)
	if len(incidents) == 0 {
		t.Fatalf("expected NodeNotReady incident detected, got 0")
	}
	if incidents[0].Severity != "critical" {
		t.Errorf("expected critical severity for NodeNotReady, got %s", incidents[0].Severity)
	}
}

func TestRCAEngineRanking(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := store.NewDB(filepath.Join(tmpDir, "rca_test.db"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()

	// 1. Create Deployment resource
	dep, err := db.EnsureResource("deploy-uid-1", "default", "Deployment", "web-app")
	if err != nil {
		t.Fatalf("failed to ensure resource: %v", err)
	}

	// 2. Insert baseline revision (15 mins ago by ArgoCD)
	_, err = db.InsertRevision(
		dep.ID, "100", now.Add(-15*time.Minute), store.ActorGitOps, "argocd-controller", store.ChangeAdded,
		`{"apiVersion":"apps/v1","kind":"Deployment","spec":{"template":{"spec":{"containers":[{"name":"web","resources":{"limits":{"memory":"1Gi"}}}]}}}}`,
	)
	if err != nil {
		t.Fatalf("failed to insert rev 1: %v", err)
	}

	// 3. Insert direct human edit revision (5 mins ago by kubectl-edit reducing memory to 16Mi)
	humanRev, err := db.InsertRevision(
		dep.ID, "101", now.Add(-5*time.Minute), store.ActorHuman, "kubectl-edit", store.ChangeModified,
		`{"apiVersion":"apps/v1","kind":"Deployment","spec":{"template":{"spec":{"containers":[{"name":"web","resources":{"limits":{"memory":"16Mi"}}}]}}}}`,
	)
	if err != nil {
		t.Fatalf("failed to insert rev 2: %v", err)
	}

	// 4. Create an incident for OOMKilled on Pod web-app-789
	inc := &store.Incident{
		Title:        "OOMKilled on container 'web'",
		Severity:     "critical",
		Namespace:    "default",
		ResourceKind: "Pod",
		ResourceName: "web-app-789-xyz",
		DetectedAt:   now.Add(-3 * time.Minute),
		Description:  "Memory limit exceeded (exitCode: 137)",
	}

	rca := NewRCAEngine(db)
	ranked, err := rca.AnalyzeIncident(inc, 30)
	if err != nil {
		t.Fatalf("RCA analysis failed: %v", err)
	}

	if len(ranked) == 0 {
		t.Fatalf("expected candidate causes, got 0")
	}

	// The top candidate should be the human edit (revision 101) with a high confidence score
	top := ranked[0]
	if top.RevisionID != humanRev.ID {
		t.Errorf("expected top candidate to be revision %d, got %d", humanRev.ID, top.RevisionID)
	}
	if top.ActorType != store.ActorHuman {
		t.Errorf("expected top candidate to be Human actor, got %s", top.ActorType)
	}
	if top.Score < 80 {
		t.Errorf("expected top candidate score >= 80, got %d", top.Score)
	}
}
