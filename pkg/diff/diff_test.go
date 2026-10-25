package diff

import (
	"strings"
	"testing"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/store"
)

func TestJSONToCleanYAML(t *testing.T) {
	jsonStr := `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"test"},"data":{"KEY":"val"}}`
	yamlStr, err := JSONToCleanYAML(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(yamlStr, "kind: ConfigMap") || !strings.Contains(yamlStr, "KEY: val") {
		t.Errorf("yaml output missing expected content:\n%s", yamlStr)
	}
}

func TestComputeDiff(t *testing.T) {
	oldRev := &store.ResourceRevision{
		ID:        1,
		Timestamp: time.Now().Add(-10 * time.Minute),
		RawJSON: `{
			"apiVersion": "apps/v1",
			"kind": "Deployment",
			"spec": {
				"replicas": 2,
				"template": {
					"spec": {
						"containers": [
							{
								"name": "web",
								"image": "nginx:1.24",
								"resources": {
									"limits": {"memory": "512Mi"}
								}
							}
						]
					}
				}
			}
		}`,
	}

	newRev := &store.ResourceRevision{
		ID:        2,
		Timestamp: time.Now(),
		RawJSON: `{
			"apiVersion": "apps/v1",
			"kind": "Deployment",
			"spec": {
				"replicas": 5,
				"template": {
					"spec": {
						"containers": [
							{
								"name": "web",
								"image": "nginx:1.25",
								"resources": {
									"limits": {"memory": "16Mi"}
								}
							}
						]
					}
				}
			}
		}`,
	}

	diffRes, err := ComputeDiff(oldRev, newRev)
	if err != nil {
		t.Fatalf("unexpected diff error: %v", err)
	}

	if len(diffRes.FieldChanges) == 0 {
		t.Fatalf("expected field changes, got 0")
	}

	summary := SummarizeChanges(diffRes.FieldChanges)
	if summary == "" {
		t.Errorf("expected non-empty diff summary")
	}

	if !strings.Contains(diffRes.UnifiedDiff, "-") || !strings.Contains(diffRes.UnifiedDiff, "+") {
		t.Errorf("expected unified diff diff markers, got:\n%s", diffRes.UnifiedDiff)
	}
}
