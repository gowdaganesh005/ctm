package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTimeTravelReconstruction(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(filepath.Join(tmpDir, "timetravel_test.db"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	t0 := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(10 * time.Minute) // 10:10
	t2 := t0.Add(20 * time.Minute) // 10:20
	t3 := t0.Add(30 * time.Minute) // 10:30

	// 1. Create Deployment A
	resA, err := db.EnsureResource("uid-a", "prod", "Deployment", "service-a")
	if err != nil {
		t.Fatalf("failed to create resource: %v", err)
	}

	// At t0: Added with replicas: 1
	_, err = db.InsertRevision(resA.ID, "1", t0, ActorGitOps, "argocd", ChangeAdded, `{"spec":{"replicas":1}}`)
	if err != nil {
		t.Fatalf("failed to insert rev: %v", err)
	}

	// At t2: Scaled to replicas: 5
	_, err = db.InsertRevision(resA.ID, "2", t2, ActorHuman, "kubectl-scale", ChangeModified, `{"spec":{"replicas":5}}`)
	if err != nil {
		t.Fatalf("failed to insert rev: %v", err)
	}

	// 2. Create Service B
	resB, err := db.EnsureResource("uid-b", "prod", "Service", "service-b")
	if err != nil {
		t.Fatalf("failed to create resource: %v", err)
	}

	// At t1: Added Service B
	_, err = db.InsertRevision(resB.ID, "1", t1, ActorGitOps, "helm", ChangeAdded, `{"spec":{"port":80}}`)
	if err != nil {
		t.Fatalf("failed to insert rev: %v", err)
	}

	// At t3: Service B is deleted
	_, err = db.InsertRevision(resB.ID, "2", t3, ActorHuman, "kubectl-delete", ChangeDeleted, `{}`)
	if err != nil {
		t.Fatalf("failed to insert rev: %v", err)
	}

	// Test 1: Query at t0 (10:00) -> Only service-a should exist with replicas: 1
	resAtT0, err := db.GetResourcesAtTime(t0, "prod")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resAtT0) != 1 {
		t.Fatalf("expected 1 resource at t0, got %d", len(resAtT0))
	}
	if resAtT0[0].Name != "service-a" || resAtT0[0].ResourceVersion != "1" {
		t.Errorf("expected service-a rev 1, got %s rev %s", resAtT0[0].Name, resAtT0[0].ResourceVersion)
	}

	// Test 2: Query at t1 + 5m (10:15) -> Both service-a (rev 1) and service-b (rev 1) should exist
	resAtT15, err := db.GetResourcesAtTime(t1.Add(5*time.Minute), "prod")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resAtT15) != 2 {
		t.Fatalf("expected 2 resources at 10:15, got %d", len(resAtT15))
	}

	// Test 3: Query at t2 + 5m (10:25) -> service-a should be rev 2 (replicas: 5) and service-b rev 1
	resAtT25, err := db.GetResourcesAtTime(t2.Add(5*time.Minute), "prod")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resAtT25) != 2 {
		t.Fatalf("expected 2 resources at 10:25, got %d", len(resAtT25))
	}
	var aFound bool
	for _, r := range resAtT25 {
		if r.Name == "service-a" {
			aFound = true
			if r.ResourceVersion != "2" || r.ActorType != ActorHuman {
				t.Errorf("expected service-a to be rev 2 with Human actor, got rev %s actor %s", r.ResourceVersion, r.ActorType)
			}
		}
	}
	if !aFound {
		t.Errorf("service-a not found at 10:25")
	}

	// Test 4: Query at t3 + 5m (10:35) -> service-b was deleted, so only service-a should remain!
	resAtT35, err := db.GetResourcesAtTime(t3.Add(5*time.Minute), "prod")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resAtT35) != 1 {
		t.Fatalf("expected 1 resource at 10:35 (service-b deleted), got %d", len(resAtT35))
	}
	if resAtT35[0].Name != "service-a" {
		t.Errorf("expected remaining resource to be service-a, got %s", resAtT35[0].Name)
	}
}

func TestManualChangesQuery(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(filepath.Join(tmpDir, "manual_test.db"))
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	res, _ := db.EnsureResource("uid-1", "default", "ConfigMap", "app-cfg")
	now := time.Now().UTC()

	// 1 GitOps rev, 1 Human rev
	_, _ = db.InsertRevision(res.ID, "1", now.Add(-10*time.Minute), ActorGitOps, "argocd", ChangeAdded, `{}`)
	_, _ = db.InsertRevision(res.ID, "2", now.Add(-5*time.Minute), ActorHuman, "kubectl-edit", ChangeModified, `{"data":{"debug":"true"}}`)

	manuals, err := db.GetManualChanges(50, 0)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(manuals) != 1 {
		t.Fatalf("expected 1 manual change, got %d", len(manuals))
	}
	if manuals[0].ActorType != ActorHuman || manuals[0].ManagerName != "kubectl-edit" {
		t.Errorf("unexpected manual change: %+v", manuals[0])
	}
}
