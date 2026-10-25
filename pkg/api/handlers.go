package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ctm/cluster-time-machine/pkg/analyzer"
	"github.com/ctm/cluster-time-machine/pkg/demo"
	"github.com/ctm/cluster-time-machine/pkg/diff"
	"github.com/ctm/cluster-time-machine/pkg/recorder"
	"github.com/ctm/cluster-time-machine/pkg/store"
)

type API struct {
	db        *store.DB
	rca       *analyzer.RCAEngine
	recorder  *recorder.Recorder
	simulator *demo.Simulator
	hub       *SSEHub
}

func NewAPI(db *store.DB, rca *analyzer.RCAEngine, rec *recorder.Recorder, sim *demo.Simulator, hub *SSEHub) *API {
	return &API{
		db:        db,
		rca:       rca,
		recorder:  rec,
		simulator: sim,
		hub:       hub,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// GET /api/status
func (a *API) HandleStatus(w http.ResponseWriter, r *http.Request) {
	stats, err := a.db.GetStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	clusterInfo := recorder.ClusterInfo{}
	if a.recorder != nil {
		clusterInfo = a.recorder.GetClusterInfo()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "online",
		"stats":             stats,
		"cluster_connected": clusterInfo.Connected,
		"cluster_info":      clusterInfo,
		"now":               time.Now().UTC(),
	})
}

// GET /api/revisions
func (a *API) HandleListRevisions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ns := q.Get("namespace")
	kind := q.Get("kind")
	actor := q.Get("actor_type")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	revs, err := a.db.ListRevisions(ns, kind, actor, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if revs == nil {
		revs = []store.ResourceRevision{}
	}
	writeJSON(w, http.StatusOK, revs)
}

// GET /api/revisions/{id}
func (a *API) HandleGetRevision(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusBadRequest, "Invalid revision ID")
		return
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid revision ID")
		return
	}

	rev, err := a.db.GetRevisionByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rev == nil {
		writeError(w, http.StatusNotFound, "Revision not found")
		return
	}

	yamlStr, _ := diff.JSONToCleanYAML(rev.RawJSON)

	writeJSON(w, http.StatusOK, map[string]any{
		"revision": rev,
		"yaml":     yamlStr,
	})
}

// GET /api/revisions/{id}/diff?compare_to={other_id}
func (a *API) HandleGetRevisionDiff(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusBadRequest, "Invalid revision ID")
		return
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid revision ID")
		return
	}

	newRev, err := a.db.GetRevisionByID(id)
	if err != nil || newRev == nil {
		writeError(w, http.StatusNotFound, "Revision not found")
		return
	}

	var oldRev *store.ResourceRevision
	compareTo := r.URL.Query().Get("compare_to")
	if compareTo != "" {
		otherID, err := strconv.ParseInt(compareTo, 10, 64)
		if err == nil {
			oldRev, _ = a.db.GetRevisionByID(otherID)
		}
	} else {
		oldRev, _ = a.db.GetPreviousRevision(newRev.ID, newRev.ResourceID)
	}

	diff, err := diff.ComputeDiff(oldRev, newRev)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to compute diff: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, diff)
}

// GET /api/manual-changes
func (a *API) HandleManualChanges(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	changes, err := a.db.GetManualChanges(limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if changes == nil {
		changes = []store.ResourceRevision{}
	}

	// Augment with diff summary
	type ManualChangeItem struct {
		store.ResourceRevision
		DiffSummary string `json:"diff_summary"`
	}

	var results []ManualChangeItem
	for _, ch := range changes {
		item := ManualChangeItem{ResourceRevision: ch}
		prev, _ := a.db.GetPreviousRevision(ch.ID, ch.ResourceID)
		revDiff, _ := diff.ComputeDiff(prev, &ch)
		if revDiff != nil {
			item.DiffSummary = diff.SummarizeChanges(revDiff.FieldChanges)
		}
		results = append(results, item)
	}

	writeJSON(w, http.StatusOK, results)
}

// GET /api/timetravel?t={RFC3339}&namespace={ns}
func (a *API) HandleTimeTravel(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tStr := q.Get("t")
	ns := q.Get("namespace")

	var targetTime time.Time
	if tStr == "" {
		targetTime = time.Now().UTC()
	} else {
		parsed, err := time.Parse(time.RFC3339Nano, tStr)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, tStr)
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid timestamp format. Use RFC3339 (e.g. 2026-10-22T17:00:00Z)")
			return
		}
		targetTime = parsed
	}

	resources, err := a.db.GetResourcesAtTime(targetTime, ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if resources == nil {
		resources = []store.TimeTravelResource{}
	}

	// Add clean YAML representations for viewing
	for i := range resources {
		y, _ := diff.JSONToCleanYAML(resources[i].RawJSON)
		resources[i].RawYAML = y
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": targetTime,
		"namespace": ns,
		"total":     len(resources),
		"resources": resources,
	})
}

// GET /api/resources/{id}/history
func (a *API) HandleResourceHistory(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusBadRequest, "Invalid resource ID")
		return
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid resource ID")
		return
	}

	history, err := a.db.GetResourceHistory(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if history == nil {
		history = []store.ResourceRevision{}
	}
	writeJSON(w, http.StatusOK, history)
}

// GET /api/namespaces
func (a *API) HandleNamespaces(w http.ResponseWriter, r *http.Request) {
	list, err := a.db.GetNamespaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []string{}
	}
	writeJSON(w, http.StatusOK, list)
}

// GET /api/incidents
func (a *API) HandleListIncidents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	list, err := a.db.ListIncidents(limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []store.Incident{}
	}
	writeJSON(w, http.StatusOK, list)
}

// GET /api/incidents/{id}
func (a *API) HandleGetIncident(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusBadRequest, "Invalid incident ID")
		return
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid incident ID")
		return
	}

	inc, err := a.db.GetIncidentByID(id)
	if err != nil || inc == nil {
		writeError(w, http.StatusNotFound, "Incident not found")
		return
	}

	writeJSON(w, http.StatusOK, inc)
}

// POST /api/incidents/{id}/rca
func (a *API) HandleIncidentRCA(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusBadRequest, "Invalid incident ID")
		return
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid incident ID")
		return
	}

	inc, err := a.db.GetIncidentByID(id)
	if err != nil || inc == nil {
		writeError(w, http.StatusNotFound, "Incident not found")
		return
	}

	lookback := 30
	if lb := r.URL.Query().Get("lookback"); lb != "" {
		if val, err := strconv.Atoi(lb); err == nil && val > 0 {
			lookback = val
		}
	}

	ranked, err := a.rca.AnalyzeIncident(inc, lookback)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = a.db.UpdateIncidentRankedChanges(inc.ID, ranked)
	inc.RankedChanges = ranked

	writeJSON(w, http.StatusOK, inc)
}

// POST /api/scenarios/run?name={scenario} or body {"scenario":"..."}
func (a *API) HandleRunScenario(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	scenario := r.URL.Query().Get("name")
	if scenario == "" {
		scenario = r.URL.Query().Get("scenario")
	}

	if scenario == "" {
		var body struct {
			Scenario string `json:"scenario"`
			Name     string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if body.Scenario != "" {
				scenario = body.Scenario
			} else if body.Name != "" {
				scenario = body.Name
			}
		}
	}

	now := time.Now().UTC()

	switch scenario {
	case "memory-oom", "oom", "memory_oom":
		a.simulator.RunScenarioMemoryOOM(now)
	case "image-pull", "image_pull":
		a.simulator.RunScenarioImagePull(now)
	case "config-crash", "config_crash", "crash-loop", "crash_loop":
		a.simulator.RunScenarioConfigCrashLoop(now)
	case "node-drain", "node_drain":
		a.simulator.RunScenarioNodeDrain(now)
	case "seed-all", "seed_all":
		_ = a.simulator.SeedDemoData()
	default:
		writeError(w, http.StatusBadRequest, "Unknown scenario name. Options: memory-oom, image-pull, config-crash, node-drain, seed-all")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "success",
		"scenario": scenario,
		"message":  "Scenario triggered successfully. Check Incidents and Live Stream.",
	})
}
