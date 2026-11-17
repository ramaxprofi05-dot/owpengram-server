package domain

import "testing"

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
		{"missing owner", func(d *Document) { d.OwnerUserID = 0 }},
		{"wrong MIME", func(d *Document) { d.MimeType = "video/webm" }},
		{"oversized", func(d *Document) { d.Size = MaxProfileVideoBannerBytes + 1 }},
		{"zero duration", func(d *Document) { d.Attributes[0].Duration = 0 }},
		{"long duration", func(d *Document) { d.Attributes[0].Duration = MaxProfileVideoBannerDuration + 1 }},
		{"round video", func(d *Document) { d.Attributes[0].RoundMessage = true }},
		{"missing dimensions", func(d *Document) { d.Attributes[0].W = 0 }},
		{"excessive dimensions", func(d *Document) { d.Attributes[0].W = 3841 }},
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
