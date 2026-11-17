-- One public, user-selected video banner per profile. The media itself stays
-- in the existing documents/object-storage pipeline; this table only owns the
-- profile-to-document relationship and lifecycle metadata.
CREATE TABLE public.user_profile_video_banners (
    owner_user_id bigint PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    document_id bigint NOT NULL UNIQUE REFERENCES public.documents(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_profile_video_banners_owner_check CHECK (owner_user_id > 0)
);

CREATE INDEX user_profile_video_banners_document_idx
    ON public.user_profile_video_banners(document_id);

-- Keep the existing storage-retention system aware that this document is
-- still referenced while it is used as a profile banner.
INSERT INTO public.media_references(media_kind, media_id, ref_kind, ref_key)
SELECT 'document', document_id, 'profile_video_banner', owner_user_id::text
FROM public.user_profile_video_banners
ON CONFLICT DO NOTHING;
