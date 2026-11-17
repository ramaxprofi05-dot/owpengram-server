package domain

import (
	"strings"
	"time"
)

// ProfileVideoBanner is the persisted association between a user profile and
// an already-uploaded video document. The media bytes remain in MediaStore's
// normal document/blob pipeline.
type ProfileVideoBanner struct {
	OwnerUserID int64
	Document    Document
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProfileVideoBannerLimits bound media accepted for profile-header playback.
const (
	MaxProfileVideoBannerBytes    int64   = 30 << 20
	MaxProfileVideoBannerDuration float64 = 60
)

// IsProfileVideoBanner reports whether a document has the format and metadata
// required by the current profile-banner playback contract.
func (d Document) IsProfileVideoBanner() bool {
	if d.ID <= 0 || d.AccessHash == 0 || d.OwnerUserID <= 0 ||
		d.Size <= 0 || d.Size > MaxProfileVideoBannerBytes {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(d.MimeType), "video/mp4") {
		return false
	}
	for _, attr := range d.Attributes {
		if attr.Kind != DocAttrVideo || attr.RoundMessage ||
			attr.Duration <= 0 || attr.Duration > MaxProfileVideoBannerDuration ||
			attr.W <= 0 || attr.H <= 0 || attr.W > 3840 || attr.H > 3840 {
			continue
		}
		return true
	}
	return false
}
