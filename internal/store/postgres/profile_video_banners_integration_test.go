package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"telesrv/internal/domain"
)

// TestProfileVideoBannerMediaReferences exercises the migration's database
// triggers: the active banner must keep its document live, replacement must
// release only the old document, and removal must release the final reference.
func TestProfileVideoBannerMediaReferences(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewUserStore(pool)
	media := NewMediaStore(pool)
	suffix := time.Now().UnixNano() % 1_000_000_000

	owner, err := users.Create(ctx, domain.User{
		AccessHash: 1,
		Phone:      fmt.Sprintf("1888%d", suffix),
		FirstName:  "BannerOwner",
	})
	if err != nil {
		t.Fatalf("create banner owner: %v", err)
	}
	otherOwner, err := users.Create(ctx, domain.User{
		AccessHash: 2,
		Phone:      fmt.Sprintf("1999%d", suffix),
		FirstName:  "OtherBannerOwner",
	})
	if err != nil {
		t.Fatalf("create second banner owner: %v", err)
	}

	docA := int64(9_100_000_000_000_000_000 + suffix*2)
	docB := docA + 1
	for i, id := range []int64{docA, docB} {
		if err := media.PutDocument(ctx, domain.Document{
			ID:          id,
			AccessHash:  int64(i + 10),
			MimeType:    "video/mp4",
			Size:        1024,
			OwnerUserID: owner.ID,
		}); err != nil {
			t.Fatalf("put banner document %d: %v", id, err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = ANY($1::bigint[])", []int64{owner.ID, otherOwner.ID})
		_, _ = pool.Exec(context.Background(), "DELETE FROM documents WHERE id = ANY($1::bigint[])", []int64{docA, docB})
	})

	if _, err := pool.Exec(ctx, `
INSERT INTO public.user_profile_video_banners(owner_user_id, document_id)
VALUES ($1, $2)`, owner.ID, docA); err != nil {
		t.Fatalf("insert initial profile banner: %v", err)
	}
	assertProfileBannerReference(t, ctx, pool, owner.ID, docA, true, false)

	// The database guard must reject a document owned by another account.
	if _, err := pool.Exec(ctx, `
INSERT INTO public.user_profile_video_banners(owner_user_id, document_id)
VALUES ($1, $2)`, otherOwner.ID, docA); err == nil {
		t.Fatal("expected cross-account profile banner document to be rejected")
	}

	// A document cannot be deleted while the profile still references it.
	if _, err := pool.Exec(ctx, "DELETE FROM public.documents WHERE id = $1", docA); err == nil {
		t.Fatal("expected document deletion to be blocked while banner is active")
	}

	if _, err := pool.Exec(ctx, `
UPDATE public.user_profile_video_banners
SET document_id = $2
WHERE owner_user_id = $1`, owner.ID, docB); err != nil {
		t.Fatalf("replace profile banner: %v", err)
	}
	assertProfileBannerReference(t, ctx, pool, owner.ID, docA, false, true)
	assertProfileBannerReference(t, ctx, pool, owner.ID, docB, true, false)

	if _, err := pool.Exec(ctx, `
DELETE FROM public.user_profile_video_banners
WHERE owner_user_id = $1`, owner.ID); err != nil {
		t.Fatalf("remove profile banner: %v", err)
	}
	assertProfileBannerReference(t, ctx, pool, owner.ID, docB, false, true)
}

func assertProfileBannerReference(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	ownerID, documentID int64,
	wantReference, wantOrphaned bool,
) {
	t.Helper()
	var references int
	err := pool.QueryRow(ctx, `
SELECT count(*)::int
FROM public.media_references
WHERE media_kind = 'document'
  AND media_id = $1
  AND ref_kind = 'profile_video_banner'
  AND ref_key = $2::text`, documentID, ownerID).Scan(&references)
	if err != nil {
		t.Fatalf("query banner media reference: %v", err)
	}
	if got := references == 1; got != wantReference {
		t.Fatalf("document %d reference present=%v, want %v (count=%d)", documentID, got, wantReference, references)
	}

	var orphaned bool
	if err := pool.QueryRow(ctx, `
SELECT orphaned_at IS NOT NULL
FROM public.documents
WHERE id = $1`, documentID).Scan(&orphaned); err != nil {
		t.Fatalf("query banner document orphan state: %v", err)
	}
	if orphaned != wantOrphaned {
		t.Fatalf("document %d orphaned=%v, want %v", documentID, orphaned, wantOrphaned)
	}
}
