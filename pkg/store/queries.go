package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// EnsureResource inserts or returns an existing resource by UID.
// If UID already exists, it updates namespace/kind/name in case of subtle rename.
func (db *DB) EnsureResource(uid, namespace, kind, name string) (*Resource, error) {
	if uid == "" {
		uid = fmt.Sprintf("synth-%s-%s-%s", namespace, kind, name)
	}

	var res Resource
	var createdAtStr string
	err := db.conn.QueryRow(
		`SELECT id, uid, namespace, kind, name, created_at FROM resources WHERE uid = ?`,
		uid,
	).Scan(&res.ID, &res.UID, &res.Namespace, &res.Kind, &res.Name, &createdAtStr)

	if err == nil {
		res.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if res.CreatedAt.IsZero() {
			res.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		return &res, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("error querying resource by uid %s: %w", uid, err)
	}

	// Insert new resource
	resCreatedAt := time.Now().UTC()
	result, err := db.conn.Exec(
		`INSERT INTO resources (uid, namespace, kind, name, created_at) VALUES (?, ?, ?, ?, ?)`,
		uid, namespace, kind, name, resCreatedAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		// In case of race condition
		err2 := db.conn.QueryRow(
			`SELECT id, uid, namespace, kind, name, created_at FROM resources WHERE uid = ?`,
			uid,
		).Scan(&res.ID, &res.UID, &res.Namespace, &res.Kind, &res.Name, &createdAtStr)
		if err2 == nil {
			return &res, nil
		}
		return nil, fmt.Errorf("failed to insert resource: %w", err)
	}

	resID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	res = Resource{
		ID:        resID,
		UID:       uid,
		Namespace: namespace,
		Kind:      kind,
		Name:      name,
		CreatedAt: resCreatedAt,
	}
	return &res, nil
}

// InsertRevision inserts an append-only revision for a resource.
func (db *DB) InsertRevision(
	resourceID int64,
	resourceVersion string,
	timestamp time.Time,
	actorType ActorType,
	managerName string,
	changeType ChangeType,
	rawJSON string,
) (*ResourceRevision, error) {
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	res, err := db.conn.Exec(
		`INSERT INTO resource_revisions 
		 (resource_id, resource_version, timestamp, actor_type, manager_name, change_type, raw_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		resourceID, resourceVersion, timestamp.Format(time.RFC3339Nano), string(actorType), managerName, string(changeType), rawJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert resource revision: %w", err)
	}

	revID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ResourceRevision{
		ID:              revID,
		ResourceID:      resourceID,
		ResourceVersion: resourceVersion,
		Timestamp:       timestamp,
		ActorType:       actorType,
		ManagerName:     managerName,
		ChangeType:      changeType,
		RawJSON:         rawJSON,
	}, nil
}

// GetLatestRevision returns the most recent revision for a given resource.
func (db *DB) GetLatestRevision(resourceID int64) (*ResourceRevision, error) {
	row := db.conn.QueryRow(
		`SELECT id, resource_id, resource_version, timestamp, actor_type, manager_name, change_type, raw_json
		 FROM resource_revisions
		 WHERE resource_id = ?
		 ORDER BY timestamp DESC, id DESC
		 LIMIT 1`,
		resourceID,
	)

	var rev ResourceRevision
	var tsStr, actorStr, changeStr string
	err := row.Scan(
		&rev.ID, &rev.ResourceID, &rev.ResourceVersion, &tsStr, &actorStr, &rev.ManagerName, &changeStr, &rev.RawJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rev.Timestamp = parseDBTime(tsStr)
	rev.ActorType = ActorType(actorStr)
	rev.ChangeType = ChangeType(changeStr)
	return &rev, nil
}

// GetRevisionByID retrieves a revision along with resource details.
func (db *DB) GetRevisionByID(revID int64) (*ResourceRevision, error) {
	row := db.conn.QueryRow(
		`SELECT r.id, r.resource_id, r.resource_version, r.timestamp, r.actor_type, r.manager_name, r.change_type, r.raw_json,
		        res.uid, res.namespace, res.kind, res.name
		 FROM resource_revisions r
		 JOIN resources res ON r.resource_id = res.id
		 WHERE r.id = ?`,
		revID,
	)

	var rev ResourceRevision
	var tsStr, actorStr, changeStr string
	err := row.Scan(
		&rev.ID, &rev.ResourceID, &rev.ResourceVersion, &tsStr, &actorStr, &rev.ManagerName, &changeStr, &rev.RawJSON,
		&rev.UID, &rev.Namespace, &rev.Kind, &rev.Name,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rev.Timestamp = parseDBTime(tsStr)
	rev.ActorType = ActorType(actorStr)
	rev.ChangeType = ChangeType(changeStr)
	return &rev, nil
}

// GetPreviousRevision returns the revision immediately preceding the specified revision for the same resource.
func (db *DB) GetPreviousRevision(currentRevID int64, resourceID int64) (*ResourceRevision, error) {
	row := db.conn.QueryRow(
		`SELECT r.id, r.resource_id, r.resource_version, r.timestamp, r.actor_type, r.manager_name, r.change_type, r.raw_json,
		        res.uid, res.namespace, res.kind, res.name
		 FROM resource_revisions r
		 JOIN resources res ON r.resource_id = res.id
		 WHERE r.resource_id = ? AND r.id < ?
		 ORDER BY r.id DESC
		 LIMIT 1`,
		resourceID, currentRevID,
	)

	var rev ResourceRevision
	var tsStr, actorStr, changeStr string
	err := row.Scan(
		&rev.ID, &rev.ResourceID, &rev.ResourceVersion, &tsStr, &actorStr, &rev.ManagerName, &changeStr, &rev.RawJSON,
		&rev.UID, &rev.Namespace, &rev.Kind, &rev.Name,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rev.Timestamp = parseDBTime(tsStr)
	rev.ActorType = ActorType(actorStr)
	rev.ChangeType = ChangeType(changeStr)
	return &rev, nil
}

// ListRevisions returns recent revisions matching optional filters.
func (db *DB) ListRevisions(namespace, kind, actorType string, limit, offset int) ([]ResourceRevision, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT r.id, r.resource_id, r.resource_version, r.timestamp, r.actor_type, r.manager_name, r.change_type, r.raw_json,
	                 res.uid, res.namespace, res.kind, res.name
	          FROM resource_revisions r
	          JOIN resources res ON r.resource_id = res.id
	          WHERE 1=1`
	var args []any

	if namespace != "" {
		query += " AND res.namespace = ?"
		args = append(args, namespace)
	}
	if kind != "" {
		query += " AND res.kind = ?"
		args = append(args, kind)
	}
	if actorType != "" {
		query += " AND r.actor_type = ?"
		args = append(args, actorType)
	}

	query += " ORDER BY r.timestamp DESC, r.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []ResourceRevision
	for rows.Next() {
		var rev ResourceRevision
		var tsStr, aStr, cStr string
		err := rows.Scan(
			&rev.ID, &rev.ResourceID, &rev.ResourceVersion, &tsStr, &aStr, &rev.ManagerName, &cStr, &rev.RawJSON,
			&rev.UID, &rev.Namespace, &rev.Kind, &rev.Name,
		)
		if err != nil {
			return nil, err
		}
		rev.Timestamp = parseDBTime(tsStr)
		rev.ActorType = ActorType(aStr)
		rev.ChangeType = ChangeType(cStr)
		revisions = append(revisions, rev)
	}

	return revisions, nil
}

// GetManualChanges returns revisions where actor_type = 'Human'.
func (db *DB) GetManualChanges(limit, offset int) ([]ResourceRevision, error) {
	return db.ListRevisions("", "", string(ActorHuman), limit, offset)
}

// GetResourcesAtTime reconstructs the state of all active resources at timestamp T.
// Returns latest revision <= T where change_type != 'DELETED'.
func (db *DB) GetResourcesAtTime(t time.Time, namespace string) ([]TimeTravelResource, error) {
	tStr := t.UTC().Format(time.RFC3339Nano)

	query := `
		SELECT res.id, res.uid, res.namespace, res.kind, res.name,
		       r.id, r.resource_version, r.timestamp, r.actor_type, r.manager_name, r.change_type, r.raw_json
		FROM resources res
		JOIN resource_revisions r ON r.id = (
			SELECT rr.id
			FROM resource_revisions rr
			WHERE rr.resource_id = res.id AND rr.timestamp <= ?
			ORDER BY rr.timestamp DESC, rr.id DESC
			LIMIT 1
		)
		WHERE r.change_type != 'DELETED'`
	var args = []any{tStr}

	if namespace != "" {
		query += " AND res.namespace = ?"
		args = append(args, namespace)
	}

	query += " ORDER BY res.namespace, res.kind, res.name"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query time travel resources: %w", err)
	}
	defer rows.Close()

	var results []TimeTravelResource
	for rows.Next() {
		var item TimeTravelResource
		var tsStr, actorStr, changeType string
		err := rows.Scan(
			&item.ResourceID, &item.UID, &item.Namespace, &item.Kind, &item.Name,
			&item.RevisionID, &item.ResourceVersion, &tsStr, &actorStr, &item.ManagerName, &changeType, &item.RawJSON,
		)
		if err != nil {
			return nil, err
		}
		item.Timestamp = parseDBTime(tsStr)
		item.ActorType = ActorType(actorStr)
		results = append(results, item)
	}

	return results, nil
}

// GetResourceHistory returns all revisions of a specific resource.
func (db *DB) GetResourceHistory(resourceID int64) ([]ResourceRevision, error) {
	rows, err := db.conn.Query(
		`SELECT r.id, r.resource_id, r.resource_version, r.timestamp, r.actor_type, r.manager_name, r.change_type, r.raw_json,
		        res.uid, res.namespace, res.kind, res.name
		 FROM resource_revisions r
		 JOIN resources res ON r.resource_id = res.id
		 WHERE r.resource_id = ?
		 ORDER BY r.timestamp ASC, r.id ASC`,
		resourceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []ResourceRevision
	for rows.Next() {
		var rev ResourceRevision
		var tsStr, aStr, cStr string
		if err := rows.Scan(
			&rev.ID, &rev.ResourceID, &rev.ResourceVersion, &tsStr, &aStr, &rev.ManagerName, &cStr, &rev.RawJSON,
			&rev.UID, &rev.Namespace, &rev.Kind, &rev.Name,
		); err != nil {
			return nil, err
		}
		rev.Timestamp = parseDBTime(tsStr)
		rev.ActorType = ActorType(aStr)
		rev.ChangeType = ChangeType(cStr)
		revisions = append(revisions, rev)
	}

	return revisions, nil
}

// GetRevisionsInWindow queries revisions between startTime and endTime (e.g. for RCA).
func (db *DB) GetRevisionsInWindow(startTime, endTime time.Time, namespace string) ([]ResourceRevision, error) {
	startStr := startTime.UTC().Format(time.RFC3339Nano)
	endStr := endTime.UTC().Format(time.RFC3339Nano)

	query := `SELECT r.id, r.resource_id, r.resource_version, r.timestamp, r.actor_type, r.manager_name, r.change_type, r.raw_json,
	                 res.uid, res.namespace, res.kind, res.name
	          FROM resource_revisions r
	          JOIN resources res ON r.resource_id = res.id
	          WHERE r.timestamp >= ? AND r.timestamp <= ?`
	args := []any{startStr, endStr}

	if namespace != "" {
		// Include matching namespace OR cluster-scoped (empty namespace)
		query += " AND (res.namespace = ? OR res.namespace = '')"
		args = append(args, namespace)
	}

	query += " ORDER BY r.timestamp DESC, r.id DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []ResourceRevision
	for rows.Next() {
		var rev ResourceRevision
		var tsStr, aStr, cStr string
		if err := rows.Scan(
			&rev.ID, &rev.ResourceID, &rev.ResourceVersion, &tsStr, &aStr, &rev.ManagerName, &cStr, &rev.RawJSON,
			&rev.UID, &rev.Namespace, &rev.Kind, &rev.Name,
		); err != nil {
			return nil, err
		}
		rev.Timestamp = parseDBTime(tsStr)
		rev.ActorType = ActorType(aStr)
		rev.ChangeType = ChangeType(cStr)
		revisions = append(revisions, rev)
	}

	return revisions, nil
}

// InsertIncident inserts a detected cluster incident.
func (db *DB) InsertIncident(inc *Incident) (*Incident, error) {
	if inc.DetectedAt.IsZero() {
		inc.DetectedAt = time.Now().UTC()
	}

	if inc.RankedChangesJSON == "" && len(inc.RankedChanges) > 0 {
		b, _ := json.Marshal(inc.RankedChanges)
		inc.RankedChangesJSON = string(b)
	}

	res, err := db.conn.Exec(
		`INSERT INTO incidents (title, severity, namespace, resource_kind, resource_name, detected_at, description, ranked_changes_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		inc.Title, inc.Severity, inc.Namespace, inc.ResourceKind, inc.ResourceName,
		inc.DetectedAt.Format(time.RFC3339Nano), inc.Description, inc.RankedChangesJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert incident: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	inc.ID = id
	return inc, nil
}

// UpdateIncidentRankedChanges updates the ranked root causes for an incident.
func (db *DB) UpdateIncidentRankedChanges(incidentID int64, rankedChanges []RankedChange) error {
	b, err := json.Marshal(rankedChanges)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(`UPDATE incidents SET ranked_changes_json = ? WHERE id = ?`, string(b), incidentID)
	return err
}

// ListIncidents returns recorded cluster incidents.
func (db *DB) ListIncidents(limit, offset int) ([]Incident, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.conn.Query(
		`SELECT id, title, severity, namespace, resource_kind, resource_name, detected_at, description, ranked_changes_json
		 FROM incidents
		 ORDER BY detected_at DESC, id DESC
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var inc Incident
		var dtStr string
		var rjson sql.NullString
		var desc sql.NullString
		if err := rows.Scan(
			&inc.ID, &inc.Title, &inc.Severity, &inc.Namespace, &inc.ResourceKind, &inc.ResourceName,
			&dtStr, &desc, &rjson,
		); err != nil {
			return nil, err
		}
		inc.DetectedAt = parseDBTime(dtStr)
		if desc.Valid {
			inc.Description = desc.String
		}
		if rjson.Valid && rjson.String != "" {
			inc.RankedChangesJSON = rjson.String
			_ = json.Unmarshal([]byte(rjson.String), &inc.RankedChanges)
		}
		incidents = append(incidents, inc)
	}

	return incidents, nil
}

// GetIncidentByID returns a single incident.
func (db *DB) GetIncidentByID(id int64) (*Incident, error) {
	row := db.conn.QueryRow(
		`SELECT id, title, severity, namespace, resource_kind, resource_name, detected_at, description, ranked_changes_json
		 FROM incidents
		 WHERE id = ?`,
		id,
	)

	var inc Incident
	var dtStr string
	var rjson sql.NullString
	var desc sql.NullString
	err := row.Scan(
		&inc.ID, &inc.Title, &inc.Severity, &inc.Namespace, &inc.ResourceKind, &inc.ResourceName,
		&dtStr, &desc, &rjson,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	inc.DetectedAt = parseDBTime(dtStr)
	if desc.Valid {
		inc.Description = desc.String
	}
	if rjson.Valid && rjson.String != "" {
		inc.RankedChangesJSON = rjson.String
		_ = json.Unmarshal([]byte(rjson.String), &inc.RankedChanges)
	}
	return &inc, nil
}

// GetNamespaces returns all unique namespaces observed.
func (db *DB) GetNamespaces() ([]string, error) {
	rows, err := db.conn.Query(`SELECT DISTINCT namespace FROM resources WHERE namespace != '' ORDER BY namespace`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nsList []string
	for rows.Next() {
		var ns string
		if err := rows.Scan(&ns); err == nil {
			nsList = append(nsList, ns)
		}
	}
	return nsList, nil
}

// GetStats returns summary metrics.
func (db *DB) GetStats() (*ClusterStats, error) {
	stats := &ClusterStats{}

	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM resources`).Scan(&stats.TotalResources)
	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM resource_revisions`).Scan(&stats.TotalRevisions)
	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM resource_revisions WHERE actor_type = 'Human'`).Scan(&stats.HumanChanges)
	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM resource_revisions WHERE actor_type = 'GitOps'`).Scan(&stats.GitOpsChanges)
	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM resource_revisions WHERE actor_type = 'Controller'`).Scan(&stats.CtrlChanges)
	_ = db.conn.QueryRow(`SELECT COUNT(*) FROM incidents`).Scan(&stats.IncidentsCount)

	var earliestStr, latestStr sql.NullString
	_ = db.conn.QueryRow(`SELECT MIN(timestamp), MAX(timestamp) FROM resource_revisions`).Scan(&earliestStr, &latestStr)

	if earliestStr.Valid && earliestStr.String != "" {
		t := parseDBTime(earliestStr.String)
		stats.EarliestTime = &t
	}
	if latestStr.Valid && latestStr.String != "" {
		t := parseDBTime(latestStr.String)
		stats.LatestTime = &t
	}

	return stats, nil
}

func parseDBTime(val string) time.Time {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, val); err == nil {
			return t
		}
	}
	return time.Time{}
}
