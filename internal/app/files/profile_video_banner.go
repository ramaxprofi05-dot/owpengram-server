package files

import (
	"context"
	"errors"

	"telesrv/internal/domain"
	"telesrv/internal/store"
)

// ProfileVideoBannerService exposes the owner-scoped profile banner association
// through the existing media service. The document bytes remain in the normal
// media/blob pipeline; this service only manages the durable association.
func (s *Service) SetProfileVideoBanner(ctx context.Context, ownerUserID, documentID int64) (domain.ProfileVideoBanner, error) {
	if s == nil || s.media == nil || ownerUserID <= 0 || documentID <= 0 {
		return domain.ProfileVideoBanner{}, domain.ErrInvalidInput
	}
	banner, err := s.media.SetProfileVideoBanner(ctx, ownerUserID, documentID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrProfileVideoBannerDocumentNotFound),
			errors.Is(err, store.ErrProfileVideoBannerNotOwned),
			errors.Is(err, store.ErrProfileVideoBannerInvalidMedia):
			return domain.ProfileVideoBanner{}, domain.ErrInvalidInput
		default:
			return domain.ProfileVideoBanner{}, err
		}
	}
	return banner, nil
}

func (s *Service) GetProfileVideoBanner(ctx context.Context, ownerUserID int64) (domain.ProfileVideoBanner, bool, error) {
	if s == nil || s.media == nil || ownerUserID <= 0 {
		return domain.ProfileVideoBanner{}, false, domain.ErrInvalidInput
	}
	return s.media.GetProfileVideoBanner(ctx, ownerUserID)
}

func (s *Service) DeleteProfileVideoBanner(ctx context.Context, ownerUserID int64) error {
	if s == nil || s.media == nil || ownerUserID <= 0 {
		return domain.ErrInvalidInput
	}
	return s.media.DeleteProfileVideoBanner(ctx, ownerUserID)
}
