package domain

import (
	"math"
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
	MaxProfileVideoBannerDimension        = 3840
)

// IsProfileVideoBanner reports whether a document has the format and metadata
// required by the current profile-banner playback contract. Reject non-finite
// durations explicitly: comparisons with NaN are false, so range checks alone
// would otherwise accept malformed metadata.
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
			math.IsNaN(attr.Duration) || math.IsInf(attr.Duration, 0) ||
			attr.Duration <= 0 || attr.Duration > MaxProfileVideoBannerDuration ||
			attr.W <= 0 || attr.H <= 0 ||
			attr.W > MaxProfileVideoBannerDimension || attr.H > MaxProfileVideoBannerDimension {
			continue
		}
		return true
	}
	return false
}
