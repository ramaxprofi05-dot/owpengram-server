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

	validID := int64(9_200_000_000_000_000_000 + suffix*2)
	invalidID := validID + 1
	for _, doc := range []domain.Document{
		{ID: validID, AccessHash: 20, MimeType: "video/mp4", Size: 2048, OwnerUserID: owner.ID,
			Attributes: []domain.DocumentAttribute{{Kind: domain.DocAttrVideo, Duration: 25, W: 1280, H: 720}}},
		{ID: invalidID, AccessHash: 21, MimeType: "image/jpeg", Size: 2048, OwnerUserID: owner.ID},
	} {
		if err := media.PutDocument(ctx, doc); err != nil {
			t.Fatalf("put test document %d: %v", doc.ID, err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = ANY($1::bigint[])", []int64{owner.ID, other.ID})
		_, _ = pool.Exec(context.Background(), "DELETE FROM documents WHERE id = ANY($1::bigint[])", []int64{validID, invalidID})
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
	if err := media.DeleteProfileVideoBanner(ctx, owner.ID); err != nil {
		t.Fatalf("delete profile banner: %v", err)
	}
	if _, found, err := media.GetProfileVideoBanner(ctx, owner.ID); err != nil || found {
		t.Fatalf("banner should be absent after delete: found=%v err=%v", found, err)
	}
}
