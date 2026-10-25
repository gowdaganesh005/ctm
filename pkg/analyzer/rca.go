package analyzer

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/diff"
	"github.com/ctm/cluster-time-machine/pkg/store"
)

type RCAEngine struct {
	db *store.DB
}

func NewRCAEngine(db *store.DB) *RCAEngine {
	return &RCAEngine{db: db}
}

// AnalyzeIncident investigates an incident and ranks candidate root-cause revisions
func (rca *RCAEngine) AnalyzeIncident(inc *store.Incident, lookbackMinutes int) ([]store.RankedChange, error) {
	if lookbackMinutes <= 0 {
		lookbackMinutes = 30
	}

	windowStart := inc.DetectedAt.Add(-time.Duration(lookbackMinutes) * time.Minute)
	windowEnd := inc.DetectedAt.Add(2 * time.Minute) // Allow small clock skew

	revisions, err := rca.db.GetRevisionsInWindow(windowStart, windowEnd, inc.Namespace)
	if err != nil {
		return nil, err
	}

	var candidates []store.RankedChange

	for _, rev := range revisions {
		// Fetch previous revision to compute accurate diff
		prevRev, _ := rca.db.GetPreviousRevision(rev.ID, rev.ResourceID)
		diffRes, _ := diff.ComputeDiff(prevRev, &rev)

		score, reasons := scoreCandidate(inc, &rev, diffRes)
		if score > 0 {
			diffSummary := ""
			if diffRes != nil {
				diffSummary = diff.SummarizeChanges(diffRes.FieldChanges)
			}

			var fieldChanges []store.FieldDifference
			if diffRes != nil {
				fieldChanges = diffRes.FieldChanges
			}

			candidates = append(candidates, store.RankedChange{
				Score:        int(math.Min(float64(score), 99)),
				RevisionID:   rev.ID,
				ResourceKind: rev.Kind,
				ResourceName: rev.Name,
				Namespace:    rev.Namespace,
				Timestamp:    rev.Timestamp,
				ActorType:    rev.ActorType,
				ManagerName:  rev.ManagerName,
				ChangeType:   rev.ChangeType,
				Reasons:      reasons,
				DiffSummary:  diffSummary,
				FieldChanges: fieldChanges,
			})
		}
	}

	// Sort descending by Score, then by Timestamp descending
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Timestamp.After(candidates[j].Timestamp)
		}
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) > 10 {
		candidates = candidates[:10]
	}

	return candidates, nil
}

func scoreCandidate(inc *store.Incident, rev *store.ResourceRevision, diff *store.RevisionDiff) (int, []string) {
	score := 0
	var reasons []string

	titleLower := strings.ToLower(inc.Title)
	descLower := strings.ToLower(inc.Description)
	isOOM := strings.Contains(titleLower, "oom") || strings.Contains(descLower, "oom")
	isCrashLoop := strings.Contains(titleLower, "crashloop") || strings.Contains(descLower, "crashloop")
	isImagePull := strings.Contains(titleLower, "imagepull") || strings.Contains(descLower, "image")
	isScheduling := strings.Contains(titleLower, "schedul") || strings.Contains(descLower, "unschedulable")
	isNodeNotReady := strings.Contains(titleLower, "nodenotready") || rev.Kind == "Node"

	// 1. Actor Type Heuristics
	switch rev.ActorType {
	case store.ActorHuman:
		score += 50
		reasons = append(reasons, "Direct Human manual change ("+rev.ManagerName+") outside GitOps pipeline")
	case store.ActorGitOps:
		score += 25
		reasons = append(reasons, "Automated GitOps/Pipeline deployment ("+rev.ManagerName+")")
	case store.ActorController:
		score += 5
	}

	// 2. Resource Target Proximity
	isSameResource := rev.Kind == inc.ResourceKind && rev.Name == inc.ResourceName
	isRelatedWorkload := isRelated(inc.ResourceKind, inc.ResourceName, rev.Kind, rev.Name)

	if isSameResource {
		score += 40
		reasons = append(reasons, "Direct modification on the failing resource")
	} else if isRelatedWorkload {
		score += 35
		reasons = append(reasons, "Modification on parent workload/controller ("+rev.Kind+"/"+rev.Name+")")
	} else if rev.Kind == "ConfigMap" || rev.Kind == "Secret" {
		score += 20
		reasons = append(reasons, "Shared configuration ("+rev.Kind+"/"+rev.Name+") modified in same namespace")
	} else if rev.Kind == "Node" && isScheduling {
		score += 40
		reasons = append(reasons, "Cluster Node ("+rev.Name+") state modified prior to scheduling failure")
	}

	// 3. Field-Level Causal Matches
	if diff != nil && len(diff.FieldChanges) > 0 {
		for _, fc := range diff.FieldChanges {
			path := strings.ToLower(fc.Path)

			// OOMKilled Trigger: memory limits/requests lowered
			if isOOM && (strings.Contains(path, "limits.memory") || strings.Contains(path, "requests.memory")) {
				score += 60
				reasons = append(reasons, "Memory limit adjusted directly prior to OOM kill: "+fc.Path)
				break
			}

			// ImagePullBackOff: image changed
			if isImagePull && strings.Contains(path, "image") {
				score += 60
				reasons = append(reasons, "Container image modified: "+fc.Path)
				break
			}

			// CrashLoopBackOff: env, command, args, image, or config
			if isCrashLoop && (strings.Contains(path, "image") || strings.Contains(path, "env") || strings.Contains(path, "command") || strings.Contains(path, "args")) {
				score += 50
				reasons = append(reasons, "Runtime environment / entrypoint altered: "+fc.Path)
				break
			}

			// FailedScheduling: cpu/memory requests, nodeSelector, tolerations, cordoned
			if isScheduling {
				if strings.Contains(path, "requests.cpu") || strings.Contains(path, "requests.memory") {
					score += 50
					reasons = append(reasons, "Workload resource requests updated prior to scheduling failure")
					break
				}
				if strings.Contains(path, "nodeselector") || strings.Contains(path, "tolerations") || strings.Contains(path, "affinity") {
					score += 50
					reasons = append(reasons, "Node placement constraint modified: "+fc.Path)
					break
				}
				if strings.Contains(path, "unschedulable") || strings.Contains(path, "taints") {
					score += 60
					reasons = append(reasons, "Node cordoned/tainted preventing pod placement")
					break
				}
			}

			// NodeNotReady
			if isNodeNotReady && rev.Kind == "Node" {
				score += 40
				reasons = append(reasons, "Node condition updated")
				break
			}
		}
	}

	// 4. Temporal Decay: Closer in time = higher causal confidence
	delta := inc.DetectedAt.Sub(rev.Timestamp)
	if delta < 0 {
		delta = -delta
	}

	if delta <= 2*time.Minute {
		score += 25
		reasons = append(reasons, "Occurred within 2 minutes of incident trigger")
	} else if delta <= 10*time.Minute {
		score += 15
		reasons = append(reasons, "Occurred within 10 minutes of incident trigger")
	} else if delta <= 30*time.Minute {
		score += 5
	}

	return score, reasons
}

func isRelated(incKind, incName, revKind, revName string) bool {
	// e.g. Pod 'frontend-89f8c6b97-5k2qm' is child of ReplicaSet 'frontend-89f8c6b97' which is child of Deployment 'frontend'
	if incKind == "Pod" {
		if revKind == "Deployment" && strings.HasPrefix(incName, revName+"-") {
			return true
		}
		if revKind == "ReplicaSet" && strings.HasPrefix(incName, revName+"-") {
			return true
		}
		if revKind == "StatefulSet" && strings.HasPrefix(incName, revName+"-") {
			return true
		}
		if revKind == "DaemonSet" && strings.HasPrefix(incName, revName+"-") {
			return true
		}
	}
	return false
}
