package analyzer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type DetectedIncident struct {
	Title        string
	Severity     string
	Namespace    string
	ResourceKind string
	ResourceName string
	DetectedAt   time.Time
	Description  string
}

// InspectPodForIncidents checks a Pod JSON snapshot for critical failure states
func InspectPodForIncidents(podJSON string) []*DetectedIncident {
	var pod map[string]any
	if err := json.Unmarshal([]byte(podJSON), &pod); err != nil {
		return nil
	}

	metadata, _ := pod["metadata"].(map[string]any)
	if metadata == nil {
		return nil
	}
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)

	status, _ := pod["status"].(map[string]any)
	if status == nil {
		return nil
	}

	var incidents []*DetectedIncident

	// Check container statuses
	checkContainers := func(containerList []any) {
		for _, item := range containerList {
			cs, ok := item.(map[string]any)
			if !ok {
				continue
			}
			cName, _ := cs["name"].(string)
			restartCount, _ := cs["restartCount"].(float64)

			// 1. Check current state waiting
			if state, ok := cs["state"].(map[string]any); ok {
				if waiting, ok := state["waiting"].(map[string]any); ok {
					reason, _ := waiting["reason"].(string)
					message, _ := waiting["message"].(string)

					if reason == "CrashLoopBackOff" {
						incidents = append(incidents, &DetectedIncident{
							Title:        fmt.Sprintf("CrashLoopBackOff on container '%s'", cName),
							Severity:     "critical",
							Namespace:    namespace,
							ResourceKind: "Pod",
							ResourceName: name,
							DetectedAt:   time.Now().UTC(),
							Description:  fmt.Sprintf("Container '%s' is in CrashLoopBackOff (restartCount: %.0f). %s", cName, restartCount, message),
						})
					} else if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
						incidents = append(incidents, &DetectedIncident{
							Title:        fmt.Sprintf("ImagePullBackOff on container '%s'", cName),
							Severity:     "critical",
							Namespace:    namespace,
							ResourceKind: "Pod",
							ResourceName: name,
							DetectedAt:   time.Now().UTC(),
							Description:  fmt.Sprintf("Failed to pull container image for '%s': %s", cName, message),
						})
					}
				}

				if terminated, ok := state["terminated"].(map[string]any); ok {
					reason, _ := terminated["reason"].(string)
					exitCode, _ := terminated["exitCode"].(float64)

					if reason == "OOMKilled" || exitCode == 137 {
						incidents = append(incidents, &DetectedIncident{
							Title:        fmt.Sprintf("OOMKilled on container '%s'", cName),
							Severity:     "critical",
							Namespace:    namespace,
							ResourceKind: "Pod",
							ResourceName: name,
							DetectedAt:   time.Now().UTC(),
							Description:  fmt.Sprintf("Container '%s' exceeded memory limits and was killed by kernel OOM-killer (exitCode: 137).", cName),
						})
					}
				}
			}

			// 2. Check lastState for recent OOM
			if lastState, ok := cs["lastState"].(map[string]any); ok {
				if terminated, ok := lastState["terminated"].(map[string]any); ok {
					reason, _ := terminated["reason"].(string)
					exitCode, _ := terminated["exitCode"].(float64)

					if (reason == "OOMKilled" || exitCode == 137) && len(incidents) == 0 {
						incidents = append(incidents, &DetectedIncident{
							Title:        fmt.Sprintf("OOMKilled on container '%s'", cName),
							Severity:     "critical",
							Namespace:    namespace,
							ResourceKind: "Pod",
							ResourceName: name,
							DetectedAt:   time.Now().UTC(),
							Description:  fmt.Sprintf("Container '%s' previously died with OOMKilled (exit code 137, restarts: %.0f).", cName, restartCount),
						})
					}
				}
			}
		}
	}

	if csList, ok := status["containerStatuses"].([]any); ok {
		checkContainers(csList)
	}
	if initList, ok := status["initContainerStatuses"].([]any); ok {
		checkContainers(initList)
	}

	// 3. Check Pod conditions (FailedScheduling / Unschedulable)
	if conditions, ok := status["conditions"].([]any); ok {
		for _, c := range conditions {
			condMap, ok := c.(map[string]any)
			if !ok {
				continue
			}
			cType, _ := condMap["type"].(string)
			cStatus, _ := condMap["status"].(string)
			cReason, _ := condMap["reason"].(string)
			cMsg, _ := condMap["message"].(string)

			if cType == "PodScheduled" && cStatus == "False" && cReason == "Unschedulable" {
				incidents = append(incidents, &DetectedIncident{
					Title:        "FailedScheduling: Pod Unschedulable",
					Severity:     "warning",
					Namespace:    namespace,
					ResourceKind: "Pod",
					ResourceName: name,
					DetectedAt:   time.Now().UTC(),
					Description:  fmt.Sprintf("Pod could not be scheduled to any node: %s", cMsg),
				})
			}
		}
	}

	return incidents
}

// InspectNodeForIncidents checks Node status for conditions like NodeNotReady
func InspectNodeForIncidents(nodeJSON string) []*DetectedIncident {
	var node map[string]any
	if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
		return nil
	}

	metadata, _ := node["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)

	status, _ := node["status"].(map[string]any)
	if status == nil {
		return nil
	}

	var incidents []*DetectedIncident
	if conditions, ok := status["conditions"].([]any); ok {
		for _, c := range conditions {
			condMap, ok := c.(map[string]any)
			if !ok {
				continue
			}
			cType, _ := condMap["type"].(string)
			cStatus, _ := condMap["status"].(string)
			cReason, _ := condMap["reason"].(string)
			cMsg, _ := condMap["message"].(string)

			if cType == "Ready" && cStatus != "True" {
				incidents = append(incidents, &DetectedIncident{
					Title:        fmt.Sprintf("NodeNotReady on '%s'", name),
					Severity:     "critical",
					Namespace:    "",
					ResourceKind: "Node",
					ResourceName: name,
					DetectedAt:   time.Now().UTC(),
					Description:  fmt.Sprintf("Node '%s' is not ready (reason: %s): %s", name, cReason, cMsg),
				})
			}
		}
	}

	return incidents
}

// InspectEventForIncidents checks Warning/Error Kubernetes Events
func InspectEventForIncidents(eventJSON string) *DetectedIncident {
	var ev map[string]any
	if err := json.Unmarshal([]byte(eventJSON), &ev); err != nil {
		return nil
	}

	evType, _ := ev["type"].(string)
	if evType != "Warning" && evType != "Error" {
		return nil
	}

	reason, _ := ev["reason"].(string)
	message, _ := ev["message"].(string)

	involvedObj, _ := ev["involvedObject"].(map[string]any)
	if involvedObj == nil {
		return nil
	}

	kind, _ := involvedObj["kind"].(string)
	name, _ := involvedObj["name"].(string)
	namespace, _ := involvedObj["namespace"].(string)

	// Filter for actionable incidents
	criticalReasons := []string{"OOMKilled", "BackOff", "Failed", "FailedScheduling", "Unhealthy", "Evicted"}
	isMatch := false
	for _, r := range criticalReasons {
		if strings.EqualFold(reason, r) || strings.Contains(strings.ToLower(message), strings.ToLower(r)) {
			isMatch = true
			break
		}
	}

	if !isMatch {
		return nil
	}

	severity := "warning"
	if reason == "OOMKilled" || reason == "BackOff" || reason == "Evicted" {
		severity = "critical"
	}

	return &DetectedIncident{
		Title:        fmt.Sprintf("%s on %s/%s", reason, kind, name),
		Severity:     severity,
		Namespace:    namespace,
		ResourceKind: kind,
		ResourceName: name,
		DetectedAt:   time.Now().UTC(),
		Description:  message,
	}
}
