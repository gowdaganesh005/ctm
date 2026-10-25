package demo

import (
	"encoding/json"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/recorder"
	"github.com/ctm/cluster-time-machine/pkg/store"
)

type Simulator struct {
	recorder *recorder.Recorder
	db       *store.DB
}

func NewSimulator(rec *recorder.Recorder, db *store.DB) *Simulator {
	return &Simulator{
		recorder: rec,
		db:       db,
	}
}

// SeedDemoData populates realistic cluster history and incidents if database is empty
func (s *Simulator) SeedDemoData() error {
	stats, err := s.db.GetStats()
	if err == nil && stats.TotalRevisions > 0 {
		return nil // Already populated
	}

	now := time.Now().UTC()

	// 1. Initial GitOps baseline 45 minutes ago
	t0 := now.Add(-45 * time.Minute)
	s.seedBaseline(t0)

	// 2. Run Scenario 1: Manual Memory Limit Squeeze -> OOMKilled (20 minutes ago)
	s.RunScenarioMemoryOOM(now.Add(-20 * time.Minute))

	// 3. Run Scenario 2: Container Image Typo -> ImagePullBackOff (10 minutes ago)
	s.RunScenarioImagePull(now.Add(-10 * time.Minute))

	// 4. Run Scenario 3: Corrupt ConfigMap -> CrashLoopBackOff (5 minutes ago)
	s.RunScenarioConfigCrashLoop(now.Add(-5 * time.Minute))

	return nil
}

func (s *Simulator) seedBaseline(t time.Time) {
	// Baseline deployment: payment-processor
	s.processRaw(map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"uid":       "uid-deploy-payment-01",
			"name":      "payment-processor",
			"namespace": "production",
			"managedFields": []any{
				map[string]any{
					"manager":    "argocd-application-controller",
					"operation":  "Apply",
					"apiVersion": "apps/v1",
					"time":       t.Format(time.RFC3339),
				},
			},
		},
		"spec": map[string]any{
			"replicas": 2,
			"selector": map[string]any{
				"matchLabels": map[string]any{"app": "payment-processor"},
			},
			"template": map[string]any{
				"metadata": map[string]any{"labels": map[string]any{"app": "payment-processor"}},
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "payment-processor",
							"image": "redis:7.2-alpine",
							"resources": map[string]any{
								"limits":   map[string]any{"memory": "512Mi", "cpu": "500m"},
								"requests": map[string]any{"memory": "256Mi", "cpu": "250m"},
							},
							"env": []any{
								map[string]any{"name": "PORT", "value": "8080"},
								map[string]any{"name": "DATABASE_PASSWORD", "value": "supersecretpassword123"},
							},
						},
					},
				},
			},
		},
	}, "Deployment", store.ChangeAdded, t)

	// Baseline Secret: payment-secret
	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]any{
			"uid":       "uid-secret-payment-01",
			"name":      "payment-secret",
			"namespace": "production",
			"managedFields": []any{
				map[string]any{"manager": "argocd-application-controller", "operation": "Apply", "apiVersion": "v1", "time": t.Format(time.RFC3339)},
			},
		},
		"type": "Opaque",
		"data": map[string]any{
			"stripe_api_key": "c2tfbGl2ZV85OTk4ODg3NzY2NTU0NA==",
		},
	}, "Secret", store.ChangeAdded, t)

	// Baseline Nodes
	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata": map[string]any{
			"uid":  "uid-node-01",
			"name": "kind-worker",
			"managedFields": []any{
				map[string]any{"manager": "kubelet", "operation": "Update", "apiVersion": "v1", "time": t.Format(time.RFC3339)},
			},
		},
		"status": map[string]any{
			"conditions": []any{
				map[string]any{"type": "Ready", "status": "True", "reason": "KubeletReady"},
			},
		},
	}, "Node", store.ChangeAdded, t)

	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata": map[string]any{
			"uid":  "uid-node-02",
			"name": "kind-worker2",
			"managedFields": []any{
				map[string]any{"manager": "kubelet", "operation": "Update", "apiVersion": "v1", "time": t.Format(time.RFC3339)},
			},
		},
		"status": map[string]any{
			"conditions": []any{
				map[string]any{"type": "Ready", "status": "True", "reason": "KubeletReady"},
			},
		},
	}, "Node", store.ChangeAdded, t)
}

// Scenario 1: Manual Memory Limit Squeeze -> OOMKilled
func (s *Simulator) RunScenarioMemoryOOM(baseTime time.Time) {
	// Step 1: Human runs kubectl edit deployment payment-processor and slashes memory limit to 16Mi
	tEdit := baseTime
	s.processRaw(map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"uid":       "uid-deploy-payment-01",
			"name":      "payment-processor",
			"namespace": "production",
			"managedFields": []any{
				map[string]any{
					"manager":    "kubectl-edit",
					"operation":  "Update",
					"apiVersion": "apps/v1",
					"time":       tEdit.Format(time.RFC3339),
				},
			},
		},
		"spec": map[string]any{
			"replicas": 2,
			"selector": map[string]any{
				"matchLabels": map[string]any{"app": "payment-processor"},
			},
			"template": map[string]any{
				"metadata": map[string]any{"labels": map[string]any{"app": "payment-processor"}},
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "payment-processor",
							"image": "redis:7.2-alpine",
							"resources": map[string]any{
								"limits":   map[string]any{"memory": "16Mi", "cpu": "500m"},
								"requests": map[string]any{"memory": "16Mi", "cpu": "100m"},
							},
						},
					},
				},
			},
		},
	}, "Deployment", store.ChangeModified, tEdit)

	// Step 2: Pod starts, exceeds 16Mi limit, and is OOMKilled 2 minutes later
	tCrash := baseTime.Add(2 * time.Minute)
	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"uid":       "uid-pod-payment-oom-99",
			"name":      "payment-processor-78d49f6cb-x49zl",
			"namespace": "production",
			"managedFields": []any{
				map[string]any{"manager": "kubelet", "operation": "Update", "apiVersion": "v1", "time": tCrash.Format(time.RFC3339)},
			},
		},
		"status": map[string]any{
			"phase": "Running",
			"containerStatuses": []any{
				map[string]any{
					"name":         "payment-processor",
					"restartCount": 3,
					"state": map[string]any{
						"terminated": map[string]any{
							"exitCode": 137,
							"reason":   "OOMKilled",
							"message":  "The container exceeded its memory limit (16Mi) and was terminated by the OOM killer.",
						},
					},
				},
			},
		},
	}, "Pod", store.ChangeModified, tCrash)
}

// Scenario 2: Container Image Typo -> ImagePullBackOff
func (s *Simulator) RunScenarioImagePull(baseTime time.Time) {
	tEdit := baseTime
	s.processRaw(map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"uid":       "uid-deploy-order-01",
			"name":      "order-service",
			"namespace": "staging",
			"managedFields": []any{
				map[string]any{
					"manager":    "kubectl-set",
					"operation":  "Update",
					"apiVersion": "apps/v1",
					"time":       tEdit.Format(time.RFC3339),
				},
			},
		},
		"spec": map[string]any{
			"replicas": 1,
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "order-service",
							"image": "redis:v999.invalid-tag",
						},
					},
				},
			},
		},
	}, "Deployment", store.ChangeAdded, tEdit)

	tBackOff := baseTime.Add(1 * time.Minute)
	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"uid":       "uid-pod-order-err-12",
			"name":      "order-service-67fb99-m7qpr",
			"namespace": "staging",
			"managedFields": []any{
				map[string]any{"manager": "kubelet", "operation": "Update", "apiVersion": "v1", "time": tBackOff.Format(time.RFC3339)},
			},
		},
		"status": map[string]any{
			"phase": "Pending",
			"containerStatuses": []any{
				map[string]any{
					"name":         "order-service",
					"restartCount": 0,
					"state": map[string]any{
						"waiting": map[string]any{
							"reason":  "ImagePullBackOff",
							"message": "Back-off pulling image 'redis:v999.invalid-tag': manifest unknown",
						},
					},
				},
			},
		},
	}, "Pod", store.ChangeAdded, tBackOff)
}

// Scenario 3: Corrupt ConfigMap -> CrashLoopBackOff
func (s *Simulator) RunScenarioConfigCrashLoop(baseTime time.Time) {
	tEdit := baseTime
	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"uid":       "uid-cm-auth-01",
			"name":      "auth-config",
			"namespace": "production",
			"managedFields": []any{
				map[string]any{
					"manager":    "kubectl-edit",
					"operation":  "Update",
					"apiVersion": "v1",
					"time":       tEdit.Format(time.RFC3339),
				},
			},
		},
		"data": map[string]any{
			"AUTH_DEBUG": "true",
			// DATABASE_URL was mistakenly deleted!
		},
	}, "ConfigMap", store.ChangeAdded, tEdit)

	tCrash := baseTime.Add(1 * time.Minute)
	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"uid":       "uid-pod-auth-crash-01",
			"name":      "auth-service-54d9f78cc-wq2pn",
			"namespace": "production",
			"managedFields": []any{
				map[string]any{"manager": "kubelet", "operation": "Update", "apiVersion": "v1", "time": tCrash.Format(time.RFC3339)},
			},
		},
		"status": map[string]any{
			"phase": "Running",
			"containerStatuses": []any{
				map[string]any{
					"name":         "auth-service",
					"restartCount": 4,
					"state": map[string]any{
						"waiting": map[string]any{
							"reason":  "CrashLoopBackOff",
							"message": "back-off 2m40s restarting failed container: missing required environment variable DATABASE_URL",
						},
					},
				},
			},
		},
	}, "Pod", store.ChangeAdded, tCrash)
}

// Scenario 4: Node Drained -> FailedScheduling
func (s *Simulator) RunScenarioNodeDrain(baseTime time.Time) {
	tCordon := baseTime
	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata": map[string]any{
			"uid":  "uid-node-01",
			"name": "kind-worker",
			"managedFields": []any{
				map[string]any{
					"manager":    "kubectl-drain",
					"operation":  "Update",
					"apiVersion": "v1",
					"time":       tCordon.Format(time.RFC3339),
				},
			},
		},
		"spec": map[string]any{
			"unschedulable": true,
		},
		"status": map[string]any{
			"conditions": []any{
				map[string]any{"type": "Ready", "status": "True", "reason": "KubeletReady"},
			},
		},
	}, "Node", store.ChangeModified, tCordon)

	tSchedule := baseTime.Add(1 * time.Minute)
	s.processRaw(map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"uid":       "uid-pod-batch-fail-01",
			"name":      "batch-analytics-worker-0",
			"namespace": "default",
			"managedFields": []any{
				map[string]any{"manager": "kube-scheduler", "operation": "Update", "apiVersion": "v1", "time": tSchedule.Format(time.RFC3339)},
			},
		},
		"status": map[string]any{
			"phase": "Pending",
			"conditions": []any{
				map[string]any{
					"type":    "PodScheduled",
					"status":  "False",
					"reason":  "Unschedulable",
					"message": "0/3 nodes are available: 1 node(s) were unschedulable, 2 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }",
				},
			},
		},
	}, "Pod", store.ChangeAdded, tSchedule)
}

func (s *Simulator) processRaw(obj map[string]any, kindHint string, changeType store.ChangeType, t time.Time) {
	b, err := json.Marshal(obj)
	if err != nil {
		return
	}
	_, _ = s.recorder.ProcessRawResource(b, kindHint, changeType, t)
}
