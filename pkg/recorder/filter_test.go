package recorder

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFilterAndScrubSecretNeverStoresPlaintext(t *testing.T) {
	rawSecret := `{
		"apiVersion": "v1",
		"kind": "Secret",
		"metadata": {
			"name": "db-credentials",
			"namespace": "production",
			"uid": "12345-abcde",
			"creationTimestamp": "2026-10-22T10:00:00Z",
			"resourceVersion": "999888"
		},
		"type": "Opaque",
		"data": {
			"password": "bXlTdXBlclNlY3JldFBhc3N3b3JkIQ=="
		},
		"stringData": {
			"token": "sensitive-api-token-xyz"
		}
	}`

	scrubbed, err := FilterAndScrubResource([]byte(rawSecret), "Secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if scrubbed.IsIgnored {
		t.Fatalf("expected secret to not be ignored")
	}

	// Ensure plaintext password is NOT in the clean JSON
	if strings.Contains(scrubbed.CleanJSON, "bXlTdXBlclNlY3JldFBhc3N3b3JkIQ==") {
		t.Fatalf("SECURITY VIOLATION: plaintext secret found in clean JSON output!")
	}
	if strings.Contains(scrubbed.CleanJSON, "sensitive-api-token-xyz") {
		t.Fatalf("SECURITY VIOLATION: plaintext stringData secret found in clean JSON output!")
	}

	// Must contain sha256 prefix
	if !strings.Contains(scrubbed.CleanJSON, "sha256:") {
		t.Errorf("expected SHA256 hashed secret value")
	}

	// Ensure metadata noise was stripped
	var cleanMap map[string]any
	_ = json.Unmarshal([]byte(scrubbed.CleanJSON), &cleanMap)
	meta, _ := cleanMap["metadata"].(map[string]any)
	if _, exists := meta["creationTimestamp"]; exists {
		t.Errorf("creationTimestamp should have been stripped")
	}
	if _, exists := meta["resourceVersion"]; exists {
		t.Errorf("resourceVersion should have been stripped")
	}
}

func TestFilterAndScrubPodRedactsSensitiveEnvAndNoise(t *testing.T) {
	rawPod := `{
		"apiVersion": "v1",
		"kind": "Pod",
		"metadata": {
			"name": "auth-service",
			"namespace": "default",
			"managedFields": [
				{
					"manager": "kubectl-edit",
					"operation": "Update",
					"apiVersion": "v1",
					"time": "2026-10-22T12:00:00Z",
					"fieldsV1": {"f:metadata": {"f:labels": {}}}
				}
			]
		},
		"spec": {
			"containers": [
				{
					"name": "auth",
					"image": "auth:v1",
					"env": [
						{"name": "DATABASE_PASSWORD", "value": "supersecretpassword123"},
						{"name": "PORT", "value": "8080"}
					]
				}
			]
		},
		"status": {
			"phase": "Running",
			"startTime": "2026-10-22T11:00:00Z",
			"podIP": "10.244.0.5",
			"containerStatuses": [
				{
					"name": "auth",
					"containerID": "containerd://1234567890abcdef",
					"imageID": "docker.io/auth@sha256:fedcba",
					"restartCount": 2,
					"state": {
						"running": {
							"startedAt": "2026-10-22T11:00:05Z"
						}
					}
				}
			]
		}
	}`

	scrubbed, err := FilterAndScrubResource([]byte(rawPod), "Pod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sensitive env should be [REDACTED]
	if strings.Contains(scrubbed.CleanJSON, "supersecretpassword123") {
		t.Fatalf("SECURITY VIOLATION: sensitive environment variable value was not redacted!")
	}
	if !strings.Contains(scrubbed.CleanJSON, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder in clean JSON")
	}

	// Status routine noise (podIP, containerID, imageID, startedAt) should be stripped
	if strings.Contains(scrubbed.CleanJSON, "containerd://") || strings.Contains(scrubbed.CleanJSON, "10.244.0.5") {
		t.Errorf("expected pod status routine identifiers to be stripped")
	}

	// Managed fields: fieldsV1 should be stripped
	if strings.Contains(scrubbed.CleanJSON, "fieldsV1") {
		t.Errorf("expected fieldsV1 to be stripped from managedFields")
	}
}

func TestFilterAndScrubEventsNoise(t *testing.T) {
	// Normal routine event should be ignored
	normalEvent := `{
		"apiVersion": "v1",
		"kind": "Event",
		"type": "Normal",
		"reason": "Scheduled",
		"message": "Successfully assigned default/mypod to kind-worker"
	}`

	scrubbedNorm, err := FilterAndScrubResource([]byte(normalEvent), "Event")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !scrubbedNorm.IsIgnored {
		t.Errorf("expected Normal event to be ignored")
	}

	// Warning OOMKilled event should NOT be ignored
	warningEvent := `{
		"apiVersion": "v1",
		"kind": "Event",
		"type": "Warning",
		"reason": "OOMKilled",
		"message": "Memory limit exceeded; process killed",
		"involvedObject": {
			"kind": "Pod",
			"name": "worker-1",
			"namespace": "default"
		}
	}`

	scrubbedWarn, err := FilterAndScrubResource([]byte(warningEvent), "Event")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scrubbedWarn.IsIgnored {
		t.Errorf("expected Warning OOMKilled event to be captured")
	}
}
