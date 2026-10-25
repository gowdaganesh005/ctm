package recorder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ScrubbedResource represents a cleaned Kubernetes resource ready for storage
type ScrubbedResource struct {
	UID             string               `json:"uid"`
	Namespace       string               `json:"namespace"`
	Kind            string               `json:"kind"`
	Name            string               `json:"name"`
	ResourceVersion string               `json:"resourceVersion"`
	ManagedFields   []ManagedFieldEntry  `json:"managedFields,omitempty"`
	CleanJSON       string               `json:"cleanJson"`
	ContentHash     string               `json:"contentHash"`
	IsIgnored       bool                 `json:"isIgnored"`
	IgnoreReason    string               `json:"ignoreReason,omitempty"`
}

var sensitiveKeyWords = []string{
	"password", "secret", "token", "apikey", "api_key",
	"auth", "private", "credential", "cert", "key",
}

// FilterAndScrubResource cleans a raw Kubernetes resource JSON according to strict noise and security rules.
func FilterAndScrubResource(raw []byte, kindHint string) (*ScrubbedResource, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("failed to parse resource json: %w", err)
	}

	kind, _ := obj["kind"].(string)
	if kind == "" {
		kind = kindHint
	}

	// 1. Ignore noise resources (Leases, endpointslices heartbeat noise)
	if kind == "Lease" || kind == "Endpoints" || kind == "EndpointSlice" {
		return &ScrubbedResource{IsIgnored: true, IgnoreReason: "Ignored resource kind (Lease/Endpoint)"}, nil
	}

	// 2. Filter Events: Capture WARNING and ERROR events only (Failed, BackOff, OOMKilled, FailedScheduling, Evicted, Unhealthy)
	if kind == "Event" {
		evType, _ := obj["type"].(string)
		reason, _ := obj["reason"].(string)
		message, _ := obj["message"].(string)
		if evType != "Warning" && evType != "Error" {
			return &ScrubbedResource{IsIgnored: true, IgnoreReason: "Routine normal event filtered"}, nil
		}
		isCritical := false
		criticalReasons := []string{"OOMKilled", "BackOff", "Failed", "FailedScheduling", "Unhealthy", "Evicted"}
		for _, r := range criticalReasons {
			if strings.EqualFold(reason, r) || strings.Contains(strings.ToLower(message), strings.ToLower(r)) {
				isCritical = true
				break
			}
		}
		if !isCritical {
			return &ScrubbedResource{IsIgnored: true, IgnoreReason: "Non-critical warning event filtered"}, nil
		}
	}

	metadata, _ := obj["metadata"].(map[string]any)
	if metadata == nil {
		metadata = make(map[string]any)
		obj["metadata"] = metadata
	}

	uid, _ := metadata["uid"].(string)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	resourceVersion, _ := metadata["resourceVersion"].(string)

	// Extract and sanitize managedFields
	rawMF, _ := metadata["managedFields"].([]any)
	cleanedMF, extractedEntries := sanitizeManagedFields(rawMF)
	metadata["managedFields"] = cleanedMF

	// Strip noise from metadata: uid, creationTimestamp, resourceVersion
	delete(metadata, "uid")
	delete(metadata, "creationTimestamp")
	delete(metadata, "resourceVersion")
	delete(metadata, "selfLink")
	delete(metadata, "generation") // strip generation from diff comparison

	// Security: NEVER store plaintext secrets
	if kind == "Secret" {
		sanitizeSecret(obj)
	}

	// Scrub sensitive env vars in Pods, Deployments, StatefulSets, DaemonSets, ReplicaSets
	sanitizeWorkloadEnv(obj)

	// Pod Status Noise: Filter out routine timestamps
	if kind == "Pod" {
		sanitizePodStatus(obj)
	}

	// Node status noise: filter routine heartbeats
	if kind == "Node" {
		sanitizeNodeStatus(obj)
	}

	// Format cleaned JSON
	cleanBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scrubbed resource: %w", err)
	}

	// Calculate deterministic content hash
	hasher := sha256.New()
	hasher.Write(cleanBytes)
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	return &ScrubbedResource{
		UID:             uid,
		Namespace:       namespace,
		Kind:            kind,
		Name:            name,
		ResourceVersion: resourceVersion,
		ManagedFields:   extractedEntries,
		CleanJSON:       string(cleanBytes),
		ContentHash:     contentHash,
	}, nil
}

// sanitizeManagedFields retains only manager, operation, apiVersion, and time.
// Strips fieldsV1, fieldsType, etc.
func sanitizeManagedFields(raw []any) ([]any, []ManagedFieldEntry) {
	if len(raw) == 0 {
		return nil, nil
	}

	var cleaned []any
	var entries []ManagedFieldEntry

	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		manager, _ := m["manager"].(string)
		operation, _ := m["operation"].(string)
		apiVersion, _ := m["apiVersion"].(string)
		timeStr, _ := m["time"].(string)

		var t *time.Time
		if timeStr != "" {
			if parsed, err := time.Parse(time.RFC3339, timeStr); err == nil {
				t = &parsed
			}
		}

		entry := ManagedFieldEntry{
			Manager:    manager,
			Operation:  operation,
			APIVersion: apiVersion,
			Time:       t,
		}
		entries = append(entries, entry)

		// Retain scrubbed representation without massive fieldsV1 trees
		scrubbed := map[string]any{
			"manager":    manager,
			"operation":  operation,
			"apiVersion": apiVersion,
		}
		if timeStr != "" {
			scrubbed["time"] = timeStr
		}
		cleaned = append(cleaned, scrubbed)
	}

	return cleaned, entries
}

// sanitizeSecret replaces all plaintext Secret data with key name + SHA256 of the value.
func sanitizeSecret(obj map[string]any) {
	if data, ok := obj["data"].(map[string]any); ok {
		scrubbedData := make(map[string]any)
		for k, v := range data {
			valStr := fmt.Sprintf("%v", v)
			h := sha256.Sum256([]byte(valStr))
			scrubbedData[k] = fmt.Sprintf("sha256:%s (redacted)", hex.EncodeToString(h[:8]))
		}
		obj["data"] = scrubbedData
	}

	if stringData, ok := obj["stringData"].(map[string]any); ok {
		scrubbedStringData := make(map[string]any)
		for k, v := range stringData {
			valStr := fmt.Sprintf("%v", v)
			h := sha256.Sum256([]byte(valStr))
			scrubbedStringData[k] = fmt.Sprintf("sha256:%s (redacted)", hex.EncodeToString(h[:8]))
		}
		obj["stringData"] = scrubbedStringData
	}
}

// sanitizeWorkloadEnv masks sensitive environment variable values
func sanitizeWorkloadEnv(obj map[string]any) {
	traverseContainers(obj, func(container map[string]any) {
		if envList, ok := container["env"].([]any); ok {
			for _, e := range envList {
				envMap, ok := e.(map[string]any)
				if !ok {
					continue
				}
				name, _ := envMap["name"].(string)
				val, exists := envMap["value"]
				if exists && isSensitiveKey(name) {
					envMap["value"] = "[REDACTED]"
				} else if exists && val != nil {
					// Check if value itself looks like a token/secret
					strVal := fmt.Sprintf("%v", val)
					if len(strVal) > 60 && !strings.Contains(strVal, " ") {
						envMap["value"] = "[REDACTED-LONG-KEY]"
					}
				}
			}
		}
	})
}

// sanitizePodStatus filters out routine status timestamps (startTime, container image IDs)
// Keeps phase, restartCount, containerStatuses[*].state/lastState (reasons like OOMKilled, CrashLoopBackOff)
func sanitizePodStatus(obj map[string]any) {
	status, ok := obj["status"].(map[string]any)
	if !ok {
		return
	}

	delete(status, "startTime")
	delete(status, "hostIP")
	delete(status, "podIP")
	delete(status, "podIPs")

	// Filter conditions routine timestamps
	if conditions, ok := status["conditions"].([]any); ok {
		for _, c := range conditions {
			if condMap, ok := c.(map[string]any); ok {
				delete(condMap, "lastProbeTime")
				delete(condMap, "lastTransitionTime")
			}
		}
	}

	// Filter containerStatuses: strip containerID, imageID, routine time
	cleanContainerStatuses := func(key string) {
		if csList, ok := status[key].([]any); ok {
			for _, cs := range csList {
				if csMap, ok := cs.(map[string]any); ok {
					delete(csMap, "containerID")
					delete(csMap, "imageID")
					// Clean state timestamps
					if state, ok := csMap["state"].(map[string]any); ok {
						for _, s := range state {
							if sMap, ok := s.(map[string]any); ok {
								delete(sMap, "startedAt")
								delete(sMap, "finishedAt")
							}
						}
					}
					if lastState, ok := csMap["lastState"].(map[string]any); ok {
						for _, s := range lastState {
							if sMap, ok := s.(map[string]any); ok {
								delete(sMap, "startedAt")
								delete(sMap, "finishedAt")
							}
						}
					}
				}
			}
		}
	}

	cleanContainerStatuses("containerStatuses")
	cleanContainerStatuses("initContainerStatuses")
}

// sanitizeNodeStatus strips routine heartbeat timestamps (lastHeartbeatTime)
func sanitizeNodeStatus(obj map[string]any) {
	status, ok := obj["status"].(map[string]any)
	if !ok {
		return
	}

	if conditions, ok := status["conditions"].([]any); ok {
		for _, c := range conditions {
			if condMap, ok := c.(map[string]any); ok {
				delete(condMap, "lastHeartbeatTime")
				delete(condMap, "lastTransitionTime")
			}
		}
	}
}

func traverseContainers(obj map[string]any, fn func(container map[string]any)) {
	// 1. Pod spec directly
	if spec, ok := obj["spec"].(map[string]any); ok {
		inspectContainersInSpec(spec, fn)
		// Pod template spec in Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs
		if template, ok := spec["template"].(map[string]any); ok {
			if tmplSpec, ok := template["spec"].(map[string]any); ok {
				inspectContainersInSpec(tmplSpec, fn)
			}
		}
	}
}

func inspectContainersInSpec(spec map[string]any, fn func(container map[string]any)) {
	if containers, ok := spec["containers"].([]any); ok {
		for _, c := range containers {
			if cMap, ok := c.(map[string]any); ok {
				fn(cMap)
			}
		}
	}
	if initContainers, ok := spec["initContainers"].([]any); ok {
		for _, c := range initContainers {
			if cMap, ok := c.(map[string]any); ok {
				fn(cMap)
			}
		}
	}
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, kw := range sensitiveKeyWords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
