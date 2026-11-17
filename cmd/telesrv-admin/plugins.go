package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type pluginPublisherRow struct {
	KeyID     string     `json:"key_id"`
	Label     string     `json:"label"`
	Status    string     `json:"status"`
	CreatedBy string     `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedBy string     `json:"revoked_by"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type pluginCatalogEventRow struct {
	ID        int64     `json:"id"`
	PluginID  string    `json:"plugin_id"`
	Version   string    `json:"version"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type pluginCatalogRow struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Author      string          `json:"author"`
	Description string          `json:"description"`
	Target      string          `json:"target"`
	Entrypoint  string          `json:"entrypoint"`
	Permissions json.RawMessage `json:"permissions"`
	SHA256      string          `json:"sha256"`
	SignerKeyID string          `json:"signer_key_id"`
	Status      string          `json:"status"`
	SubmittedBy string          `json:"submitted_by"`
	ReviewedBy  string          `json:"reviewed_by"`
	ReviewNote  string          `json:"review_note"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ReviewedAt  *time.Time      `json:"reviewed_at,omitempty"`
}

func (s *server) handlePluginsAPI(w http.ResponseWriter, r *http.Request) {
	if s.read == nil || s.read.pool == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "plugin catalog database is unavailable")
		return
	}
	rows, err := s.read.pool.Query(r.Context(), `
SELECT plugin_id, name, version, author, description, target, entrypoint,
       permissions, package_sha256, signer_key_id, status, submitted_by,
       reviewed_by, review_note, created_at, updated_at, reviewed_at
FROM public.plugin_catalog
ORDER BY updated_at DESC, plugin_id ASC
LIMIT 500`)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "could not read plugin catalog")
		return
	}
	defer rows.Close()
	result := make([]pluginCatalogRow, 0)
	for rows.Next() {
		var row pluginCatalogRow
		if err := rows.Scan(
			&row.ID, &row.Name, &row.Version, &row.Author, &row.Description,
			&row.Target, &row.Entrypoint, &row.Permissions, &row.SHA256,
			&row.SignerKeyID, &row.Status, &row.SubmittedBy, &row.ReviewedBy,
			&row.ReviewNote, &row.CreatedAt, &row.UpdatedAt, &row.ReviewedAt,
		); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "could not read plugin catalog row")
			return
		}
		if len(row.Permissions) == 0 {
			row.Permissions = json.RawMessage("[]")
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "could not finish reading plugin catalog")
		return
	}
	publisherRows, err := s.read.pool.Query(r.Context(), `
SELECT signer_key_id, label, status, created_by, created_at, revoked_by, revoked_at
FROM public.plugin_publishers
ORDER BY created_at DESC, signer_key_id ASC
LIMIT 500`)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "could not read plugin publishers")
		return
	}
	defer publisherRows.Close()
	publishers := make([]pluginPublisherRow, 0)
	for publisherRows.Next() {
		var row pluginPublisherRow
		if err := publisherRows.Scan(&row.KeyID, &row.Label, &row.Status, &row.CreatedBy, &row.CreatedAt, &row.RevokedBy, &row.RevokedAt); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "could not read plugin publisher row")
			return
		}
		publishers = append(publishers, row)
	}
	if err := publisherRows.Err(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "could not finish reading plugin publishers")
		return
	}

	eventRows, err := s.read.pool.Query(r.Context(), `
SELECT id, plugin_id, version, action, actor, note, created_at
FROM public.plugin_catalog_events
ORDER BY created_at DESC, id DESC
LIMIT 200`)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "could not read plugin review history")
		return
	}
	defer eventRows.Close()
	events := make([]pluginCatalogEventRow, 0)
	for eventRows.Next() {
		var row pluginCatalogEventRow
		if err := eventRows.Scan(&row.ID, &row.PluginID, &row.Version, &row.Action, &row.Actor, &row.Note, &row.CreatedAt); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "could not read plugin review event")
			return
		}
		events = append(events, row)
	}
	if err := eventRows.Err(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "could not finish reading plugin review history")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": result, "publishers": publishers, "events": events})
}
