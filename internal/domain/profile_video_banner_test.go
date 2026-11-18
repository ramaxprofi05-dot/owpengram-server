package domain

import (
	"math"
	"testing"
)

func TestDocumentIsProfileVideoBanner(t *testing.T) {
	valid := Document{
		ID: 1, AccessHash: 2, OwnerUserID: 3,
		MimeType: "video/mp4", Size: 1024,
		Attributes: []DocumentAttribute{{
			Kind: DocAttrVideo, Duration: 30, W: 1920, H: 1080,
		}},
	}
	if !valid.IsProfileVideoBanner() {
		t.Fatal("expected valid MP4 video document to be accepted")
	}

	tests := []struct {
		name string
		edit func(*Document)
	}{
		{"missing document id", func(d *Document) { d.ID = 0 }},
		{"missing access hash", func(d *Document) { d.AccessHash = 0 }},
		{"missing owner", func(d *Document) { d.OwnerUserID = 0 }},
		{"zero size", func(d *Document) { d.Size = 0 }},
		{"negative size", func(d *Document) { d.Size = -1 }},
		{"wrong MIME", func(d *Document) { d.MimeType = "video/webm" }},
		{"oversized", func(d *Document) { d.Size = MaxProfileVideoBannerBytes + 1 }},
		{"zero duration", func(d *Document) { d.Attributes[0].Duration = 0 }},
		{"long duration", func(d *Document) { d.Attributes[0].Duration = MaxProfileVideoBannerDuration + 1 }},
		{"NaN duration", func(d *Document) { d.Attributes[0].Duration = math.NaN() }},
		{"positive infinite duration", func(d *Document) { d.Attributes[0].Duration = math.Inf(1) }},
		{"negative infinite duration", func(d *Document) { d.Attributes[0].Duration = math.Inf(-1) }},
		{"round video", func(d *Document) { d.Attributes[0].RoundMessage = true }},
		{"missing dimensions", func(d *Document) { d.Attributes[0].W = 0 }},
		{"excessive width", func(d *Document) { d.Attributes[0].W = MaxProfileVideoBannerDimension + 1 }},
		{"excessive height", func(d *Document) { d.Attributes[0].H = MaxProfileVideoBannerDimension + 1 }},
		{"negative dimensions", func(d *Document) { d.Attributes[0].H = -1 }},
		{"no video attribute", func(d *Document) { d.Attributes = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := valid
			doc.Attributes = append([]DocumentAttribute(nil), valid.Attributes...)
			tt.edit(&doc)
			if doc.IsProfileVideoBanner() {
				t.Fatal("expected invalid banner document to be rejected")
			}
		})
	}
}
