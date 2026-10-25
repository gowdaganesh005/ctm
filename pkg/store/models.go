package store

import (
	"time"
)

type ActorType string

const (
	ActorHuman      ActorType = "Human"
	ActorGitOps     ActorType = "GitOps"
	ActorController ActorType = "Controller"
)

type ChangeType string

const (
	ChangeAdded    ChangeType = "ADDED"
	ChangeModified ChangeType = "MODIFIED"
	ChangeDeleted  ChangeType = "DELETED"
)

type Resource struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Namespace string    `json:"namespace"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type ResourceRevision struct {
	ID              int64      `json:"id"`
	ResourceID      int64      `json:"resource_id"`
	ResourceVersion string     `json:"resource_version"`
	Timestamp       time.Time  `json:"timestamp"`
	ActorType       ActorType  `json:"actor_type"`
	ManagerName     string     `json:"manager_name"`
	ChangeType      ChangeType `json:"change_type"`
	RawJSON         string     `json:"raw_json"`

	// Joined fields for convenience
	UID       string `json:"uid,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name,omitempty"`
}

type Incident struct {
	ID                int64          `json:"id"`
	Title             string         `json:"title"`
	Severity          string         `json:"severity"` // "warning", "critical"
	Namespace         string         `json:"namespace"`
	ResourceKind      string         `json:"resource_kind"`
	ResourceName      string         `json:"resource_name"`
	DetectedAt        time.Time      `json:"detected_at"`
	Description       string         `json:"description"`
	RankedChangesJSON string         `json:"ranked_changes_json"`
	RankedChanges     []RankedChange `json:"ranked_changes,omitempty"`
}

type RankedChange struct {
	Score          int               `json:"score"`           // 0-100 confidence
	RevisionID     int64             `json:"revision_id"`
	ResourceKind   string            `json:"resource_kind"`
	ResourceName   string            `json:"resource_name"`
	Namespace      string            `json:"namespace"`
	Timestamp      time.Time         `json:"timestamp"`
	ActorType      ActorType         `json:"actor_type"`
	ManagerName    string            `json:"manager_name"`
	ChangeType     ChangeType        `json:"change_type"`
	Reasons        []string          `json:"reasons"`          // Explanations why this was ranked
	DiffSummary    string            `json:"diff_summary"`     // Brief diff note, e.g. "resources.limits.memory changed from 512Mi to 16Mi"
	FieldChanges   []FieldDifference `json:"field_changes,omitempty"`
}

type FieldDifference struct {
	Path     string `json:"path"`
	OldValue any    `json:"old_value"`
	NewValue any    `json:"new_value"`
	Type     string `json:"type"` // "added", "removed", "modified"
}

type RevisionDiff struct {
	OldRevision  *ResourceRevision `json:"old_revision,omitempty"`
	NewRevision  *ResourceRevision `json:"new_revision"`
	OldYAML      string            `json:"old_yaml"`
	NewYAML      string            `json:"new_yaml"`
	UnifiedDiff  string            `json:"unified_diff"`
	FieldChanges []FieldDifference `json:"field_changes"`
}

type TimeTravelResource struct {
	ResourceID      int64     `json:"resource_id"`
	UID             string    `json:"uid"`
	Namespace       string    `json:"namespace"`
	Kind            string    `json:"kind"`
	Name            string    `json:"name"`
	RevisionID      int64     `json:"revision_id"`
	ResourceVersion string    `json:"resource_version"`
	Timestamp       time.Time `json:"timestamp"`
	ActorType       ActorType `json:"actor_type"`
	ManagerName     string    `json:"manager_name"`
	RawJSON         string    `json:"raw_json"`
	RawYAML         string    `json:"raw_yaml,omitempty"`
}

type ClusterStats struct {
	TotalResources int64 `json:"total_resources"`
	TotalRevisions int64 `json:"total_revisions"`
	HumanChanges   int64 `json:"human_changes"`
	GitOpsChanges  int64 `json:"gitops_changes"`
	CtrlChanges    int64 `json:"ctrl_changes"`
	IncidentsCount int64 `json:"incidents_count"`
	EarliestTime   *time.Time `json:"earliest_time,omitempty"`
	LatestTime     *time.Time `json:"latest_time,omitempty"`
}
