package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"telesrv/internal/domain"
)

func TestProfileVideoBannerStoreValidatesOwnerAndMedia(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewUserStore(pool)
	media := NewMediaStore(pool)
	suffix := time.Now().UnixNano() % 1_000_000_000

	owner, err := users.Create(ctx, domain.User{
		AccessHash: 1,
		Phone:      fmt.Sprintf("1889%d", suffix),
		FirstName:  "BannerOwner",
	})
	if err != nil {
		t.Fatalf("create banner owner: %v", err)
	}
	other, err := users.Create(ctx, domain.User{
		AccessHash: 2,
		Phone:      fmt.Sprintf("1890%d", suffix),
		FirstName:  "OtherOwner",
	})
	if err != nil {
		t.Fatalf("create other owner: %v", err)
	}

	validID := int64(9_200_000_000_000_000_000 + suffix*3)
	replacementID := validID + 1
	invalidID := validID + 2
	for _, doc := range []domain.Document{
		{ID: validID, AccessHash: 20, MimeType: "video/mp4", Size: 2048, OwnerUserID: owner.ID,
			Attributes: []domain.DocumentAttribute{{Kind: domain.DocAttrVideo, Duration: 25, W: 1280, H: 720}}},
		{ID: replacementID, AccessHash: 22, MimeType: "video/mp4", Size: 3072, OwnerUserID: owner.ID,
			Attributes: []domain.DocumentAttribute{{Kind: domain.DocAttrVideo, Duration: 18, W: 1920, H: 1080}}},
		{ID: invalidID, AccessHash: 21, MimeType: "image/jpeg", Size: 2048, OwnerUserID: owner.ID},
	} {
		if err := media.PutDocument(ctx, doc); err != nil {
			t.Fatalf("put test document %d: %v", doc.ID, err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = ANY($1::bigint[])", []int64{owner.ID, other.ID})
		_, _ = pool.Exec(context.Background(), "DELETE FROM documents WHERE id = ANY($1::bigint[])", []int64{validID, replacementID, invalidID})
	})

	if _, err := media.SetProfileVideoBanner(ctx, other.ID, validID); !errors.Is(err, ErrProfileVideoBannerNotOwned) {
		t.Fatalf("set banner with another user's document: got %v, want ownership error", err)
	}
	if _, err := media.SetProfileVideoBanner(ctx, owner.ID, invalidID); !errors.Is(err, ErrProfileVideoBannerInvalidMedia) {
		t.Fatalf("set banner with non-video document: got %v, want media validation error", err)
	}
	if _, err := media.SetProfileVideoBanner(ctx, owner.ID, validID); err != nil {
		t.Fatalf("set valid profile banner: %v", err)
	}
	got, found, err := media.GetProfileVideoBanner(ctx, owner.ID)
	if err != nil || !found {
		t.Fatalf("get profile banner: found=%v err=%v", found, err)
	}
	if got.Document.ID != validID || got.OwnerUserID != owner.ID {
		t.Fatalf("unexpected profile banner: owner=%d document=%d", got.OwnerUserID, got.Document.ID)
	}
	assertMediaReference := func(documentID int64, wantReference, wantOrphaned bool) {
		t.Helper()
		var hasReference, orphaned bool
		if err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM public.media_references
    WHERE media_kind = 'document' AND media_id = $1
      AND ref_kind = 'profile_video_banner' AND ref_key = $2
), orphaned_at IS NOT NULL
FROM public.documents WHERE id = $1`, documentID, fmt.Sprint(owner.ID)).Scan(&hasReference, &orphaned); err != nil {
			t.Fatalf("inspect media lifecycle for document %d: %v", documentID, err)
		}
		if hasReference != wantReference || orphaned != wantOrphaned {
			t.Fatalf("document %d media lifecycle: reference=%v orphaned=%v, want reference=%v orphaned=%v",
				documentID, hasReference, orphaned, wantReference, wantOrphaned)
		}
	}
	assertMediaReference(validID, true, false)

	if _, err := media.SetProfileVideoBanner(ctx, owner.ID, invalidID); !errors.Is(err, ErrProfileVideoBannerInvalidMedia) {
		t.Fatalf("replace with invalid media: got %v, want media validation error", err)
	}
	afterRejectedReplacement, stillFound, err := media.GetProfileVideoBanner(ctx, owner.ID)
	if err != nil || !stillFound || afterRejectedReplacement.Document.ID != validID {
		t.Fatalf("invalid replacement must preserve previous banner: found=%v document=%d err=%v",
			stillFound, afterRejectedReplacement.Document.ID, err)
	}
	assertMediaReference(validID, true, false)

	if _, err := media.SetProfileVideoBanner(ctx, owner.ID, replacementID); err != nil {
		t.Fatalf("replace profile banner: %v", err)
	}
	replaced, found, err := media.GetProfileVideoBanner(ctx, owner.ID)
	if err != nil || !found || replaced.Document.ID != replacementID {
		t.Fatalf("get replacement banner: found=%v document=%d err=%v", found, replaced.Document.ID, err)
	}
	assertMediaReference(validID, false, true)
	assertMediaReference(replacementID, true, false)

	if err := media.DeleteProfileVideoBanner(ctx, owner.ID); err != nil {
		t.Fatalf("delete profile banner: %v", err)
	}
	if _, found, err := media.GetProfileVideoBanner(ctx, owner.ID); err != nil || found {
		t.Fatalf("banner should be absent after delete: found=%v err=%v", found, err)
	}
	assertMediaReference(replacementID, false, true)
}
