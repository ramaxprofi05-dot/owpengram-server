package main

import (
	"encoding/json"
	"net/http"
	"time"
)

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
	writeJSON(w, http.StatusOK, map[string]any{"rows": result})
}
