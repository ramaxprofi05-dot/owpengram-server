package files

import (
	"context"

	"telesrv/internal/domain"
)

// SetProfileVideoBanner selects an already-uploaded, owner-owned MP4 document.
// The media store and database trigger both enforce the ownership boundary.
func (s *Service) SetProfileVideoBanner(ctx context.Context, ownerUserID, documentID int64) (domain.ProfileVideoBanner, error) {
	if s == nil || s.media == nil || ownerUserID <= 0 || documentID <= 0 {
		return domain.ProfileVideoBanner{}, domain.ErrDocumentInvalid
	}
	return s.media.SetProfileVideoBanner(ctx, ownerUserID, documentID)
}

func (s *Service) GetProfileVideoBanner(ctx context.Context, ownerUserID int64) (domain.ProfileVideoBanner, bool, error) {
	if s == nil || s.media == nil || ownerUserID <= 0 {
		return domain.ProfileVideoBanner{}, false, domain.ErrDocumentInvalid
	}
	return s.media.GetProfileVideoBanner(ctx, ownerUserID)
}

func (s *Service) DeleteProfileVideoBanner(ctx context.Context, ownerUserID int64) error {
	if s == nil || s.media == nil || ownerUserID <= 0 {
		return domain.ErrDocumentInvalid
	}
	return s.media.DeleteProfileVideoBanner(ctx, ownerUserID)
}
