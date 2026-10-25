package recorder

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/analyzer"
	"github.com/ctm/cluster-time-machine/pkg/store"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type EventBroadcaster interface {
	Broadcast(msgType string, payload any)
}

type ClusterInfo struct {
	Connected bool   `json:"connected"`
	Host      string `json:"host"`
	InCluster bool   `json:"in_cluster"`
	Version   string `json:"version"`
}

type Recorder struct {
	db          *store.DB
	rca         *analyzer.RCAEngine
	broadcaster func(eventType string, payload any)
	clientset   *kubernetes.Clientset
	kubeconfig  string
	stopCh      chan struct{}
	mu          sync.Mutex
	lastHashes  map[string]string // resource UID -> content hash
	host        string
	inCluster   bool
}

func NewRecorder(db *store.DB, rca *analyzer.RCAEngine, kubeconfig string, broadcaster func(eventType string, payload any)) *Recorder {
	return &Recorder{
		db:          db,
		rca:         rca,
		kubeconfig:  kubeconfig,
		broadcaster: broadcaster,
		stopCh:      make(chan struct{}),
		lastHashes:  make(map[string]string),
	}
}

// Start attempts to connect to Kubernetes cluster and watch resources
func (r *Recorder) Start(ctx context.Context) error {
	config, err := r.buildKubeConfig()
	if err != nil {
		log.Printf("[Recorder] No active Kubernetes connection: %v (Time Machine running in standalone/offline mode)", err)
		return nil
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}
	r.clientset = clientset
	r.host = config.Host
	if _, err := rest.InClusterConfig(); err == nil {
		r.inCluster = true
	}

	log.Printf("[Recorder] Connected to Kubernetes cluster at %s. Starting API watchers...", config.Host)

	// Start watchers for standard resources
	go r.watchCoreResources(ctx)
	go r.watchAppsResources(ctx)
	go r.watchNetworkingResources(ctx)
	go r.watchAutoscalingResources(ctx)

	return nil
}

func (r *Recorder) GetClusterInfo() ClusterInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := ClusterInfo{
		Connected: r.clientset != nil,
		Host:      r.host,
		InCluster: r.inCluster,
	}

	if r.clientset != nil {
		if ver, err := r.clientset.Discovery().ServerVersion(); err == nil {
			info.Version = ver.GitVersion
		}
	}
	return info
}

func (r *Recorder) Stop() {
	close(r.stopCh)
}

func (r *Recorder) buildKubeConfig() (*rest.Config, error) {
	if r.kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", r.kubeconfig)
	}
	// Try in-cluster first
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	// Try default kubeconfig loading rules
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
}

// ProcessRawResource processes a raw Kubernetes JSON object, applies noise filters,
// persists revisions to SQLite, checks for incidents, and notifies listeners.
func (r *Recorder) ProcessRawResource(rawBytes []byte, kindHint string, changeType store.ChangeType, timestamp time.Time) (*store.ResourceRevision, error) {
	scrubbed, err := FilterAndScrubResource(rawBytes, kindHint)
	if err != nil {
		return nil, fmt.Errorf("failed to scrub resource: %w", err)
	}

	if scrubbed.IsIgnored {
		return nil, nil // Silently ignore leases / endpoint noise
	}

	// Content deduplication check: if hash matches previous revision, skip creating noisy revision!
	r.mu.Lock()
	uidKey := scrubbed.Namespace + "/" + scrubbed.Kind + "/" + scrubbed.Name
	if scrubbed.UID != "" {
		uidKey = scrubbed.UID
	}
	lastHash, exists := r.lastHashes[uidKey]
	if exists && lastHash == scrubbed.ContentHash && changeType == store.ChangeModified {
		r.mu.Unlock()
		return nil, nil // Spec and vital status did not change, skip duplicate
	}
	r.lastHashes[uidKey] = scrubbed.ContentHash
	r.mu.Unlock()

	// Ensure resource row in catalog
	res, err := r.db.EnsureResource(scrubbed.UID, scrubbed.Namespace, scrubbed.Kind, scrubbed.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure resource: %w", err)
	}

	// Classify Actor
	actorType, managerName := ClassifyActor(scrubbed.ManagedFields, nil)

	// Insert append-only revision
	rev, err := r.db.InsertRevision(
		res.ID,
		scrubbed.ResourceVersion,
		timestamp,
		actorType,
		managerName,
		changeType,
		scrubbed.CleanJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert revision: %w", err)
	}

	rev.UID = res.UID
	rev.Namespace = res.Namespace
	rev.Kind = res.Kind
	rev.Name = res.Name

	// Broadcast revision to live stream
	if r.broadcaster != nil {
		r.broadcaster("REVISION_ADDED", rev)
	}

	// Incident detection
	r.checkIncidents(scrubbed.Kind, scrubbed.CleanJSON)

	return rev, nil
}

func (r *Recorder) checkIncidents(kind, cleanJSON string) {
	var incidents []*analyzer.DetectedIncident

	switch kind {
	case "Pod":
		incidents = analyzer.InspectPodForIncidents(cleanJSON)
	case "Node":
		incidents = analyzer.InspectNodeForIncidents(cleanJSON)
	case "Event":
		if inc := analyzer.InspectEventForIncidents(cleanJSON); inc != nil {
			incidents = append(incidents, inc)
		}
	}

	for _, inc := range incidents {
		dbInc := &store.Incident{
			Title:        inc.Title,
			Severity:     inc.Severity,
			Namespace:    inc.Namespace,
			ResourceKind: inc.ResourceKind,
			ResourceName: inc.ResourceName,
			DetectedAt:   inc.DetectedAt,
			Description:  inc.Description,
		}

		// Run Root Cause Analysis heuristic ranking
		rankedChanges, err := r.rca.AnalyzeIncident(dbInc, 30)
		if err == nil {
			dbInc.RankedChanges = rankedChanges
		}

		savedInc, err := r.db.InsertIncident(dbInc)
		if err != nil {
			log.Printf("[Recorder] Error saving incident: %v", err)
			continue
		}

		log.Printf("[Incident Alert] %s on %s/%s (Ranked causes: %d)", savedInc.Title, savedInc.ResourceKind, savedInc.ResourceName, len(savedInc.RankedChanges))
		if r.broadcaster != nil {
			r.broadcaster("INCIDENT_ALERT", savedInc)
		}
	}
}

func (r *Recorder) watchCoreResources(ctx context.Context) {
	// Watch Pods
	r.runInformerWatcher("Pod", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.CoreV1().Pods(metav1.NamespaceAll).Watch(ctx, options)
	}, func() runtime.Object { return &v1.Pod{} })

	// Watch ConfigMaps
	r.runInformerWatcher("ConfigMap", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).Watch(ctx, options)
	}, func() runtime.Object { return &v1.ConfigMap{} })

	// Watch Secrets
	r.runInformerWatcher("Secret", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.CoreV1().Secrets(metav1.NamespaceAll).Watch(ctx, options)
	}, func() runtime.Object { return &v1.Secret{} })

	// Watch Services
	r.runInformerWatcher("Service", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.CoreV1().Services(metav1.NamespaceAll).Watch(ctx, options)
	}, func() runtime.Object { return &v1.Service{} })

	// Watch Nodes
	r.runInformerWatcher("Node", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.CoreV1().Nodes().Watch(ctx, options)
	}, func() runtime.Object { return &v1.Node{} })

	// Watch PersistentVolumeClaims
	r.runInformerWatcher("PersistentVolumeClaim", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).Watch(ctx, options)
	}, func() runtime.Object { return &v1.PersistentVolumeClaim{} })

	// Watch Events (Warning and Error)
	r.runInformerWatcher("Event", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.CoreV1().Events(metav1.NamespaceAll).Watch(ctx, options)
	}, func() runtime.Object { return &v1.Event{} })
}

func (r *Recorder) watchAppsResources(ctx context.Context) {
	// Watch Deployments
	r.runInformerWatcher("Deployment", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.AppsV1().Deployments(metav1.NamespaceAll).Watch(ctx, options)
	}, nil)

	// Watch StatefulSets
	r.runInformerWatcher("StatefulSet", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.AppsV1().StatefulSets(metav1.NamespaceAll).Watch(ctx, options)
	}, nil)

	// Watch DaemonSets
	r.runInformerWatcher("DaemonSet", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.AppsV1().DaemonSets(metav1.NamespaceAll).Watch(ctx, options)
	}, nil)

	// Watch ReplicaSets
	r.runInformerWatcher("ReplicaSet", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.AppsV1().ReplicaSets(metav1.NamespaceAll).Watch(ctx, options)
	}, nil)
}

func (r *Recorder) watchNetworkingResources(ctx context.Context) {
	r.runInformerWatcher("Ingress", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.NetworkingV1().Ingresses(metav1.NamespaceAll).Watch(ctx, options)
	}, nil)
}

func (r *Recorder) watchAutoscalingResources(ctx context.Context) {
	r.runInformerWatcher("HorizontalPodAutoscaler", func(options metav1.ListOptions) (watch.Interface, error) {
		return r.clientset.AutoscalingV2().HorizontalPodAutoscalers(metav1.NamespaceAll).Watch(ctx, options)
	}, nil)
}

func (r *Recorder) runInformerWatcher(
	kind string,
	watchFunc func(options metav1.ListOptions) (watch.Interface, error),
	objType func() runtime.Object,
) {
	lw := &cache.ListWatch{
		WatchFunc: watchFunc,
	}

	_, controller := cache.NewInformer(
		lw,
		nil,
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) {
				rawBytes, _ := json.Marshal(obj)
				_, _ = r.ProcessRawResource(rawBytes, kind, store.ChangeAdded, time.Now().UTC())
			},
			UpdateFunc: func(oldObj, newObj any) {
				rawBytes, _ := json.Marshal(newObj)
				_, _ = r.ProcessRawResource(rawBytes, kind, store.ChangeModified, time.Now().UTC())
			},
			DeleteFunc: func(obj any) {
				rawBytes, _ := json.Marshal(obj)
				_, _ = r.ProcessRawResource(rawBytes, kind, store.ChangeDeleted, time.Now().UTC())
			},
		},
	)

	go controller.Run(r.stopCh)
}
