package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"telesrv/internal/domain"
)

var (
	ErrProfileVideoBannerDocumentNotFound = errors.New("profile video banner document not found")
	ErrProfileVideoBannerNotOwned          = errors.New("profile video banner document is not owned by profile user")
	ErrProfileVideoBannerInvalidMedia      = errors.New("document is not a supported profile video banner")
)

// SetProfileVideoBanner atomically selects an already-uploaded document for a
// user's profile. The database trigger independently enforces ownership, so
// callers cannot bypass the check by using another write path.
func (s *MediaStore) SetProfileVideoBanner(ctx context.Context, ownerUserID, documentID int64) (domain.ProfileVideoBanner, error) {
	if ownerUserID <= 0 || documentID <= 0 {
		return domain.ProfileVideoBanner{}, fmt.Errorf("set profile video banner: invalid owner or document")
	}
	doc, found, err := s.GetDocument(ctx, documentID)
	if err != nil {
		return domain.ProfileVideoBanner{}, fmt.Errorf("load profile video banner document: %w", err)
	}
	if !found {
		return domain.ProfileVideoBanner{}, ErrProfileVideoBannerDocumentNotFound
	}
	if doc.OwnerUserID != ownerUserID {
		return domain.ProfileVideoBanner{}, ErrProfileVideoBannerNotOwned
	}
	if !doc.IsProfileVideoBanner() {
		return domain.ProfileVideoBanner{}, ErrProfileVideoBannerInvalidMedia
	}

	var createdAt, updatedAt time.Time
	err = s.db.QueryRow(ctx, `
INSERT INTO public.user_profile_video_banners(owner_user_id, document_id)
VALUES ($1, $2)
ON CONFLICT (owner_user_id) DO UPDATE
SET document_id = EXCLUDED.document_id
RETURNING created_at, updated_at`, ownerUserID, documentID).Scan(&createdAt, &updatedAt)
	if err != nil {
		return domain.ProfileVideoBanner{}, fmt.Errorf("save profile video banner: %w", err)
	}
	return domain.ProfileVideoBanner{
		OwnerUserID: ownerUserID,
		Document:    doc,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

// GetProfileVideoBanner resolves the active banner through the canonical
// document registry; the caller still applies profile visibility policy.
func (s *MediaStore) GetProfileVideoBanner(ctx context.Context, ownerUserID int64) (domain.ProfileVideoBanner, bool, error) {
	if ownerUserID <= 0 {
		return domain.ProfileVideoBanner{}, false, nil
	}
	var out domain.ProfileVideoBanner
	var documentID int64
	err := s.db.QueryRow(ctx, `
SELECT owner_user_id, document_id, created_at, updated_at
FROM public.user_profile_video_banners
WHERE owner_user_id = $1`, ownerUserID).Scan(
		&out.OwnerUserID, &documentID, &out.CreatedAt, &out.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ProfileVideoBanner{}, false, nil
	}
	if err != nil {
		return domain.ProfileVideoBanner{}, false, fmt.Errorf("get profile video banner: %w", err)
	}
	doc, found, err := s.GetDocument(ctx, documentID)
	if err != nil {
		return domain.ProfileVideoBanner{}, false, fmt.Errorf("load profile video banner media: %w", err)
	}
	if !found || !doc.IsProfileVideoBanner() || doc.OwnerUserID != ownerUserID {
		return domain.ProfileVideoBanner{}, false, fmt.Errorf("profile video banner %d has invalid or missing document %d", ownerUserID, documentID)
	}
	out.Document = doc
	return out, true, nil
}

// DeleteProfileVideoBanner removes only the profile association. The migration
// trigger releases its media reference and lets the existing retention system
// collect the blob when no other feature references it.
func (s *MediaStore) DeleteProfileVideoBanner(ctx context.Context, ownerUserID int64) error {
	if ownerUserID <= 0 {
		return fmt.Errorf("delete profile video banner: invalid owner")
	}
	_, err := s.db.Exec(ctx, `
DELETE FROM public.user_profile_video_banners
WHERE owner_user_id = $1`, ownerUserID)
	if err != nil {
		return fmt.Errorf("delete profile video banner: %w", err)
	}
	return nil
}
